package camping

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/cookbook"
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

type sharedRecipeCargo struct {
	*fakeCargo
	user   *users.UserRecord
	err    error
	inputs []items.Item
}

func (c *sharedRecipeCargo) CargoContents(int) []encumbrance.CargoStack {
	var out []encumbrance.CargoStack
	for _, itm := range c.user.Character.Items {
		out = append(out, encumbrance.CargoStack{ItemId: itm.ItemId, Count: 1})
	}
	return out
}

func (c *sharedRecipeCargo) TransformCargo(_ int, inputs, outputs []items.Item) error {
	c.inputs = inputs
	if c.err != nil {
		return c.err
	}
	for _, itm := range inputs {
		if !c.user.Character.RemoveItem(itm) {
			return encumbrance.ErrInsufficientCargo
		}
	}
	c.user.Character.Items = append(c.user.Character.Items, outputs...)
	return nil
}

func TestSharedCampCookCountsRepeatedIngredientsOnceAndHandlesSaveFailure(t *testing.T) {
	forageSpecs(t)
	w := newRaidWorld(t, 0)
	w.m.inBattle = func(int) bool { return false }
	w.m.campCfg.Recipes = []campRecipe{{Output: 30020, Inputs: []int{29, 29, 30018}}}
	w.user.Character.CompanyCargo = true
	a, b, herb := items.New(29), items.New(29), items.New(30018)
	cargo := &sharedRecipeCargo{fakeCargo: newFakeCargo(100000), user: w.user}
	encumbrance.SetProvider(cargo)
	t.Cleanup(func() { encumbrance.SetProvider(nil) })
	w.user.Character.Items = []items.Item{a, herb}
	assert.Contains(t, w.m.cook(w.user, w.room, nil), "nothing to cook", "one meat isn't counted through two inventory paths")
	assert.Len(t, w.user.Character.Items, 2)
	w.user.Character.Items = []items.Item{a, b, herb}
	cargo.err = errors.New("atomic save failed")
	assert.Contains(t, w.m.cook(w.user, w.room, nil), "nothing was cooked")
	assert.Len(t, w.user.Character.Items, 3)
	assert.Len(t, cargo.inputs, 3)
	assert.NotEqual(t, cargo.inputs[0].UUID, cargo.inputs[1].UUID)
	cargo.err = nil
	messages := captureMessages(t)
	_, err := w.m.userCommand("cook", w.user, w.room, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*messages, ""), "company cargo")
	require.Len(t, w.user.Character.Items, 1)
	assert.Equal(t, 30020, w.user.Character.Items[0].ItemId)
}

// fakeCargo is an encumbrance provider with a cargo and a capacity, keeping
// deposit operations once, as the real module does.
type fakeCargo struct {
	failWithdraw map[int]bool
	capacity     int
	stacks       map[int]int
	applied      map[string]bool
	grams        int
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
	if f.failWithdraw[itemID] || f.stacks[itemID] < count {
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

func (l *loyaltyStub) FormationFor(int) (company.Formation, bool) { return company.Formation{}, false }
func (l *loyaltyStub) InstanceFor(int, int) (int, bool)           { return 0, false }
func (l *loyaltyStub) LeaderAndKeyForInstance(int) (int, company.MemberKey, bool) {
	return 0, "", false
}
func (l *loyaltyStub) CompanionArchetype(int, int) (string, bool) { return "", false }
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
	w.room = &rooms.Room{RoomId: 100, Zone: "Road", Title: "A Clearing", Tags: []string{"camping"}, Resources: []string{"firewood"}}
	rooms.SetTestRoom(w.room)
	t.Cleanup(func() { rooms.RemoveTestRoom(100) })
	w.user = campUser(t, 7, 100)
	// Phase 56: these tests are about cooking, so the leader knows the dishes.
	for _, dish := range []int{30019, 30020, 30021, 30024} {
		cookbook.Learn(w.user.Character, dish)
	}
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

	assert.Contains(t, w.m.cook(w.user, w.room, nil), "it needs cooking 1", "no Cooking at all")
	w.user.Character.Skills = map[string]int{"cooking": 1}
	assert.Contains(t, w.m.cook(w.user, w.room, nil), "seared game meat", "Cooking 1 makes the simpler dish")
	assert.Equal(t, 0, cargo.stacks[29], "the meat came from the cargo")
	assert.Equal(t, 1, cargo.stacks[30021], "the dish went into it")

	cargo.stacks[29] = 1
	w.user.Character.Skills["cooking"] = 2
	assert.Contains(t, w.m.cook(w.user, w.room, nil), "thyme-roasted game")
	_, kept := packItem(w.user, 30018)
	assert.False(t, kept, "the thyme came from the pack")

	w.m.inBattle = func(int) bool { return true }
	assert.Contains(t, w.m.cook(w.user, w.room, nil), "middle of a fight")
	w.m.inBattle = func(int) bool { return false }
	other := &rooms.Room{RoomId: 555, Zone: "Road"}
	assert.Contains(t, w.m.cook(w.user, other, nil), "your own camp here")
}

// TestShippedCampConfigParses (33f3).
func TestShippedCampConfigParses(t *testing.T) {
	cfg := shippedCampSettings(t)
	assert.Equal(t, campRaid{MobID: 86, ChancePct: 15}, cfg.Raids["Old Kings Road"])
	assert.Equal(t, 25, cfg.WatchPctPerLevel)
	assert.Equal(t, 60, cfg.VigilCap)
	assert.Len(t, cfg.Forage["Old Kings Road"], 3)
	assert.Len(t, cfg.Recipes, 4, "game meat dishes plus grilled fish (40a2)")
}

func shippedCampSettings(t *testing.T) campSettings {
	t.Helper()
	data, err := files.ReadFile("files/data-overlays/config.yaml")
	require.NoError(t, err)
	var cfg map[string]any
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	return parseCampSettings(func(k string) any { return cfg[k] })
}

// rewardWorld is a raid-free camp whose rest pays a level-4 forage.
func rewardWorld(t *testing.T) (*raidWorld, *fakeCargo) {
	t.Helper()
	forageSpecs(t)
	w := newRaidWorld(t, 0)
	w.m.campCfg.Forage["Road"] = []forageFind{{ItemID: 29, Weight: 1}}
	w.m.specialist = specialistsAt(t, 100, map[string]archetypes.Specialist{archetypes.UtilityForage: {Name: "Mira", Level: 4}})
	w.m.inBattle = func(int) bool { return false }
	cargo := newFakeCargo(100000)
	useCargo(t, cargo)
	return w, cargo
}

// TestCampRewardsWaitForTheFightToEnd (33f3 review finding 2): rewards owed
// while the company still fights a spotted raid are paid after the fight.
func TestCampRewardsWaitForTheFightToEnd(t *testing.T) {
	w, cargo := rewardWorld(t)
	w.rest(t)
	w.at(camping.RestDuration + time.Second)
	w.sched.fireLatest()
	w.m.inBattle = func(int) bool { return true }
	w.m.onNewRound(events.NewRound{})
	assert.Zero(t, cargo.stacks[29], "not during the fight")
	assert.NotEmpty(t, w.m.campRewards[7], "still owed")
	w.m.inBattle = func(int) bool { return false }
	w.m.onNewRound(events.NewRound{})
	assert.Equal(t, 3, cargo.stacks[29])
}

// TestCampRewardsComeOncePerCooldown (33f3 review finding 3): a second
// rest inside 15 minutes is still a rest, but forages nothing.
func TestCampRewardsComeOncePerCooldown(t *testing.T) {
	w, cargo := rewardWorld(t)
	w.rest(t)
	w.at(camping.RestDuration + time.Second)
	w.sched.fireLatest()
	w.m.onNewRound(events.NewRound{})
	require.Equal(t, 3, cargo.stacks[29])

	w.m.breakCamp(w.user, w.room)
	w.at(5 * time.Minute)
	w.m.establish(w.user, w.room)
	w.m.lightFire(w.user, w.room)
	w.rest(t)
	w.at(5*time.Minute + camping.RestDuration + time.Second)
	w.sched.fireLatest()
	assert.True(t, w.m.restedPending[7], "Rested still comes")
	assert.Empty(t, w.m.campRewards, "no rewards inside the cooldown")

	w.m.onNewRound(events.NewRound{})
	w.m.breakCamp(w.user, w.room)
	w.at(16 * time.Minute)
	w.m.establish(w.user, w.room)
	w.m.lightFire(w.user, w.room)
	w.rest(t)
	w.at(16*time.Minute + camping.RestDuration + time.Second)
	w.sched.fireLatest()
	w.m.onNewRound(events.NewRound{})
	assert.Equal(t, 6, cargo.stacks[29], "after the cooldown, again")
	assert.Equal(t, baseTime().Add(16*time.Minute), w.store.saved.LastCampRewards[7], "durable")
}

// TestLeavingTheCampForfeitsItsRewards (33f3 review finding 7): rewards are
// collected at the camp, against its forage and its company.
func TestLeavingTheCampForfeitsItsRewards(t *testing.T) {
	w, cargo := rewardWorld(t)
	w.rest(t)
	w.at(camping.RestDuration + time.Second)
	w.sched.fireLatest()
	w.user.Character.RoomId = 555
	w.m.onNewRound(events.NewRound{})
	assert.Zero(t, cargo.stacks[29])
	assert.Empty(t, w.m.campRewards, "the debt is dropped, not kept forever")
}

// TestLoggingOutDoesNotDodgeARaid (33f3 review finding 5).
func TestLoggingOutDoesNotDodgeARaid(t *testing.T) {
	w := newRaidWorld(t, 100, 0, 0)
	w.rest(t)
	users.RemoveTestUser(7)
	w.at(camping.RestDuration / 2)
	w.m.onNewRound(events.NewRound{})
	assert.Empty(t, *w.spawns, "nobody there to fight")
	assert.True(t, w.camp().Rest.Broken, "but the rest is spoiled")
}

// TestFailedRaidSpawnLeavesTheRestWhole (33f3 review finding 9).
func TestFailedRaidSpawnLeavesTheRestWhole(t *testing.T) {
	w := newRaidWorld(t, 100, 0, 0)
	w.m.spawnRaid = func(int, int, int) (int, error) { return 0, assert.AnError }
	w.rest(t)
	messages := captureMessages(t)
	w.at(camping.RestDuration / 2)
	w.m.onNewRound(events.NewRound{})
	events.ProcessEvents()
	assert.False(t, w.camp().Rest.Broken)
	assert.False(t, w.store.saved.Camps[7].Rest.Broken, "durably")
	assert.NotContains(t, strings.Join(*messages, ""), "Raiders fall")
}

// TestRaidersLeaveWhenTheirTargetIsGone (33f3 review finding 4): raiders
// still standing when the leader falls or leaves are sent off, so they
// never turn on other players camping in the room.
func TestRaidersLeaveWhenTheirTargetIsGone(t *testing.T) {
	w := newRaidWorld(t, 100, 0, 0)
	alive := map[int]bool{41: true, 42: true}
	var despawned []int
	w.m.raidGroup = func(room, first int) []int {
		assert.Equal(t, 100, room)
		return []int{41, 42}
	}
	w.m.spawnRaid = func(int, int, int) (int, error) { return 41, nil }
	w.m.raiderAlive = func(id int) bool { return alive[id] }
	w.m.despawnRaider = func(id int) { despawned = append(despawned, id); alive[id] = false }
	w.rest(t)
	w.at(camping.RestDuration / 2)
	w.m.onNewRound(events.NewRound{})
	w.m.onNewRound(events.NewRound{})
	assert.Empty(t, despawned, "the leader is still there to fight them")

	w.user.Character.Health = 0
	w.m.onNewRound(events.NewRound{})
	assert.ElementsMatch(t, []int{41, 42}, despawned)
	assert.Empty(t, w.m.raiders, "forgotten")
}

// TestCampCookCommandAndItsSafety (33f3 review findings 6 and 12): through
// "camp cook"; a failed cargo withdrawal puts back what was taken; a dish
// heavier than its makings is refused when the company is full.
func TestCampCookCommandAndItsSafety(t *testing.T) {
	forageSpecs(t)
	w := newRaidWorld(t, 0)
	w.m.campCfg.Recipes = []campRecipe{{Output: 30021, Inputs: []int{29}, Skill: "cooking", MinLevel: 1}}
	w.m.inBattle = func(int) bool { return false }
	skills.SetTestData([]*skills.Skill{{SkillId: "cooking", Name: "Cooking", MaxLevel: 4}}, nil)
	t.Cleanup(func() { skills.SetTestData(nil, nil) })
	w.user.Character.Skills = map[string]int{"cooking": 1}
	cargo := newFakeCargo(100000)
	useCargo(t, cargo)
	cargo.stacks[29] = 1

	messages := captureMessages(t)
	_, err := w.m.userCommand("cook", w.user, w.room, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*messages, ""), "seared game meat")
	assert.Equal(t, 1, cargo.stacks[30021])

	// A dish heavier than its makings, in a full company.
	items.SetTestItemSpec(&items.ItemSpec{ItemId: 30021, Name: "seared game meat", Type: items.Food, Weight: 900})
	cargo.stacks[29] = 1
	cargo.capacity, cargo.grams = 100, 100
	assert.Contains(t, w.m.cook(w.user, w.room, nil), "too much for your company to carry")
	assert.Equal(t, 1, cargo.stacks[29], "nothing taken")
}

// TestCampCookPutsBackWhatAFailedWithdrawalTook (33f3 review finding 6).
func TestCampCookPutsBackWhatAFailedWithdrawalTook(t *testing.T) {
	forageSpecs(t)
	w := newRaidWorld(t, 0)
	w.m.campCfg.Recipes = []campRecipe{{Output: 30020, Inputs: []int{29, 30018}}}
	w.m.inBattle = func(int) bool { return false }
	cargo := newFakeCargo(100000)
	useCargo(t, cargo)
	cargo.stacks[29], cargo.stacks[30018] = 1, 1
	cargo.failWithdraw = map[int]bool{30018: true}
	assert.Contains(t, w.m.cook(w.user, w.room, nil), "couldn't be gathered")
	assert.Equal(t, 1, cargo.stacks[29], "the meat was put back")
	assert.Equal(t, 1, cargo.stacks[30018])
	assert.Zero(t, cargo.stacks[30020])
}

// TestCampRegistryRoundTripsThroughYAML (33f3 review finding 12): a raid,
// a broken rest, owed rewards, and the reward clock survive the real
// save format.
func TestCampRegistryRoundTripsThroughYAML(t *testing.T) {
	at := baseTime()
	reg := NewRegistry()
	reg.Camps[7] = camping.Camp{LeaderUserID: 7, RoomID: 100, FireLit: true, Rest: &camping.RestSession{
		StartedAtUTC: at, Broken: true, Raid: &camping.Raid{AtUTC: at.Add(time.Second), MobID: 86, Fired: true},
	}}
	reg.CampRewards[8] = campReward{Op: "rest-8", RoomID: 100}
	reg.LastCampRewards[8] = at
	data, err := yaml.Marshal(reg)
	require.NoError(t, err)
	var loaded Registry
	require.NoError(t, decodeRegistry(data, &loaded))
	rest := loaded.Camps[7].Rest
	require.NotNil(t, rest)
	assert.True(t, rest.Broken)
	require.NotNil(t, rest.Raid)
	assert.Equal(t, 86, rest.Raid.MobID)
	assert.True(t, rest.Raid.Fired)
	assert.Equal(t, campReward{Op: "rest-8", RoomID: 100}, loaded.CampRewards[8])
	assert.True(t, at.Equal(loaded.LastCampRewards[8]))
}

// TestTutorialCampsAreNeverRaided (33f3 review finding 12).
func TestTutorialCampsAreNeverRaided(t *testing.T) {
	cfg := shippedCampSettings(t)
	_, raided := cfg.Raids["Tutorial"]
	assert.False(t, raided)
	assert.Len(t, cfg.Raids, 2, "the Old Kings Road and the admin Test Area Road")
	_, road := cfg.Raids["Old Kings Road"]
	assert.True(t, road)
	assert.Equal(t, 15*time.Minute, cfg.RewardCooldown)
}

// restEnds is what the story events' camp trigger heard (Phase 60).
var (
	restEndsOnce sync.Once
	restEnds     [][2]int
)

// TestAFinishedRestTellsTheCampListeners (Phase 60): a rest that pays its
// rewards at the camp reaches the story events' camp trigger once; one the
// leader walked away from does not.
func TestAFinishedRestTellsTheCampListeners(t *testing.T) {
	restEndsOnce.Do(func() {
		camping.AddRestEndListener(func(leader, room int) { restEnds = append(restEnds, [2]int{leader, room}) })
	})
	restEnds = nil
	w, _ := rewardWorld(t)
	w.rest(t)
	w.at(camping.RestDuration + time.Second)
	w.sched.fireLatest()
	w.m.onNewRound(events.NewRound{})
	assert.Equal(t, [][2]int{{7, 100}}, restEnds)
	w.m.onNewRound(events.NewRound{})
	assert.Len(t, restEnds, 1, "heard once")

	restEnds = nil
	w, _ = rewardWorld(t)
	w.rest(t)
	w.at(camping.RestDuration + time.Second)
	w.sched.fireLatest()
	w.user.Character.RoomId = 555
	w.m.onNewRound(events.NewRound{})
	assert.Empty(t, restEnds, "walking away forfeits the camp scene too")
}
