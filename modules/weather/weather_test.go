package weather

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/weather"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		biomes: map[string]biomeTable{"forest": forestTable()},
		zones:  map[string]weather.ZoneWeather{},
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
