package hooks

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// The real idle turn (HandleIdleMobs) applies the chatter limits: a town NPC
// whose every idle command is a line of speech used to speak most rounds.
func TestIdleMobChatterIsRareThroughTheIdleHook(t *testing.T) {
	freshEvents(t)
	mobs.ForgetChatterForTest()
	t.Cleanup(mobs.ForgetChatterForTest)

	gameplay := configs.GetGamePlayConfig()
	gameplay.MobChatterCooldownRounds = 60
	gameplay.MobChatterMemoryRounds = 900
	gameplay.MobConverseChance = 0
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))

	prevRound := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCount(prevRound) })

	room := &rooms.Room{RoomId: 990301, Title: "Market square"}
	room.SetTestOccupants([]int{41}, []int{990301})
	rooms.SetTestRoom(room)
	t.Cleanup(func() { rooms.RemoveTestRoom(room.RoomId) })

	crier := &mobs.Mob{
		InstanceId:    990301,
		MobId:         990301,
		HomeRoomId:    room.RoomId,
		MaxWander:     -1,
		ActivityLevel: 100,
		IdleCommands:  []string{"say Fresh eels!", "say Eels, two a copper!"},
	}
	crier.Character.Name = "crier"
	crier.Character.RoomId = room.RoomId
	mobs.SetTestInstance(crier)
	t.Cleanup(func() { mobs.RemoveTestInstance(crier.InstanceId) })

	var said []string
	id := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
		if in, ok := e.(events.Input); ok && in.MobInstanceId == crier.InstanceId && strings.HasPrefix(in.InputText, "say ") {
			said = append(said, in.InputText)
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Input{}, id) })

	// Ten minutes of 4-second rounds: before the fix about 150 lines.
	for round := uint64(10000); round < 10150; round++ {
		util.SetRoundCount(round)
		HandleIdleMobs(events.MobIdle{MobInstanceId: crier.InstanceId})
		events.ProcessEvents()
	}

	// Each line once (memory), at least 60 rounds apart (cooldown).
	if len(said) != 2 {
		t.Fatalf("the crier speaks each of its two lines once in ten minutes, got %d: %v", len(said), said)
	}
	if said[0] == said[1] {
		t.Fatalf("player 41 never hears the same line twice within the hour: %v", said)
	}
}
