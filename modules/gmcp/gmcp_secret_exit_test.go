package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/encounters"
	"github.com/GoMudEngine/GoMud/internal/events"
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

// TestRoomInfoCarriesTheZoneLevelBand (37b): the web client's zone header
// reads levelband, rated against the player's level; a zone with no band
// sends none.
func TestRoomInfoCarriesTheZoneLevelBand(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "gmcpband", Name: "Band", LitArea: true})
	t.Cleanup(func() { rooms.RemoveTestBiome("gmcpband") })
	t.Cleanup(rooms.SetTestZoneConfig(&rooms.ZoneConfig{Name: "Banded", Encounters: encounters.ZoneConfig{Band: encounters.Band{Low: 10, High: 12}}}))
	for _, r := range []*rooms.Room{{RoomId: 990311, Zone: "Banded", Biome: "gmcpband"}, {RoomId: 990312, Zone: "Unbanded", Biome: "gmcpband"}} {
		rooms.SetTestRoom(r)
		id := r.RoomId
		t.Cleanup(func() { rooms.RemoveTestRoom(id) })
	}
	user := users.NewUserRecord(8, 1)
	user.Character.Level = 5
	g := &GMCPRoomModule{}
	info := func(room int) GMCPRoomModule_Payload {
		user.Character.RoomId = room
		data, _ := g.GetRoomNode(user, `Room.Info`)
		payload, ok := data.(GMCPRoomModule_Payload)
		require.True(t, ok)
		return payload
	}
	band := info(990311).LevelBand
	require.NotNil(t, band)
	assert.Equal(t, GMCPRoomModule_Payload_LevelBand{Low: 10, High: 12, Rating: "dangerous"}, *band)
	user.Character.Level = 12
	assert.Equal(t, "easy", info(990311).LevelBand.Rating)
	assert.Nil(t, info(990312).LevelBand)
}

// TestRoomInfoResendsWhenTheCompanyLevelChanges (37c): the web header's zone
// rating follows a level-up (or a recruit) at once. The first refresh only
// records the level; a change in a banded zone queues Room.Info; the same
// level, or an unbanded zone, queues nothing.
func TestRoomInfoResendsWhenTheCompanyLevelChanges(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "gmcpband2", Name: "Band", LitArea: true})
	t.Cleanup(func() { rooms.RemoveTestBiome("gmcpband2") })
	t.Cleanup(rooms.SetTestZoneConfig(&rooms.ZoneConfig{Name: "Banded Two", Encounters: encounters.ZoneConfig{Band: encounters.Band{Low: 10, High: 12}}}))
	for _, r := range []*rooms.Room{{RoomId: 990321, Zone: "Banded Two", Biome: "gmcpband2"}, {RoomId: 990322, Zone: "Unbanded Two", Biome: "gmcpband2"}} {
		rooms.SetTestRoom(r)
		id := r.RoomId
		t.Cleanup(func() { rooms.RemoveTestRoom(id) })
	}
	var resent []GMCPRoomUpdate
	freshEvents(t)
	lid := events.RegisterListener(GMCPRoomUpdate{}, func(e events.Event) events.ListenerReturn {
		resent = append(resent, e.(GMCPRoomUpdate))
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(GMCPRoomUpdate{}, lid) })

	user := users.NewUserRecord(9, 1)
	watch := &bandWatch{last: map[int]int{}}
	refresh := func(room, level int, companions ...int) {
		user.Character.RoomId = room
		s := companyview.Summary{CompanyKnown: true, Leader: companyview.Member{Level: level}}
		for _, l := range companions {
			s.Companions = append(s.Companions, companyview.Member{Level: l})
		}
		watch.onRefresh(companyview.Refreshed{User: user, Summary: s})
		events.ProcessEvents()
	}

	refresh(990321, 5)
	assert.Empty(t, resent, "the first sight only records the level")
	refresh(990321, 5)
	assert.Empty(t, resent, "the same level changes nothing")
	refresh(990321, 6)
	require.Len(t, resent, 1, "a level-up resends the room")
	assert.Equal(t, GMCPRoomUpdate{UserId: 9, Identifier: "Room.Info"}, resent[0])
	refresh(990321, 6, 12, 12)
	assert.Len(t, resent, 2, "a stronger company is a new level too")
	refresh(990322, 7)
	assert.Len(t, resent, 2, "no band here, nothing to refresh")
	watch.forget(9)
	refresh(990321, 9)
	assert.Len(t, resent, 2, "a user seen again starts fresh")
}
