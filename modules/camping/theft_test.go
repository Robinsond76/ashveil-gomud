package camping

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 40a4: camp theft.

type theftCall struct {
	share, max int
	protected  []int
}

// theftWorld is a raid world whose Road zone draws thieves at a chance.
// Rolls are consumed in order: the raid (none: its chance is 0) and then
// the theft roll at rest start, and then any watch roll.
func theftWorld(t *testing.T, chance int, rolls ...int) (*raidWorld, *[]theftCall) {
	t.Helper()
	w := newRaidWorld(t, 0, rolls...)
	w.m.campCfg.Thefts["Road"] = chance
	calls := &[]theftCall{}
	w.m.theft = func(_, share, maxUnits int, pick func(int) int, protect func(int) bool) []company.TheftLoss {
		call := theftCall{share: share, max: maxUnits}
		for _, id := range []int{bedrollItemID, tentItemID, fireSteelItemID, cookpotItemID, campBellsItemID, surgeonKitItemID, 29, 30} {
			if protect(id) {
				call.protected = append(call.protected, id)
			}
		}
		*calls = append(*calls, call)
		return []company.TheftLoss{{ItemID: 29, Name: "raw game meat", Count: 2}, {ItemID: 30018, Name: "wild thyme", Count: 1}}
	}
	return w, calls
}

func TestTheftIsRolledAtRestStartOnlyWithoutBells(t *testing.T) {
	w, _ := theftWorld(t, 20, 19)
	w.rest(t)
	require.NotNil(t, w.camp().Rest.Theft, "19 < 20")
	assert.False(t, w.camp().Rest.Theft.Done)
	require.NotNil(t, w.store.saved.Camps[7].Rest.Theft, "durable")

	quiet, _ := theftWorld(t, 20, 20)
	quiet.rest(t)
	assert.Nil(t, quiet.camp().Rest.Theft, "20 is not under 20")

	bells, _ := theftWorld(t, 100, 0)
	stock{campBellsItemID: 1}.install(bells.m)
	bells.rest(t)
	assert.Nil(t, bells.camp().Rest.Theft, "bells and trip lines keep thieves out")

	unlisted := newRaidWorld(t, 0, 0)
	unlisted.rest(t)
	assert.Nil(t, unlisted.camp().Rest.Theft, "a zone with no theft chance is never robbed")
}

func TestTheftHappensOnceAfterTheRestAndIsReported(t *testing.T) {
	w, calls := theftWorld(t, 100, 0)
	w.rest(t)
	messages := captureMessages(t)

	w.m.onNewRound(events.NewRound{})
	assert.Empty(t, *calls, "thieves work while the company sleeps; nothing yet")

	finishRest(w)
	// The theft is saved as done before anything is taken.
	w.m.theft = func(leader, share, maxUnits int, pick func(int) int, protect func(int) bool) []company.TheftLoss {
		assert.True(t, w.store.saved.Camps[7].Rest.Theft.Done, "saved before it is taken")
		assert.Equal(t, 7, leader)
		return []company.TheftLoss{{ItemID: 29, Name: "raw game meat", Count: 2}, {ItemID: 30018, Name: "wild thyme", Count: 1}}
	}
	w.m.onNewRound(events.NewRound{})
	events.ProcessEvents()
	out := strings.Join(*messages, "\n")
	assert.Contains(t, out, "Missing: 2 raw game meat, wild thyme.")
	assert.Contains(t, out, "help camp gear")
	assert.True(t, w.camp().Rest.Theft.Done)

	n := len(*messages)
	w.m.onNewRound(events.NewRound{})
	assert.Len(t, *messages, n, "once")

	// A restart never robs the same rest twice.
	reloaded := newTestModule(w.store, &fakeScheduler{}, &fakeSurvival{}, func() time.Time { return *w.now })
	reloaded.load()
	require.True(t, reloaded.camps[7].Rest.Theft.Done)
	reloaded.theft = func(int, int, int, func(int) int, func(int) bool) []company.TheftLoss {
		t.Fatal("robbed twice")
		return nil
	}
	reloaded.resolveCampTheft()
}

func TestTheftLeavesCampGearAlone(t *testing.T) {
	w, calls := theftWorld(t, 100, 0)
	w.rest(t)
	finishRest(w)
	w.m.onNewRound(events.NewRound{})
	require.Len(t, *calls, 1)
	assert.Equal(t, []int{bedrollItemID, tentItemID, fireSteelItemID, cookpotItemID, campBellsItemID, surgeonKitItemID}, (*calls)[0].protected)
	assert.Equal(t, 10, (*calls)[0].share)
	assert.Equal(t, 4, (*calls)[0].max)
}

func TestTheftWaitsForAnOnlineLeaderOutOfBattle(t *testing.T) {
	w, calls := theftWorld(t, 100, 0)
	w.rest(t)
	finishRest(w)

	w.m.inBattle = func(int) bool { return true }
	w.m.onNewRound(events.NewRound{})
	assert.Empty(t, *calls, "not mid-fight")
	assert.False(t, w.camp().Rest.Theft.Done)

	w.m.inBattle = nil
	w.m.onNewRound(events.NewRound{})
	assert.Len(t, *calls, 1)
}

func TestAWatchMayCatchTheThieves(t *testing.T) {
	level1 := archetypes.Specialist{Name: "Bran", Level: 1} // 25%
	run := func(roll int) (*raidWorld, *[]theftCall, string) {
		w, calls := theftWorld(t, 100, 0, roll)
		w.m.specialist = func(int, string, ...int) (archetypes.Specialist, bool) { return level1, true }
		w.rest(t)
		messages := captureMessages(t)
		finishRest(w)
		w.m.onNewRound(events.NewRound{})
		events.ProcessEvents()
		return w, calls, strings.Join(*messages, "\n")
	}
	w, calls, out := run(24)
	assert.Empty(t, *calls, "caught: nothing is taken")
	assert.Contains(t, out, "nothing is missing")
	assert.True(t, w.camp().Rest.Theft.Done)

	_, calls, out = run(25)
	assert.Len(t, *calls, 1, "25 is not under 25")
	assert.Contains(t, out, "Missing:")
}

func TestThievesFindNothingWorthTaking(t *testing.T) {
	w, _ := theftWorld(t, 100, 0)
	w.m.theft = func(int, int, int, func(int) int, func(int) bool) []company.TheftLoss { return nil }
	w.rest(t)
	messages := captureMessages(t)
	finishRest(w)
	w.m.onNewRound(events.NewRound{})
	events.ProcessEvents()
	assert.NotContains(t, strings.Join(*messages, "\n"), "Missing")
	assert.True(t, w.camp().Rest.Theft.Done)
}

func TestCampTheftSettingsParse(t *testing.T) {
	cfg := parseCampSettings(func(key string) any {
		switch key {
		case "CampTheft":
			return []any{map[string]any{"Zone": "Old Kings Road", "ChancePct": 20}, map[string]any{"Zone": "Bad", "ChancePct": 200}}
		case "TheftSharePct":
			return 25
		case "TheftMaxItems":
			return 0 // out of range: default stays
		}
		return nil
	})
	assert.Equal(t, map[string]int{"Old Kings Road": 20}, cfg.Thefts)
	assert.Equal(t, 25, cfg.TheftSharePct)
	assert.Equal(t, 4, cfg.TheftMaxItems)
}

func TestTheftTextAndGearLabels(t *testing.T) {
	assert.Contains(t, theftText([]company.TheftLoss{{Name: "a firewood bundle", Count: 1}}), "Missing: a firewood bundle.")
	g := campGear{Bedrolls: []string{"leader"}, Tent: true, Bells: true}
	assert.Equal(t, []string{"Bedrolls 1/3", "Canvas tent", "Bells and trip lines"}, g.labels(3))
	assert.Contains(t, strings.Join(g.lines(3), "\n"), "thieves keep out")
}

func TestCampStateCarriesGearLabels(t *testing.T) {
	w := newRaidWorld(t, 0)
	stock{tentItemID: 1, cookpotItemID: 1}.install(w.m)
	s, ok := w.m.CampStateOf(7, 100, nil)
	require.True(t, ok)
	assert.Equal(t, []string{"Canvas tent", "Cookpot"}, s.Gear)
	none, _ := w.m.CampStateOf(99, 100, nil)
	assert.Empty(t, none.Gear, "no camp, no gear line")
}

// 40a4 review: breaking camp, or resting again, straight after a rest does
// not slip between the rest's end and the next round to dodge thieves,
// even before the rest's timer has fired.
func TestBreakingCampOrRestingAgainDoesNotDodgeThieves(t *testing.T) {
	w, calls := theftWorld(t, 100, 0)
	w.rest(t)
	messages := captureMessages(t)
	w.at(camping.RestDuration + time.Second) // due, timer not yet fired
	assert.Equal(t, "You break camp.", w.m.breakCamp(w.user, w.room))
	events.ProcessEvents()
	require.Len(t, *calls, 1, "robbed as camp is broken")
	assert.Contains(t, strings.Join(*messages, "\n"), "Missing: 2 raw game meat")

	again, againCalls := theftWorld(t, 50, 0) // robbed; later rolls are 99, so the next rest draws none
	again.rest(t)
	finishRest(again)
	again.m.lightFire(again.user, again.room)
	again.rest(t)
	require.Len(t, *againCalls, 1, "the first rest's thieves come before the next rest")
	assert.Nil(t, again.camp().Rest.Theft)
}

// 40a4 review: the player is warned on a road thieves work, at rest start
// and on the web Camp tab, unless bells are carried.
func TestThievesAreWarnedOfAtRestStartAndOnTheCampTab(t *testing.T) {
	const warning = "thieves may slip into the camp"
	w, _ := theftWorld(t, 20, 99)
	s, _ := w.m.CampStateOf(7, 100, nil)
	assert.True(t, s.TheftRisk, "camp tab: thieves work this road")
	assert.Contains(t, w.m.startRest(w.user, w.room), warning)

	bells, _ := theftWorld(t, 20, 99)
	stock{campBellsItemID: 1}.install(bells.m)
	s, _ = bells.m.CampStateOf(7, 100, nil)
	assert.False(t, s.TheftRisk, "bells carried")
	assert.NotContains(t, bells.m.startRest(bells.user, bells.room), warning)

	safe := newRaidWorld(t, 0, 99)
	s, _ = safe.m.CampStateOf(7, 100, nil)
	assert.False(t, s.TheftRisk, "no thieves on this road")
	assert.NotContains(t, safe.m.startRest(safe.user, safe.room), warning)
}
