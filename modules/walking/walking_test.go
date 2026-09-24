package walking

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/walking"
	"github.com/GoMudEngine/GoMud/internal/weather"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "low", "", false)
	os.Exit(m.Run())
}

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
	m         *WalkingModule
	store     *fakeStore
	rooms     map[int]*rooms.Room
	users     map[int]*users.UserRecord
	comps     map[int][]member
	exposure  map[string]int
	band      *encumbrance.LoadBand
	condition map[string]weather.Condition
	mountPct  int
	riders    int
	fatigue   map[survival.MemberKey]int
	drains    []drainCall
	messages  *[]string
}

// Rooms: 1 forest (Wood), 2 forest (Wood), 3 city (Town), 4 snow (Tundra).
func setup(t *testing.T) *env {
	t.Helper()
	buffs.SetTestFlag(WellRestedFlag)
	buffs.SetTestFlag(RestedFlag)
	for _, id := range []int{WellRestedBuffId, ExhaustedBuffId, CollapsedBuffId, RestedBuffId} {
		spec := &buffs.BuffSpec{BuffId: id, Name: "band", TriggerCount: 1}
		if id == WellRestedBuffId {
			spec.Flags = []string{WellRestedFlag}
			spec.TriggerCount = 450
		}
		if id == RestedBuffId {
			spec.Flags = []string{RestedFlag}
			spec.TriggerCount = 225
		}
		buffs.SetTestBuffSpec(spec)
	}
	for _, b := range []*rooms.BiomeInfo{
		{BiomeId: "forest", Name: "Forest", Symbol: "f"},
		{BiomeId: "city", Name: "City", Symbol: "c", LitArea: true},
		{BiomeId: "snow", Name: "Snow", Symbol: "*"},
	} {
		rooms.SetTestBiome(b)
	}
	users.ResetActiveUsers()
	t.Cleanup(func() {
		for _, id := range []int{WellRestedBuffId, ExhaustedBuffId, CollapsedBuffId, RestedBuffId} {
			buffs.RemoveTestBuffSpec(id)
		}
		for _, id := range []string{"forest", "city", "snow"} {
			rooms.RemoveTestBiome(id)
		}
		users.ResetActiveUsers()
	})

	e := &env{
		store:     &fakeStore{},
		rooms:     map[int]*rooms.Room{},
		users:     map[int]*users.UserRecord{},
		comps:     map[int][]member{},
		exposure:  map[string]int{},
		condition: map[string]weather.Condition{},
		fatigue:   map[survival.MemberKey]int{},
	}
	e.rooms[1] = &rooms.Room{RoomId: 1, Zone: "Wood", Biome: "forest"}
	e.rooms[2] = &rooms.Room{RoomId: 2, Zone: "Wood", Biome: "forest"}
	e.rooms[3] = &rooms.Room{RoomId: 3, Zone: "Town", Biome: "city"}
	e.rooms[4] = &rooms.Room{RoomId: 4, Zone: "Tundra", Biome: "snow"}

	m := newModule()
	m.store = e.store
	m.lookupUser = func(id int) *users.UserRecord { return e.users[id] }
	m.onlineUsers = func() []*users.UserRecord {
		out := []*users.UserRecord{}
		for _, u := range e.users {
			out = append(out, u)
		}
		return out
	}
	m.companions = func(leader int) ([]member, bool) {
		roster, known := e.comps[leader]
		return roster, known
	}
	m.loadRoom = func(id int) *rooms.Room { return e.rooms[id] }
	m.weatherIn = func(zone string) (weather.Condition, bool) {
		c, ok := e.condition[zone]
		return c, ok
	}
	m.loadBand = func(int) (encumbrance.LoadBand, bool) {
		if e.band == nil {
			return encumbrance.LoadBand{}, false
		}
		return *e.band, true
	}
	m.mountRelief = func(int) (int, int) {
		if e.mountPct == 0 {
			return 100, 0
		}
		return e.mountPct, e.riders
	}
	m.exposureOf = func(_ int, key string) (int, bool) {
		v, ok := e.exposure[key]
		return v, ok
	}
	m.drain = func(leader int, key survival.MemberKey, cost survival.Exertion) error {
		e.drains = append(e.drains, drainCall{leader, key, cost})
		return nil
	}
	m.companyNeeds = func(int) []survival.MemberNeeds {
		out := []survival.MemberNeeds{}
		for key, value := range e.fatigue {
			out = append(out, survival.MemberNeeds{Key: key, Needs: survival.Needs{Hunger: 100, Thirst: 100, Fatigue: value}})
		}
		return out
	}
	e.m = m
	e.messages = captureMessages(t)
	return e
}

func (e *env) addUser(t *testing.T, id, roomId int) *users.UserRecord {
	t.Helper()
	u := users.NewUserRecord(id, uint64(id))
	u.Character.Name = "Hero"
	u.Character.RoomId = roomId
	u.Character.Validate()
	users.SetTestUser(u)
	e.users[id] = u
	return u
}

// addCompanion adds a spawned companion in roomId (0 = not spawned).
func (e *env) addCompanion(leader, id int, name string, roomId int) *characters.Character {
	mb := member{Key: survival.CompanionMemberKey(id), CompanionID: id, Name: name}
	if roomId != 0 {
		c := characters.New()
		c.Name = name
		c.RoomId = roomId
		c.Validate()
		mb.Character = c
		mb.RoomId = roomId
	}
	e.comps[leader] = append(e.comps[leader], mb)
	return mb.Character
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

// fatigueDrained sums drained fatigue per member.
func (e *env) fatigueDrained() map[survival.MemberKey]int {
	out := map[survival.MemberKey]int{}
	for _, d := range e.drains {
		out[d.key] += d.cost.Fatigue
	}
	return out
}

func (e *env) steps(leader, from, to, n int) {
	for i := 0; i < n; i++ {
		e.m.Stepped(leader, from, to)
	}
}

func TestParseSettingsReadsConfigAndKeepsDefaults(t *testing.T) {
	s := parseSettings(func(name string) any {
		return map[string]any{
			"TickRounds":    3,
			"DefaultStrain": 30,
			"Settlements":   []any{"city", "Hamlet"},
			"Biomes":        []any{map[any]any{"Biome": "Forest", "Strain": 60}, map[any]any{"Biome": "bog", "Strain": -1}},
			"ChilledPct":    110,
			"WellRestedPct": 40,
			"RestedPct":     80,
		}[name]
	})
	assert.Equal(t, 3, s.TickRounds)
	assert.Equal(t, 30, s.Terrain.Default)
	assert.Equal(t, map[string]bool{"city": true, "hamlet": true}, s.Terrain.Settlements)
	assert.Equal(t, map[string]int{"forest": 60}, s.Terrain.Biomes, "the invalid entry is dropped")
	assert.Equal(t, 110, s.Cold.ChilledPct)
	assert.Equal(t, 150, s.Cold.FrostbittenPct, "unset keeps the default")
	assert.Equal(t, 40, s.WellRestedPct)
	assert.Equal(t, 80, s.RestedPct)

	d := parseSettings(func(string) any { return nil })
	assert.Equal(t, DefaultSettings(), d)
}

// TestShippedConfigMatchesDefaults parses the real overlay.
func TestShippedConfigMatchesDefaults(t *testing.T) {
	data, err := files.ReadFile("files/data-overlays/config.yaml")
	require.NoError(t, err)
	var cfg map[string]any
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	s := parseSettings(func(name string) any { return cfg[name] })
	assert.Equal(t, DefaultSettings(), s)
}

func TestSteppedDrainsEachMemberAndKeepsCarries(t *testing.T) {
	e := setup(t)
	e.addUser(t, 7, 1)
	e.addCompanion(7, 1, "Bran", 1)

	e.steps(7, 1, 2, 3) // forest: 50 each step
	drained := e.fatigueDrained()
	assert.Equal(t, 1, drained[survival.LeaderMemberKey])
	assert.Equal(t, 1, drained[survival.CompanionMemberKey(1)])
	assert.Equal(t, 50, e.m.registry.Carry[7][string(survival.LeaderMemberKey)])
	assert.Equal(t, 50, e.m.registry.Carry[7][string(survival.CompanionMemberKey(1))])
	assert.Zero(t, e.store.saveCalls, "steps never write; the save callback does")
}

func TestCarryPersistedBySaveAndReloaded(t *testing.T) {
	e := setup(t)
	e.addUser(t, 7, 1)
	e.steps(7, 1, 2, 1)
	require.NoError(t, e.m.save())
	assert.Equal(t, 50, e.store.saved.Carry[7][string(survival.LeaderMemberKey)])
	require.NoError(t, e.m.save())
	assert.Equal(t, 1, e.store.saveCalls, "a clean registry isn't rewritten")

	reloaded := newModule()
	reloaded.store = e.store
	reloaded.load()
	require.NoError(t, reloaded.loadErr)
	assert.Equal(t, 50, reloaded.registry.Carry[7][string(survival.LeaderMemberKey)])

	var decoded Registry
	require.NoError(t, decodeRegistry([]byte("carry:\n  7:\n    leader: 40\n    companion:2: 150\n    bogus: 10\n  0:\n    leader: 5\n"), &decoded))
	assert.Equal(t, map[int]map[string]int{7: {"leader": 40}}, decoded.Carry)
}

func TestWellRestedHalvesStrain(t *testing.T) {
	e := setup(t)
	u := e.addUser(t, 7, 1)
	require.NoError(t, u.Character.AddBuff(WellRestedBuffId, false))
	e.steps(7, 1, 2, 4) // forest 50 × 50% = 25
	assert.Equal(t, 1, e.fatigueDrained()[survival.LeaderMemberKey])
	assert.Zero(t, e.m.registry.Carry[7][string(survival.LeaderMemberKey)])
}

// TestRestedCutsStrainByAQuarter: the camp tier (Phase 23a).
func TestRestedCutsStrainByAQuarter(t *testing.T) {
	e := setup(t)
	u := e.addUser(t, 7, 1)
	require.NoError(t, u.Character.AddBuff(RestedBuffId, false))
	e.steps(7, 1, 2, 4) // forest 50 × 75% = 38 (37.5 rounds half up) × 4 = 152
	assert.Equal(t, 1, e.fatigueDrained()[survival.LeaderMemberKey])
	assert.Equal(t, 52, e.m.registry.Carry[7][string(survival.LeaderMemberKey)])
}

// TestWellRestedWinsOverRested: a member somehow holding both tiers walks
// at the better one, never both.
func TestWellRestedWinsOverRested(t *testing.T) {
	e := setup(t)
	u := e.addUser(t, 7, 1)
	require.NoError(t, u.Character.AddBuff(RestedBuffId, false))
	require.NoError(t, u.Character.AddBuff(WellRestedBuffId, false))
	e.steps(7, 1, 2, 4) // forest 50 × 50% = 25, not 50 × 50% × 75%
	assert.Equal(t, 1, e.fatigueDrained()[survival.LeaderMemberKey])
	assert.Zero(t, e.m.registry.Carry[7][string(survival.LeaderMemberKey)])

	lines := strings.Join(e.m.report(u, e.rooms[1]), "\n")
	assert.Contains(t, lines, "well rested 50%")
	assert.NotContains(t, lines, "rested 75%")
}

func TestStrainViewNamesRestedTier(t *testing.T) {
	e := setup(t)
	u := e.addUser(t, 7, 1)
	require.NoError(t, u.Character.AddBuff(RestedBuffId, false))
	lines := strings.Join(e.m.report(u, e.rooms[1]), "\n")
	assert.Contains(t, lines, "You: 38 strain per step")
	assert.Contains(t, lines, "— rested 75%")
}

func TestColdExposureRaisesStrain(t *testing.T) {
	e := setup(t)
	e.addUser(t, 7, 1)
	e.exposure[string(survival.LeaderMemberKey)] = -55 // frostbitten: 150%
	e.steps(7, 1, 2, 2)                                // 75 × 2 = 150
	assert.Equal(t, 1, e.fatigueDrained()[survival.LeaderMemberKey])
	assert.Equal(t, 50, e.m.registry.Carry[7][string(survival.LeaderMemberKey)])
}

func TestHeatExposureAddsNoStrain(t *testing.T) {
	e := setup(t)
	e.addUser(t, 7, 1)
	e.exposure[string(survival.LeaderMemberKey)] = 80
	e.steps(7, 1, 2, 1)
	assert.Equal(t, 50, e.m.registry.Carry[7][string(survival.LeaderMemberKey)])
}

// TestMountReliefCoversTwoRiders: the leader and the lowest companion id
// ride; a third member walks at full cost; an unspawned companion never
// takes a seat.
func TestMountReliefCoversTwoRiders(t *testing.T) {
	e := setup(t)
	e.addUser(t, 7, 1)
	e.addCompanion(7, 5, "Cara", 1)
	e.addCompanion(7, 2, "Absent", 0) // lowest id, but not spawned
	e.addCompanion(7, 3, "Bran", 1)
	e.mountPct, e.riders = 50, 2

	e.steps(7, 1, 2, 4) // riders 25/step, walker 50/step
	drained := e.fatigueDrained()
	assert.Equal(t, 1, drained[survival.LeaderMemberKey], "the leader rides")
	assert.Equal(t, 1, drained[survival.CompanionMemberKey(3)], "the lowest spawned companion id rides")
	assert.Equal(t, 2, drained[survival.CompanionMemberKey(5)], "a third member walks at full cost")
	assert.Zero(t, drained[survival.CompanionMemberKey(2)], "an unspawned companion didn't walk")
}

func TestCompanionElsewhereIsNotCharged(t *testing.T) {
	e := setup(t)
	e.addUser(t, 7, 1)
	e.addCompanion(7, 1, "Stray", 4)
	e.steps(7, 1, 2, 2)
	assert.Zero(t, e.fatigueDrained()[survival.CompanionMemberKey(1)])
}

func TestLoadAndWeatherMultiply(t *testing.T) {
	e := setup(t)
	e.addUser(t, 7, 1)
	e.band = &encumbrance.LoadBand{TravelDurationPct: 125, FatiguePct: 115}
	e.condition["Wood"] = weather.Condition{Name: "rain", ExertionPct: 115}
	e.steps(7, 1, 2, 1) // 50 × 1.15 × 1.15 = 66
	assert.Equal(t, 66, e.m.registry.Carry[7][string(survival.LeaderMemberKey)])

	// Indoors, the weather doesn't reach you.
	e.rooms[2].Tags = []string{"strain:50", "indoor"}
	e.steps(7, 1, 2, 1) // 50 × 1.15 = 58 (57.5 half up)
	assert.Equal(t, 24, e.m.registry.Carry[7][string(survival.LeaderMemberKey)], "66 + 58 = 124")
}

func TestSettlementStepCostsNothingAndTouchesNoSurvival(t *testing.T) {
	e := setup(t)
	e.addUser(t, 7, 1)
	e.addCompanion(7, 1, "Bran", 1)
	e.band = &encumbrance.LoadBand{FatiguePct: 130}
	e.steps(7, 1, 3, 20)
	assert.Empty(t, e.drains)
	assert.Empty(t, e.m.registry.Carry)
	assert.False(t, e.m.dirty)
}

func TestPruningDropsMembersWhoLeftTheRoster(t *testing.T) {
	e := setup(t)
	e.addUser(t, 7, 1)
	e.m.registry.Carry[7] = map[string]int{string(survival.CompanionMemberKey(9)): 40}
	e.comps[7] = []member{} // known, empty roster
	e.steps(7, 1, 2, 1)
	assert.NotContains(t, e.m.registry.Carry[7], string(survival.CompanionMemberKey(9)))

	// With no roster provider at all, stored companion carries are kept.
	e.m.registry.Carry[7][string(survival.CompanionMemberKey(9))] = 40
	delete(e.comps, 7)
	e.steps(7, 1, 2, 1)
	assert.Equal(t, 40, e.m.registry.Carry[7][string(survival.CompanionMemberKey(9))])
}

func TestTickAppliesAndSwapsBandBuffs(t *testing.T) {
	e := setup(t)
	u := e.addUser(t, 7, 1)
	bran := e.addCompanion(7, 1, "Bran", 1)
	e.fatigue[survival.LeaderMemberKey] = 20
	e.fatigue[survival.CompanionMemberKey(1)] = 60

	e.m.tick()
	assert.True(t, hasActiveBuff(u.Character, ExhaustedBuffId))
	assert.False(t, hasActiveBuff(bran, ExhaustedBuffId))
	assert.Contains(t, e.text(), "You are exhausted")

	e.fatigue[survival.LeaderMemberKey] = 0
	e.fatigue[survival.CompanionMemberKey(1)] = 25
	e.m.tick()
	assert.True(t, hasActiveBuff(u.Character, CollapsedBuffId))
	assert.False(t, hasActiveBuff(u.Character, ExhaustedBuffId), "the old band buff is swapped out")
	assert.True(t, hasActiveBuff(bran, ExhaustedBuffId))
	text := e.text()
	assert.Contains(t, text, "You collapse with exhaustion")
	assert.Contains(t, text, "Bran is exhausted")

	e.fatigue[survival.LeaderMemberKey] = 80
	e.m.tick()
	assert.False(t, hasActiveBuff(u.Character, CollapsedBuffId))
	assert.Contains(t, e.text(), "You feel less exhausted")
}

func TestOnNewRoundTicksOnlyOnInterval(t *testing.T) {
	e := setup(t)
	u := e.addUser(t, 7, 1)
	e.fatigue[survival.LeaderMemberKey] = 10
	e.m.onNewRound(events.NewRound{RoundNumber: 4})
	assert.False(t, hasActiveBuff(u.Character, ExhaustedBuffId))
	e.m.onNewRound(events.NewRound{RoundNumber: 5})
	assert.True(t, hasActiveBuff(u.Character, ExhaustedBuffId))
}

func TestBandBuffSurvivesPermabuffReconciliation(t *testing.T) {
	e := setup(t)
	u := e.addUser(t, 7, 1)
	syncBandBuff(u.Character, 10, 11)
	u.Character.Validate(true)
	assert.True(t, hasActiveBuff(u.Character, ExhaustedBuffId))
}

// TestShippedBuffsLoadAndNeverTouchVitality loads the module's real buff
// and flag files with the real game config.
func TestShippedBuffsLoadAndNeverTouchVitality(t *testing.T) {
	e := setup(t)
	_, thisFile, _, _ := runtime.Caller(0)
	t.Chdir(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	require.NoError(t, configs.ReloadConfig())

	for path, want := range map[string]string{
		"files/datafiles/buffs-flags/well-rested.yaml": WellRestedFlag,
		"files/datafiles/buffs-flags/rested.yaml":      RestedFlag,
	} {
		flagData, err := files.ReadFile(path)
		require.NoError(t, err)
		var flag buffs.FlagSpec
		require.NoError(t, yaml.Unmarshal(flagData, &flag))
		require.NoError(t, flag.Validate())
		assert.Equal(t, want, flag.Flag)
		assert.True(t, strings.HasSuffix(path, flag.Filepath()))
	}

	paths, err := fs.Glob(files, "files/datafiles/buffs/*.yaml")
	require.NoError(t, err)
	require.Len(t, paths, 4)
	for _, path := range paths {
		data, err := files.ReadFile(path)
		require.NoError(t, err)
		var spec buffs.BuffSpec
		require.NoError(t, yaml.Unmarshal(data, &spec))
		require.NoError(t, spec.Validate(), path)
		require.True(t, strings.HasSuffix(path, spec.Filepath()), "plugin loader requires the id-name filename: %s", path)
		_, hasVitality := spec.StatMods["vitality"]
		assert.False(t, hasVitality, "%s must not touch vitality", path)
		buffs.SetTestBuffSpec(&spec)
	}

	u := e.addUser(t, 7, 1)
	u.Character.Level = 5
	u.Character.Stats.Vitality.Base = 10
	u.Character.Validate()
	baseMax := u.Character.HealthMax.Value
	for _, fatigue := range []int{20, 0} {
		syncBandBuff(u.Character, fatigue, 11)
		assert.Equal(t, baseMax, u.Character.HealthMax.Value)
		assert.Less(t, u.Character.StatMod("speed"), 0)
	}
	syncBandBuff(u.Character, 100, 11)
	require.NoError(t, u.Character.AddBuff(WellRestedBuffId, false))
	assert.True(t, u.Character.HasBuffFlag(WellRestedFlag))
	assert.Equal(t, 450, u.Character.GetBuffs(WellRestedBuffId)[0].TriggersLeft)
	require.NoError(t, u.Character.AddBuff(RestedBuffId, false))
	assert.True(t, u.Character.HasBuffFlag(RestedFlag))
	assert.Equal(t, "Rested", u.Character.GetBuffs(RestedBuffId)[0].Name())
	assert.Equal(t, 225, u.Character.GetBuffs(RestedBuffId)[0].TriggersLeft, "15 minutes at 4-second rounds")
}

func TestStrainCommandListsCostAndFactors(t *testing.T) {
	e := setup(t)
	u := e.addUser(t, 7, 4) // snow, 90
	require.NoError(t, u.Character.AddBuff(WellRestedBuffId, false))
	e.addCompanion(7, 1, "Bran", 4)
	e.addCompanion(7, 2, "Cara", 4)
	e.band = &encumbrance.LoadBand{FatiguePct: 130}
	e.exposure[string(survival.CompanionMemberKey(1))] = -55
	e.mountPct, e.riders = 75, 2

	lines := strings.Join(e.m.report(u, e.rooms[4]), "\n")
	assert.Contains(t, lines, "Walking here (snow) costs 90 strain per step")
	assert.Contains(t, lines, "load 130%")
	assert.Contains(t, lines, "mount 75% for 2 rider(s)")
	assert.Contains(t, lines, "You: 44 strain per step")   // 90 × 1.3 × .75 × .5 = 43.9
	assert.Contains(t, lines, "Bran: 132 strain per step") // 90 × 1.3 × 1.5 × .75 = 131.6
	assert.Contains(t, lines, "Cara: 117 strain per step") // 90 × 1.3 = 117, walking
	assert.Contains(t, lines, "riding, well rested 50%")
	assert.Contains(t, lines, "cold 150%")

	town := strings.Join(e.m.report(u, e.rooms[3]), "\n")
	assert.Contains(t, town, "costs no fatigue")

	_, err := e.m.userCommand("", u, e.rooms[4], 0)
	require.NoError(t, err)
	assert.Contains(t, e.text(), "strain per step")
}

func TestSteppedWithoutProviderDataIsSafe(t *testing.T) {
	e := setup(t)
	e.m.Stepped(99, 1, 2) // unknown user
	e.addUser(t, 7, 1)
	e.m.Stepped(7, 1, 999) // unknown room
	assert.Empty(t, e.drains)
	_ = walking.CentiPerPoint
}
