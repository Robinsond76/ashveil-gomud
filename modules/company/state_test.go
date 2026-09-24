package company

import (
	"errors"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func geared(level int, ids ...int) domain.MemberState {
	s := domain.MemberState{Level: level}
	if len(ids) == 0 {
		return s
	}
	s.Equipment.Weapon = items.Item{ItemId: ids[0]}
	for _, id := range ids[1:] {
		s.Items = append(s.Items, items.Item{ItemId: id})
	}
	return s
}

func companionState(t *testing.T, m *CompanyModule, leader, id int) *domain.MemberState {
	t.Helper()
	record, ok := m.registry.Get(leader)
	require.True(t, ok)
	for _, c := range record.Companions {
		if c.ID == id {
			return c.State
		}
	}
	t.Fatalf("no companion #%d", id)
	return nil
}

func legacyModule(runtime *fakeRuntime) *CompanyModule {
	return newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}}, NextCompanionID: 2},
	}}, runtime)
}

func TestSummonRecordsTemplateState(t *testing.T) {
	useFakeLifecycle(t, &fakeLifecycle{})
	tpl := geared(2, 10002, 30004)
	runtime := &fakeRuntime{nextInstanceID: 101, templateState: &tpl}
	m := newTestModule(*domain.NewRegistry(), runtime)
	store := m.store.(*fakeStore)

	_, err := m.summon(7, 12, "58")
	require.NoError(t, err)
	assert.Nil(t, runtime.spawnedStates[0], "a new recruit spawns from its template")
	saved, ok := store.saved.Get(7)
	require.True(t, ok)
	require.NotNil(t, saved.Companions[0].State, "the minted gear is in the summon's own save")
	assert.Equal(t, tpl, *saved.Companions[0].State)
}

func TestSummonSaveFailureLeavesNoState(t *testing.T) {
	useFakeLifecycle(t, &fakeLifecycle{})
	tpl := geared(2, 10002)
	runtime := &fakeRuntime{nextInstanceID: 101, templateState: &tpl}
	m := newTestModule(*domain.NewRegistry(), runtime)
	m.store.(*fakeStore).saveErr = errors.New("disk full")

	_, err := m.summon(7, 12, "58")
	require.Error(t, err)
	assert.False(t, runtime.live[101], "the mob and its minted gear are destroyed")
	record, _ := m.registry.Get(7)
	assert.Empty(t, record.Companions)
}

func TestRestoreSpawnsWithSavedState(t *testing.T) {
	runtime := &fakeRuntime{nextInstanceID: 101}
	s := geared(5, 10007, 30015)
	m := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58, State: &s}}},
	}}, runtime)
	require.NoError(t, m.restoreForLeader(7, 12))
	require.Len(t, runtime.spawnedStates, 1)
	require.NotNil(t, runtime.spawnedStates[0])
	assert.Equal(t, s, *runtime.spawnedStates[0])
	assert.Zero(t, m.store.(*fakeStore).saveCalls, "nothing to upgrade, nothing saved")
}

func TestLegacyCompanionUpgradedBeforeSpawn(t *testing.T) {
	tpl := geared(1, 10001)
	runtime := &fakeRuntime{nextInstanceID: 101, templateState: &tpl}
	m := legacyModule(runtime)
	store := m.store.(*fakeStore)

	require.NoError(t, m.restoreForLeader(7, 12))
	saved, _ := store.saved.Get(7)
	require.NotNil(t, saved.Companions[0].State, "the upgrade is durable")
	assert.Equal(t, tpl, *saved.Companions[0].State)
	require.NotNil(t, runtime.spawnedStates[0])
	assert.Equal(t, tpl, *runtime.spawnedStates[0], "spawned from the saved upgrade, not the template")

	// A second restore uses the stored state and doesn't save again.
	m.clearInstance(7, 1)
	before := store.saveCalls
	require.NoError(t, m.restoreForLeader(7, 12))
	assert.Equal(t, before, store.saveCalls)
}

func TestLegacyUpgradeSaveFailureDoesNotSpawn(t *testing.T) {
	runtime := &fakeRuntime{nextInstanceID: 101}
	m := legacyModule(runtime)
	m.store.(*fakeStore).saveErr = errors.New("disk full")

	assert.Error(t, m.restoreForLeader(7, 12))
	assert.Zero(t, runtime.spawnCalls, "no template gear without a durable record")
	assert.Nil(t, companionState(t, m, 7, 1), "the old record is kept for retry")

	m.store.(*fakeStore).saveErr = nil
	require.NoError(t, m.restoreForLeader(7, 12))
	assert.Equal(t, 1, runtime.spawnCalls, "retried on the next spawn")
}

func TestLegacyWithoutTemplateSpawnsAsBefore(t *testing.T) {
	runtime := &fakeRuntime{nextInstanceID: 101, noTemplateState: true, spawnErr: errors.New("unknown template")}
	m := legacyModule(runtime)
	assert.Error(t, m.restoreForLeader(7, 12))
	assert.Nil(t, companionState(t, m, 7, 1))
	_, exists := m.registry.Get(7)
	assert.True(t, exists, "the record is kept")
}

// liveModule is a module with companion #1 restored as instance 101.
func liveModule(t *testing.T) (*CompanyModule, *fakeRuntime, *fakeStore) {
	t.Helper()
	s := geared(3, 10002)
	runtime := &fakeRuntime{nextInstanceID: 101}
	m := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58, State: &s}}},
	}}, runtime)
	require.NoError(t, m.restoreForLeader(7, 12))
	return m, runtime, m.store.(*fakeStore)
}

// Review finding 1: a gear change is recorded in memory only. The store is
// written by the same saves that write user and room files, so a crash
// can't leave an item in both the company file and a player's file.
func TestItemOwnershipRefreshesInMemoryOnly(t *testing.T) {
	m, runtime, store := liveModule(t)
	runtime.liveState[101] = geared(3, 10002, 30004)
	before := store.saveCalls

	m.onItemOwnership(events.ItemOwnership{MobInstanceId: 101, Item: items.Item{ItemId: 30004}, Gained: true})
	assert.Equal(t, before, store.saveCalls, "no store write on a gear change")
	assert.Equal(t, geared(3, 10002, 30004), *companionState(t, m, 7, 1))

	runtime.liveState[101] = geared(3)
	m.onItemOwnership(events.ItemOwnership{MobInstanceId: 999, Gained: true})
	m.onItemOwnership(events.ItemOwnership{UserId: 7, Gained: true})
	assert.Equal(t, geared(3, 10002, 30004), *companionState(t, m, 7, 1), "other mobs and players are ignored")
}

// Review finding 6: a companion befriended away by another player keeps
// its gear and isn't destroyed; the record's gear is cleared.
func TestCompanionCharmedByOtherIsLost(t *testing.T) {
	m, runtime, store := liveModule(t)
	runtime.stolen = map[int]bool{101: true}

	m.onPlayerDespawn(events.PlayerDespawn{UserId: 7})
	assert.True(t, runtime.live[101], "another player's mob is never destroyed")
	_, tracked := m.instance(7, 1)
	assert.False(t, tracked)
	saved, _ := store.saved.Get(7)
	state := saved.Companions[0].State
	assert.Zero(t, state.Equipment.Weapon.ItemId, "the gear went with the mob")
	assert.Equal(t, 3, state.Level)
}

func TestRestoreKeepsCompanionCharmedByOther(t *testing.T) {
	m, runtime, _ := liveModule(t)
	runtime.stolen = map[int]bool{101: true}
	runtime.nextInstanceID = 102
	require.NoError(t, m.restoreForLeader(7, 12))
	assert.True(t, runtime.live[101], "the other player's mob stays")
	last := runtime.spawnedStates[len(runtime.spawnedStates)-1]
	assert.Zero(t, last.Equipment.Weapon.ItemId, "the replacement doesn't duplicate its gear")
}

func TestGoldIsDurableAndClearedOnDeath(t *testing.T) {
	m, runtime, _ := liveModule(t)
	s := geared(3, 10002)
	s.Gold = 50
	runtime.liveState[101] = s
	m.refreshAll()
	assert.Equal(t, 50, companionState(t, m, 7, 1).Gold)
	m.onMobDeath(events.MobDeath{InstanceId: 101, Level: 3})
	assert.Zero(t, companionState(t, m, 7, 1).Gold, "gold drops with the body")
}

func TestOnSaveRefreshesLiveCompanions(t *testing.T) {
	m, runtime, _ := liveModule(t)
	runtime.liveState[101] = geared(4, 10007)
	m.refreshAll()
	assert.Equal(t, geared(4, 10007), *companionState(t, m, 7, 1))

	// A dead (untracked-but-stale) instance isn't read.
	runtime.live[101] = false
	runtime.liveState[101] = geared(9)
	m.refreshAll()
	assert.Equal(t, 4, companionState(t, m, 7, 1).Level)
}

func TestLeaderDespawnSnapshotsAndRemovesCompanions(t *testing.T) {
	m, runtime, store := liveModule(t)
	runtime.liveState[101] = geared(3, 10002, 30015)

	m.onPlayerDespawn(events.PlayerDespawn{UserId: 7})
	saved, _ := store.saved.Get(7)
	assert.Equal(t, geared(3, 10002, 30015), *saved.Companions[0].State)
	assert.False(t, runtime.live[101], "the live mob can't linger to be looted")
	_, tracked := m.instance(7, 1)
	assert.False(t, tracked)

	// The next login restores the recorded gear.
	runtime.nextInstanceID = 102
	require.NoError(t, m.restoreForLeader(7, 12))
	assert.Equal(t, geared(3, 10002, 30015), *runtime.spawnedStates[len(runtime.spawnedStates)-1])
}

func TestLeaderDespawnSaveFailureStillRemoves(t *testing.T) {
	m, runtime, store := liveModule(t)
	runtime.liveState[101] = geared(3, 10002, 30015)
	store.saveErr = errors.New("disk full")
	m.onPlayerDespawn(events.PlayerDespawn{UserId: 7})
	assert.False(t, runtime.live[101])
	assert.Equal(t, geared(3, 10002, 30015), *companionState(t, m, 7, 1), "kept in memory for the next autosave")
}

// TestCompanionDeathKeepsOnlyWhatTheBodyKept: a companion's death keeps its
// level and only the gear the engine's drop rules left on the body (Phase
// 25b); it is dead, so the next restore doesn't spawn it.
func TestCompanionDeathKeepsOnlyWhatTheBodyKept(t *testing.T) {
	m, runtime, store := liveModule(t)
	m.onMobDeath(events.MobDeath{InstanceId: 101, Level: 4})
	_, tracked := m.instance(7, 1)
	assert.False(t, tracked)
	saved, _ := store.saved.Get(7)
	require.True(t, saved.Companions[0].Dead())
	state := saved.Companions[0].State
	require.NotNil(t, state)
	assert.Equal(t, 4, state.Level)
	assert.Zero(t, state.Equipment.Weapon.ItemId, "gear dropped with the body isn't restored")
	assert.Empty(t, state.Items)

	spawns := runtime.spawnCalls
	require.NoError(t, m.restoreForLeader(7, 12))
	assert.Equal(t, spawns, runtime.spawnCalls, "the dead aren't restored")

	// A body that kept its sword keeps it on the record.
	m2, _, store2 := liveModule(t)
	m2.onMobDeath(events.MobDeath{InstanceId: 101, Level: 3, KeptWorn: map[items.ItemType]items.Item{items.Weapon: {ItemId: 10002}}})
	saved, _ = store2.saved.Get(7)
	assert.Equal(t, 10002, saved.Companions[0].State.Equipment.Weapon.ItemId)
}

func TestDismissDropsState(t *testing.T) {
	useFakeLifecycle(t, &fakeLifecycle{})
	m, runtime, store := liveModule(t)
	_, err := m.dismiss(7, "#1")
	require.NoError(t, err)
	assert.False(t, runtime.live[101], "the companion leaves with its gear")
	record, _ := store.saved.Get(7)
	assert.Empty(t, record.Companions)
}

func TestStatusShowsLevelAndGearView(t *testing.T) {
	m, runtime, _ := liveModule(t)
	assert.Contains(t, m.status(7), "level 3")

	runtime.liveState[101] = geared(3, 10002, 30004)
	view := m.gearView(7, "#1")
	assert.Contains(t, view, "level 3")
	assert.Contains(t, view, "Weapon:")
	assert.Contains(t, view, "Carrying:")
	assert.Equal(t, geared(3, 10002, 30004), *companionState(t, m, 7, 1), "the view reads the live mob")
	assert.Contains(t, m.gearView(7, "#9"), "No companion matches")

	legacy := legacyModule(&fakeRuntime{})
	assert.Contains(t, legacy.gearView(7, "#1"), "recorded the next time")
	assert.Contains(t, legacy.status(7), "level ?")
}
