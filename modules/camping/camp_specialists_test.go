package camping

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// fakeCargo is an encumbrance provider with a cargo and a capacity, keeping
// deposit operations once, as the real module does.
type fakeCargo struct {
	capacity int
	stacks   map[int]int
	applied  map[string]bool
	grams    int
}

func newFakeCargo(capacity int) *fakeCargo {
	return &fakeCargo{capacity: capacity, stacks: map[int]int{}, applied: map[string]bool{}}
}

func (f *fakeCargo) CurrentLoad(int) (encumbrance.Load, bool) {
	return encumbrance.Load{CargoGrams: f.grams, CapacityGrams: f.capacity, MemberCapacityGrams: f.capacity}, true
}
func (f *fakeCargo) CargoContents(int) []encumbrance.CargoStack {
	var out []encumbrance.CargoStack
	for id, n := range f.stacks {
		if n > 0 {
			out = append(out, encumbrance.CargoStack{ItemId: id, Count: n})
		}
	}
	return out
}
func (f *fakeCargo) ConsumeCargoUse(int, int) error { return nil }
func (f *fakeCargo) DepositCargo(_ int, op string, deposits []encumbrance.CargoStack) error {
	if op != "" && f.applied[op] {
		return nil
	}
	for _, d := range deposits {
		f.stacks[d.ItemId] += d.Count
		if spec := items.GetItemSpec(d.ItemId); spec != nil {
			f.grams += spec.Weight * d.Count
		}
	}
	if op != "" {
		f.applied[op] = true
	}
	return nil
}
func (f *fakeCargo) WithdrawCargo(_ int, itemID, count int) error {
	if f.stacks[itemID] < count {
		return encumbrance.ErrInsufficientCargo
	}
	f.stacks[itemID] -= count
	return nil
}

func useCargo(t *testing.T, f *fakeCargo) {
	t.Helper()
	encumbrance.SetProvider(f)
	t.Cleanup(func() { encumbrance.SetProvider(nil) })
}

// loyaltyStub is a company formation provider that keeps companions'
// loyalty and vigil operations.
type loyaltyStub struct {
	loyalty map[int]int
	names   map[int]string
	applied map[string]bool
}

func (l *loyaltyStub) FormationFor(int) (company.Formation, bool)                 { return company.Formation{}, false }
func (l *loyaltyStub) InstanceFor(int, int) (int, bool)                            { return 0, false }
func (l *loyaltyStub) LeaderAndKeyForInstance(int) (int, company.MemberKey, bool) { return 0, "", false }
func (l *loyaltyStub) CompanionArchetype(int, int) (string, bool)                  { return "", false }
func (l *loyaltyStub) RaiseLoyaltyOnce(_ int, op string, ids []int, delta, cap int) ([]string, error) {
	if l.applied[op] {
		return nil, nil
	}
	l.applied[op] = true
	var raised []string
	for _, id := range ids {
		if l.loyalty[id] < cap {
			l.loyalty[id] = min(cap, l.loyalty[id]+delta)
			raised = append(raised, l.names[id])
		}
	}
	return raised, nil
}

// specialistsAt is a specialist seam with fixed specialists by utility.
func specialistsAt(t *testing.T, roomID int, byUtility map[string]archetypes.Specialist) func(int, string, ...int) (archetypes.Specialist, bool) {
	return func(_ int, utility string, roomIDs ...int) (archetypes.Specialist, bool) {
		require.Contains(t, roomIDs, roomID, "specialists must be at the camp")
		sp, ok := byUtility[utility]
		return sp, ok
	}
}

type spawnCall struct{ room, mob, leader int }

// raidWorld is a camp in a raiding zone with a lit fire, ready to rest.
type raidWorld struct {
	m      *CampingModule
	store  *fakeStore
	sched  *fakeScheduler
	user   *users.UserRecord
	room   *rooms.Room
	now    *time.Time
	spawns *[]spawnCall
	rolls  []int
}

func newRaidWorld(t *testing.T, chance int, rolls ...int) *raidWorld {
	t.Helper()
	now := baseTime()
	store := &fakeStore{}
	sched := &fakeScheduler{}
	m := newTestModule(store, sched, &fakeSurvival{}, func() time.Time { return now })
	m.campCfg = defaultCampSettings()
	m.campCfg.Raids["Road"] = campRaid{MobID: 86, ChancePct: chance}
	m.campCfgLoaded = true
	w := &raidWorld{m: m, store: store, sched: sched, now: &now, spawns: &[]spawnCall{}}
	w.rolls = rolls
	m.roll = func(n int) int {
		if len(w.rolls) == 0 {
			return n - 1
		}
		r := w.rolls[0]
		w.rolls = w.rolls[1:]
		return r
	}
	m.spawnRaid = func(room, mob, leader int) (int, error) {
		*w.spawns = append(*w.spawns, spawnCall{room, mob, leader})
		return 1, nil
	}
	m.specialist = func(int, string, ...int) (archetypes.Specialist, bool) { return archetypes.Specialist{}, false }
	w.room = &rooms.Room{RoomId: 100, Zone: "Road", Title: "A Clearing", Tags: []string{"camping"}}
	rooms.SetTestRoom(w.room)
	t.Cleanup(func() { rooms.RemoveTestRoom(100) })
	w.user = campUser(t, 7, 100)
	require.NotContains(t, m.establish(w.user, w.room), "can't")
	m.lightFire(w.user, w.room)
	return w
}

func (w *raidWorld) rest(t *testing.T) {
	t.Helper()
	assert.Contains(t, w.m.startRest(w.user, w.room), "You settle in")
}

func (w *raidWorld) at(d time.Duration) { *w.now = baseTime().Add(d) }

func (w *raidWorld) camp() camping.Camp { return w.m.camps[7] }

// TestRaidIsRolledAtRestStartAndSaved (33f3): whether raiders come, and
// when, is decided as the rest begins and saved with it.
func TestRaidIsRolledAtRestStartAndSaved(t *testing.T) {
	w := newRaidWorld(t, 15, 14, 20) // 14 < 15: raiders; at 30% + 20% of the rest
	w.rest(t)
	raid := w.camp().Rest.Raid
	require.NotNil(t, raid)
	assert.Equal(t, 86, raid.MobID)
	assert.Equal(t, baseTime().Add(camping.RestDuration/2), raid.AtUTC)
	saved := w.store.saved.Camps[7].Rest.Raid
	require.NotNil(t, saved, "durable")
	assert.Equal(t, raid.AtUTC, saved.AtUTC)

	quiet := newRaidWorld(t, 15, 15)
	quiet.rest(t)
	assert.Nil(t, quiet.camp().Rest.Raid, "a roll of 15 against 15% brings no raid")
}

// TestUnspottedRaidSpoilsTheRest (33f3): with no watch, the raiders come
// once, attack the leader, and the rest earns no Rested and no rewards.
func TestUnspottedRaidSpoilsTheRest(t *testing.T) {
	w := newRaidWorld(t, 100, 0, 0)
	w.rest(t)
	messages := captureMessages(t)
	w.at(camping.RestDuration * 29 / 100)
	w.m.onNewRound(events.NewRound{})
	assert.Empty(t, *w.spawns, "not yet")

	w.at(camping.RestDuration * 31 / 100)
	w.m.onNewRound(events.NewRound{})
	w.m.onNewRound(events.NewRound{})
	events.ProcessEvents()
	assert.Equal(t, []spawnCall{{100, 86, 7}}, *w.spawns, "once, at the leader")
	assert.True(t, w.camp().Rest.Broken)
	assert.True(t, w.store.saved.Camps[7].Rest.Raid.Fired, "resolved durably")
	assert.Contains(t, strings.Join(*messages, ""), "Raiders fall on your sleeping camp")

	w.at(camping.RestDuration + time.Second)
	w.sched.fireLatest()
	assert.False(t, w.m.restedPending[7], "no Rested")
	assert.Empty(t, w.m.campRewards, "no camp rewards")
}

// TestWatchSpotsRaidersAndSavesTheRest (33f3): a level-4 watch (100%)
// spots the raiders: the fight still comes, the rest still counts.
func TestWatchSpotsRaidersAndSavesTheRest(t *testing.T) {
	w := newRaidWorld(t, 100, 0, 0, 99)
	w.m.specialist = specialistsAt(t, 100, map[string]archetypes.Specialist{archetypes.UtilityWatch: {Name: "Bran", Level: 4}})
	w.rest(t)
	messages := captureMessages(t)
	w.at(camping.RestDuration / 2)
	w.m.onNewRound(events.NewRound{})
	events.ProcessEvents()
	assert.Len(t, *w.spawns, 1)
	assert.False(t, w.camp().Rest.Broken)
	assert.Contains(t, strings.Join(*messages, ""), "Bran spots raiders creeping toward the fire")

	w.at(camping.RestDuration + time.Second)
	w.sched.fireLatest()
	assert.True(t, w.m.restedPending[7])
	assert.NotEmpty(t, w.m.campRewards[7])
}

// TestRaidLapsesWhenTheLeaderIsAway (33f3): nobody at the camp, nobody to
// raid: no spawn, no broken rest.
func TestRaidLapsesWhenTheLeaderIsAway(t *testing.T) {
	w := newRaidWorld(t, 100, 0, 0)
	w.rest(t)
	w.user.Character.RoomId = 101
	w.at(camping.RestDuration / 2)
	w.m.onNewRound(events.NewRound{})
	assert.Empty(t, *w.spawns)
	assert.False(t, w.camp().Rest.Broken)
	assert.True(t, w.camp().Rest.Raid.Fired)
}

// TestPlannedRaidSurvivesARestart (33f3).
func TestPlannedRaidSurvivesARestart(t *testing.T) {
	w := newRaidWorld(t, 100, 0, 0)
	w.rest(t)
	reloaded := newTestModule(w.store, &fakeScheduler{}, &fakeSurvival{}, func() time.Time { return *w.now })
	reloaded.load()
	require.NotNil(t, reloaded.camps[7].Rest.Raid)
	assert.Equal(t, 86, reloaded.camps[7].Rest.Raid.MobID)
}

func forageSpecs(t *testing.T) {
	t.Helper()
	for _, spec := range []*items.ItemSpec{
		{ItemId: 29, Name: "raw game meat", Type: items.Food, Weight: 500},
		{ItemId: 30018, Name: "wild thyme", Type: items.Food, Weight: 50},
		{ItemId: 30021, Name: "seared game meat", Type: items.Food, Weight: 400},
		{ItemId: 30020, Name: "thyme-roasted game", Type: items.Food, Weight: 500},
	} {
		items.SetTestItemSpec(spec)
		id := spec.ItemId
		t.Cleanup(func() { items.RemoveTestItemSpec(id) })
	}
}

// TestForageAndVigilAfterAnUnbrokenRest (33f3): at the next round after
// the rest, the ranger forages into the cargo and the cleric's vigil
// steadies present companions, each once even if retried.
func TestForageAndVigilAfterAnUnbrokenRest(t *testing.T) {
	forageSpecs(t)
	w := newRaidWorld(t, 0)
	w.m.campCfg.Forage["Road"] = []forageFind{{ItemID: 29, Weight: 1}}
	w.m.specialist = specialistsAt(t, 100, map[string]archetypes.Specialist{
		archetypes.UtilityForage: {Name: "Mira", Level: 4},
		archetypes.UtilityVigil:  {Name: "Hild", Level: 3},
	})
	cargo := newFakeCargo(100000)
	useCargo(t, cargo)
	bran := characters.New()
	bran.RoomId, bran.Health = 100, 10
	away := characters.New()
	away.RoomId, away.Health = 999, 10
	w.m.companionsOf = func(int) (map[int]*characters.Character, []int) {
		return map[int]*characters.Character{1: bran, 2: away}, []int{1, 2}
	}
	stub := &loyaltyStub{loyalty: map[int]int{1: 40, 2: 40}, names: map[int]string{1: "Bran", 2: "Cara"}, applied: map[string]bool{}}
	company.SetFormationProvider(stub)
	t.Cleanup(func() { company.SetFormationProvider(nil) })

	w.rest(t)
	messages := captureMessages(t)
	w.at(camping.RestDuration + time.Second)
	w.sched.fireLatest()
	op := w.m.campRewards[7]
	require.NotEmpty(t, op)
	w.m.onNewRound(events.NewRound{})
	events.ProcessEvents()
	text := strings.Join(*messages, "")

	assert.Equal(t, 3, cargo.stacks[29], "1 + level 4 / 2 finds")
	assert.Contains(t, text, `Mira foraged <ansi fg="itemname">3 raw game meat</ansi>, now in the company's cargo.`)
	assert.Equal(t, 43, stub.loyalty[1], "+ the cleric's level")
	assert.Equal(t, 40, stub.loyalty[2], "a companion away from the camp keeps none")
	assert.Contains(t, text, "Hild keeps a vigil by the fire; Bran is steadier for it.")
	assert.Empty(t, w.m.campRewards, "paid")
	assert.Empty(t, w.store.saved.CampRewards, "durably")

	// A retry of the same rest (a restart before the debt cleared) pays
	// nothing twice.
	w.m.campRewards[7] = op
	w.m.onNewRound(events.NewRound{})
	assert.Equal(t, 3, cargo.stacks[29])
	assert.Equal(t, 43, stub.loyalty[1])
}

// TestForageLeavesWhatTheCompanyCantCarry (33f3).
func TestForageLeavesWhatTheCompanyCantCarry(t *testing.T) {
	forageSpecs(t)
	w := newRaidWorld(t, 0)
	w.m.campCfg.Forage["Road"] = []forageFind{{ItemID: 29, Weight: 1}}
	w.m.specialist = specialistsAt(t, 100, map[string]archetypes.Specialist{archetypes.UtilityForage: {IsLeader: true, Level: 4}})
	cargo := newFakeCargo(1200) // room for two
	useCargo(t, cargo)
	w.rest(t)
	messages := captureMessages(t)
	w.at(camping.RestDuration + time.Second)
	w.sched.fireLatest()
	w.m.onNewRound(events.NewRound{})
	events.ProcessEvents()
	assert.Equal(t, 2, cargo.stacks[29])
	assert.Contains(t, strings.Join(*messages, ""), "the company can carry no more")
}

// TestCampCookUsesPackAndCargoAndSkill (33f3): the first recipe the cook
// can make, from the pack first and then the cargo, into the cargo.
func TestCampCookUsesPackAndCargoAndSkill(t *testing.T) {
	forageSpecs(t)
	w := newRaidWorld(t, 0)
	w.m.campCfg.Recipes = []campRecipe{
		{Output: 30020, Inputs: []int{29, 30018}, Skill: "cooking", MinLevel: 2},
		{Output: 30021, Inputs: []int{29}, Skill: "cooking", MinLevel: 1},
	}
	w.m.inBattle = func(int) bool { return false }
	cargo := newFakeCargo(100000)
	useCargo(t, cargo)
	cargo.stacks[29] = 1
	w.user.Character.StoreItem(items.New(30018))
	skills.SetTestData([]*skills.Skill{{SkillId: "cooking", Name: "Cooking", MaxLevel: 4}}, nil)
	t.Cleanup(func() { skills.SetTestData(nil, nil) })

	assert.Contains(t, w.m.cook(w.user, w.room), "it needs cooking 1", "no Cooking at all")
	w.user.Character.Skills = map[string]int{"cooking": 1}
	assert.Contains(t, w.m.cook(w.user, w.room), "seared game meat", "Cooking 1 makes the simpler dish")
	assert.Equal(t, 0, cargo.stacks[29], "the meat came from the cargo")
	assert.Equal(t, 1, cargo.stacks[30021], "the dish went into it")

	cargo.stacks[29] = 1
	w.user.Character.Skills["cooking"] = 2
	assert.Contains(t, w.m.cook(w.user, w.room), "thyme-roasted game")
	_, kept := packItem(w.user, 30018)
	assert.False(t, kept, "the thyme came from the pack")

	w.m.inBattle = func(int) bool { return true }
	assert.Contains(t, w.m.cook(w.user, w.room), "middle of a fight")
	w.m.inBattle = func(int) bool { return false }
	other := &rooms.Room{RoomId: 555, Zone: "Road"}
	assert.Contains(t, w.m.cook(w.user, other), "your own camp here")
}

// TestShippedCampConfigParses (33f3).
func TestShippedCampConfigParses(t *testing.T) {
	cfg := shippedCampSettings(t)
	assert.Equal(t, campRaid{MobID: 86, ChancePct: 15}, cfg.Raids["Old Kings Road"])
	assert.Equal(t, 25, cfg.WatchPctPerLevel)
	assert.Equal(t, 60, cfg.VigilCap)
	assert.Len(t, cfg.Forage["Old Kings Road"], 3)
	assert.Len(t, cfg.Recipes, 3)
}

func shippedCampSettings(t *testing.T) campSettings {
	t.Helper()
	data, err := files.ReadFile("files/data-overlays/config.yaml")
	require.NoError(t, err)
	var cfg map[string]any
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	return parseCampSettings(func(k string) any { return cfg[k] })
}
