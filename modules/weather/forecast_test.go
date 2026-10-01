package weather

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/weather"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRecoveryForetellsAZoneSavedBeforeForecasts (33f2): a zone saved
// before forecasts keeps its weather and gains a foretold condition, saved.
func TestRecoveryForetellsAZoneSavedBeforeForecasts(t *testing.T) {
	saved := Registry{Zones: map[string]weather.ZoneWeather{
		"dunmar": {Zone: "dunmar", Current: "storm", NextChangeRound: 2000},
	}}
	store := &fakeStore{saved: saved}
	module := newTestModule(store, func() uint64 { return 1000 }, zeroRNG)

	module.load()

	zw := module.zones["dunmar"]
	assert.Equal(t, "storm", zw.Current, "the current weather is untouched")
	assert.Equal(t, uint64(2000), zw.NextChangeRound)
	assert.Equal(t, "clear", zw.Next)
	assert.Equal(t, 1, store.saveCalls)
	assert.Equal(t, "clear", store.saved.Zones["dunmar"].Next, "durable")
}

// TestWeatherBecomesWhatWasForetold (33f2): on the round a zone's weather
// changes it becomes its foretold condition, not a fresh roll, and
// foretells the next.
func TestWeatherBecomesWhatWasForetold(t *testing.T) {
	store := &fakeStore{}
	round := uint64(1000)
	module := newTestModule(store, func() uint64 { return round }, zeroRNG)
	module.load()
	module.zones["dunmar"] = weather.ZoneWeather{Zone: "dunmar", Current: "clear", NextChangeRound: 1040, Next: "storm"}

	module.onNewRound(events.NewRound{RoundNumber: 1040})

	zw := module.zones["dunmar"]
	assert.Equal(t, "storm", zw.Current, "a zero roll would have given clear: the foretold storm came")
	assert.Equal(t, "clear", zw.Next)
	assert.Equal(t, uint64(1080), zw.NextChangeRound)
}

// TestWeatherCommandForecastsByLevel (33f2): the weather command adds the
// forecaster's reading, more of it at each level, and nothing without one.
func TestWeatherCommandForecastsByLevel(t *testing.T) {
	module := newTestModule(&fakeStore{}, func() uint64 { return 1000 }, zeroRNG)
	module.roundsPerHour = func() uint64 { return 20 }
	module.zones["dunmar"] = weather.ZoneWeather{Zone: "dunmar", Current: "clear", NextChangeRound: 1060, Next: "storm"}
	user := weatherUser(t, 7)
	messages := captureMessages(t)
	read := func(level int, present bool) string {
		module.forecaster = func(leader int, roomIDs ...int) (archetypes.Specialist, bool) {
			assert.Equal(t, 7, leader)
			assert.Equal(t, []int{2001}, roomIDs, "the forecaster must be in the leader's room")
			return archetypes.Specialist{Name: "Mira", Level: level}, present
		}
		*messages = nil
		module.userCommand("", user, weatherRoom("dunmar"), 0)
		events.ProcessEvents()
		return joinMessages(*messages)
	}

	assert.NotContains(t, read(0, false), "the sky")
	one := read(1, true)
	assert.Contains(t, one, "Mira reads the sky: harder going ahead.")
	assert.NotContains(t, one, "storm is coming")
	two := read(2, true)
	assert.Contains(t, two, "storm is coming (harder going ahead).")
	assert.NotContains(t, two, "hours")
	assert.Contains(t, read(3, true), "storm is coming (harder going ahead), in about 3 hours.")
	require.NotContains(t, read(4, true), "Over in", "a room with no exits has no neighbours to read")
}

// TestNeighbourForecastsUseWeatherZones (33f2 review finding 8): a level-4
// forecaster reads the weather zones the exits lead into, by each room's
// sky, never repeating its own.
func TestNeighbourForecastsUseWeatherZones(t *testing.T) {
	module := newTestModule(&fakeStore{}, func() uint64 { return 1000 }, zeroRNG)
	module.zones["dunmar"] = weather.ZoneWeather{Zone: "dunmar", Current: "clear", NextChangeRound: 1060, Next: "storm"}
	module.forecaster = func(int, ...int) (archetypes.Specialist, bool) {
		return archetypes.Specialist{Name: "Mira", Level: 4}, true
	}
	// The yard's own zone has no weather; its sky is Dunmar's.
	module.skyView = func(r *rooms.Room) weather.SkyView {
		if r.Zone == "Yard" {
			return weather.SkyView{WeatherZone: "dunmar"}
		}
		return weather.SkyView{WeatherZone: r.Zone}
	}
	gate := &rooms.Room{RoomId: 98101, Zone: "Yard", Exits: map[string]exit.RoomExit{"north": {RoomId: 98102}}}
	street := &rooms.Room{RoomId: 98102, Zone: "dunmar"}
	rooms.SetTestRoom(gate)
	rooms.SetTestRoom(street)
	t.Cleanup(func() { rooms.RemoveTestRoom(98101); rooms.RemoveTestRoom(98102) })
	user := weatherUser(t, 7)
	messages := captureMessages(t)
	module.userCommand("", user, gate, 0)
	events.ProcessEvents()
	text := joinMessages(*messages)
	assert.Contains(t, text, "storm is coming")
	assert.NotContains(t, text, "Over in", "the street shares the yard's weather zone")
}
