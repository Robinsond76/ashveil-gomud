package mount

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	domain "github.com/GoMudEngine/GoMud/internal/mount"
	"github.com/stretchr/testify/assert"
)

// TestUserPurgedDropsTheHerd (Phase 32b, herds from 32f).
func TestUserPurgedDropsTheHerd(t *testing.T) {
	store := &fakeStore{}
	m := &MountModule{store: store, herds: map[int]domain.Herd{
		7: {LeaderUserID: 7, NextID: 2, Horses: []domain.Horse{{ID: 1, Type: "horse"}}},
		8: {LeaderUserID: 8, NextID: 2, Horses: []domain.Horse{{ID: 1, Type: "horse"}}},
	}}
	m.onUserPurged(events.UserPurged{UserId: 7})
	assert.Equal(t, map[int]domain.Herd{8: {LeaderUserID: 8, NextID: 2, Horses: []domain.Horse{{ID: 1, Type: "horse"}}}}, store.saved.Herds)
}
