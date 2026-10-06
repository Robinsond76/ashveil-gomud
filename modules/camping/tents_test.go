package camping

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/weather"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Phase 52: tents with trade-offs.

const (
	furTentItemID  = 300
	campTentItemID = 301
	bigTentItemID  = 302
)

func tentStock(items ...int) stock {
	s := stock{}
	for _, id := range items {
		s[id]++
	}
	return s
}

// The fur-lined tent ends the weather's penalty to a rest; canvas halves it.
func TestFurTentEndsTheWeathersPenaltyToARest(t *testing.T) {
	storm := func(string) (weather.Condition, bool) {
		return weather.Condition{Name: "storm", RestRecoveryPct: 50}, true
	}
	for name, tc := range map[string]struct {
		items []int
		want  int
		note  string
	}{
		"none":   {nil, 10, ""},
		"canvas": {[]int{tentItemID}, 15, "The tent softens it."},
		"fur":    {[]int{furTentItemID}, 20, ""},
	} {
		w := newRaidWorld(t, 0)
		w.m.weatherIn = storm
		tentStock(tc.items...).install(w.m)
		text := w.m.startRest(w.user, w.room)
		assert.Equal(t, tc.want, w.camp().Rest.Recovery, name)
		if tc.note != "" {
			assert.Contains(t, text, tc.note, name)
		}
	}
	w := newRaidWorld(t, 0)
	w.m.weatherIn = storm
	tentStock(furTentItemID).install(w.m)
	assert.NotContains(t, w.m.startRest(w.user, w.room), "poorer rest", "a fur tent shuts the weather out, so there is nothing to report")
	assert.Equal(t, camping.TentFur, w.camp().Rest.Tent)
}

// A camouflaged tent halves the raid chance and a large one raises it half
// again; both are rolled in the same single roll at the rest's start.
func TestTentsScaleTheRaidChance(t *testing.T) {
	raids := func(items []int, chance, roll int) bool {
		w := newRaidWorld(t, chance, roll, 0)
		tentStock(items...).install(w.m)
		w.rest(t)
		return w.camp().Rest.Raid != nil
	}
	assert.True(t, raids([]int{tentItemID}, 30, 20), "canvas leaves the 30% alone: 20 raids")
	assert.False(t, raids([]int{campTentItemID}, 30, 20), "camouflaged: 15%, a roll of 20 is safe")
	assert.True(t, raids([]int{campTentItemID}, 30, 14), "but 14 still raids")
	assert.False(t, raids([]int{tentItemID}, 30, 40), "canvas: 40 is safe")
	assert.True(t, raids([]int{bigTentItemID}, 30, 40), "large: 45%, 40 raids")
	assert.False(t, raids(nil, 30, 40), "no tent leaves it alone")
}

func TestTentsScaleTheThiefChanceAndBellsStillWin(t *testing.T) {
	thieves := func(items []int, chance, roll int) bool {
		w, _ := theftWorld(t, chance, roll)
		tentStock(items...).install(w.m)
		w.rest(t)
		return w.camp().Rest.Theft != nil
	}
	assert.True(t, thieves([]int{tentItemID}, 40, 30))
	assert.False(t, thieves([]int{campTentItemID}, 40, 30), "camouflaged: 20%")
	assert.False(t, thieves([]int{tentItemID}, 40, 50))
	assert.True(t, thieves([]int{bigTentItemID}, 40, 50), "large: 60%")

	w, _ := theftWorld(t, 100, 0)
	tentStock(bigTentItemID, campBellsItemID).install(w.m)
	w.rest(t)
	assert.Nil(t, w.camp().Rest.Theft, "bells keep thieves out whatever the tent")
}

func TestThievesNeverTakeATent(t *testing.T) {
	for _, tent := range camping.Tents {
		assert.True(t, campGearItem(tent.ItemID), tent.Name)
	}
	assert.False(t, campGearItem(29))
}

// A large tent's rest grants Well Rested to everyone who slept (the leader
// and companions), for the inn's length; wounds and vitals still follow the
// camp rest, so only the buff changes.
func TestLargeTentGrantsWellRestedToEveryoneWhoSlept(t *testing.T) {
	e, user := heroEnv(t)
	tentStock(bigTentItemID).install(e.module)
	messages := captureMessages(t)
	e.completeCamp(t, user)
	assert.Equal(t, camping.TentLarge, e.store.saved.RestedTents[7], "saved with the pending grant")

	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Equal(t, []buffCall{{"Hero", 1030}, {"Bran", 1030}}, *e.buffs)
	assert.Equal(t, 450, e.ledger.rounds[buffCall{"Hero", 1030}], "30 minutes at 4-second rounds")
	assert.Empty(t, e.store.saved.RestedTents, "settled once, then cleared")
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*messages, "\n"), "well rested")
	assert.NotContains(t, strings.Join(*messages, "\n"), "Your company is Rested")
}

func TestLargeTentStillExcludesMembersOnADuty(t *testing.T) {
	d := newDutyWorld(t, 0)
	tentStock(bigTentItemID).install(d.m)
	d.duty(t, "mira", "watch")
	d.rest(t)
	d.finishAndGrant()
	assert.Contains(t, d.buffs[d.bran], 1030, "the sleeper wakes Well Rested")
	assert.NotContains(t, d.buffs[d.bran], 1033)
	assert.Contains(t, d.buffs[d.user.Character], 1030)
	assert.Empty(t, d.buffs[d.mira], "the watcher stayed up")
}

// A camouflaged tent costs rest: Rested lasts half as long.
func TestCamouflagedTentHalvesRestedAndKeepsTheTier(t *testing.T) {
	e, user := heroEnv(t)
	tentStock(campTentItemID).install(e.module)
	e.completeCamp(t, user)
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Equal(t, []buffCall{{"Hero", 1033}, {"Bran", 1033}}, *e.buffs, "still Rested, never Well Rested")
	assert.Equal(t, 113, e.ledger.rounds[buffCall{"Hero", 1033}], "7 and a half minutes at 4-second rounds (half of 225, rounded)")
}

func TestPlainAndCanvasRestsGrantTheUnchangedRested(t *testing.T) {
	for name, items := range map[string][]int{"none": nil, "canvas": {tentItemID}, "fur": {furTentItemID}} {
		e, user := heroEnv(t)
		tentStock(items...).install(e.module)
		e.completeCamp(t, user)
		e.module.onNewRound(events.NewRound{RoundNumber: 1})
		assert.Equal(t, []buffCall{{"Hero", 1033}, {"Bran", 1033}}, *e.buffs, name)
		assert.Equal(t, 225, e.ledger.rounds[buffCall{"Hero", 1033}], name)
	}
}

// The tent a pending grant was slept in is saved with it: breaking camp, or
// resting under another tent, before the grant neither drops nor swaps it,
// and a restart keeps it.
func TestPendingGrantKeepsItsTentAcrossBreakCampAndReload(t *testing.T) {
	e, user := heroEnv(t)
	tentStock(bigTentItemID).install(e.module)
	e.completeCamp(t, user)
	require.Equal(t, camping.TentLarge, e.store.saved.RestedTents[7])

	// Break camp before the round grants the buff.
	assert.Contains(t, e.module.breakCamp(user, eligibleRoom()), "break camp")
	assert.Equal(t, camping.TentLarge, e.store.saved.RestedTents[7], "the grant still owes the large tent's buff")

	child := newTestModule(e.store, &fakeScheduler{}, &fakeSurvival{}, func() time.Time { return *e.now })
	child.load()
	require.NoError(t, child.loadErr)
	calls := []buffCall{}
	installLedger(child, &calls)
	child.companionsOf = e.module.companionsOf
	child.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Equal(t, []buffCall{{"Hero", 1030}, {"Bran", 1030}}, calls)
}

func TestRestedTentIsKeptWhenTheGrantSaveFails(t *testing.T) {
	e, user := heroEnv(t)
	tentStock(bigTentItemID).install(e.module)
	e.completeCamp(t, user)
	e.store.failNextSave = true
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Equal(t, camping.TentLarge, e.module.restedTents[7], "a failed save restores the tent with the marker")
	assert.True(t, e.module.restedPending[7])
}

func TestPurgeDropsThePendingTent(t *testing.T) {
	e, user := heroEnv(t)
	tentStock(bigTentItemID).install(e.module)
	e.completeCamp(t, user)
	e.module.mu.Lock()
	dropped := e.module.purgeLocked(7)
	e.module.mu.Unlock()
	assert.True(t, dropped)
	assert.Empty(t, e.module.restedTents)
}

// camp tent: view, choose, refuse, persist.
func TestCampTentChoosesAmongCarriedTents(t *testing.T) {
	w := newRaidWorld(t, 0)
	tentStock(tentItemID, furTentItemID).install(w.m)

	w.m.tentCommand(w.user, w.room, []string{"canvas"})
	view := w.m.tentCommand(w.user, w.room, nil)
	assert.Contains(t, view, "Oiled canvas tent (canvas)")
	assert.Contains(t, view, "Fur-lined tent (fur)")
	assert.Contains(t, view, "* Oiled canvas tent", "canvas is pitched by default")
	assert.NotContains(t, view, "pavilion", "only tents carried are listed")

	text := w.m.tentCommand(w.user, w.room, []string{"fur"})
	assert.Contains(t, text, "You pitch the fur-lined tent")
	camp := w.camp()
	assert.Equal(t, camping.TentFur, camp.TentKind)
	assert.Equal(t, camping.TentFur, w.m.tentChoices[7])
	assert.True(t, camp.Tent)
	assert.Equal(t, camping.TentFur, w.store.saved.TentChoices[7], "durable")
	assert.Contains(t, w.m.status(7), "A fur-lined tent is pitched here")
	assert.Contains(t, camping.CampLines(w.m.RoomCamps(100), 7, nil)[0], "a fur-lined tent")

	// The choice carries into the rest.
	w.rest(t)
	assert.Equal(t, camping.TentFur, w.camp().Rest.Tent)
	assert.Contains(t, w.m.status(7), "the fur-lined tent keeps out the weather")
}

func TestCampTentRefusals(t *testing.T) {
	w := newRaidWorld(t, 0)
	tentStock(tentItemID).install(w.m)
	assert.Contains(t, w.m.tentCommand(w.user, w.room, []string{"large"}), "You carry no large pavilion tent")
	assert.Contains(t, w.m.tentCommand(w.user, w.room, []string{"palace"}), "There is no tent called")

	away := *w.room
	away.RoomId = 101
	assert.Contains(t, w.m.tentCommand(w.user, &away, []string{"canvas"}), "not here")

	tentStock(tentItemID, bigTentItemID).install(w.m)
	w.rest(t)
	assert.Contains(t, w.m.tentCommand(w.user, w.room, []string{"large"}), "The company is resting")
	assert.Equal(t, camping.TentCanvas, w.camp().Rest.Tent, "locked for the rest")

	e := newInnEnv(t)
	assert.Contains(t, e.module.tentCommand(campUser(t, 9, 100), eligibleRoom(), nil), "You carry no tent")
}

// 52 review: the choice outlives the camp, so a company carrying two tents
// need not choose again at every camp, and it can choose before camping.
func TestTentChoiceOutlivesTheCamp(t *testing.T) {
	w := newRaidWorld(t, 0)
	tentStock(tentItemID, furTentItemID).install(w.m)
	w.m.tentCommand(w.user, w.room, []string{"fur"})

	delete(w.m.camps, 7)
	view := w.m.tentCommand(w.user, w.room, nil)
	assert.Contains(t, view, "* Fur-lined tent", "with no camp up the view marks the next camp's tent")
	assert.Contains(t, view, "your next camp pitches")
	assert.Contains(t, w.m.establish(w.user, w.room), "fur-lined tent", "the next camp pitches the chosen tent")

	delete(w.m.camps, 7)
	assert.Contains(t, w.m.tentCommand(w.user, w.room, []string{"canvas"}), "Your next camp will pitch the oiled canvas tent")
	assert.Equal(t, camping.TentCanvas, w.store.saved.TentChoices[7], "chosen without a camp, and saved")

	// It survives a reload.
	data, err := yaml.Marshal(w.store.saved)
	require.NoError(t, err)
	loaded := NewRegistry()
	require.NoError(t, decodeRegistry(data, loaded))
	assert.Equal(t, camping.TentCanvas, loaded.TentChoices[7])
}

func TestCampTentNoTentCarried(t *testing.T) {
	w := newRaidWorld(t, 0)
	tentStock().install(w.m)
	assert.Contains(t, w.m.tentCommand(w.user, w.room, nil), "You carry no tent")
	assert.False(t, w.camp().Tent)
	w.rest(t)
	assert.Empty(t, w.camp().Rest.Tent)
}

// A chosen tent that is later lost or sold falls back to one still carried,
// and clear forgets the choice.
func TestChosenTentFallsBackAndClears(t *testing.T) {
	w := newRaidWorld(t, 0)
	s := tentStock(tentItemID, bigTentItemID)
	s.install(w.m)
	w.m.tentCommand(w.user, w.room, []string{"large"})
	require.Equal(t, camping.TentLarge, w.camp().TentKind)

	delete(s, bigTentItemID)
	w.rest(t)
	assert.Equal(t, camping.TentCanvas, w.camp().Rest.Tent, "the large tent is gone: canvas")
}

func TestCampTentClearResetsToTheDefault(t *testing.T) {
	w := newRaidWorld(t, 0)
	tentStock(tentItemID, bigTentItemID).install(w.m)
	w.m.tentCommand(w.user, w.room, []string{"large"})
	require.Equal(t, camping.TentLarge, w.camp().TentKind)
	assert.Contains(t, w.m.tentCommand(w.user, w.room, []string{"clear"}), "You pitch the oiled canvas tent")
	assert.Empty(t, w.m.tentChoices)
	assert.Equal(t, camping.TentCanvas, w.camp().TentKind)
}

// Lighting the fire or making camp pitches the chosen or default tent.
func TestEstablishPitchesTheDefaultTent(t *testing.T) {
	w := newRaidWorld(t, 0)
	w.m.camps = map[int]camping.Camp{}
	tentStock(furTentItemID).install(w.m)
	assert.Contains(t, w.m.establish(w.user, w.room), "fur-lined tent")
	assert.Equal(t, camping.TentFur, w.camp().TentKind)
}

// An old camp saved before tent kinds (Tent set, no kind) reads as canvas.
func TestOldCampWithATentReadsAsCanvas(t *testing.T) {
	w := newRaidWorld(t, 0)
	c := w.camp()
	c.Tent, c.TentKind = true, ""
	w.m.camps[7] = c
	assert.Contains(t, w.m.status(7), "An oiled canvas tent is pitched here")
}

// The Camp tab's state lists the tents carried and the one pitched.
func TestCampStateListsTheTentsCarried(t *testing.T) {
	w := newRaidWorld(t, 0)
	tentStock(tentItemID, bigTentItemID).install(w.m)
	w.m.tentCommand(w.user, w.room, []string{"large"})
	s, ok := w.m.CampStateOf(7, 100, nil)
	require.True(t, ok)
	assert.Equal(t, camping.TentLarge, s.TentKind)
	assert.Contains(t, s.TentNote, "Well Rested")
	require.Len(t, s.Tents, 2)
	assert.Equal(t, camping.TentChoice{Kind: camping.TentCanvas, Name: "oiled canvas tent", Effect: camping.TentOf(camping.TentCanvas).Effect}, s.Tents[0])
	assert.True(t, s.Tents[1].Pitched)
	assert.Contains(t, s.Gear, "Large tent")
}

func TestGearLinesNameThePitchedTentAndTheOthers(t *testing.T) {
	g := campGear{Tent: true, TentKind: camping.TentCamouflaged, Tents: []camping.TentKind{camping.TentCanvas, camping.TentCamouflaged}}
	lines := strings.Join(g.lines(2), "\n")
	assert.Contains(t, lines, "Camouflaged tent: raiders and thieves come half as often")
	assert.Contains(t, lines, "Tents carried: canvas, camouflaged")
}

// Tents are bought, never sold back; each shipped item matches the design;
// and each shop line is supply-only.
func TestShippedTentItemsAndShopsMatchTheDesign(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	dir := filepath.Join(root, "_datafiles", "world", "default", "items", "other-0")
	for _, want := range []struct {
		kind          camping.TentKind
		weight, value int
	}{
		{camping.TentCanvas, 9000, 40}, {camping.TentFur, 12000, 90}, {camping.TentCamouflaged, 7000, 70}, {camping.TentLarge, 20000, 120},
	} {
		tent := camping.TentOf(want.kind)
		matches, err := filepath.Glob(filepath.Join(dir, fmt.Sprintf("%d-*.yaml", tent.ItemID)))
		require.NoError(t, err)
		require.Len(t, matches, 1, "item %d", tent.ItemID)
		data, err := os.ReadFile(matches[0])
		require.NoError(t, err)
		var spec struct {
			Name   string `yaml:"name"`
			Weight int    `yaml:"weight"`
			Value  int    `yaml:"value"`
		}
		require.NoError(t, yaml.Unmarshal(data, &spec))
		assert.Equal(t, tent.Name, spec.Name)
		assert.Equal(t, want.weight, spec.Weight, tent.Name)
		assert.Equal(t, want.value, spec.Value, tent.Name)
	}

	data, err := os.ReadFile(filepath.Join(root, "modules", "market", "files", "data-overlays", "config.yaml"))
	require.NoError(t, err)
	var cfg map[string]any
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	counts := map[int]int{}
	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for _, child := range x {
				walk(child)
			}
		case map[any]any:
			if id, ok := x["ItemId"].(int); ok {
				for _, tent := range camping.Tents {
					if tent.ItemID == id {
						counts[id]++
						assert.Equal(t, true, x["SupplyOnly"], "tent %d is sold and never bought back", id)
						assert.LessOrEqual(t, x["MinPrice"], x["BasePrice"])
					}
				}
			}
			for _, child := range x {
				walk(child)
			}
		case []any:
			for _, child := range x {
				walk(child)
			}
		}
	}
	walk(cfg)
	for _, tent := range camping.Tents {
		assert.GreaterOrEqual(t, counts[tent.ItemID], 1, "%s is on sale somewhere", tent.Name)
	}
}
