package weather

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/climate"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/sky"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/weather"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "low", "", false)
	os.Exit(m.Run())
}

func forestTable() biomeTable {
	return biomeTable{
		ChangeMin: 40,
		ChangeMax: 40, // deterministic in tests
		Conditions: []weightedCondition{
			{Condition: weather.Condition{Name: "clear", Description: "Clear skies.", TravelDurationPct: 100, ExertionPct: 100, RestRecoveryPct: 100}, Weight: 5},
			{Condition: weather.Condition{Name: "storm", Description: "A storm rolls in.", TravelDurationPct: 140, ExertionPct: 130, RestRecoveryPct: 75}, Weight: 1},
		},
	}
}

type fakeStore struct {
	saved     Registry
	loadErr   error
	saveErr   error
	loadCalls int
	saveCalls int
}

func (f *fakeStore) Load(registry *Registry) error {
	f.loadCalls++
	if f.loadErr != nil {
		return f.loadErr
	}
	*registry = f.saved.Clone()
	return nil
}

func (f *fakeStore) Save(registry Registry) error {
	f.saveCalls++
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = registry.Clone()
	return nil
}

func newTestModule(store Store, roundFn func() uint64, rng func(int) int) *WeatherModule {
	return &WeatherModule{
		store:   store,
		roundFn: roundFn,
		rng:     rng,
		zoneNames: func() []string {
			return []string{"dunmar"}
		},
		zoneBiome: func(zone string) string {
			if zone == "dunmar" {
				return "forest"
			}
			return ""
		},
		biomes:  map[string]biomeTable{"forest": forestTable()},
		zones:   map[string]weather.ZoneWeather{},
		skyView: func(r *rooms.Room) weather.SkyView { return weather.SkyView{WeatherZone: r.Zone} },
		timeOfDay: func() string {
			return "9:00PM"
		},
	}
}

func zeroRNG(_ int) int { return 0 }

func TestRecoveryEstablishesUntrackedZone(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store, func() uint64 { return 1000 }, zeroRNG)

	module.load()

	require.Contains(t, module.zones, "dunmar")
	zw := module.zones["dunmar"]
	assert.Equal(t, "clear", zw.Current)
	assert.Equal(t, uint64(1040), zw.NextChangeRound)
	assert.Equal(t, 1, store.saveCalls)
}

func TestRecoveryAdvancesOverdueZoneExactlyOnce(t *testing.T) {
	saved := Registry{Zones: map[string]weather.ZoneWeather{
		"dunmar": {Zone: "dunmar", Current: "clear", NextChangeRound: 900},
	}}
	store := &fakeStore{saved: saved}
	module := newTestModule(store, func() uint64 { return 1000 }, zeroRNG)

	module.load()

	zw := module.zones["dunmar"]
	assert.Equal(t, "clear", zw.Current, "zeroRNG always rolls the first weighted condition")
	assert.Equal(t, uint64(1040), zw.NextChangeRound, "rescheduled from the current round, not the missed one")
}

func TestRecoveryDoesNotAdvanceZoneNotYetDue(t *testing.T) {
	saved := Registry{Zones: map[string]weather.ZoneWeather{
		"dunmar": {Zone: "dunmar", Current: "storm", NextChangeRound: 2000},
	}}
	store := &fakeStore{saved: saved}
	module := newTestModule(store, func() uint64 { return 1000 }, zeroRNG)

	module.load()

	zw := module.zones["dunmar"]
	assert.Equal(t, "storm", zw.Current)
	assert.Equal(t, uint64(2000), zw.NextChangeRound)
	assert.Zero(t, store.saveCalls, "nothing changed, so nothing should be persisted")
}

func TestRecoveryRetainsInvalidZoneWeatherForRepair(t *testing.T) {
	saved := Registry{Zones: map[string]weather.ZoneWeather{
		"dunmar": {Zone: "dunmar", Current: "", NextChangeRound: 900},
	}}
	store := &fakeStore{saved: saved}
	module := newTestModule(store, func() uint64 { return 1000 }, zeroRNG)

	module.load()

	require.Contains(t, module.zones, "dunmar")
	assert.Equal(t, "", module.zones["dunmar"].Current)
	assert.Zero(t, store.saveCalls)
}

func TestRecoveryRetainsUnknownConditionForRepair(t *testing.T) {
	saved := Registry{Zones: map[string]weather.ZoneWeather{
		"dunmar": {Zone: "dunmar", Current: "hurricane", NextChangeRound: 900},
	}}
	store := &fakeStore{saved: saved}
	module := newTestModule(store, func() uint64 { return 1000 }, zeroRNG)

	module.load()

	assert.Equal(t, "hurricane", module.zones["dunmar"].Current, "a condition removed from the table is left untouched, not guessed")
}

func TestUntrackedZoneNeverEstablished(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store, func() uint64 { return 1000 }, zeroRNG)
	module.zoneNames = func() []string { return []string{"untracked"} }
	module.zoneBiome = func(string) string { return "swamp" }

	module.load()

	assert.Empty(t, module.zones)
	_, ok := module.CurrentCondition("untracked")
	assert.False(t, ok)
}

func TestOnNewRoundAdvancesOnlyDueTrackedZones(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store, func() uint64 { return 0 }, zeroRNG)
	module.zones = map[string]weather.ZoneWeather{
		"dunmar": {Zone: "dunmar", Current: "clear", NextChangeRound: 100},
	}

	module.onNewRound(events.NewRound{RoundNumber: 99})
	assert.Equal(t, uint64(100), module.zones["dunmar"].NextChangeRound, "not yet due")

	module.onNewRound(events.NewRound{RoundNumber: 100})
	assert.Equal(t, uint64(140), module.zones["dunmar"].NextChangeRound, "due, advances from the round it fired on")
	assert.Equal(t, 1, store.saveCalls)
}

func TestOnNewRoundNeverEstablishesNewZone(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store, func() uint64 { return 0 }, zeroRNG)

	module.onNewRound(events.NewRound{RoundNumber: 5000})

	assert.Empty(t, module.zones, "establishment only happens on load, never mid-round")
	assert.Zero(t, store.saveCalls)
}

func TestCurrentConditionRequiresTrackedZoneWithConfiguredBiome(t *testing.T) {
	module := newTestModule(&fakeStore{}, func() uint64 { return 0 }, zeroRNG)

	_, ok := module.CurrentCondition("dunmar")
	assert.False(t, ok, "no record yet")

	module.zones["dunmar"] = weather.ZoneWeather{Zone: "dunmar", Current: "storm", NextChangeRound: 40}
	condition, ok := module.CurrentCondition("dunmar")
	require.True(t, ok)
	assert.Equal(t, "storm", condition.Name)

	module.biomes = map[string]biomeTable{}
	_, ok = module.CurrentCondition("dunmar")
	assert.False(t, ok, "biome table removed from config")
}

func weatherRoom(zone string) *rooms.Room {
	return &rooms.Room{RoomId: 2001, Zone: zone}
}

func weatherUser(t *testing.T, userId int) *users.UserRecord {
	t.Helper()
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(userId, 1)
	users.SetTestUser(user)
	return user
}

func TestUserCommandShowsCurrentConditionOrUntracked(t *testing.T) {
	module := newTestModule(&fakeStore{}, func() uint64 { return 0 }, zeroRNG)
	user := weatherUser(t, 7)
	messages := captureMessages(t)

	module.userCommand("", user, weatherRoom("dunmar"), 0)
	events.ProcessEvents()
	assert.Contains(t, joinMessages(*messages), "can't tell")

	module.zones["dunmar"] = weather.ZoneWeather{Zone: "dunmar", Current: "storm", NextChangeRound: 40}
	*messages = nil
	module.userCommand("", user, weatherRoom("dunmar"), 0)
	events.ProcessEvents()
	assert.Contains(t, joinMessages(*messages), "storm rolls in")
}

func captureMessages(t *testing.T) *[]string {
	t.Helper()
	messages := []string{}
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		messages = append(messages, e.(events.Message).Text)
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	return &messages
}

func joinMessages(messages []string) string {
	out := ""
	for _, m := range messages {
		out += m
	}
	return out
}

func TestParseBiomeTablesRejectsMalformedEntries(t *testing.T) {
	raw := []any{
		map[string]any{
			"biome":                "forest",
			"changeintervalrounds": map[string]any{"min": 40, "max": 120},
			"conditions": []any{
				map[string]any{"name": "clear", "description": "Clear.", "weight": 5, "traveldurationpct": 100, "exertionpct": 100, "restrecoverypct": 100},
				map[string]any{"name": "bad", "description": "Bad.", "weight": 1, "traveldurationpct": 1, "exertionpct": 100, "restrecoverypct": 100}, // out of range
			},
		},
		map[string]any{
			"biome":      "swamp",
			"conditions": []any{},
		},
		map[string]any{
			"biome": "",
		},
	}

	tables := parseBiomeTables(raw)

	require.Contains(t, tables, "forest")
	assert.Len(t, tables["forest"].Conditions, 1, "the out-of-range condition is rejected, not the whole table")
	assert.NotContains(t, tables, "swamp", "an empty condition list invalidates the table")
}

func TestParseBiomeTablesReadsCloudCoverAndFog(t *testing.T) {
	raw := []any{
		map[string]any{
			"biome":                "forest",
			"changeintervalrounds": map[string]any{"min": 40, "max": 120},
			"conditions": []any{
				map[string]any{"name": "fog", "description": "Fog.", "weight": 2, "traveldurationpct": 110, "exertionpct": 100, "restrecoverypct": 100, "cloudcover": 2, "visibilitymod": -1, "temperaturemod": -2},
				map[string]any{"name": "bad-cloud", "description": "Bad.", "weight": 1, "traveldurationpct": 100, "exertionpct": 100, "restrecoverypct": 100, "cloudcover": 5},
				map[string]any{"name": "bad-fog", "description": "Bad.", "weight": 1, "traveldurationpct": 100, "exertionpct": 100, "restrecoverypct": 100, "visibilitymod": -3},
			},
		},
	}

	tables := parseBiomeTables(raw)

	require.Contains(t, tables, "forest")
	require.Len(t, tables["forest"].Conditions, 1, "out-of-range cloud cover and fog are rejected")
	fog := tables["forest"].Conditions[0]
	assert.Equal(t, 2, fog.CloudCover)
	assert.Equal(t, -1, fog.VisibilityMod)
	assert.Equal(t, -2, fog.TemperatureMod)
}

func TestParseMoonCycleDays(t *testing.T) {
	assert.Equal(t, 28, parseMoonCycleDays(28))
	assert.Equal(t, sky.DefaultCycleDays, parseMoonCycleDays(nil), "missing config uses the default")
	assert.Equal(t, sky.DefaultCycleDays, parseMoonCycleDays(-4), "a non-positive cycle uses the default")
}

func TestUserCommandIndoorWithoutGlimpseHidesWeather(t *testing.T) {
	module := newTestModule(&fakeStore{}, func() uint64 { return 0 }, zeroRNG)
	module.zones["dunmar"] = weather.ZoneWeather{Zone: "dunmar", Current: "storm", NextChangeRound: 40}
	module.skyView = func(*rooms.Room) weather.SkyView { return weather.SkyView{Indoor: true} }
	user := weatherUser(t, 7)
	messages := captureMessages(t)

	module.userCommand("", user, weatherRoom("dunmar"), 0)
	events.ProcessEvents()
	out := joinMessages(*messages)
	assert.Contains(t, out, "can't see the sky")
	assert.NotContains(t, out, "storm rolls in")
}

func TestUserCommandReportsTimeMoonAndGlimpse(t *testing.T) {
	module := newTestModule(&fakeStore{}, func() uint64 { return 0 }, zeroRNG)
	module.zones["dunmar"] = weather.ZoneWeather{Zone: "dunmar", Current: "clear", NextChangeRound: 40}
	module.skyView = func(*rooms.Room) weather.SkyView {
		return weather.SkyView{Indoor: true, GlimpseExit: "north", WeatherZone: "dunmar", Night: true, HasMoon: true, Moon: sky.Full}
	}
	user := weatherUser(t, 7)
	messages := captureMessages(t)

	module.userCommand("", user, weatherRoom("inn"), 0)
	events.ProcessEvents()
	out := joinMessages(*messages)
	assert.Contains(t, out, "It is 9:00PM")
	assert.Contains(t, out, "Through the north exit: Clear skies.")
}

type fakeClimate struct{ temps map[int]int }

func (f fakeClimate) AirTemperatureIn(roomId int) (int, bool) {
	t, ok := f.temps[roomId]
	return t, ok
}

func TestUserCommandShowsAirTemperature(t *testing.T) {
	module := newTestModule(&fakeStore{}, func() uint64 { return 0 }, zeroRNG)
	climate.SetProvider(fakeClimate{temps: map[int]int{2001: -8}})
	t.Cleanup(func() { climate.SetProvider(nil) })
	user := weatherUser(t, 7)
	messages := captureMessages(t)

	module.userCommand("", user, weatherRoom("dunmar"), 0)
	events.ProcessEvents()
	assert.Contains(t, joinMessages(*messages), "Temperature here: -8°C (freezing).")

	climate.SetProvider(nil)
	*messages = nil
	module.userCommand("", user, weatherRoom("dunmar"), 0)
	events.ProcessEvents()
	assert.NotContains(t, joinMessages(*messages), "Temperature here:", "no provider, no temperature line")
}

// TestShippedWorldTracksOldKingsRoadNotDunmar loads the shipped world and
// the shipped biome tables (Phase 16 decision 6): the fork's forest zone
// gets weather, Dunmar (a city) does not.
func TestShippedWorldTracksOldKingsRoadNotDunmar(t *testing.T) {
	dataDir := filepath.Join("..", "..", "_datafiles", "world", "default")
	require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": dataDir}))
	rooms.LoadDataFiles()

	data, err := files.ReadFile("files/data-overlays/config.yaml")
	require.NoError(t, err)
	var cfg map[string]any
	require.NoError(t, yaml.Unmarshal(data, &cfg))

	module := newTestModule(&fakeStore{}, func() uint64 { return 1000 }, zeroRNG)
	module.zoneNames = rooms.GetAllZoneNames
	module.zoneBiome = rooms.GetZoneBiome
	module.biomes = parseBiomeTables(cfg["Biomes"])
	module.load()

	_, tracked := module.CurrentCondition("Old Kings Road")
	assert.True(t, tracked, "the proving route's forest zone has weather")
	_, tracked = module.CurrentCondition("Dunmar")
	assert.False(t, tracked, "Dunmar is a city: no weather")
}
