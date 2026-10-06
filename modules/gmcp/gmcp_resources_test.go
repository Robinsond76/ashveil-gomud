package gmcp

import (
	"encoding/json"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 40a: Room.Info carries the shown room resources, never reserved
// ones, and omits the field when there are none.
func TestRoomInfoCarriesShownResources(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "gmcpres", Name: "Meadow", LitArea: true})
	t.Cleanup(func() { rooms.RemoveTestBiome("gmcpres") })
	spring := &rooms.Room{RoomId: 990401, Zone: "Meadow", Biome: "gmcpres", Resources: []string{"herbs", "shelter", "water"}}
	bare := &rooms.Room{RoomId: 990402, Zone: "Meadow", Biome: "gmcpres", Resources: []string{"herbs"}}
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
	assert.Equal(t, []string{"water", "shelter"}, p.Resources, "table order, reserved herbs withheld")
	assert.Contains(t, raw, `"resources":["water","shelter"]`)

	p, raw = payload(bare.RoomId)
	assert.Empty(t, p.Resources)
	assert.NotContains(t, raw, "resources", "omitted, never null or empty")
}

// World.Map entries carry the same field for visited rooms.
func TestWorldMapEntriesCarryShownResources(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "gmcpres2", Name: "Meadow", LitArea: true})
	t.Cleanup(func() { rooms.RemoveTestBiome("gmcpres2") })
	spring := &rooms.Room{RoomId: 990411, Zone: "Meadow2", Biome: "gmcpres2", Resources: []string{"game", "forage"}}
	bare := &rooms.Room{RoomId: 990412, Zone: "Meadow2", Biome: "gmcpres2", Resources: []string{"game"}}
	valid := map[int]struct{}{}
	user := users.NewUserRecord(8, 1)
	for _, r := range []*rooms.Room{spring, bare} {
		rooms.SetTestRoom(r)
		id := r.RoomId
		t.Cleanup(func() { rooms.RemoveTestRoom(id) })
		valid[id] = struct{}{}
		user.Character.MarkVisitedRoom(id, "Meadow2", valid)
	}

	payload := (&GMCPWorldModule{}).buildWorldMap(user)
	got := map[int][]string{}
	for _, entry := range payload.Rooms {
		got[entry.Id] = entry.Resources
		raw, err := json.Marshal(entry)
		require.NoError(t, err)
		if entry.Id == bare.RoomId {
			assert.NotContains(t, string(raw), "resources")
		}
	}
	assert.Equal(t, []string{"forage"}, got[spring.RoomId], "reserved game withheld")
	assert.Empty(t, got[bare.RoomId])
	assert.Len(t, got, 2)
}
