package strategy

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	domain "github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestSetTacticsSavesAndReloads(t *testing.T) {
	m, store, _, _ := testModule(t)
	assert.Equal(t, domain.Tactics{}, m.StoredTactics(4401))
	require.NoError(t, m.SetTactics(4401, domain.Tactics{Focus: domain.Leader, Healing: 70, Patch: 65}))
	assert.Equal(t, domain.Tactics{Focus: domain.Leader, Healing: 70, Patch: 65}, store.saved.Tactics[4401])

	fresh := newModule()
	fresh.store = store
	fresh.load()
	assert.Equal(t, domain.Tactics{Focus: domain.Leader, Healing: 70, Patch: 65}, fresh.StoredTactics(4401), "the patch threshold survives a reload")

	// Back to the defaults: nothing stored. A focus of "none" is a choice
	// (Phase 35d: a blank focus takes the level's default), so it is kept.
	require.NoError(t, m.SetTactics(4401, domain.Tactics{Healing: 50, Patch: 80}))
	_, kept := store.saved.Tactics[4401]
	assert.False(t, kept)
	require.NoError(t, m.SetTactics(4401, domain.Tactics{Focus: domain.NoFocus}))
	assert.Equal(t, domain.NoFocus, store.saved.Tactics[4401].Focus)
}

func TestSetTacticsRollsBack(t *testing.T) {
	m, store, _, _ := testModule(t)
	require.NoError(t, m.SetTactics(4401, domain.Tactics{Focus: domain.Wounded}))
	store.failNow = true
	assert.Error(t, m.SetTactics(4401, domain.Tactics{Focus: domain.Leader}))
	assert.Equal(t, domain.Wounded, m.StoredTactics(4401).Focus)
	store.failNow = false

	// A fresh player's failed first save leaves nothing behind.
	store.failNow = true
	assert.Error(t, m.SetTactics(55, domain.Tactics{Healing: 30}))
	assert.Equal(t, domain.Tactics{}, m.StoredTactics(55))
}

func TestSetTacticsBlockedByLoadFailure(t *testing.T) {
	m, store, _, _ := testModule(t)
	store.loadErr = assert.AnError
	m.load()
	assert.Error(t, m.SetTactics(4401, domain.Tactics{Focus: domain.Leader}))
}

func TestDecodeRegistryDropsBadTactics(t *testing.T) {
	raw := map[string]any{"tactics": map[int]domain.Tactics{
		7:  {Focus: "strongest", Healing: 30},
		8:  {Focus: "assist"},
		9:  {Healing: 45},
		10: {Focus: "none"},
		11: {Patch: 30},
		12: {Patch: 90},
		-1: {Focus: "leader"},
	}}
	data, err := yaml.Marshal(raw)
	require.NoError(t, err)
	r := NewRegistry()
	require.NoError(t, decodeRegistry(data, r))
	assert.Equal(t, map[int]domain.Tactics{7: {Focus: domain.Strongest, Healing: 30}, 10: {Focus: domain.NoFocus}, 12: {Patch: 90}}, r.Tactics, "a patch under 50 drops the entry")
}

func TestUserPurgedForgetsTactics(t *testing.T) {
	m, store, _, _ := testModule(t)
	require.NoError(t, m.SetTactics(4401, domain.Tactics{Focus: domain.Leader}))
	id := events.RegisterListener(events.UserPurged{}, m.onUserPurged)
	t.Cleanup(func() { events.UnregisterListener(events.UserPurged{}, id) })
	events.AddToQueue(events.UserPurged{UserId: 4401})
	events.ProcessEvents()
	assert.Equal(t, domain.Tactics{}, m.StoredTactics(4401))
	_, kept := store.saved.Tactics[4401]
	assert.False(t, kept)
}

func TestListNamesTheCompanyFocus(t *testing.T) {
	m, _, u, _ := testModule(t)
	domain.SetTacticsProvider(m)
	t.Cleanup(func() { domain.SetTacticsProvider(module) })
	assert.NotContains(t, run(m, u, ""), "Your company's focus")
	require.NoError(t, m.SetTactics(4401, domain.Tactics{Focus: domain.Leader}))
	assert.Contains(t, run(m, u, ""), "Your company's focus is leader (company tactics): instead of these rules, everyone goes for their leader, the toughest of them.")
}
