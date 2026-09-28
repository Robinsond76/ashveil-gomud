package exposure

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
)

// TestUserPurgedDropsExposure (Phase 32b).
func TestUserPurgedDropsExposure(t *testing.T) {
	store := &fakeStore{}
	m := &ExposureModule{store: store, registry: newRegistry()}
	m.registry.Exposure[7] = map[string]int{"leader": 3}
	m.registry.Exposure[8] = map[string]int{"leader": 1}
	m.onUserPurged(events.UserPurged{UserId: 7})
	assert.Equal(t, 1, store.saveCalls)
	assert.Equal(t, map[int]map[string]int{8: {"leader": 1}}, store.saved.Exposure)
	m.onUserPurged(events.UserPurged{UserId: 7})
	assert.Equal(t, 1, store.saveCalls, "nothing left to save")
}
