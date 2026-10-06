package camping

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/weather"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Phase 40a3: camp gear and the fuel rule (one bundle a rest, then embers).

// stock is a stand-in for what the company carries; it counts and spends
// by item, as the company seam does.
type stock map[int]int

func (s stock) count(_, itemID int) int { return s[itemID] }
func (s stock) spend(_, itemID int) bool {
	if s[itemID] <= 0 {
		return false
	}
	s[itemID]--
	return true
}

func (s stock) install(m *CampingModule) {
	m.itemCount, m.spendItem = s.count, s.spend
}

// bonusSurvival records a bedroll-aware recovery.
type bonusFake struct {
	fakeSurvival
	bonuses []map[survival.MemberKey]int
	amounts []int
}

func (f *bonusFake) ApplyCompanyRestRecoveryBonus(_ int, _ string, fatigue int, bonus map[survival.MemberKey]int) ([]survival.ExertionResult, error) {
	f.bonuses = append(f.bonuses, bonus)
	f.amounts = append(f.amounts, fatigue)
	return nil, nil
}

// withRoster gives the camp a leader and companions 1 and 2, all present.
func withRoster(m *CampingModule) {
	m.companySize = func(int) int { return 3 }
	m.companionsOf = func(int) (map[int]*characters.Character, []int) {
		return map[int]*characters.Character{}, []int{2, 1} // unsorted on purpose
	}
}

func finishRest(w *raidWorld) {
	w.at(camping.RestDuration + time.Second)
	w.sched.fireLatest()
}

// A finished rest burns the fire down to embers: they keep the room warm,
// but another rest needs the fire fed, which spends another bundle away
// from deadfall.
func TestFinishedRestLeavesEmbersAndTheNextRestNeedsFuel(t *testing.T) {
	w := newRaidWorld(t, 0)
	w.room.Resources = nil // no deadfall: a fire burns a bundle
	stock{}.install(w.m)
	w.rest(t)
	finishRest(w)

	camp := w.camp()
	assert.False(t, camp.FireLit)
	assert.True(t, camp.Embers)
	assert.Equal(t, camping.Completed, camp.Rest.State)
	assert.True(t, w.m.RoomWarmedByFire(100), "embers keep their warmth until the camp is broken")
	assert.False(t, w.m.RoomHasLitFire(100), "but give no light")
	assert.Contains(t, w.m.status(7), "burned down to glowing embers")

	assert.Contains(t, w.m.startRest(w.user, w.room), "burned down to embers")
	assert.Equal(t, camping.Completed, w.camp().Rest.State, "no second rest on embers")

	// Feeding the embers needs a bundle, and the company has none.
	assert.Contains(t, w.m.lightFire(w.user, w.room), "no firewood")
	assert.True(t, w.camp().Embers)
	assert.Contains(t, w.m.fuelLine(w.user, w.room), "Fuel: none")
}

func TestEmbersAreFedAndTheSecondRestRuns(t *testing.T) {
	w := newRaidWorld(t, 0)
	w.room.Resources = nil
	surv := &fakeSurvival{}
	w.m.survival = surv
	supplies := stock{firewoodItemID: 2}
	supplies.install(w.m)
	w.rest(t)
	finishRest(w)
	require.True(t, w.camp().Embers)
	require.Len(t, surv.amounts, 1)

	text := w.m.lightFire(w.user, w.room)
	assert.Contains(t, text, "feed the embers")
	assert.Equal(t, 1, supplies[firewoodItemID], "one bundle for the second rest")
	camp := w.camp()
	assert.True(t, camp.FireLit)
	assert.False(t, camp.Embers)

	first := camp.Rest.StartedAtUTC
	w.at(camping.RestDuration + 2*time.Second)
	assert.Contains(t, w.m.startRest(w.user, w.room), "You settle in")
	second := w.camp().Rest
	assert.Equal(t, camping.Resting, second.State)
	assert.True(t, second.StartedAtUTC.After(first), "a new session, with its own operation id")

	w.at(2*camping.RestDuration + 3*time.Second)
	w.sched.fireLatest()
	require.Len(t, surv.amounts, 2, "the second rest recovers fatigue too")
	assert.True(t, w.camp().Embers)
	assert.Equal(t, camping.Completed, w.camp().Rest.State)
}

// A camp saved before 40a3 kept its fire lit after the rest; it burns down
// on load so resting again costs fuel.
func TestLegacyCompletedCampBurnsDownOnRecovery(t *testing.T) {
	w := newRaidWorld(t, 0)
	w.m.camps[7] = camping.Camp{LeaderUserID: 7, RoomID: 100, FireLit: true,
		Rest: &camping.RestSession{StartedAtUTC: baseTime().Add(-5 * time.Minute), State: camping.Completed}}
	w.m.recoveryApplied[7] = true
	w.m.recoverLocked()
	assert.False(t, w.camp().FireLit)
	assert.True(t, w.camp().Embers)
}

func TestFireSteelLightsDampWoodFirstTimeAtFullWarmth(t *testing.T) {
	w := newRaidWorld(t, 0)
	w.room.Resources = nil
	w.m.camps[7] = camping.Camp{LeaderUserID: 7, RoomID: 100}
	supplies := stock{dampFirewoodItemID: 1}
	supplies.install(w.m)

	// Without a fire steel, the first try fails.
	assert.Contains(t, w.m.lightFire(w.user, w.room), "will not catch")
	assert.Equal(t, 1, supplies[dampFirewoodItemID])

	w.m.dampTried = nil
	supplies[fireSteelItemID] = 1
	text := w.m.lightFire(w.user, w.room)
	assert.Contains(t, text, "fire steel")
	assert.True(t, w.camp().FireLit)
	assert.False(t, w.camp().Damp, "full warmth")
	assert.Zero(t, supplies[dampFirewoodItemID])
	assert.True(t, w.m.RoomWarmedByFire(100))
}

// Bedrolls: counted when the rest starts, assigned leader first then
// companions by number, and locked on the rest.
func TestBedrollsCoverMembersInOrderAndAreLockedAtRestStart(t *testing.T) {
	w := newRaidWorld(t, 0)
	withRoster(w.m)
	surv := &bonusFake{}
	w.m.survival = surv
	supplies := stock{bedrollItemID: 2}
	supplies.install(w.m)

	text := w.m.startRest(w.user, w.room)
	assert.Contains(t, text, "2 of 3 on bedrolls (+25% fatigue)")
	rest := w.camp().Rest
	assert.Equal(t, []string{"leader", "companion:1"}, rest.Bedrolls, "the leader, then companions by number")
	assert.Equal(t, []string{"leader", "companion:1"}, w.store.saved.Camps[7].Rest.Bedrolls, "saved with the rest")

	supplies[bedrollItemID] = 0 // lost mid-rest: nothing changes
	finishRest(w)
	require.Len(t, surv.bonuses, 1)
	assert.Equal(t, map[survival.MemberKey]int{"leader": 25, "companion:1": 25}, surv.bonuses[0])
	assert.Equal(t, 20, surv.amounts[0], "the base recovery; the bonus rides on top")
}

func TestNoBedrollsRecoverNormally(t *testing.T) {
	w := newRaidWorld(t, 0)
	surv := &bonusFake{}
	w.m.survival = surv
	w.rest(t)
	assert.Empty(t, w.camp().Rest.Bedrolls)
	finishRest(w)
	assert.Empty(t, surv.bonuses, "the plain recovery call")
	assert.Equal(t, 1, surv.fakeSurvival.applyCalls)
}

func TestMoreBedrollsThanMembersCoverEveryone(t *testing.T) {
	w := newRaidWorld(t, 0)
	withRoster(w.m)
	stock{bedrollItemID: 9}.install(w.m)
	w.rest(t)
	assert.Len(t, w.camp().Rest.Bedrolls, 3)
}

// The tent counts as shelter for the weather, does not stack with a
// shelter room, and keeps the cold off the room while the company rests.
func TestTentActsAsShelterAndDoesNotStackWithARoom(t *testing.T) {
	storm := func(string) (weather.Condition, bool) {
		return weather.Condition{Name: "storm", RestRecoveryPct: 50}, true
	}
	bare := newRaidWorld(t, 0)
	bare.m.weatherIn = storm
	bare.rest(t)
	assert.Equal(t, 10, bare.camp().Rest.Recovery)

	tent := newRaidWorld(t, 0)
	tent.m.weatherIn = storm
	stock{tentItemID: 1}.install(tent.m)
	text := tent.m.startRest(tent.user, tent.room)
	assert.Equal(t, 15, tent.camp().Rest.Recovery, "halfway from 50% to a full rest")
	assert.Contains(t, text, "The tent softens it.")

	both := newRaidWorld(t, 0)
	both.m.weatherIn = storm
	both.room.Resources = []string{rooms.ResourceShelter}
	stock{tentItemID: 1}.install(both.m)
	both.m.startRest(both.user, both.room)
	assert.Equal(t, 15, both.camp().Rest.Recovery, "the better applies; they don't stack")
}

func TestTentWarmsTheRoomOnlyWhileResting(t *testing.T) {
	w := newRaidWorld(t, 0)
	stock{tentItemID: 1}.install(w.m)
	// A damp fire gives no warmth; a tent shuts the cold out of a rest.
	camp := w.camp()
	camp.Damp = true
	w.m.camps[7] = camp
	w.m.mu.Lock()
	w.m.refreshLitRoomsLocked()
	w.m.mu.Unlock()
	assert.False(t, w.m.RoomWarmedByFire(100))
	w.rest(t)
	assert.True(t, w.camp().Tent)
	assert.True(t, w.m.RoomWarmedByFire(100), "no rest-time cold with a tent")
	finishRest(w)
	assert.False(t, w.m.RoomWarmedByFire(100), "once the rest is over the damp fire's embers give no warmth")
}

func TestTentIsPitchedAtCampAndShownInLookAndStatus(t *testing.T) {
	w := newRaidWorld(t, 0)
	w.m.camps = map[int]camping.Camp{}
	stock{tentItemID: 1}.install(w.m)
	text := w.m.establish(w.user, w.room)
	assert.Contains(t, text, "oiled canvas tent")
	assert.True(t, w.camp().Tent)
	assert.Contains(t, w.m.status(7), "oiled canvas tent")
	lines := camping.CampLines(w.m.RoomCamps(100), 7, nil)
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "oiled canvas tent")
	s, _ := w.m.CampStateOf(7, 100, nil)
	assert.True(t, s.Tent)
}

func TestCampLinesForEmbersAndTent(t *testing.T) {
	lines := camping.CampLines([]camping.RoomCamp{
		{LeaderUserID: 7, Embers: true},
		{LeaderUserID: 8, Embers: true, Tent: true},
		{LeaderUserID: 9, Tent: true},
	}, 7, func(id int) string { return map[int]string{8: "Bran", 9: "Mira"}[id] })
	assert.Equal(t, []string{
		"A camp is pitched here: bedrolls around the glowing embers of a banked fire.",
		"Bran's camp is pitched here: an oiled canvas tent and bedrolls around the glowing embers of a banked fire.",
		"Mira's camp is pitched here: an oiled canvas tent beside a cold fire pit.",
	}, lines)
}

// The locked gear effects and the ember state survive a save and load,
// and a copyover (which reloads the same registry), mid-rest.
func TestLockedGearAndEmbersSurviveSaveAndLoad(t *testing.T) {
	w := newRaidWorld(t, 0)
	withRoster(w.m)
	stock{bedrollItemID: 1, tentItemID: 1, campBellsItemID: 1, surgeonKitItemID: 1}.install(w.m)
	w.rest(t)

	reloaded := newTestModule(w.store, &fakeScheduler{}, &fakeSurvival{}, func() time.Time { return *w.now })
	reloaded.load()
	camp := reloaded.camps[7]
	require.NotNil(t, camp.Rest)
	assert.Equal(t, []string{"leader"}, camp.Rest.Bedrolls)
	assert.True(t, camp.Rest.Bells)
	assert.True(t, camp.Rest.Kit)
	assert.True(t, camp.Tent)

	finishRest(w)
	require.True(t, w.camp().Embers)
	data, err := yaml.Marshal(Registry{Camps: map[int]camping.Camp{7: w.camp()}})
	require.NoError(t, err)
	var back Registry
	require.NoError(t, decodeRegistry(data, &back))
	assert.True(t, back.Camps[7].Embers)
	assert.False(t, back.Camps[7].FireLit)
}

// Bells and trip lines: a flat chance with no watch, a bonus on a watch,
// capped; one use worn per rest.
func TestSpotChanceTable(t *testing.T) {
	for _, tc := range []struct {
		posted bool
		watch  int
		bells  bool
		want   int
	}{
		{false, 0, false, 0},
		{false, 0, true, 20},
		{true, 25, false, 25},
		{true, 25, true, 35},
		{true, 100, false, 100},
		{true, 100, true, 90},
		{true, 85, true, 90},
	} {
		assert.Equal(t, tc.want, spotChance(tc.posted, tc.watch, tc.bells), "%+v", tc)
	}
}

func raidWithBells(t *testing.T, spotRoll int, watch *archetypes.Specialist) (*raidWorld, []string) {
	t.Helper()
	// rolls: raid happens, raid at 30%, then the spot roll
	w := newRaidWorld(t, 100, 0, 0, spotRoll)
	supplies := stock{campBellsItemID: 1}
	supplies.install(w.m)
	if watch != nil {
		w.m.specialist = specialistsAt(t, 100, map[string]archetypes.Specialist{archetypes.UtilityWatch: *watch})
	}
	w.rest(t)
	assert.Zero(t, supplies[campBellsItemID], "a rest wears the bells' last use")
	assert.True(t, w.camp().Rest.Bells)
	messages := captureMessages(t)
	w.at(camping.RestDuration / 2)
	w.m.onNewRound(events.NewRound{})
	events.ProcessEvents()
	return w, *messages
}

func TestBellsSpotRaidersWithNoWatchAtTwentyPercent(t *testing.T) {
	w, out := raidWithBells(t, 19, nil)
	assert.False(t, w.camp().Rest.Broken)
	assert.Contains(t, strings.Join(out, ""), "bells and trip lines jangle")

	missed, out := raidWithBells(t, 20, nil)
	assert.True(t, missed.camp().Rest.Broken)
	assert.Contains(t, strings.Join(out, ""), "Raiders fall on your sleeping camp")
}

func TestBellsAddTenPointsToAWatchAndCapAtNinety(t *testing.T) {
	level1 := archetypes.Specialist{Name: "Bran", Level: 1} // 25% a level
	w, out := raidWithBells(t, 34, &level1)
	assert.False(t, w.camp().Rest.Broken, "34 < 25+10")
	assert.Contains(t, strings.Join(out, ""), "Bran spots raiders")
	w, _ = raidWithBells(t, 35, &level1)
	assert.True(t, w.camp().Rest.Broken, "35 is not under 35")

	level4 := archetypes.Specialist{Name: "Bran", Level: 4} // 100%
	w, _ = raidWithBells(t, 89, &level4)
	assert.False(t, w.camp().Rest.Broken)
	w, _ = raidWithBells(t, 90, &level4)
	assert.True(t, w.camp().Rest.Broken, "the bells' cap holds a sure watch to 90%")
}

func TestNoBellsAndNoWatchSpotsNothing(t *testing.T) {
	w := newRaidWorld(t, 100, 0, 0)
	w.rest(t)
	assert.False(t, w.camp().Rest.Bells)
	w.at(camping.RestDuration / 2)
	w.m.onNewRound(events.NewRound{})
	assert.True(t, w.camp().Rest.Broken)
}

func TestBellsBoughtMidRestDoNotCountAndWearOutAfterTenRests(t *testing.T) {
	w := newRaidWorld(t, 0)
	supplies := stock{campBellsItemID: 10}
	supplies.install(w.m)
	w.rest(t)
	assert.Equal(t, 9, supplies[campBellsItemID], "one use a rest")
	assert.Contains(t, w.m.status(7), "bells and trip lines strung")
}

// The surgeon's kit: locked at rest start, used at the rest's end only for
// a camp rest, never for a rest that raiders broke.
func TestSurgeonsKitTreatsAtTheRestsEnd(t *testing.T) {
	e, user := heroEnv(t)
	stock{surgeonKitItemID: 1}.install(e.module)
	calls := 0
	e.module.surgery = func(leader int) ([]string, bool) {
		calls++
		return []string{"Bran tends your cut hand, and it draws closed."}, true
	}
	messages := captureMessages(t)
	e.completeCamp(t, user)
	assert.True(t, e.module.camps[7].Rest.Kit)
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	events.ProcessEvents()
	assert.Equal(t, 1, calls)
	out := strings.Join(*messages, "\n")
	assert.Contains(t, out, "Bran tends your cut hand")
	assert.Contains(t, out, "surgeon's kit wears")

	e.module.onNewRound(events.NewRound{RoundNumber: 2})
	assert.Equal(t, 1, calls, "once a rest")
}

func TestNoKitNoSurgery(t *testing.T) {
	e, user := heroEnv(t)
	calls := 0
	e.module.surgery = func(int) ([]string, bool) { calls++; return nil, false }
	e.completeCamp(t, user)
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Zero(t, calls)
}

func TestKitIsNotUsedAfterABrokenRest(t *testing.T) {
	w := newRaidWorld(t, 0)
	stock{surgeonKitItemID: 1}.install(w.m)
	w.rest(t)
	camp := w.camp()
	rest := *camp.Rest
	rest.Broken = true
	camp.Rest = &rest
	w.m.camps[7] = camp
	assert.False(t, w.m.restKit(7))
}

// The cookpot makes one more portion of a dish with more than one
// ingredient, and nothing extra for a single-ingredient dish.
func TestCookpotAddsOnePortionToMultiIngredientDishes(t *testing.T) {
	forageSpecs(t)
	for _, tc := range []struct {
		name   string
		inputs []int
		pot    bool
		want   int
	}{
		{"two inputs, no pot", []int{29, 30018}, false, 1},
		{"two inputs, a pot", []int{29, 30018}, true, 2},
		{"one input, a pot", []int{29}, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newRaidWorld(t, 0)
			w.m.inBattle = func(int) bool { return false }
			w.m.campCfg.Recipes = []campRecipe{{Output: 30020, Inputs: tc.inputs}}
			supplies := stock{}
			if tc.pot {
				supplies[cookpotItemID] = 1
			}
			supplies.install(w.m)
			cargo := newFakeCargo(100000)
			useCargo(t, cargo)
			for _, id := range tc.inputs {
				cargo.stacks[id]++
			}
			text := w.m.cook(w.user, w.room, nil)
			assert.Contains(t, text, "cargo")
			assert.Equal(t, tc.want, cargo.stacks[30020], text)
			if tc.want == 2 {
				assert.Contains(t, text, "cookpot")
			} else {
				assert.NotContains(t, text, "cookpot")
			}
		})
	}
}

func TestGearLinesListWhatIsInUse(t *testing.T) {
	g := campGear{Bedrolls: []string{"leader"}, Tent: true, FireSteel: true, Cookpot: true, Bells: true, Kit: true}
	out := strings.Join(g.lines(3), "\n")
	for _, want := range []string{"1 of 3 members", "tent", "Fire steel", "cookpot", "bells and trip lines", "surgeon's kit"} {
		assert.Contains(t, out, want)
	}
	assert.Empty(t, campGear{}.lines(3))
}

// The shipped gear items carry the designed weights, uses and values.
func TestShippedGearItemsMatchTheDesign(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default", "items", "other-0")
	for _, want := range []struct {
		id, weight, uses, value int
		name                    string
	}{
		{bedrollItemID, 2500, 0, 8, "bedroll"},
		{tentItemID, 9000, 0, 40, "oiled canvas tent"},
		{fireSteelItemID, 200, 0, 5, "fire steel and tinder"},
		{cookpotItemID, 3000, 0, 12, "iron cookpot"},
		{campBellsItemID, 1000, 10, 6, "camp bells and trip lines"},
		{surgeonKitItemID, 1500, 5, 25, "field surgeon's kit"},
	} {
		matches, err := filepath.Glob(filepath.Join(dir, fmt.Sprintf("%d-*.yaml", want.id)))
		require.NoError(t, err)
		require.Len(t, matches, 1, "item %d", want.id)
		data, err := os.ReadFile(matches[0])
		require.NoError(t, err)
		var spec items.ItemSpec
		require.NoError(t, yaml.Unmarshal(data, &spec))
		assert.Equal(t, want.name, spec.Name)
		assert.Equal(t, want.weight, spec.Weight, want.name)
		assert.Equal(t, want.uses, spec.Uses, want.name)
		assert.Equal(t, want.value, spec.Value, want.name)
	}
}

// 40a3 review: between rests, camp status lists the gear at hand and what
// it will do, so a player sees it before resting; a running rest reports
// only the gear locked for it.
func TestCampStatusListsGearAtHandBetweenRests(t *testing.T) {
	w := newRaidWorld(t, 0)
	withRoster(w.m)
	w.m.camps = map[int]camping.Camp{}
	stock{bedrollItemID: 2, cookpotItemID: 1, surgeonKitItemID: 1}.install(w.m)
	w.m.establish(w.user, w.room)
	text := w.m.status(7)
	assert.Contains(t, text, "Camp gear at hand (help camp gear):")
	assert.Contains(t, text, "Bedrolls: 2 of 3 members sleep on one (+25% fatigue recovered).")
	assert.Contains(t, text, "Iron cookpot: a multi-ingredient dish makes one extra portion.")
	assert.Contains(t, text, "Field surgeon's kit")

	stock{}.install(w.m)
	assert.NotContains(t, w.m.status(7), "Camp gear at hand", "no gear, no list")
}
