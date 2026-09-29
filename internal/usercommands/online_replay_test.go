package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// TestOnlineNotesAReplay (Phase 32b): the online list shows a player
// replaying the tutorial under their name, with a note.
func TestOnlineNotesAReplay(t *testing.T) {
	real := users.NewUserRecord(7, 1)
	assert.Equal(t, "Aria", onlineName(real, "Aria"))
	replay := users.NewUserRecord(users.ReplayUserIdBase, 1)
	replay.ReplayOf = 7
	assert.Equal(t, "Aria (replaying the tutorial)", onlineName(replay, "Aria"))
}
