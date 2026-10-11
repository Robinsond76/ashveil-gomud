package gmcp

import (
	"encoding/json"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
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
	assert.Equal(t, map[string]any{"level": float64(0), "daylight": false, "room": float64(cave.RoomId)}, got[0]["sight"], "a dark cave, no light")

	// Moving sends at once, with the new room's sight.
	a.Character.RoomId = hall.RoomId
	events.AddToQueue(events.RoomChange{UserId: 17, FromRoomId: cave.RoomId, ToRoomId: hall.RoomId})
	events.ProcessEvents()
	require.Len(t, got, 2)
	assert.Equal(t, map[string]any{"level": float64(2), "daylight": true, "room": float64(hall.RoomId)}, got[1]["sight"], "a lit hall")

	// A buff starting or ending sends at once too.
	events.AddToQueue(events.BuffsAdded{UserId: 17, BuffIds: []int{1001}})
	events.AddToQueue(events.BuffsRemoved{UserId: 17, BuffIds: []int{1001}})
	events.ProcessEvents()
	assert.Len(t, got, 4)

	// Holding or putting something away sends at once; a gold change doesn't.
	events.AddToQueue(events.EquipmentChange{UserId: 17, ItemsWorn: []items.Item{{ItemId: 1}}})
	events.AddToQueue(events.EquipmentChange{UserId: 17, GoldChange: 5})
	events.ProcessEvents()
	assert.Len(t, got, 5)

	// A mob moving sends nothing.
	events.AddToQueue(events.RoomChange{MobInstanceId: 5, FromRoomId: cave.RoomId, ToRoomId: hall.RoomId})
	events.ProcessEvents()
	assert.Len(t, got, 5)
}

// TestGametimeSightThroughTheRealLightModel (LM2 review): the sight sent is
// worked out from the player's real buffs, carried-light providers, fixture
// providers and party, not only the pure rules.
func TestGametimeSightThroughTheRealLightModel(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "lm2cave2", Name: "Cave", DarkArea: true})
	t.Cleanup(func() { rooms.RemoveTestBiome("lm2cave2") })
	cave := &rooms.Room{RoomId: 990711, Zone: "Under", Biome: "lm2cave2"}
	rooms.SetTestRoom(cave)
	t.Cleanup(func() { rooms.RemoveTestRoom(cave.RoomId) })
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 990711, Name: "Torchlight", TriggerRate: "1 real minute", TriggerCount: 15, Flags: []string{rooms.FlagLightSource}})
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 990712, Name: "Wisp", TriggerRate: "1 real minute", TriggerCount: 15, Flags: []string{rooms.FlagPartyLight}})
	t.Cleanup(parties.UseMemoryForTest())
	t.Cleanup(rooms.ResetLightFixtures)
	rooms.ResetLightFixtures()

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	a := users.NewUserRecord(18, 1)
	a.Character.RoomId = cave.RoomId
	users.SetTestUser(a)
	b := users.NewUserRecord(19, 2)
	b.Character.RoomId = cave.RoomId
	users.SetTestUser(b)
	cave.SetTestOccupants([]int{18, 19}, nil)
	AcceptGMCPForTest(a.ConnectionId())

	freshEvents(t)
	var sight map[string]any
	id := events.RegisterListener(GMCPOut{}, func(e events.Event) events.ListenerReturn {
		if out := e.(GMCPOut); out.Module == `Gametime` && out.UserId == 18 {
			raw, _ := json.Marshal(out.Payload)
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			sight, _ = m["sight"].(map[string]any)
		}
		return events.Cancel
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(GMCPOut{}, id) })
	send := func() map[string]any {
		sight = nil
		events.AddToQueue(events.BuffsAdded{UserId: 18})
		events.ProcessEvents()
		return sight
	}

	// A lit campfire here (the camping module's kind of fixture).
	lit := false
	rooms.RegisterLightFixture(func(roomId int) bool { return lit && roomId == cave.RoomId })
	lit = true
	assert.Equal(t, "fixture", send()["source"], "a fire lights the cave")
	lit = false

	// An ally's party light, then your own.
	p := parties.New(18)
	require.NotNil(t, p)
	p.UserIds = append(p.UserIds, 19)
	_ = b.Character.AddBuff(990712, false)
	assert.Equal(t, "party", send()["source"], "an ally's light")
	_ = a.Character.AddBuff(990712, false)
	assert.Equal(t, "orb", send()["source"], "your own party light")

	// A torch you carry wins, and names itself.
	carried := ""
	t.Cleanup(func() { carried = "" }) // the registry has no unregister
	rooms.RegisterCarriedLight(func(u *users.UserRecord) string {
		if u.UserId == 18 {
			return carried
		}
		return ""
	})
	carried = "torch"
	_ = a.Character.AddBuff(990711, false)
	got := send()
	assert.Equal(t, "torch", got["source"])
	assert.Equal(t, "torch", got["carried"])
	assert.Equal(t, float64(1), got["level"], "a light alone in a cave shows your room only")
}
