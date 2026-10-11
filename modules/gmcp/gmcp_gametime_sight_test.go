package gmcp

import (
	"encoding/json"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGametimeCarriesSight (LM2): each player's Gametime carries their own
// sight, every round and at once when they move or gain or lose a buff; a
// mob's room change sends nothing.
func TestGametimeCarriesSight(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "lm2cave", Name: "Cave", DarkArea: true})
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "lm2hall", Name: "Hall", LitArea: true, Indoor: true})
	t.Cleanup(func() { rooms.RemoveTestBiome("lm2cave"); rooms.RemoveTestBiome("lm2hall") })
	cave := &rooms.Room{RoomId: 990701, Zone: "Under", Biome: "lm2cave"}
	hall := &rooms.Room{RoomId: 990702, Zone: "Under", Biome: "lm2hall"}
	for _, r := range []*rooms.Room{cave, hall} {
		rooms.SetTestRoom(r)
		id := r.RoomId
		t.Cleanup(func() { rooms.RemoveTestRoom(id) })
	}
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	a := users.NewUserRecord(17, 1)
	a.Character.RoomId = cave.RoomId
	users.SetTestUser(a)
	AcceptGMCPForTest(a.ConnectionId())

	freshEvents(t)
	var got []map[string]any
	id := events.RegisterListener(GMCPOut{}, func(e events.Event) events.ListenerReturn {
		if out := e.(GMCPOut); out.Module == `Gametime` && out.UserId == 17 {
			raw, _ := json.Marshal(out.Payload)
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			got = append(got, m)
		}
		return events.Cancel
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(GMCPOut{}, id) })
	events.ProcessEvents()

	events.AddToQueue(events.NewRound{RoundNumber: 1})
	events.ProcessEvents()
	require.Len(t, got, 1)
	assert.Equal(t, map[string]any{"level": float64(0), "daylight": false}, got[0]["sight"], "a dark cave, no light")

	// Moving sends at once, with the new room's sight.
	a.Character.RoomId = hall.RoomId
	events.AddToQueue(events.RoomChange{UserId: 17, FromRoomId: cave.RoomId, ToRoomId: hall.RoomId})
	events.ProcessEvents()
	require.Len(t, got, 2)
	assert.Equal(t, map[string]any{"level": float64(2), "daylight": true}, got[1]["sight"], "a lit hall")

	// A buff starting or ending sends at once too.
	events.AddToQueue(events.BuffsAdded{UserId: 17, BuffIds: []int{1001}})
	events.AddToQueue(events.BuffsRemoved{UserId: 17, BuffIds: []int{1001}})
	events.ProcessEvents()
	assert.Len(t, got, 4)

	// A mob moving sends nothing.
	events.AddToQueue(events.RoomChange{MobInstanceId: 5, FromRoomId: cave.RoomId, ToRoomId: hall.RoomId})
	events.ProcessEvents()
	assert.Len(t, got, 4)
}
