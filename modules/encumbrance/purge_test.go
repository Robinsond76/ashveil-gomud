package encumbrance

import (
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
)

// TestUserPurgedDropsCargo (Phase 32b).
func TestUserPurgedDropsCargo(t *testing.T) {
	store := &fakeStore{}
	m := &EncumbranceModule{store: store, cargo: map[int]domain.Cargo{7: {LeaderUserID: 7}, 8: {LeaderUserID: 8}}}
	m.onUserPurged(events.UserPurged{UserId: 7})
	assert.Equal(t, map[int]domain.Cargo{8: {LeaderUserID: 8}}, store.saved.Cargo)
	assert.NotContains(t, m.cargo, 7)
}
