package mount

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	domain "github.com/GoMudEngine/GoMud/internal/mount"
	"github.com/stretchr/testify/assert"
)

// TestUserPurgedDropsTheMount (Phase 32b).
func TestUserPurgedDropsTheMount(t *testing.T) {
	store := &fakeStore{}
	m := &MountModule{store: store, mounts: map[int]domain.Mount{7: {LeaderUserID: 7, Type: "horse"}, 8: {LeaderUserID: 8, Type: "horse"}}}
	m.onUserPurged(events.UserPurged{UserId: 7})
	assert.Equal(t, map[int]domain.Mount{8: {LeaderUserID: 8, Type: "horse"}}, store.saved.Mounts)
}
