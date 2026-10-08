package rooms

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
)

// Phase 88: a corpse left by an earlier fight crumbling mid-battle read as
// part of the current fight ("a skeleton corpse crumbles to dust" in an imp
// battle). Players in a battle are not told; the others still are.
func TestACrumblingCorpseIsNotToldToThoseInABattle(t *testing.T) {
	battle.Reset()
	t.Cleanup(battle.Reset)
	events.ClearQueueForTest()
	t.Cleanup(events.ClearQueueForTest)

	room := NewEmptyRoom()
	room.AddPlayer(88001) // in a battle
	room.AddPlayer(88002) // watching from the side
	battle.Begin(88001, room.RoomId, 1, "imps", []int{88100})
	room.Corpses = []Corpse{{MobId: 5, Character: characters.Character{Name: "skeleton"}, Prunable: true}}

	var told []events.Message
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if m := e.(events.Message); m.RoomId == room.RoomId && strings.Contains(m.Text, "crumbles to dust") {
			told = append(told, m)
		}
		return events.Cancel
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })

	room.UpdateCorpses(10)
	events.ProcessEvents()

	assert.Empty(t, room.Corpses, "the corpse still decays")
	if assert.Len(t, told, 1) {
		assert.Equal(t, []int{88001}, told[0].ExcludeUserIds, "the player in the battle is left out of the room message")
	}
}
