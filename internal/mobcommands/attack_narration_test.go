package mobcommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMobFightStartIsNarrated (Phase 29c review fix): a mob turning on
// another mob opens no battle, so the room is told, in the narration
// voice, who goes for whom.
func TestMobFightStartIsNarrated(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	room := rooms.NewEmptyRoom()
	wolf := &mobs.Mob{InstanceId: 434401}
	wolf.Character.Name = "timber wolf"
	wolf.Character.RoomId = room.RoomId
	rat := &mobs.Mob{InstanceId: 434402}
	rat.Character.Name = "big rat"
	rat.Character.RoomId = room.RoomId
	rat.Character.Health = 5
	rat.Character.SetAggro(0, wolf.InstanceId, characters.DefaultAttack)
	for _, m := range []*mobs.Mob{wolf, rat} {
		mobs.SetTestInstance(m)
		room.AddMob(m.InstanceId)
		id := m.InstanceId
		t.Cleanup(func() { mobs.RemoveTestInstance(id) })
	}

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

	handled, err := Attack("", wolf, room)
	require.NoError(t, err)
	require.True(t, handled)
	events.ProcessEvents()
	require.NotNil(t, wolf.Character.Aggro)
	assert.Equal(t, rat.InstanceId, wolf.Character.Aggro.MobInstanceId)
	assert.Contains(t, strings.Join(seen, ""), `The <ansi fg="mobname">timber wolf</ansi> goes for the <ansi fg="mobname">big rat</ansi>.`)
}
