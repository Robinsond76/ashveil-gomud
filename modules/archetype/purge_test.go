package archetype

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserPurgedDropsArchetypeState (Phase 32b): a purged user's choice,
// toggles, owed kit, and trap senses go, and are saved; others' stay. A
// second purge saves nothing.
func TestUserPurgedDropsArchetypeState(t *testing.T) {
	store := &fakeStore{}
	m := newModule()
	m.store = store
	m.registry.Players[7], m.registry.Players[8] = "rogue", "warrior"
	m.registry.Autoskill[7] = map[string]bool{"traps": false}
	m.registry.Kits[7], m.registry.Kits[8] = "rogue", "warrior"
	m.pickSensed = map[string]struct{}{"7|door": {}, "8|door": {}, "70|door": {}}

	m.onUserPurged(events.UserPurged{UserId: 7})
	require.NotNil(t, store.saved)
	assert.Equal(t, map[int]string{8: "warrior"}, store.saved.Players)
	assert.Equal(t, map[int]string{8: "warrior"}, store.saved.Kits)
	assert.Empty(t, store.saved.Autoskill)
	assert.Equal(t, map[string]struct{}{"8|door": {}, "70|door": {}}, m.pickSensed)

	saves := store.saves
	m.onUserPurged(events.UserPurged{UserId: 7})
	assert.Equal(t, saves, store.saves, "nothing left to save")
}
