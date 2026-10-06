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

// Phase 40a: Room.Info carries the shown room resources and omits the field
// when there are none. Phase 40a2: it also lists the ones picked clean.
func TestRoomInfoCarriesShownResources(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "gmcpres", Name: "Meadow", LitArea: true})
	t.Cleanup(func() { rooms.RemoveTestBiome("gmcpres") })
	spring := &rooms.Room{RoomId: 990401, Zone: "Meadow", Biome: "gmcpres", Resources: []string{"herbs", "shelter", "water"}}
	bare := &rooms.Room{RoomId: 990402, Zone: "Meadow", Biome: "gmcpres"}
	for _, r := range []*rooms.Room{spring, bare} {
		rooms.SetTestRoom(r)
		id := r.RoomId
		t.Cleanup(func() { rooms.RemoveTestRoom(id) })
	}
	user := users.NewUserRecord(7, 1)
	g := &GMCPRoomModule{}
	payload := func(roomID int) (GMCPRoomModule_Payload, string) {
		user.Character.RoomId = roomID
		data, _ := g.GetRoomNode(user, `Room.Info`)
		p, ok := data.(GMCPRoomModule_Payload)
		require.True(t, ok)
		raw, err := json.Marshal(p)
		require.NoError(t, err)
		return p, string(raw)
	}

	p, raw := payload(spring.RoomId)
	assert.Equal(t, []string{"water", "shelter", "herbs"}, p.Resources, "table order")
	assert.Contains(t, raw, `"resources":["water","shelter","herbs"]`)
	assert.NotContains(t, raw, "depleted", "omitted when nothing is picked clean")

	rooms.SetDepletedCheck(func(roomID int, resource string) bool { return roomID == spring.RoomId && resource == "herbs" })
	t.Cleanup(func() { rooms.SetDepletedCheck(nil) })
	p, raw = payload(spring.RoomId)
	assert.Equal(t, []string{"herbs"}, p.Depleted)
	assert.Contains(t, raw, `"depleted":["herbs"]`)

	p, raw = payload(bare.RoomId)
	assert.Empty(t, p.Resources)
	assert.NotContains(t, raw, "resources", "omitted, never null or empty")
	assert.NotContains(t, raw, "depleted")
}

// World.Map entries carry the same fields for visited rooms.
func TestWorldMapEntriesCarryShownResources(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "gmcpres2", Name: "Meadow", LitArea: true})
	t.Cleanup(func() { rooms.RemoveTestBiome("gmcpres2") })
	spring := &rooms.Room{RoomId: 990411, Zone: "Meadow2", Biome: "gmcpres2", Resources: []string{"game", "forage"}}
	bare := &rooms.Room{RoomId: 990412, Zone: "Meadow2", Biome: "gmcpres2"}
	valid := map[int]struct{}{}
	user := users.NewUserRecord(8, 1)
	for _, r := range []*rooms.Room{spring, bare} {
		rooms.SetTestRoom(r)
		id := r.RoomId
		t.Cleanup(func() { rooms.RemoveTestRoom(id) })
		valid[id] = struct{}{}
		user.Character.MarkVisitedRoom(id, "Meadow2", valid)
	}
	rooms.SetDepletedCheck(func(roomID int, resource string) bool { return roomID == spring.RoomId && resource == "game" })
	t.Cleanup(func() { rooms.SetDepletedCheck(nil) })

	payload := (&GMCPWorldModule{}).buildWorldMap(user)
	got := map[int][]string{}
	depleted := map[int][]string{}
	for _, entry := range payload.Rooms {
		got[entry.Id] = entry.Resources
		depleted[entry.Id] = entry.Depleted
		raw, err := json.Marshal(entry)
		require.NoError(t, err)
		if entry.Id == bare.RoomId {
			assert.NotContains(t, string(raw), "resources")
			assert.NotContains(t, string(raw), "depleted")
		}
	}
	assert.Equal(t, []string{"forage", "game"}, got[spring.RoomId])
	assert.Equal(t, []string{"game"}, depleted[spring.RoomId])
	assert.Empty(t, got[bare.RoomId])
	assert.Len(t, got, 2)
}

// Phase 40c: a resource change reaches every player online as World.Resources,
// so a map redraws a room its player is not standing in.
func TestWorldResourcesGoToEveryoneOnline(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "gmcpres2", Name: "Meadow", LitArea: true})
	t.Cleanup(func() { rooms.RemoveTestBiome("gmcpres2") })
	spring := &rooms.Room{RoomId: 990411, Zone: "Meadow", Biome: "gmcpres2", Resources: []string{"herbs", "water"}}
	rooms.SetTestRoom(spring)
	t.Cleanup(func() { rooms.RemoveTestRoom(990411) })
	rooms.SetDepletedCheck(func(roomID int, resource string) bool { return roomID == 990411 && resource == "herbs" })
	t.Cleanup(func() { rooms.SetDepletedCheck(nil) })

	far := users.NewUserRecord(990412, 1) // standing somewhere else entirely
	far.Character.RoomId = 1
	users.SetTestUser(far)
	t.Cleanup(func() { users.RemoveTestUser(far.UserId) })

	var got []GMCPWorldResources_Payload
	var to []int
	id := events.RegisterListener(GMCPOut{}, func(e events.Event) events.ListenerReturn {
		if out := e.(GMCPOut); out.Module == `World.Resources` {
			got = append(got, out.Payload.(GMCPWorldResources_Payload))
			to = append(to, out.UserId)
			return events.Cancel
		}
		return events.Continue
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(GMCPOut{}, id) })

	(&GMCPWorldModule{}).resourcesChangedHandler(events.RoomResourcesChanged{RoomId: 990411})
	events.ProcessEvents()
	require.Contains(t, to, far.UserId)
	i := 0
	for n, u := range to {
		if u == far.UserId {
			i = n
		}
	}
	assert.Equal(t, GMCPWorldResources_Payload{Id: 990411, Resources: []string{"water", "herbs"}, Depleted: []string{"herbs"}}, got[i])
}
