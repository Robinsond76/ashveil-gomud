package company

import (
	"encoding/json"
	"errors"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/modules/gmcp"
	"strings"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestFormationMigrationDurableRetryAndManualClear(t *testing.T) {
	m := newTestModule(*domain.NewRegistry(), &fakeRuntime{})
	old := domain.Record{LeaderUserID: 7, Companions: []domain.Companion{{ID: 1}, {ID: 2}, {ID: 3, Death: &domain.CompanionDeath{OpID: "dead"}}}}
	require.NoError(t, old.Formation.Place(domain.CompanionMemberKey(1), 2, 2))
	m.registry.Put(old)
	before := m.registry.Clone()
	store := m.store.(*fakeStore)
	store.saveErr = errors.New("disk full")
	require.Error(t, m.prepareFormation(7))
	assert.Equal(t, before, m.registry.Clone(), "failed migration rolls back its version and placements")
	store.saveErr = nil
	require.NoError(t, m.prepareFormation(7))
	saved, _ := store.saved.Get(7)
	assert.Equal(t, domain.LeaderMemberKey, saved.Formation.At(1, 1))
	assert.Equal(t, domain.CompanionMemberKey(1), saved.Formation.At(2, 2), "retain old arrangement")
	assert.Equal(t, domain.CompanionMemberKey(2), saved.Formation.At(0, 1))
	_, _, placed := saved.Formation.Find(domain.CompanionMemberKey(3))
	assert.False(t, placed, "dead members stay out")
	require.NoError(t, m.registry.ClearMember(7, domain.CompanionMemberKey(2)))
	require.NoError(t, m.save())
	data, err := yaml.Marshal(store.saved)
	require.NoError(t, err)
	restored := domain.NewRegistry()
	require.NoError(t, decodeCompanies(data, restored))
	m.registry = *restored
	calls := store.saveCalls
	require.NoError(t, m.prepareFormation(7))
	got, _ := m.registry.Get(7)
	_, _, placed = got.Formation.Find(domain.CompanionMemberKey(2))
	assert.False(t, placed, "later deliberate clears survive reload and refresh")
	assert.Equal(t, calls, store.saveCalls)
}

func TestSoloFormationMigrationAndFailedFirstSave(t *testing.T) {
	m := newTestModule(*domain.NewRegistry(), &fakeRuntime{})
	store := m.store.(*fakeStore)
	store.saveErr = errors.New("disk full")
	require.Error(t, m.prepareFormation(7))
	_, exists := m.registry.Get(7)
	assert.False(t, exists)
	store.saveErr = nil
	require.NoError(t, m.prepareFormation(7))
	got, _ := store.saved.Get(7)
	assert.Equal(t, domain.LeaderMemberKey, got.Formation.At(1, 1))
	assert.ErrorIs(t, m.registry.ClearMember(7, domain.LeaderMemberKey), domain.ErrSoloFormation)
	assert.ErrorIs(t, m.registry.PlaceMember(7, domain.LeaderMemberKey, 0, 0), domain.ErrSoloFormation)
}

func TestEnlistPlacementAndFailedSave(t *testing.T) {
	useDataDir(t, t.TempDir())
	m := newTestModule(*domain.NewRegistry(), &fakeRuntime{nextInstanceID: 101})
	store := m.store.(*fakeStore)
	c, err := m.enlist(7, 12, 58, map[int]struct{}{58: {}}, false, nil)
	require.NoError(t, err)
	got, _ := store.saved.Get(7)
	assert.Equal(t, domain.LeaderMemberKey, got.Formation.At(1, 1))
	assert.Equal(t, domain.CompanionMemberKey(c.ID), got.Formation.At(0, 1))
	require.NoError(t, m.registry.PlaceMember(7, domain.LeaderMemberKey, 2, 2))
	require.NoError(t, m.save())
	before, _ := m.registry.Get(7)
	store.failSaveOnCall = store.saveCalls + 1
	_, err = m.enlist(7, 12, 58, map[int]struct{}{58: {}}, false, nil)
	require.Error(t, err)
	got, _ = m.registry.Get(7)
	assert.Equal(t, before.Formation, got.Formation)
	assert.Equal(t, before.FormationVersion, got.FormationVersion)
	assert.Len(t, got.Companions, 1)
	assert.Equal(t, before.Formation, store.saved.Companies[7].Formation)
}

func TestFormationDefaultsThroughLoginAndCombatFeed(t *testing.T) {
	b := newBrawl(t)
	old, _ := module.registry.Get(7)
	old.FormationVersion = 0
	module.registry.Put(old)
	require.NoError(t, module.save())
	gmcp.AcceptGMCPForTest(users.GetConnectionId(7))
	var snapshot map[string]any
	id := events.RegisterListener(gmcp.GMCPOut{}, func(e events.Event) events.ListenerReturn {
		out := e.(gmcp.GMCPOut)
		if out.UserId == 7 && out.Module == "Company" {
			if raw, ok := out.Payload.([]byte); ok {
				require.NoError(t, json.Unmarshal(raw, &snapshot))
			}
		}
		return events.Cancel
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(gmcp.GMCPOut{}, id) })
	// The real spawn listener is shared by login and copyover.
	events.AddToQueue(events.PlayerSpawn{UserId: 7, RoomId: b.aria.Character.RoomId})
	events.ProcessEvents()
	b.refresh(7)
	got, _ := module.registry.Get(7)
	loaded := domain.NewRegistry()
	require.NoError(t, module.store.Load(loaded))
	assert.Equal(t, got.Formation, loaded.Companies[7].Formation)
	assert.Equal(t, 1, loaded.Companies[7].FormationVersion)
	require.NotNil(t, snapshot)
	for _, entry := range snapshot["members"].([]any) {
		member := entry.(map[string]any)
		row, col, placed := got.Formation.Find(domain.MemberKey(member["key"].(string)))
		require.True(t, placed)
		cell := member["cell"].(map[string]any)
		assert.Equal(t, float64(row), cell["row"])
		assert.Equal(t, float64(col), cell["col"])
	}
	assert.NotContains(t, b.cmd("formation", ""), "Unplaced:")
	assert.Contains(t, b.cmd("formation", ""), "Aria")
	// Combat's target upkeep must turn the auto-centered leader away from a rear foe.
	b.toughen()
	b.aimAt("bandit slinger")
	rearTarget := b.aria.Character.Aggro.MobInstanceId
	b.fight()
	if aggro := b.aria.Character.Aggro; aggro != nil {
		assert.NotEqual(t, rearTarget, aggro.MobInstanceId, "unreachable rear target is cleared; upkeep may select another foe")
	}
	// End combat before deliberately clearing a multi-member position.
	b.aria.Character.EndAggro()
	for i := 1; i <= 4; i++ {
		b.companion(i).Character.EndAggro()
	}
	// Avoid the battle command guard: the domain operation is saved like a command.
	require.NoError(t, module.registry.ClearMember(7, domain.CompanionMemberKey(1)))
	require.NoError(t, module.save())
	events.AddToQueue(events.PlayerSpawn{UserId: 7, RoomId: b.aria.Character.RoomId})
	events.ProcessEvents()
	got, _ = module.registry.Get(7)
	_, _, placed := got.Formation.Find(domain.CompanionMemberKey(1))
	assert.False(t, placed)
	assert.True(t, strings.Contains(b.cmd("formation", ""), "Unplaced:"))
}
