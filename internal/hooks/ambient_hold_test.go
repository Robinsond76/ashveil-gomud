package hooks

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// Phase 88: a sunset must not break into a battle log; it is told when the
// player's battle ends.
func TestAmbientBroadcastWaitsForTheBattleToEnd(t *testing.T) {
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	battle.Reset()
	t.Cleanup(battle.Reset)
	events.ClearQueueForTest()
	t.Cleanup(events.ClearQueueForTest)
	heldAmbient = map[int]([]string){}
	t.Cleanup(func() { heldAmbient = map[int][]string{} })

	u := users.NewUserRecord(88501, 0)
	u.Character.Name = "Aria"
	u.Character.RoomId = 1
	users.SetTestUser(u)
	battle.Begin(88501, 1, 1, "imps", []int{88600})

	Broadcast_SendToAll(events.Broadcast{Text: "The sun sets.", HoldInBattle: true})
	assert.Equal(t, []string{"The sun sets."}, heldAmbient[88501], "held while the battle goes on")

	var told []string
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		told = append(told, e.(events.Message).Text)
		return events.Cancel
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	battle.End(88501)
	releaseAmbient(88501)
	events.ProcessEvents()
	assert.Contains(t, strings.Join(told, ""), "The sun sets.")
	assert.Empty(t, heldAmbient[88501])
}

// Phase 88: a target that fell mid-round is told as fallen before the
// "turns toward" line for the next one, and only once.
func TestAFallenTargetIsToldBeforeTheTurn(t *testing.T) {
	events.ClearQueueForTest()
	t.Cleanup(events.ClearQueueForTest)
	room := rooms.NewEmptyRoom()
	room.RoomId = 88800
	rooms.SetTestRoom(room)
	t.Cleanup(func() { rooms.RemoveTestRoom(room.RoomId) })
	foe := &mobs.Mob{InstanceId: 88701}
	foe.Character.Name = "skeleton"
	foe.Character.RoomId = room.RoomId
	foe.Character.Health = 0
	mobs.SetTestInstance(foe)
	t.Cleanup(func() { mobs.RemoveTestInstance(88701) })
	room.AddMob(88701)

	var told []string
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if m := e.(events.Message); m.RoomId == room.RoomId {
			told = append(told, m.Text)
		}
		return events.Cancel
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })

	fallenFirst(88701)
	fallenFirst(88701) // the round's own notice, later, stays silent
	mobDeathNotice(foe)
	events.ProcessEvents()
	assert.Len(t, told, 1)
	assert.Contains(t, told[0], "skeleton")
}
