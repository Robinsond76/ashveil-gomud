package usercommands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Phase 27b: the tutorial asks for an inspection only when its command
// exists.
func TestIsRegistered(t *testing.T) {
	assert.True(t, IsRegistered("status"))
	assert.False(t, IsRegistered("no-such-command-27b"))
	RegisterCommand("tutorial-27b-probe", Status, false, false)
	t.Cleanup(func() { delete(userCommands, "tutorial-27b-probe") })
	assert.True(t, IsRegistered("tutorial-27b-probe"))
}
