package mobcommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// roomLines kills a mob in a fresh room with rest and returns what the
// room was told.
func roomLines(t *testing.T, rest string, practice bool) string {
	t.Helper()
	mudlog.SetupLogger(nil, "low", "", false)
	room := rooms.NewEmptyRoom()
	mob := &mobs.Mob{MobId: 998, Practice: practice}
	mob.Character.Name = "bandit captain"
	mob.InstanceId = 434343
	room.AddMob(mob.InstanceId)

	var seen []string
	lid := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if m := e.(events.Message); m.RoomId == room.RoomId {
			seen = append(seen, m.Text)
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, lid) })
	events.ProcessEvents()
	seen = nil

	handled, err := Suicide(rest, mob, room)
	require.NoError(t, err)
	require.True(t, handled)
	events.ProcessEvents()
	assert.NotContains(t, room.GetMobs(), 434343, "it dies either way")
	return strings.Join(seen, "")
}

// Phase 29c: a death prints its notice in the narration voice, with its
// article; "suicide quiet" (a combat death, whose notice the round already
// printed) prints none.
func TestSuicideDeathNotice(t *testing.T) {
	got := roomLines(t, "", false)
	assert.Contains(t, got, `The <ansi fg="mobname">bandit captain</ansi> `)
	assert.NotContains(t, got, "has died")

	assert.NotContains(t, roomLines(t, "quiet", false), "bandit captain")
}

func TestSuicideQuietPracticeFoe(t *testing.T) {
	assert.Contains(t, roomLines(t, "", true), "is beaten and yields the field.")
	assert.NotContains(t, roomLines(t, "quiet", true), "is beaten")
}
