package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWhoTemplateListsGroupsOnTheirOwnLines (Phase 32c): the room's "Also
// here" list keeps lone mobs and players; each enemy group has a line of
// its own below it.
func TestWhoTemplateListsGroupsOnTheirOwnLines(t *testing.T) {
	useWorld(t, "default")
	details := rooms.RoomTemplateDetails{
		VisiblePlayers: []string{"Brom"},
		VisibleMobs:    []string{"a cave troll"},
		VisibleGroups: []string{
			`<ansi fg="mobname">A band of ruffians</ansi> (3): two ruffians and a rat.`,
			`<ansi fg="mobname">A swarm of rats</ansi> (2).`,
		},
	}
	out, err := templates.Process("descriptions/who", details, 0)
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(out, "")
	assert.Equal(t, "Also here: Brom and a cave troll\nA band of ruffians (3): two ruffians and a rat.\nA swarm of rats (2).\n", plain)

	details = rooms.RoomTemplateDetails{VisibleGroups: []string{`<ansi fg="mobname">A swarm of rats</ansi> (2).`}}
	out, err = templates.Process("descriptions/who", details, 0)
	require.NoError(t, err)
	assert.Equal(t, "A swarm of rats (2).\n", tagPattern.ReplaceAllString(out, ""))
}
