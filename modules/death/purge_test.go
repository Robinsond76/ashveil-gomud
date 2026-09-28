package death

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
)

// TestUserPurgedForgetsTheChurchReturn (Phase 32b).
func TestUserPurgedForgetsTheChurchReturn(t *testing.T) {
	m := newModule()
	m.returned[7], m.returned[8] = 10, 11
	m.onUserPurged(events.UserPurged{UserId: 7})
	assert.Equal(t, map[int]uint64{8: 11}, m.returned)
}
