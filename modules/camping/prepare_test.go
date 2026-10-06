package camping

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 43a: camp supplies (fortifying broth, warming draught, cooling
// salve, watch incense), prepared with `camp prepare`.

func prepWorld(t *testing.T, supplies stock) (*raidWorld, *buffLedger, *[]buffCall) {
	t.Helper()
	w := newRaidWorld(t, 0)
	calls := &[]buffCall{}
	ledger := installLedger(w.m, calls)
	w.m.inBattle = func(int) bool { return false }
	supplies.install(w.m)
	return w, ledger, calls
}

func prepare(w *raidWorld, args ...string) string {
	return w.m.prepareCommand(w.user, w.room, args)
}

func TestSupplyWordsMatchTheirSupplies(t *testing.T) {
	for word, key := range map[string]string{"broth": "broth", "fortifying": "broth", "Warming": "warming", "draught": "warming", "salve": "cooling", "incense": "incense", "watch": "incense"} {
		s, ok := camping.FindSupply(word)
		require.True(t, ok, word)
		assert.Equal(t, key, s.Key, word)
	}
	_, ok := camping.FindSupply("poison")
	assert.False(t, ok)
}

func TestBrothSizeFollowsMaximumHealthAtAboutFivePercent(t *testing.T) {
	for _, tc := range []struct{ hp, bonus int }{{1, 2}, {30, 2}, {40, 2}, {41, 5}, {100, 5}, {101, 10}, {200, 10}, {201, 20}, {400, 20}} {
		assert.Equal(t, tc.bonus, camping.BrothTierFor(tc.hp).Bonus, "%d hp", tc.hp)
	}
	for _, tier := range camping.BrothBuffs {
		assert.True(t, camping.IsBrothBuff(tier.BuffID))
	}
	assert.False(t, camping.IsBrothBuff(camping.WarmingBuffID))
}

func TestIncenseAddsTenPointsCapsAtNinetyAndKeepsCertainty(t *testing.T) {
	for _, tc := range []struct{ in, want int }{{0, 10}, {25, 35}, {80, 90}, {85, 90}, {90, 90}, {95, 95}, {100, 100}} {
		assert.Equal(t, tc.want, camping.IncenseChance(tc.in), "%d%%", tc.in)
	}
}

func TestWarmingDraughtStartsAtOnceAndSpendsOneDose(t *testing.T) {
	supplies := stock{camping.WarmingItemID: 2}
	w, ledger, calls := prepWorld(t, supplies)
	text := prepare(w, "warming")
	assert.Contains(t, text, "You drink the warming draught")
	assert.Contains(t, text, "15 minutes")
	assert.Equal(t, []buffCall{{w.user.Character.Name, camping.WarmingBuffID}}, *calls)
	assert.Zero(t, ledger.rounds[(*calls)[0]], "the buff's own fifteen real minutes, no override")
	assert.Equal(t, 1, supplies[camping.WarmingItemID])
}

func TestCoolingSalveGivesTheCoolingBuff(t *testing.T) {
	w, _, calls := prepWorld(t, stock{camping.CoolingItemID: 1})
	assert.Contains(t, prepare(w, "cooling"), "You rub on the cooling salve")
	assert.Equal(t, camping.CoolingBuffID, (*calls)[0].id)
}

func TestPreparingWithNoStockConsumesNothing(t *testing.T) {
	supplies := stock{}
	w, _, calls := prepWorld(t, supplies)
	assert.Contains(t, prepare(w, "warming"), "You carry no warming draught. Nothing was used.")
	assert.Empty(t, *calls)
	assert.Contains(t, prepare(w, "broth"), "You carry no fortifying broth")
	assert.Nil(t, w.camp().Prepared)
}

func TestPreparingForTheWholeCompanyNeedsEnoughDosesAndSpendsNothingShort(t *testing.T) {
	e, user := heroEnv(t)
	e.companion.RoomId = 100
	user.Character.RoomId = 100
	supplies := stock{camping.WarmingItemID: 1}
	supplies.install(e.module)
	e.module.inBattle = func(int) bool { return false }
	e.module.establish(user, eligibleRoom())
	text := e.module.prepareCommand(user, eligibleRoom(), []string{"warming", "all"})
	assert.Contains(t, text, "You need 2 warming draught for that, and carry 1. Nothing was used.")
	assert.Empty(t, *e.buffs)
	assert.Equal(t, 1, supplies[camping.WarmingItemID])

	supplies[camping.WarmingItemID] = 2
	text = e.module.prepareCommand(user, eligibleRoom(), []string{"warming", "all"})
	assert.Contains(t, text, "Bran drinks the warming draught")
	assert.Equal(t, []buffCall{{"Hero", camping.WarmingBuffID}, {"Bran", camping.WarmingBuffID}}, *e.buffs)
	assert.Zero(t, supplies[camping.WarmingItemID])

	// A companion who is elsewhere is not at the camp to be prepared.
	e.companion.RoomId = 5
	assert.Contains(t, e.module.prepareCommand(user, eligibleRoom(), []string{"warming", "bran"}), "nobody called")
}

func TestOnePersonalBenefitAtATimeAndNothingIsReplacedSilently(t *testing.T) {
	supplies := stock{camping.WarmingItemID: 1, camping.CoolingItemID: 1}
	w, ledger, calls := prepWorld(t, supplies)
	require.Contains(t, prepare(w, "warming"), "You drink")
	text := prepare(w, "cooling")
	assert.Contains(t, text, "already has a warming draught")
	assert.Len(t, *calls, 1)
	assert.Equal(t, 1, supplies[camping.CoolingItemID], "the salve was not spent")

	// The same supply neither spends a dose nor refreshes its time.
	supplies[camping.WarmingItemID] = 1
	assert.Contains(t, prepare(w, "warming"), "already has a warming draught")
	assert.Equal(t, 1, supplies[camping.WarmingItemID])

	// Clearing drops it, refunds nothing, and frees the slot.
	assert.Contains(t, prepare(w, "clear"), "Nothing is refunded")
	assert.Equal(t, []buffCall{{w.user.Character.Name, camping.WarmingBuffID}}, ledger.removed)
	assert.Contains(t, prepare(w, "cooling"), "You rub")
}

func TestPreparingNeedsACampHereAndNoRestOrFight(t *testing.T) {
	w, _, calls := prepWorld(t, stock{camping.WarmingItemID: 5})
	w.m.camps[7] = withRoom(w.camp(), 999)
	assert.Equal(t, "Your camp is not here.", prepare(w, "warming"))
	w.m.camps[7] = withRoom(w.camp(), 100)

	w.m.inBattle = func(int) bool { return true }
	assert.Contains(t, prepare(w, "warming"), "middle of a fight")
	w.m.inBattle = func(int) bool { return false }

	w.rest(t)
	assert.Contains(t, prepare(w, "warming"), "while the company rests")
	assert.Empty(t, *calls)

	noCamp := newRaidWorld(t, 0)
	noCamp.m.camps = map[int]camping.Camp{}
	assert.Contains(t, noCamp.m.prepareCommand(noCamp.user, noCamp.room, []string{"warming"}), "You have no camp")
}

func withRoom(c camping.Camp, roomID int) camping.Camp {
	c.RoomID = roomID
	return c
}

func TestBrothIsQueuedThenSpentAtRestStartAndLockedOnTheRest(t *testing.T) {
	supplies := stock{camping.BrothItemID: 1}
	w, _, _ := prepWorld(t, supplies)
	text := prepare(w, "broth")
	assert.Contains(t, text, "Fortifying broth is set by for")
	assert.Equal(t, 1, supplies[camping.BrothItemID], "nothing is spent until the rest begins")
	require.NotNil(t, w.camp().Prepared)
	assert.True(t, w.camp().Prepared.HasBroth(string(survival.LeaderMemberKey)))
	assert.Contains(t, w.store.saved.Camps[7].Prepared.Broth, string(survival.LeaderMemberKey), "durable")
	assert.Contains(t, w.m.status(7), "Set by for the next rest")

	// Queued twice is refused, and queueing the broth blocks a draught.
	assert.Contains(t, prepare(w, "broth"), "already has fortifying broth queued")
	supplies[camping.WarmingItemID] = 1
	assert.Contains(t, prepare(w, "warming"), "already has fortifying broth queued")

	rest := w.m.startRest(w.user, w.room)
	assert.Contains(t, rest, "broth simmering for")
	assert.Zero(t, supplies[camping.BrothItemID], "spent once, when the rest begins")
	assert.Equal(t, []string{string(survival.LeaderMemberKey)}, w.camp().Rest.Broth)
	assert.Nil(t, w.camp().Prepared, "the queue is spent")
	assert.Equal(t, []string{string(survival.LeaderMemberKey)}, w.store.saved.Camps[7].Rest.Broth, "locked on the saved rest")
}

func TestQueuedBrothTheCompanyNoLongerCarriesIsDroppedWithANote(t *testing.T) {
	supplies := stock{camping.BrothItemID: 1}
	w, _, _ := prepWorld(t, supplies)
	require.Contains(t, prepare(w, "broth"), "set by")
	supplies[camping.BrothItemID] = 0 // sold or spilled before the rest
	rest := w.m.startRest(w.user, w.room)
	assert.Contains(t, rest, "There is no broth left for")
	assert.Empty(t, w.camp().Rest.Broth)
	assert.Nil(t, w.camp().Prepared)
}

func TestBrothTakesHoldWhenTheRestIsDoneOnceAndNotAfterARaid(t *testing.T) {
	e, user := heroEnv(t)
	supplies := stock{camping.BrothItemID: 1}
	supplies.install(e.module)
	e.module.inBattle = func(int) bool { return false }
	e.module.establish(user, eligibleRoom())
	e.module.lightFire(user, eligibleRoom())
	require.Contains(t, e.module.prepareCommand(user, eligibleRoom(), []string{"broth"}), "set by")
	require.Contains(t, e.module.startRest(user, eligibleRoom()), "settle in")
	assert.Empty(t, *e.buffs, "no benefit while the rest runs")

	messages := captureMessages(t)
	*e.now = e.now.Add(camping.RestDuration)
	e.scheduler.fireLatest()
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	events.ProcessEvents()
	assert.Contains(t, *e.buffs, buffCall{"Hero", camping.BrothBuffs[0].BuffID})
	assert.Contains(t, strings.Join(*messages, "\n"), "The broth takes hold")
	assert.Len(t, *e.buffs, 3, "Rested for two, and the broth for one")

	e.module.onNewRound(events.NewRound{RoundNumber: 2})
	assert.Len(t, *e.buffs, 3, "once a rest")

	// A rest spoiled by raiders gives no broth.
	spoiled, user2 := heroEnv(t)
	stock{camping.BrothItemID: 1}.install(spoiled.module)
	spoiled.module.inBattle = func(int) bool { return false }
	spoiled.module.establish(user2, eligibleRoom())
	spoiled.module.lightFire(user2, eligibleRoom())
	require.Contains(t, spoiled.module.prepareCommand(user2, eligibleRoom(), []string{"broth"}), "set by")
	require.Contains(t, spoiled.module.startRest(user2, eligibleRoom()), "settle in")
	camp := spoiled.module.camps[7]
	rest := *camp.Rest
	rest.Broken = true
	camp.Rest = &rest
	spoiled.module.camps[7] = camp
	spoiled.module.grantBroth(user2, nil)
	assert.Empty(t, *spoiled.buffs)
}

func TestBrothFortifiesACompanionWhoIsAtTheCamp(t *testing.T) {
	e, user := heroEnv(t)
	e.companion.RoomId = 100
	e.companion.Health = 10
	user.Character.RoomId = 100
	e.surv.needs = []survival.MemberNeeds{{Key: survival.LeaderMemberKey, Name: "Hero"}, {Key: survival.CompanionMemberKey(1), Name: "Bran"}}
	supplies := stock{camping.BrothItemID: 2}
	supplies.install(e.module)
	e.module.inBattle = func(int) bool { return false }
	e.module.establish(user, eligibleRoom())
	e.module.lightFire(user, eligibleRoom())
	text := e.module.prepareCommand(user, eligibleRoom(), []string{"broth", "bran"})
	assert.Contains(t, text, "Bran")
	assert.Contains(t, e.module.suppliesCommand(user), "fortifying broth x2")
	assert.Contains(t, e.module.prepareStatus(user, e.module.camps[7].Prepared), "Bran: fortifying broth queued")

	require.Contains(t, e.module.startRest(user, eligibleRoom()), "broth simmering for Bran")
	*e.now = e.now.Add(camping.RestDuration)
	e.scheduler.fireLatest()
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Contains(t, *e.buffs, buffCall{"Bran", camping.BrothBuffs[0].BuffID})
	assert.NotContains(t, *e.buffs, buffCall{"Hero", camping.BrothBuffs[0].BuffID}, "only the one who drank it")
	assert.Equal(t, 1, supplies[camping.BrothItemID])
}

func TestWatchIncenseNeedsAPostedWatchAndIsKeptWithoutOne(t *testing.T) {
	supplies := stock{camping.IncenseItemID: 1}
	w, _, _ := prepWorld(t, supplies)
	assert.Contains(t, prepare(w, "incense"), "needs a Camp Watch")
	assert.Nil(t, w.camp().Prepared)
	assert.Equal(t, 1, supplies[camping.IncenseItemID])

	watch := archetypes.Specialist{Name: "Bran", Level: 1}
	w.m.specialist = specialistsAt(t, 100, map[string]archetypes.Specialist{archetypes.UtilityWatch: watch})
	assert.Contains(t, prepare(w, "incense"), "watchsage")
	assert.True(t, w.camp().Prepared.Incense)
	assert.Contains(t, prepare(w, "incense"), "already set")

	// The watch leaves before the rest: the incense stays unlit and kept.
	w.m.specialist = func(int, string, ...int) (archetypes.Specialist, bool) { return archetypes.Specialist{}, false }
	text := w.m.startRest(w.user, w.room)
	assert.Contains(t, text, "No watch is posted, so the incense stays unlit")
	assert.False(t, w.camp().Rest.Incense)
	assert.Equal(t, 1, supplies[camping.IncenseItemID])
	assert.True(t, w.camp().Prepared.Incense, "kept for another rest")
}

func raidWithIncense(t *testing.T, spotRoll int, watch archetypes.Specialist) *raidWorld {
	t.Helper()
	w := newRaidWorld(t, 100, 0, 0, spotRoll)
	supplies := stock{camping.IncenseItemID: 1}
	supplies.install(w.m)
	w.m.inBattle = func(int) bool { return false }
	w.m.specialist = specialistsAt(t, 100, map[string]archetypes.Specialist{archetypes.UtilityWatch: watch})
	require.Contains(t, prepare(w, "incense"), "watchsage")
	w.rest(t)
	assert.True(t, w.camp().Rest.Incense, "locked on the rest")
	assert.Zero(t, supplies[camping.IncenseItemID], "spent when the rest began")
	assert.Nil(t, w.camp().Prepared)
	w.at(camping.RestDuration / 2)
	w.m.onNewRound(events.NewRound{})
	events.ProcessEvents()
	return w
}

func TestIncenseRaisesAWatchesChanceByTenPoints(t *testing.T) {
	level1 := archetypes.Specialist{Name: "Bran", Level: 1} // 25% a level
	assert.False(t, raidWithIncense(t, 34, level1).camp().Rest.Broken, "34 < 25+10")
	assert.True(t, raidWithIncense(t, 35, level1).camp().Rest.Broken, "35 is not under 35")
}

func TestIncenseNeverLowersACertainWatch(t *testing.T) {
	level4 := archetypes.Specialist{Name: "Bran", Level: 4} // 100%
	assert.False(t, raidWithIncense(t, 99, level4).camp().Rest.Broken, "a sure watch stays sure")
}

func TestIncenseChangesNothingWithoutIt(t *testing.T) {
	w := newRaidWorld(t, 100, 0, 0, 34)
	w.m.specialist = specialistsAt(t, 100, map[string]archetypes.Specialist{archetypes.UtilityWatch: {Name: "Bran", Level: 1}})
	w.rest(t)
	w.at(camping.RestDuration / 2)
	w.m.onNewRound(events.NewRound{})
	assert.True(t, w.camp().Rest.Broken, "a level 1 watch spots 25%: 34 misses")
}

func TestClearingSetsQueuedSuppliesAsideWithoutRefund(t *testing.T) {
	supplies := stock{camping.BrothItemID: 1, camping.IncenseItemID: 1}
	w, _, _ := prepWorld(t, supplies)
	w.m.specialist = specialistsAt(t, 100, map[string]archetypes.Specialist{archetypes.UtilityWatch: {Name: "Bran", Level: 1}})
	require.Contains(t, prepare(w, "broth"), "set by")
	require.Contains(t, prepare(w, "incense"), "watchsage")
	assert.Contains(t, prepare(w, "clear", "incense"), "watch incense is set aside")
	assert.False(t, w.camp().Prepared.Incense)
	assert.True(t, w.camp().Prepared.HasBroth(string(survival.LeaderMemberKey)))
	assert.Contains(t, prepare(w, "clear"), "queued broth is set aside")
	assert.Nil(t, w.camp().Prepared)
	assert.Equal(t, 1, supplies[camping.BrothItemID], "the dose was never spent")
	assert.Contains(t, prepare(w, "clear"), "Nothing to clear")
}

func TestUnknownSuppliesAndMembersAreRefusedWithUsage(t *testing.T) {
	w, _, _ := prepWorld(t, stock{camping.WarmingItemID: 1})
	assert.Contains(t, prepare(w, "poison"), "Prepare what?")
	assert.Contains(t, prepare(w, "warming", "nobody"), "nobody called")
	assert.Contains(t, prepare(w), "Usage: camp supplies")
	assert.Contains(t, prepare(w, "status"), "Prepared supplies:")
}

func TestSuppliesCommandListsDosesAndTheQueue(t *testing.T) {
	w, _, _ := prepWorld(t, stock{camping.BrothItemID: 2, camping.IncenseItemID: 1})
	text := w.m.suppliesCommand(w.user)
	assert.Contains(t, text, "fortifying broth x2 (camp prepare broth)")
	assert.Contains(t, text, "watch incense x1")
	assert.NotContains(t, text, "warming draught")
	none, _, _ := prepWorld(t, stock{})
	assert.Contains(t, none.m.suppliesCommand(none.user), "None.")
}

func TestCampTabListsSuppliesAndTheQueue(t *testing.T) {
	supplies := stock{camping.BrothItemID: 1, camping.WarmingItemID: 3}
	w, _, _ := prepWorld(t, supplies)
	state, ok := w.m.CampStateOf(7, 100, []string{"camping"})
	require.True(t, ok)
	assert.Equal(t, []string{"fortifying broth x1", "warming draught x3"}, state.Supplies)
	assert.Empty(t, state.Prepared)
	prepare(w, "broth")
	state, _ = w.m.CampStateOf(7, 100, []string{"camping"})
	require.Len(t, state.Prepared, 1)
	assert.Contains(t, state.Prepared[0], "fortifying broth for")
}

// The real `camp` command routes the new subcommands, and the usage names
// them.
func TestCampCommandRoutesSuppliesAndPrepare(t *testing.T) {
	supplies := stock{camping.WarmingItemID: 1}
	w, _, calls := prepWorld(t, supplies)
	messages := captureMessages(t)
	for _, rest := range []string{"supplies", "prepare warming", "prepare status", "bogus"} {
		handled, err := w.m.userCommand(rest, w.user, w.room, 0)
		require.NoError(t, err)
		assert.True(t, handled)
	}
	events.ProcessEvents()
	out := strings.Join(*messages, "\n")
	assert.Contains(t, out, "warming draught x1")
	assert.Contains(t, out, "You drink the warming draught")
	assert.Contains(t, out, "Prepared supplies:")
	assert.Contains(t, out, "camp supplies | camp prepare")
	assert.Len(t, *calls, 1)
}
