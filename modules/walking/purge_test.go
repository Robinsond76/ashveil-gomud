package walking

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
)

// TestUserPurgedDropsWalkingFatigue (Phase 32b).
func TestUserPurgedDropsWalkingFatigue(t *testing.T) {
	store := &fakeStore{}
	m := newModule()
	m.store = store
	m.registry = newRegistry()
	m.registry.Carry[7] = map[string]int{"leader": 3}
	m.registry.Carry[8] = map[string]int{"leader": 1}
	m.onUserPurged(events.UserPurged{UserId: 7})
	assert.Equal(t, 1, store.saveCalls)
	assert.Equal(t, map[int]map[string]int{8: {"leader": 1}}, store.saved.Carry)
	m.onUserPurged(events.UserPurged{UserId: 7})
	assert.Equal(t, 1, store.saveCalls, "nothing left to save")
}
