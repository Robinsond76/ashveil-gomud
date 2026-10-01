package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRoomInfoShowsASpottedSecretExit (33f2 review finding 5): Room.Info
// lists a secret exit once the player has spotted it with Keen Eye, as
// well as one they have been through.
func TestRoomInfoShowsASpottedSecretExit(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "gmcphall", Name: "Hall", LitArea: true})
	t.Cleanup(func() { rooms.RemoveTestBiome("gmcphall") })
	hall := &rooms.Room{RoomId: 990301, Zone: "Hall", Biome: "gmcphall", Exits: map[string]exit.RoomExit{
		"north": {RoomId: 990302},
		"west":  {RoomId: 990303, Secret: true},
	}}
	for _, r := range []*rooms.Room{hall, {RoomId: 990302, Zone: "Hall", Biome: "gmcphall"}, {RoomId: 990303, Zone: "Hall", Biome: "gmcphall"}} {
		rooms.SetTestRoom(r)
		id := r.RoomId
		t.Cleanup(func() { rooms.RemoveTestRoom(id) })
	}
	user := users.NewUserRecord(7, 1)
	user.Character.RoomId = 990301
	g := &GMCPRoomModule{}
	exits := func() map[string]int {
		data, _ := g.GetRoomNode(user, `Room.Info`)
		payload, ok := data.(GMCPRoomModule_Payload)
		require.True(t, ok)
		return payload.Exits
	}
	assert.NotContains(t, exits(), "west")
	user.Character.LearnSecretExit(990301, "west")
	assert.Contains(t, exits(), "west")
}
