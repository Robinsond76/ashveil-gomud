package exposure

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/climate"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/weather"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "low", "", false)
	os.Exit(m.Run())
}

const (
	testWarmedBuff = 9201
	testFurCoat    = 9301
	testTunic      = 9302
)

type fakeStore struct {
	saved     Registry
	saveErr   error
	saveCalls int
}

func (f *fakeStore) Load(r *Registry) error { *r = f.saved.clone(); return nil }
func (f *fakeStore) Save(r Registry) error {
	f.saveCalls++
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = r.clone()
	return nil
}

type drainCall struct {
	leader int
	key    survival.MemberKey
	cost   survival.Exertion
}

type env struct {
	m        *ExposureModule
	store    *fakeStore
	rooms    map[int]*rooms.Room
	users    []*users.UserRecord
	comps    map[int][]member
	drains   []drainCall
	night    bool
	messages *[]string
	// rosterAlwaysKnown reports an (empty) known roster for leaders with no
	// entry in comps, so off-roster companions get pruned.
	rosterAlwaysKnown bool
}

// setup registers test biomes, buffs, and items, and a module whose seams
// read from the returned env.
func setup(t *testing.T) *env {
	t.Helper()
	buffs.SetTestFlag("warmed")
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: testWarmedBuff, Name: "cold tolerant", TriggerCount: 1, Flags: []string{"warmed"}})
	ids := []int{testWarmedBuff}
	for _, table := range []map[climate.Band]int{coldBuffs, heatBuffs} {
		for _, id := range table {
			buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: id, Name: "band", TriggerCount: 1})
			ids = append(ids, id)
		}
	}
	items.SetTestItemSpec(&items.ItemSpec{ItemId: testFurCoat, Name: "fur coat", Type: items.Body, Warmth: 30})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: testTunic, Name: "tunic", Type: items.Body})
	for _, b := range []*rooms.BiomeInfo{
		{BiomeId: "snow", Name: "Snow", Symbol: "*"},
		{BiomeId: "desert", Name: "Desert", Symbol: "d"},
		{BiomeId: "forest", Name: "Forest", Symbol: "f"},
		{BiomeId: "house", Name: "House", Symbol: "h", LitArea: true, Indoor: true},
		{BiomeId: "cave", Name: "Cave", Symbol: "c", DarkArea: true, Indoor: true},
	} {
		rooms.SetTestBiome(b)
	}
	users.ResetActiveUsers()
	t.Cleanup(func() {
		for _, id := range ids {
			buffs.RemoveTestBuffSpec(id)
		}
		items.RemoveTestItemSpec(testFurCoat)
		items.RemoveTestItemSpec(testTunic)
		for _, id := range []string{"snow", "desert", "forest", "house", "cave"} {
			rooms.RemoveTestBiome(id)
		}
		climate.ResetHeatSources()
		users.ResetActiveUsers()
	})

	e := &env{store: &fakeStore{}, rooms: map[int]*rooms.Room{}, comps: map[int][]member{}}
	e.rooms[1] = &rooms.Room{RoomId: 1, Zone: "Tundra", Biome: "snow"}
	e.rooms[2] = &rooms.Room{RoomId: 2, Zone: "Dunes", Biome: "desert"}
	e.rooms[3] = &rooms.Room{RoomId: 3, Zone: "Wood", Biome: "forest"}
	e.rooms[4] = &rooms.Room{RoomId: 4, Zone: "Town", Biome: "house"}
	e.rooms[5] = &rooms.Room{RoomId: 5, Zone: "Deep", Biome: "cave"}

	m := newModule()
	m.store = e.store
	m.settings = parseSettings(func(name string) any { return testConfig[name] })
	m.onlineUsers = func() []*users.UserRecord { return e.users }
	m.companions = func(leader int) ([]member, bool) {
		roster, known := e.comps[leader]
		return roster, known || e.rosterAlwaysKnown
	}
	m.loadRoom = func(id int) *rooms.Room { return e.rooms[id] }
	m.isNight = func() bool { return e.night }
	m.drain = func(leader int, key survival.MemberKey, cost survival.Exertion) error {
		e.drains = append(e.drains, drainCall{leader, key, cost})
		return nil
	}
	e.m = m
	e.messages = captureMessages(t)
	return e
}

var testConfig = map[string]any{
	"Biomes": []any{
		map[any]any{"Biome": "snow", "Base": -12, "NightDrop": 10},
		map[any]any{"Biome": "desert", "Base": 36, "NightDrop": 22},
		map[any]any{"Biome": "forest", "Base": 12, "NightDrop": 8},
		map[any]any{"Biome": "cave", "Base": 10, "NightDrop": 0},
	},
}

func (e *env) addUser(t *testing.T, id, roomId int) *users.UserRecord {
	t.Helper()
	u := users.NewUserRecord(id, uint64(id))
	u.Character.Name = "Hero"
	u.Character.RoomId = roomId
	u.Character.Validate()
	u.Character.Health = u.Character.HealthMax.Value
	users.SetTestUser(u)
	e.users = append(e.users, u)
	return u
}

func captureMessages(t *testing.T) *[]string {
	t.Helper()
	messages := []string{}
	id := events.RegisterListener(events.Message{}, func(ev events.Event) events.ListenerReturn {
		messages = append(messages, ev.(events.Message).Text)
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	return &messages
}

func (e *env) text() string {
	events.ProcessEvents()
	return strings.Join(*e.messages, "\n")
}

func TestParseSettingsReadsConfigAndKeepsDefaults(t *testing.T) {
	s := parseSettings(func(name string) any {
		return map[string]any{
			"TickRounds": 3,
			"HeatFactor": 0.25,
			"Recovery":   -4, // invalid, keep default
			"Biomes":     []any{map[any]any{"Biome": "Snow", "Base": -20, "NightDrop": 5}, map[any]any{"Biome": ""}},
			"SlotWarmth": []any{map[any]any{"Slot": "body", "Warmth": 9}},
			"ComfortLow": 40, // above ComfortHigh: invalid band, keep defaults
		}[name]
	})
	assert.Equal(t, 3, s.TickRounds)
	assert.Equal(t, 0.25, s.Comfort.HeatFactor)
	assert.Equal(t, DefaultSettings().Exposure.Recovery, s.Exposure.Recovery)
	assert.Equal(t, climate.BiomeTemperature{Base: -20, NightDrop: 5}, s.Biomes["snow"])
	assert.Len(t, s.Biomes, 1)
	assert.Equal(t, map[string]int{"body": 9}, s.Warmth.SlotDefaults)
	assert.Equal(t, DefaultSettings().Comfort, climate.ComfortSettings{ComfortLow: s.Comfort.ComfortLow, ComfortHigh: s.Comfort.ComfortHigh, HeatFactor: DefaultSettings().Comfort.HeatFactor})
}

func TestNakedOnSnowfieldEscalatesToLethalWithDamage(t *testing.T) {
	e := setup(t)
	e.night = true
	u := e.addUser(t, 7, 1) // naked on a snowfield at night: -22°C, stress -38

	e.m.tick()
	assert.Equal(t, -19, e.m.exposureFor(7, survival.LeaderMemberKey))
	for i := 0; i < 10; i++ {
		e.m.tick()
	}
	assert.Equal(t, -100, e.m.exposureFor(7, survival.LeaderMemberKey))
	assert.True(t, hasActiveBuff(u.Character, coldBuffs[climate.BandCritical]))
	for _, band := range []climate.Band{climate.BandMild, climate.BandModerate, climate.BandSevere} {
		assert.False(t, hasActiveBuff(u.Character, coldBuffs[band]), "only the current band's buff is kept")
	}
	assert.Less(t, u.Character.Health, u.Character.HealthMax.Value, "severe and critical bands deal damage")
	out := e.text()
	assert.Contains(t, out, "You are getting chilled.")
	assert.Contains(t, out, "You are freezing to death!")
	assert.Contains(t, out, "The cold bites you for")

	// Moderate cold or worse drains fatigue; cold never drains thirst.
	require.NotEmpty(t, e.drains)
	last := e.drains[len(e.drains)-1]
	assert.Equal(t, survival.Exertion{Fatigue: 7}, last.cost)
}

func TestDownedCharacterTakesNoFurtherExposureDamage(t *testing.T) {
	e := setup(t)
	e.night = true
	u := e.addUser(t, 7, 1)
	e.m.registry.Exposure[7] = map[string]int{string(survival.LeaderMemberKey): -100}
	u.Character.Health = 0

	e.m.tick()
	assert.Equal(t, 0, u.Character.Health, "the normal bleed-out handles a downed character")
}

func TestWarmClothingKeepsSnowfieldSurvivable(t *testing.T) {
	e := setup(t)
	e.night = true
	u := e.addUser(t, 7, 1)
	u.Character.Equipment.Body = items.New(testFurCoat)           // warmth 30: comfortable down to -14°C
	require.NoError(t, u.Character.AddBuff(testWarmedBuff, true)) // +20: down to -34°C

	for i := 0; i < 20; i++ {
		e.m.tick()
	}
	assert.Equal(t, 0, e.m.exposureFor(7, survival.LeaderMemberKey))
	assert.Equal(t, u.Character.HealthMax.Value, u.Character.Health)
}

func TestModerateColdSettlesBelowLethal(t *testing.T) {
	e := setup(t)
	e.night = true
	u := e.addUser(t, 7, 3)                           // forest night 4°C
	u.Character.Equipment.Body = items.New(testTunic) // slot default 5 warmth: comfortable to 11°C, stress -7

	for i := 0; i < 50; i++ {
		e.m.tick()
	}
	assert.Equal(t, -28, e.m.exposureFor(7, survival.LeaderMemberKey), "mild mismatch settles at a penalty-only level")
	assert.True(t, hasActiveBuff(u.Character, coldBuffs[climate.BandMild]))
	assert.Equal(t, u.Character.HealthMax.Value, u.Character.Health)
}

func TestShelterAndFireSpeedRecovery(t *testing.T) {
	e := setup(t)
	e.addUser(t, 7, 4) // furnished house: 18°C, comfortable naked
	e.m.registry.Exposure[7] = map[string]int{string(survival.LeaderMemberKey): -60}
	e.m.tick()
	assert.Equal(t, -48, e.m.exposureFor(7, survival.LeaderMemberKey), "indoors recovers at the shelter rate")

	e2 := setup(t)
	e2.addUser(t, 8, 3) // forest noon 12°C, naked: stress -4
	climate.RegisterHeatSource(func(roomId int) bool { return roomId == 3 })
	e2.m.registry.Exposure[8] = map[string]int{string(survival.LeaderMemberKey): -60}
	e2.m.tick()
	assert.Equal(t, -48, e2.m.exposureFor(8, survival.LeaderMemberKey), "a fire warms the air and speeds recovery")
}

func TestHeatDrainsThirstAndHeavyClothingMakesItWorse(t *testing.T) {
	e := setup(t)
	naked := e.addUser(t, 7, 2)   // desert noon 36°C, naked: stress 6
	clothed := e.addUser(t, 8, 2) // fur coat 30: high edge 15°C, stress 21
	clothed.Character.Equipment.Body = items.New(testFurCoat)

	e.m.tick()
	byLeader := map[int]survival.Exertion{}
	for _, d := range e.drains {
		byLeader[d.leader] = d.cost
	}
	assert.Equal(t, survival.Exertion{Thirst: 1}, byLeader[7])
	assert.Equal(t, survival.Exertion{Thirst: 5}, byLeader[8])
	assert.Greater(t, e.m.exposureFor(8, survival.LeaderMemberKey), e.m.exposureFor(7, survival.LeaderMemberKey))
	_ = naked
}

func TestBandRecoverySwapsBuffAndAnnounces(t *testing.T) {
	e := setup(t)
	u := e.addUser(t, 7, 4) // indoors, comfortable
	e.m.registry.Exposure[7] = map[string]int{string(survival.LeaderMemberKey): -30}
	require.NoError(t, u.Character.AddBuff(coldBuffs[climate.BandMild], true))

	e.m.tick() // -30 -> -18: band none
	assert.False(t, hasActiveBuff(u.Character, coldBuffs[climate.BandMild]))
	assert.Contains(t, e.text(), "You feel warmer.")
}

func TestCompanionExposureUsesItsOwnClothingAndTellsLeader(t *testing.T) {
	e := setup(t)
	e.night = true
	e.addUser(t, 7, 4) // leader is indoors
	companion := &characters.Character{Name: "Bran", RoomId: 1, Buffs: buffs.New()}
	companion.HealthMax.Value, companion.Health = 50, 50
	e.comps[7] = []member{{Key: survival.CompanionMemberKey(1), Name: "Bran", Character: companion, RoomId: 1}}

	for i := 0; i < 2; i++ {
		e.m.tick()
	}
	assert.Equal(t, 0, e.m.exposureFor(7, survival.LeaderMemberKey))
	assert.Equal(t, -38, e.m.exposureFor(7, survival.CompanionMemberKey(1)))
	assert.True(t, hasActiveBuff(companion, coldBuffs[climate.BandMild]))
	assert.Contains(t, e.text(), "Bran is getting chilled.")
}

// TestBandBuffRevivedWhenBandReturnsBeforePrune is a regression test: a
// removed buff stays in the list (expired) until pruned, and the band
// returning in that window must re-activate it.
func TestBandBuffRevivedWhenBandReturnsBeforePrune(t *testing.T) {
	e := setup(t)
	u := e.addUser(t, 7, 4)
	syncBandBuff(u.Character, -30, 11)
	syncBandBuff(u.Character, 0, 11)
	require.False(t, hasActiveBuff(u.Character, coldBuffs[climate.BandMild]))
	syncBandBuff(u.Character, -30, 11)
	assert.True(t, hasActiveBuff(u.Character, coldBuffs[climate.BandMild]))
}

func TestDismissedCompanionIsForgotten(t *testing.T) {
	e := setup(t)
	e.rosterAlwaysKnown = true
	e.addUser(t, 7, 4)
	e.m.registry.Exposure[7] = map[string]int{string(survival.CompanionMemberKey(3)): -40}
	e.m.tick()
	_, stillTracked := e.m.registry.Exposure[7]
	assert.False(t, stillTracked)
}

func TestExposurePersistsAndReloadReappliesBuff(t *testing.T) {
	e := setup(t)
	e.night = true
	e.addUser(t, 7, 1)
	e.m.tick()
	e.m.tick()
	require.Equal(t, -38, e.store.saved.Exposure[7][string(survival.LeaderMemberKey)])

	// A restart: a fresh module loads the stored registry; the next tick
	// re-applies the band buff to the (reloaded) character.
	reloaded := setup(t)
	reloaded.store.saved = e.store.saved.clone()
	reloaded.m.load()
	reloaded.night = true
	u := reloaded.addUser(t, 7, 1)
	assert.Equal(t, -38, reloaded.m.exposureFor(7, survival.LeaderMemberKey))
	reloaded.m.tick() // -38 continues toward the lethal ceiling: -57
	assert.Equal(t, -57, reloaded.m.exposureFor(7, survival.LeaderMemberKey))
	assert.True(t, hasActiveBuff(u.Character, coldBuffs[climate.BandModerate]), "the fresh character gets the stored band's buff")
}

func TestTickSaveFailureIsLoggedNotFatal(t *testing.T) {
	e := setup(t)
	e.night = true
	e.addUser(t, 7, 1)
	e.store.saveErr = errors.New("disk full")
	e.m.tick()
	assert.Equal(t, 1, e.store.saveCalls)
	assert.Equal(t, -19, e.m.exposureFor(7, survival.LeaderMemberKey), "live state still advances; the next tick retries the save")
}

func TestDecodeRegistryDropsInvalidEntries(t *testing.T) {
	var r Registry
	require.NoError(t, decodeRegistry([]byte("exposure:\n  7:\n    leader: -250\n    companion:2: 30\n    bogus: 10\n    companion:3: 0\n  0:\n    leader: 5\n"), &r))
	assert.Equal(t, map[int]map[string]int{7: {"leader": -100, "companion:2": 30}}, r.Exposure)
}

func TestTemperatureCommandReport(t *testing.T) {
	e := setup(t)
	e.night = true
	u := e.addUser(t, 7, 1)
	u.Character.Equipment.Body = items.New(testTunic)
	e.m.registry.Exposure[7] = map[string]int{string(survival.LeaderMemberKey): -55}
	e.comps[7] = []member{{Key: survival.CompanionMemberKey(1), Name: "Bran", Character: &characters.Character{Buffs: buffs.New()}, RoomId: 1}}

	_, err := e.m.userCommand("", u, e.rooms[1], 0)
	require.NoError(t, err)
	out := e.text()
	assert.Contains(t, out, "Air temperature here: -22°C (freezing).")
	assert.Contains(t, out, "it is night")
	assert.Contains(t, out, "Your clothing gives 5 warmth: you are comfortable from 11°C to 27°C.")
	assert.Contains(t, out, "You are frostbitten (exposure 55/100).")
	assert.Contains(t, out, "Bran is comfortable.")
}

func TestAirTemperatureProvider(t *testing.T) {
	e := setup(t)
	temp, ok := e.m.AirTemperatureIn(5)
	require.True(t, ok)
	assert.Equal(t, 10, temp, "a cave holds its base temperature")
	_, ok = e.m.AirTemperatureIn(999)
	assert.False(t, ok)
}

func TestOnNewRoundTicksOnlyOnInterval(t *testing.T) {
	e := setup(t)
	e.night = true
	e.addUser(t, 7, 1)
	e.m.onNewRound(events.NewRound{RoundNumber: 4})
	assert.Equal(t, 0, e.m.exposureFor(7, survival.LeaderMemberKey))
	e.m.onNewRound(events.NewRound{RoundNumber: 5})
	assert.Equal(t, -19, e.m.exposureFor(7, survival.LeaderMemberKey))
}

type stormProvider struct{}

func (stormProvider) CurrentCondition(zone string) (weather.Condition, bool) {
	if zone != "Tundra" {
		return weather.Condition{}, false
	}
	return weather.Condition{Name: "blizzard", Description: "A blizzard.", TravelDurationPct: 100, ExertionPct: 100, RestRecoveryPct: 100, TemperatureMod: -8}, true
}

// TestTickWiringWithRealRoomClockAndWeather drives a tick through the
// module's production seams for rooms, the clock, and weather: the room
// comes from the room manager, night from the shared clock, and the
// temperature modifier from the registered weather provider.
func TestTickWiringWithRealRoomClockAndWeather(t *testing.T) {
	e := setup(t)
	e.m.loadRoom = rooms.LoadRoom
	e.m.isNight = func() bool { return gametime.GetDate().Night }
	room := &rooms.Room{RoomId: 93001, Zone: "Tundra", Biome: "snow"}
	rooms.SetTestRoom(room)
	t.Cleanup(func() { rooms.RemoveTestRoom(93001) })
	weather.SetProvider(stormProvider{})
	t.Cleanup(func() { weather.SetProvider(nil) })

	saved := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCount(saved) })
	day := uint64(0)
	for gametime.GetDate(day).Night {
		day += 5
	}
	util.SetRoundCount(day)

	u := e.addUser(t, 7, 93001)
	u.Character.Equipment.Body = items.New(testTunic)

	temp, ok := e.m.AirTemperatureIn(93001)
	require.True(t, ok)
	assert.Equal(t, -20, temp, "snow base -12 by day, blizzard -8")

	e.m.tick() // warmth 5: comfortable down to 11°C, stress -31: grows 15
	assert.Equal(t, -15, e.m.exposureFor(7, survival.LeaderMemberKey))
}

func TestUnspawnedCompanionKeepsStoredExposure(t *testing.T) {
	e := setup(t)
	e.addUser(t, 7, 4)
	key := string(survival.CompanionMemberKey(1))
	e.m.registry.Exposure[7] = map[string]int{key: -60}
	// On the roster but not spawned yet (e.g. mid-restore after a restart).
	e.comps[7] = []member{{Key: survival.CompanionMemberKey(1), Name: "Bran"}}
	e.m.tick()
	assert.Equal(t, -60, e.m.registry.Exposure[7][key], "untouched until the companion is back")

	// With no roster provider at all, stored companion exposure is kept too.
	delete(e.comps, 7)
	e.m.tick()
	assert.Equal(t, -60, e.m.registry.Exposure[7][key])
}

func TestExposureDamage(t *testing.T) {
	cases := []struct {
		name                            string
		band                            climate.Band
		health, max, pct, regen, expect int
	}{
		{"no damage below severe", climate.BandModerate, 50, 50, 2, 0, 0},
		{"severe wears down", climate.BandSevere, 50, 100, 2, 0, 2},
		{"severe never downs", climate.BandSevere, 2, 100, 2, 0, 1},
		{"severe at 1 HP deals nothing", climate.BandSevere, 1, 100, 2, 0, 0},
		{"critical outpaces regen", climate.BandCritical, 50, 50, 10, 5, 10},
		{"critical at least 1 plus regen", climate.BandCritical, 5, 5, 10, 3, 4},
		{"downed takes none", climate.BandCritical, 0, 50, 10, 5, 0},
	}
	for _, c := range cases {
		if got := exposureDamage(c.band, c.health, c.max, c.pct, c.regen); got != c.expect {
			t.Errorf("%s: got %d want %d", c.name, got, c.expect)
		}
	}
}

func TestSevereColdNeverDownsAPlayer(t *testing.T) {
	e := setup(t)
	u := e.addUser(t, 7, 1)
	u.Character.Equipment.Body = items.New(testTunic) // by day -12°C, warmth 5: stress -23, settles at -92 (severe)
	for i := 0; i < 200; i++ {
		e.m.tick()
	}
	assert.Equal(t, -92, e.m.exposureFor(7, survival.LeaderMemberKey))
	assert.GreaterOrEqual(t, u.Character.Health, 1, "only the critical band can down anyone")
}

func TestPlayerDeathClearsExposure(t *testing.T) {
	e := setup(t)
	u := e.addUser(t, 7, 1)
	e.m.registry.Exposure[7] = map[string]int{string(survival.LeaderMemberKey): -100, string(survival.CompanionMemberKey(1)): -40}
	syncBandBuff(u.Character, -100, 11)

	e.m.onPlayerDeath(events.PlayerDeath{UserId: 7})
	assert.Equal(t, 0, e.m.exposureFor(7, survival.LeaderMemberKey))
	assert.Equal(t, -40, e.m.exposureFor(7, survival.CompanionMemberKey(1)), "companions keep theirs")
	assert.False(t, hasActiveBuff(u.Character, coldBuffs[climate.BandCritical]))
	assert.Equal(t, 1, e.store.saveCalls)
}

func TestIndoorTaggedCaveChamberIsFurnished(t *testing.T) {
	e := setup(t)
	e.rooms[6] = &rooms.Room{RoomId: 6, Zone: "Deep", Biome: "cave", Tags: []string{rooms.TagIndoor}}
	temp, ok := e.m.AirTemperatureIn(6)
	require.True(t, ok)
	assert.Equal(t, 18, temp, "a tagged, furnished chamber in a cave is kept at indoor temperature")
}

// TestShippedBandBuffsPenaliseWithoutShrinkingMaxHealth loads the module's
// real band-buff files and the real game config. Regression: the severe and
// critical buffs once cut vitality by 30, which with HPPerVitality 4 wiped
// out max HP and made a penalty band lethal.
func TestShippedBandBuffsPenaliseWithoutShrinkingMaxHealth(t *testing.T) {
	e := setup(t)
	_, thisFile, _, _ := runtime.Caller(0)
	t.Chdir(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	require.NoError(t, configs.ReloadConfig())

	paths, err := fs.Glob(files, "files/datafiles/buffs/*.yaml")
	require.NoError(t, err)
	require.Len(t, paths, 8)
	for _, path := range paths {
		data, err := files.ReadFile(path)
		require.NoError(t, err)
		var spec buffs.BuffSpec
		require.NoError(t, yaml.Unmarshal(data, &spec))
		require.NoError(t, spec.Validate(), path)
		require.True(t, strings.HasSuffix(path, spec.Filepath()), "plugin loader requires the id-name filename: %s", path)
		buffs.SetTestBuffSpec(&spec)
	}

	u := e.addUser(t, 7, 1)
	u.Character.Level = 5
	u.Character.Stats.Vitality.Base = 10
	u.Character.Validate()
	baseMax := u.Character.HealthMax.Value
	require.Greater(t, baseMax, 1)

	for _, exposure := range []int{-30, -60, -80, -100, 30, 60, 80, 100} {
		syncBandBuff(u.Character, exposure, 11)
		assert.Equal(t, baseMax, u.Character.HealthMax.Value, "band at exposure %d must not shrink max HP", exposure)
		assert.Less(t, u.Character.StatMod("speed"), 0, "band at exposure %d penalises speed", exposure)
	}
}

// TestBandBuffSurvivesPermabuffReconciliation is a regression test: a
// permanent buff with no item/race/pet source is stripped whenever the
// engine reconciles permabuffs (equip, remove, login). Band buffs must not
// be.
func TestBandBuffSurvivesPermabuffReconciliation(t *testing.T) {
	e := setup(t)
	u := e.addUser(t, 7, 1)
	syncBandBuff(u.Character, -60, 11)
	u.Character.Validate(true) // the login path reconciles permabuffs
	assert.True(t, hasActiveBuff(u.Character, coldBuffs[climate.BandModerate]))
}

// TestExposureOfThroughClimateSeam reads a member's exposure through the
// registered climate seam, as modules/walking does.
func TestExposureOfThroughClimateSeam(t *testing.T) {
	e := setup(t)
	climate.SetProvider(nil)
	_, ok := climate.ExposureOf(7, string(survival.LeaderMemberKey))
	assert.False(t, ok, "absent without a provider")

	climate.SetProvider(e.m)
	t.Cleanup(func() { climate.SetProvider(nil) })
	e.m.registry.Exposure[7] = map[string]int{string(survival.CompanionMemberKey(2)): -55}
	value, ok := climate.ExposureOf(7, string(survival.CompanionMemberKey(2)))
	require.True(t, ok)
	assert.Equal(t, -55, value)
	value, ok = climate.ExposureOf(7, string(survival.LeaderMemberKey))
	assert.True(t, ok, "a comfortable member is known: 0 (Phase 26a)")
	assert.Zero(t, value)
	e.m.loadErr = assert.AnError
	_, ok = climate.ExposureOf(7, string(survival.LeaderMemberKey))
	assert.False(t, ok, "unreadable data: unknown")
}
