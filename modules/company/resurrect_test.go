package company

import (
	"errors"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResurrectCostsLevelAndSpawns: one level, the kept gear, the old
// cell, one save, then one live mob in the room, attached.
func TestResurrectCostsLevelAndSpawns(t *testing.T) {
	module, _, runtime, _ := newDeathModule(t)
	killOne(module)
	saves := module.store.(*fakeStore).saveCalls
	spawns := runtime.spawnCalls

	result, err := module.ResurrectCompanion(7, "#1", 2007)
	require.NoError(t, err)
	assert.Equal(t, domain.ResurrectionResult{ID: 1, Name: "#1", Level: 4, Spawned: true}, result)
	c := companion(t, module, 1)
	assert.False(t, c.Dead())
	assert.Equal(t, 4, c.State.Level)
	assert.Zero(t, c.State.Experience)
	assert.Equal(t, 1, c.State.Equipment.Head.ItemId, "the kept helmet returns")
	record, _ := module.registry.Get(7)
	r, col, placed := record.Formation.Find(domain.CompanionMemberKey(1))
	assert.True(t, placed)
	assert.Equal(t, [2]int{0, 1}, [2]int{r, col})
	assert.Equal(t, saves+1, module.store.(*fakeStore).saveCalls)
	assert.False(t, module.store.(*fakeStore).saved.Companies[7].Companions[0].Dead(), "saved alive")
	assert.Equal(t, spawns+1, runtime.spawnCalls)
	assert.Equal(t, 2007, runtime.spawnedRoomID)
	assert.Equal(t, 4, runtime.spawnedStates[len(runtime.spawnedStates)-1].Level)
	instanceID, tracked := module.instance(7, 1)
	assert.True(t, tracked)
	assert.Equal(t, 201, instanceID)
	assert.Empty(t, module.DeadCompanions(7))

	// Raised once: a second attempt finds no one dead.
	_, err = module.ResurrectCompanion(7, "#1", 2007)
	assert.ErrorIs(t, err, domain.ErrNotDead)
	assert.Equal(t, spawns+1, runtime.spawnCalls)
}

// TestResurrectLevelOneFloor: a level 1 companion stays level 1.
func TestResurrectLevelOneFloor(t *testing.T) {
	module, _, _, _ := newDeathModule(t)
	module.onMobDeath(events.MobDeath{InstanceId: 102, Level: 1})
	result, err := module.ResurrectCompanion(7, "2", 2007)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Level)
	assert.Equal(t, 1, companion(t, module, 2).State.Level)
}

// TestResurrectSaveFailureRollsBack: nothing changes and nothing spawns.
func TestResurrectSaveFailureRollsBack(t *testing.T) {
	module, _, runtime, _ := newDeathModule(t)
	killOne(module)
	before, _ := module.registry.Get(7)
	spawns := runtime.spawnCalls
	module.store.(*fakeStore).saveErr = errors.New("disk full")
	_, err := module.ResurrectCompanion(7, "#1", 2007)
	require.Error(t, err)
	after, _ := module.registry.Get(7)
	assert.Equal(t, before, after)
	assert.Equal(t, spawns, runtime.spawnCalls)
}

// TestResurrectSpawnFailureAwaitsRestoration: alive in the record, and
// restored with the leader later.
func TestResurrectSpawnFailureAwaitsRestoration(t *testing.T) {
	module, _, runtime, _ := newDeathModule(t)
	killOne(module)
	runtime.spawnErr = errors.New("no room")
	result, err := module.ResurrectCompanion(7, "#1", 2007)
	require.NoError(t, err)
	assert.False(t, result.Spawned)
	assert.False(t, companion(t, module, 1).Dead())
	_, tracked := module.instance(7, 1)
	assert.False(t, tracked)
	runtime.spawnErr = nil
	require.NoError(t, module.restoreForLeader(7, 12))
	_, tracked = module.instance(7, 1)
	assert.True(t, tracked)
}

// TestResurrectExpiredIsLost: a companion whose time ran out is lost, not
// raised, even if the round hasn't charged it yet.
func TestResurrectExpiredIsLost(t *testing.T) {
	module, _, runtime, clock := newDeathModule(t)
	killOne(module)
	record, _ := module.registry.Get(7)
	record.Companions[0].Death.Remaining = 3
	module.registry.Put(record)
	clock.advance(5_000_000_000) // 5 seconds, not yet charged
	spawns := runtime.spawnCalls
	_, err := module.ResurrectCompanion(7, "#1", 2007)
	assert.ErrorIs(t, err, domain.ErrCompanionLost)
	record, _ = module.registry.Get(7)
	assert.Len(t, record.Lost, 1)
	assert.Equal(t, spawns, runtime.spawnCalls)
	_, err = module.ResurrectCompanion(7, "#1", 2007)
	assert.ErrorIs(t, err, domain.ErrCompanionLost, "still lost")
}

// TestResurrectRefusesLivingAndUnknown.
func TestResurrectRefusesLivingAndUnknown(t *testing.T) {
	module, _, _, _ := newDeathModule(t)
	_, err := module.ResurrectCompanion(7, "#2", 2007)
	assert.ErrorIs(t, err, domain.ErrNotDead)
	_, err = module.ResurrectCompanion(7, "#9", 2007)
	assert.ErrorIs(t, err, domain.ErrUnknownMember)
	_, err = module.ResurrectCompanion(8, "#1", 2007)
	assert.ErrorIs(t, err, domain.ErrUnknownMember)
}

// TestDeadCompanionsListing.
func TestDeadCompanionsListing(t *testing.T) {
	module, _, _, _ := newDeathModule(t)
	killOne(module)
	assert.Equal(t, []domain.DeadCompanionView{{ID: 1, Name: "#1", Level: 5, Remaining: 3600}}, module.DeadCompanions(7))
	assert.Empty(t, module.DeadCompanions(8))
}

