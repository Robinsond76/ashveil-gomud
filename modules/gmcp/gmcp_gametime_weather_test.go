package gmcp

import (
	"encoding/json"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/weather"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeWeather map[string]weather.Condition

func (f fakeWeather) CurrentCondition(zone string) (weather.Condition, bool) {
	c, ok := f[zone]
	return c, ok
}

// The time panel's weather follows the zone the player stands in, is seen
// through an exit from indoors, and is absent with no view out or no weather.
func TestGametimeWeatherFollowsTheRoom(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	weather.SetProvider(fakeWeather{
		"Wet":  {Name: "rain", Description: "Rain falls.", CloudCover: 3},
		"Mist": {Name: "fog", Description: "Fog.", CloudCover: 2, VisibilityMod: -1},
	})
	t.Cleanup(func() { weather.SetProvider(nil) })
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "gtwx", Name: "Meadow", LitArea: true})
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "gtwxin", Name: "Hall", LitArea: true, Indoor: true})
	t.Cleanup(func() { rooms.RemoveTestBiome("gtwx"); rooms.RemoveTestBiome("gtwxin") })
	outside := &rooms.Room{RoomId: 990601, Zone: "Wet", Biome: "gtwx"}
	untracked := &rooms.Room{RoomId: 990602, Zone: "Dry", Biome: "gtwx"}
	hall := &rooms.Room{RoomId: 990603, Zone: "Wet", Biome: "gtwxin", Exits: map[string]exit.RoomExit{"out": {RoomId: 990601}}}
	cellar := &rooms.Room{RoomId: 990604, Zone: "Wet", Biome: "gtwxin"}
	for _, r := range []*rooms.Room{outside, untracked, hall, cellar} {
		rooms.SetTestRoom(r)
		id := r.RoomId
		t.Cleanup(func() { rooms.RemoveTestRoom(id) })
	}

	w := gametimeWeatherFor(outside)
	require.NotNil(t, w)
	assert.Equal(t, "rain", w.Name)
	assert.False(t, w.Indoor)
	assert.Nil(t, gametimeWeatherFor(untracked), "a zone with no weather shows none")
	w = gametimeWeatherFor(hall)
	require.NotNil(t, w, "an exit to the outdoors shows the weather beyond")
	assert.True(t, w.Indoor)
	assert.Nil(t, gametimeWeatherFor(cellar), "no view out, no weather")
	assert.Nil(t, gametimeWeatherFor(nil))
}

// The real round listener sends each player the weather of their own room,
// and omits the field when there is none.
func TestGametimeRoundCarriesWeather(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	weather.SetProvider(fakeWeather{"Wet": {Name: "storm", CloudCover: 3}})
	t.Cleanup(func() { weather.SetProvider(nil) })
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "gtwx2", Name: "Meadow", LitArea: true})
	t.Cleanup(func() { rooms.RemoveTestBiome("gtwx2") })
	wet := &rooms.Room{RoomId: 990611, Zone: "Wet", Biome: "gtwx2"}
	dry := &rooms.Room{RoomId: 990612, Zone: "Dry", Biome: "gtwx2"}
	for _, r := range []*rooms.Room{wet, dry} {
		rooms.SetTestRoom(r)
		id := r.RoomId
		t.Cleanup(func() { rooms.RemoveTestRoom(id) })
	}
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	a := users.NewUserRecord(7, 1)
	a.Character.RoomId = wet.RoomId
	b := users.NewUserRecord(8, 2)
	b.Character.RoomId = dry.RoomId
	users.SetTestUser(a)
	users.SetTestUser(b)
	AcceptGMCPForTest(a.ConnectionId())
	AcceptGMCPForTest(b.ConnectionId())

	freshEvents(t)
	got := map[int]string{}
	id := events.RegisterListener(GMCPOut{}, func(e events.Event) events.ListenerReturn {
		if out := e.(GMCPOut); out.Module == `Gametime` {
			raw, _ := json.Marshal(out.Payload)
			got[out.UserId] = string(raw)
		}
		return events.Cancel
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(GMCPOut{}, id) })
	events.ProcessEvents()

	events.AddToQueue(events.NewRound{RoundNumber: 1})
	events.ProcessEvents()
	assert.Contains(t, got[7], `"weather":{"name":"storm"`)
	require.Contains(t, got, 8)
	assert.NotContains(t, got[8], `"weather"`)
}
