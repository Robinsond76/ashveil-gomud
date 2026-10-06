package camping

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/flasks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cappedFake records a duty-aware recovery (Phase 51).
type cappedFake struct {
	fakeSurvival
	ceilings []map[survival.MemberKey]int
	capCalls int
}

func (f *cappedFake) ApplyCompanyRestRecoveryCapped(_ int, _ string, _ int, _ map[survival.MemberKey]int, ceilings map[survival.MemberKey]int) ([]survival.ExertionResult, error) {
	f.capCalls++
	f.ceilings = append(f.ceilings, ceilings)
	return nil, nil
}

// dutyWorld is a lit camp with the leader (user 7) and two live companions,
// Mira (1) and Bran (2), at the camp.
type dutyWorld struct {
	*raidWorld
	mira, bran *characters.Character
	surv       *cappedFake
	buffs      map[*characters.Character][]int
}

func newDutyWorld(t *testing.T, raidChance int, rolls ...int) *dutyWorld {
	t.Helper()
	w := newRaidWorld(t, raidChance, rolls...)
	d := &dutyWorld{raidWorld: w, buffs: map[*characters.Character][]int{}}
	d.mira, d.bran = characters.New(), characters.New()
	d.mira.Name, d.bran.Name = "Mira", "Bran"
	for _, c := range []*characters.Character{d.mira, d.bran} {
		c.RoomId, c.Health = 100, 10
	}
	w.m.companionsOf = func(int) (map[int]*characters.Character, []int) {
		return map[int]*characters.Character{1: d.mira, 2: d.bran}, []int{1, 2}
	}
	w.m.companySize = func(int) int { return 3 }
	d.surv = &cappedFake{}
	d.surv.needs = []survival.MemberNeeds{
		{Key: survival.LeaderMemberKey, Name: w.user.Character.Name},
		{Key: survival.CompanionMemberKey(1), Name: "Mira"},
		{Key: survival.CompanionMemberKey(2), Name: "Bran"},
	}
	w.m.survival = d.surv
	w.m.inBattle = func(int) bool { return false }
	w.m.memberLevel = func(int, int, string) int { return 0 }
	w.m.grantBuff = func(c *characters.Character, id, _ int) error { d.buffs[c] = append(d.buffs[c], id); return nil }
	w.m.hasBuff = func(*characters.Character, int) bool { return false }
	return d
}

func (d *dutyWorld) duty(t *testing.T, words ...string) string {
	t.Helper()
	return d.m.dutiesCommand(d.user, d.room, words)
}

func (d *dutyWorld) finishAndGrant() {
	finishRest(d.raidWorld)
	d.m.onNewRound(events.NewRound{})
	events.ProcessEvents()
}

func TestDutiesDefaultToSleepAndLeaveStatusAlone(t *testing.T) {
	d := newDutyWorld(t, 0)
	view := d.duty(t)
	assert.Contains(t, view, "Mira: sleep (default)")
	assert.Contains(t, view, "Bran: sleep (default)")
	assert.NotContains(t, d.m.status(7), "Duty", "a camp with no duties reads as before")

	d.rest(t)
	assert.Empty(t, d.camp().Rest.Duties)
	d.finishAndGrant()
	assert.Zero(t, d.surv.capCalls, "no watcher: the plain recovery")
	assert.NotEmpty(t, d.buffs[d.mira], "a sleeper gets the Rested buff, as before")
	assert.NotEmpty(t, d.buffs[d.user.Character])
}

func TestSetDutiesPersistsLocksOnRestAndSurvivesARestart(t *testing.T) {
	d := newDutyWorld(t, 0)
	assert.Contains(t, d.duty(t, "mira", "watch"), "Mira keeps watch")
	assert.Contains(t, d.duty(t, "me", "forage"), "forages")
	assert.Equal(t, "watch", d.camp().Duties[string(survival.CompanionMemberKey(1))])
	require.Equal(t, "watch", d.store.saved.Camps[7].Duties[string(survival.CompanionMemberKey(1))], "durable")

	d.rest(t)
	locked := d.camp().Rest.Duties
	assert.Equal(t, map[string]string{string(survival.CompanionMemberKey(1)): "watch", "leader": "forage"}, locked)
	assert.Contains(t, d.m.status(7), "Mira: Hunger")
	assert.Contains(t, d.m.status(7), "Duty watch")

	// Duties are fixed for the running rest.
	assert.Contains(t, d.duty(t, "mira", "sleep"), "fixed when the rest began")
	assert.Equal(t, locked, d.camp().Rest.Duties)

	reloaded := newTestModule(d.store, &fakeScheduler{}, &fakeSurvival{}, func() time.Time { return *d.now })
	reloaded.load()
	assert.Equal(t, locked, reloaded.camps[7].Rest.Duties, "a restart never re-rolls them")
	assert.Equal(t, "watch", reloaded.camps[7].Duties[string(survival.CompanionMemberKey(1))])
}

func TestDutyForAMemberAwayFromTheCampIsNotLocked(t *testing.T) {
	d := newDutyWorld(t, 0)
	d.duty(t, "mira", "watch")
	d.duty(t, "bran", "tend")
	d.bran.RoomId = 999
	d.rest(t)
	assert.Equal(t, map[string]string{string(survival.CompanionMemberKey(1)): "watch"}, d.camp().Rest.Duties, "Bran was away, so he slept")
}

func TestDutiesCommandRefusals(t *testing.T) {
	d := newDutyWorld(t, 0)
	assert.Contains(t, d.duty(t, "nobody", "watch"), "nobody called")
	assert.Contains(t, d.duty(t, "mira", "dance"), "no duty called")
	assert.Contains(t, d.duty(t, "mira"), "Usage")
	assert.Contains(t, d.duty(t, "mira", "brew"), "no flask satchel", "only an Alchemist can brew")
	assert.Empty(t, d.camp().Duties)

	d.mira.HPArchetype = "alchemist"
	assert.Contains(t, d.duty(t, "mira", "brew"), "Mira brews")
	assert.Contains(t, d.duty(t, "all", "tend"), "Bran tends")
	assert.Equal(t, "tend", d.camp().Duties[string(survival.CompanionMemberKey(2))])
	// 51 review: a bare duty word is the leader's own.
	d.duty(t, "watch")
	assert.Equal(t, "watch", d.camp().Duties[string(survival.LeaderMemberKey)])
	assert.Contains(t, d.duty(t, "clear"), "sleep through")
	assert.Empty(t, d.camp().Duties)

	elsewhere := &rooms.Room{RoomId: 101}
	assert.Contains(t, d.m.dutiesCommand(d.user, elsewhere, []string{"mira", "watch"}), "not here")
	delete(d.m.camps, 7)
	assert.Contains(t, d.m.dutiesCommand(d.user, d.room, nil), "no camp")
}

func TestTwoWatchersSpotRaidersMoreOftenThanOne(t *testing.T) {
	assert.Equal(t, 20, combineChances(20))
	assert.Equal(t, 36, combineChances(20, 20))
	assert.Equal(t, 100, combineChances(100, 50))

	// A roll of 25 beats one base watcher (20%) but not two (36%).
	spotted := func(watchers ...string) bool {
		d := newDutyWorld(t, 100, 0, 0, 25)
		for _, w := range watchers {
			d.duty(t, w, "watch")
		}
		d.rest(t)
		d.at(camping.RestDuration / 2)
		d.m.onNewRound(events.NewRound{})
		return d.camp().Rest.Raid.Spotted
	}
	assert.False(t, spotted("mira"))
	assert.True(t, spotted("mira", "bran"))
}

func TestWatcherEndsCappedAtReadyAndDutyMembersMissRested(t *testing.T) {
	d := newDutyWorld(t, 0)
	d.duty(t, "mira", "watch")
	d.duty(t, "bran", "tend")
	d.rest(t)
	messages := captureMessages(t)
	d.finishAndGrant()

	require.Equal(t, 1, d.surv.capCalls)
	assert.Equal(t, map[survival.MemberKey]int{survival.CompanionMemberKey(1): camping.WatchFatigueCap}, d.surv.ceilings[0], "only the watcher is capped, at the top of Ready")
	assert.Equal(t, survival.BandSteady, survival.BandFor(camping.WatchFatigueCap))
	assert.Equal(t, survival.BandFull, survival.BandFor(camping.WatchFatigueCap+1))
	assert.Empty(t, d.buffs[d.mira], "a watcher is not Rested")
	assert.Empty(t, d.buffs[d.bran], "nor is a tender")
	assert.NotEmpty(t, d.buffs[d.user.Character], "the sleeper is")
	text := strings.Join(*messages, "\n")
	assert.Contains(t, text, "Rest duties:")
	assert.Contains(t, text, "Mira kept watch through the night.")
	assert.Contains(t, text, "Bran finds nothing to tend")
}

func TestSpoiledRestSettlesNoDuties(t *testing.T) {
	d := newDutyWorld(t, 100, 0, 0, 99)
	d.duty(t, "bran", "tend")
	d.rest(t)
	messages := captureMessages(t)
	d.at(camping.RestDuration / 2)
	d.m.onNewRound(events.NewRound{})
	require.True(t, d.camp().Rest.Broken, "nobody spotted the raiders")
	d.finishAndGrant()
	assert.NotContains(t, strings.Join(*messages, "\n"), "Rest duties")
	assert.Empty(t, d.buffs[d.user.Character], "no Rested at all after a spoiled rest")
}

func TestTendersTreatWoundsWithTheKitOnePerTender(t *testing.T) {
	d := newDutyWorld(t, 0)
	stock{surgeonKitItemID: 3}.install(d.m)
	calls := 0
	d.m.surgery = func(int) ([]string, bool) {
		calls++
		return []string{"A wound is treated."}, calls <= 3
	}
	d.duty(t, "mira", "tend")
	d.duty(t, "bran", "tend")
	d.rest(t)
	require.True(t, d.camp().Rest.Kit)
	messages := captureMessages(t)
	d.finishAndGrant()
	assert.Equal(t, 3, calls, "the kit's own pass, then one treatment for each tender")
	assert.Equal(t, 3, strings.Count(strings.Join(*messages, "\n"), "A wound is treated."))
}

func TestTendersSharpenOneMembersBladesWhenThereIsNoWound(t *testing.T) {
	d := newDutyWorld(t, 0)
	sharpenSpecs(t)
	armed(d.mira, testSwordID, 0)
	armed(d.bran, testDaggerID, 0)
	d.user.Character.Items = append(d.user.Character.Items, stone(10))
	d.duty(t, "bran", "tend")
	d.duty(t, "me", "tend")
	d.rest(t)
	messages := captureMessages(t)
	d.finishAndGrant()
	text := strings.Join(*messages, "\n")
	assert.Contains(t, text, "puts a fresh edge on Mira's blades")
	assert.Contains(t, text, "puts a fresh edge on Bran's blades")
	assert.True(t, d.mira.Equipment.Weapon.Sharpened())
	assert.True(t, d.bran.Equipment.Weapon.Sharpened())
	assert.Equal(t, []int{8}, stoneUses(d.user.Character), "one whetstone use for each tender")
}

func TestCookDutyCooksAtRestEndWithTheMembersOwnRanks(t *testing.T) {
	d := newDutyWorld(t, 0)
	forageSpecs(t)
	d.m.campCfg.Recipes = []campRecipe{
		{Output: 30020, Inputs: []int{29, 30018}, Skill: "cooking", MinLevel: 2},
		{Output: 30021, Inputs: []int{29}, Skill: "cooking", MinLevel: 1},
	}
	cargo := newFakeCargo(100000)
	useCargo(t, cargo)
	cargo.stacks[29] = 2
	skills.SetTestData([]*skills.Skill{{SkillId: "cooking", Name: "Cooking", MaxLevel: 4}}, nil)
	t.Cleanup(func() { skills.SetTestData(nil, nil) })
	d.mira.Skills = map[string]int{"cooking": 1}
	d.duty(t, "mira", "cook")
	d.duty(t, "bran", "cook") // Bran cannot cook yet
	d.rest(t)
	messages := captureMessages(t)
	d.finishAndGrant()
	text := strings.Join(*messages, "\n")
	assert.Contains(t, text, "Mira cooks")
	assert.Contains(t, text, "seared game meat")
	assert.Contains(t, text, "it needs cooking 1", "Bran has no Cooking: the refusal says so")
	assert.Equal(t, 1, cargo.stacks[29], "one cook's dish used one meat")
	assert.Equal(t, 1, cargo.stacks[30021])
}

func TestForageDutyFindsFoodUnderTheForageCooldown(t *testing.T) {
	d := newDutyWorld(t, 0)
	forageSpecs(t)
	d.m.campCfg.Forage["Road"] = []forageFind{{ItemID: 29, Weight: 1}}
	cargo := newFakeCargo(100000)
	useCargo(t, cargo)
	d.m.memberLevel = func(_ int, companionID int, _ string) int { return map[int]int{1: 4}[companionID] }
	d.duty(t, "mira", "forage")
	d.duty(t, "bran", "forage")
	d.rest(t)
	messages := captureMessages(t)
	d.finishAndGrant()
	d.m.onNewRound(events.NewRound{})
	events.ProcessEvents()
	text := strings.Join(*messages, "\n")
	assert.Contains(t, text, "Mira foraged")
	assert.Contains(t, text, "Bran foraged")
	assert.Equal(t, 3+1, cargo.stacks[29], "Mira at level 4 finds 1+4/2, Bran 1")
	assert.Empty(t, d.m.campRewards, "paid")

	// A second rest inside the cooldown finds nothing, and says why.
	d.at(camping.RestDuration + 2*time.Minute)
	d.m.lightFire(d.user, d.room)
	messages = captureMessages(t)
	d.rest(t)
	d.at(2*camping.RestDuration + 3*time.Minute)
	d.sched.fireLatest()
	d.m.onNewRound(events.NewRound{})
	d.m.onNewRound(events.NewRound{})
	events.ProcessEvents()
	assert.Equal(t, 4, cargo.stacks[29], "no second forage inside the cooldown")
	assert.Contains(t, strings.Join(*messages, "\n"), "still picked over")
}

func TestBrewDutyServesTheBrewersReagentsFirst(t *testing.T) {
	leader := &users.UserRecord{UserId: 7, Character: &characters.Character{Name: "Aria", Level: 5}}
	first := &characters.Character{Name: "Ione", Level: 5, HPArchetype: "alchemist"}
	second := &characters.Character{Name: "Mira", Level: 5, HPArchetype: "alchemist"}
	first.FlasksSpent, second.FlasksSpent = 2, 2
	live := map[int]*characters.Character{1: first, 2: second}
	for i := 0; i < 2; i++ {
		leader.Character.Items = append(leader.Character.Items, items.Item{ItemId: flasks.ReagentItemID})
	}
	lines := brewRestFlasks(leader, live, map[*characters.Character]bool{second: true})
	assert.Equal(t, []string{"Mira brews through the night: 2 flask(s) from 2 reagent(s)."}, lines)
	assert.Equal(t, 0, second.FlasksSpent)
	assert.Equal(t, 2, first.FlasksSpent, "reagents were short, so the brewer was served first")
}

func TestDutyHookAndCampTabRows(t *testing.T) {
	d := newDutyWorld(t, 0)
	d.mira.HPArchetype = "alchemist"
	var hooked map[string]string
	d.m.onDuties = func(_ int, duties map[string]string) { hooked = duties }
	d.rest(t)
	assert.Nil(t, hooked, "no duties, no hook")
	finishRest(d.raidWorld)
	d.m.lightFire(d.user, d.room)
	d.duty(t, "mira", "brew")
	d.rest(t)
	assert.Equal(t, map[string]string{string(survival.CompanionMemberKey(1)): "brew"}, hooked)

	state, ok := d.m.CampStateOf(7, 100, []string{"camping"})
	require.True(t, ok)
	require.True(t, state.DutiesLocked, "a running rest's duties are fixed")
	require.Len(t, state.Duties, 3)
	assert.Equal(t, "me", state.Duties[0].Command)
	assert.Equal(t, "brew", state.Duties[1].Duty)
	assert.Contains(t, state.Duties[1].Options, "brew", "an Alchemist can brew")
	assert.NotContains(t, state.Duties[2].Options, "brew", "Bran cannot")
	assert.Equal(t, "sleep", state.Duties[2].Duty)
}

func TestWatchDutyUsesTheMembersOwnWatchLevel(t *testing.T) {
	d := newDutyWorld(t, 0)
	d.m.memberLevel = func(_ int, companionID int, utility string) int {
		require.Equal(t, archetypes.UtilityWatch, utility)
		return map[int]int{1: 3}[companionID]
	}
	assert.Equal(t, []int{75, 20}, d.m.watchChances(7, []string{string(survival.CompanionMemberKey(1)), string(survival.CompanionMemberKey(2))}), "level 3 at 25 a level; a base 20 for the rest")
}

func TestCampCommandRoutesDutiesAndParsesTheWatcherBase(t *testing.T) {
	d := newDutyWorld(t, 0)
	messages := captureMessages(t)
	_, err := d.m.userCommand("duties mira watch", d.user, d.room, 0)
	require.NoError(t, err)
	_, err = d.m.userCommand("duties", d.user, d.room, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	text := strings.Join(*messages, "\n")
	assert.Contains(t, text, "Mira keeps watch")
	assert.Contains(t, text, "Mira: watch")
	assert.Contains(t, campUsage, "camp duties")

	assert.Equal(t, 20, parseCampSettings(nil).WatcherBasePct)
	assert.Equal(t, 35, parseCampSettings(func(k string) any { return map[string]any{"WatcherBasePct": 35}[k] }).WatcherBasePct)
	assert.Equal(t, 20, parseCampSettings(func(k string) any { return map[string]any{"WatcherBasePct": 300}[k] }).WatcherBasePct, "out of range keeps the default")
}

// 51 review: the duties go with the pending Rested grant, so breaking camp
// between the rest's end and the grant neither hands the members on duty
// the Rested buff nor drops their work, and the owed duties are saved.
func TestBreakingCampBeforeTheGrantKeepsTheRestsDuties(t *testing.T) {
	d := newDutyWorld(t, 0)
	d.duty(t, "mira", "watch")
	d.duty(t, "bran", "tend")
	d.rest(t)
	finishRest(d.raidWorld)
	assert.Equal(t, map[string]string{"companion:1": "watch", "companion:2": "tend"}, d.store.saved.RestedDuties[7], "saved with the pending grant")
	messages := captureMessages(t)
	d.m.breakCamp(d.user, d.room)
	require.NotContains(t, d.m.camps, 7)
	d.m.onNewRound(events.NewRound{})
	events.ProcessEvents()

	assert.Empty(t, d.buffs[d.mira], "a watcher is not Rested, camp or no camp")
	assert.Empty(t, d.buffs[d.bran])
	assert.NotEmpty(t, d.buffs[d.user.Character])
	text := strings.Join(*messages, "\n")
	assert.Contains(t, text, "Mira kept watch through the night.")
	assert.Contains(t, text, "Bran finds nothing to tend")
	assert.Empty(t, d.m.restedDuties, "settled once, then cleared")
}

// 51 review: the camp's watch specialist on the watch duty counts once.
func TestWatchSpecialistOnWatchDutyCountsOnce(t *testing.T) {
	spotted := func(watcher string) bool {
		d := newDutyWorld(t, 100, 0, 0, 30)
		d.m.specialist = specialistsAt(t, 100, map[string]archetypes.Specialist{archetypes.UtilityWatch: {Name: "Mira", Level: 1}})
		d.m.memberLevel = func(_ int, companionID int, _ string) int { return map[int]int{1: 1}[companionID] }
		d.duty(t, watcher, "watch")
		d.rest(t)
		d.at(camping.RestDuration / 2)
		d.m.onNewRound(events.NewRound{})
		return d.camp().Rest.Raid.Spotted
	}
	assert.False(t, spotted("mira"), "Mira's 25% once, not 25% twice (44%): a roll of 30 gets past")
	assert.True(t, spotted("bran"), "Mira 25% and Bran's 20%: 40%")
}

// 51 review: the company's forager on the forage duty forages once.
func TestForageSpecialistOnForageDutyForagesOnce(t *testing.T) {
	d := newDutyWorld(t, 0)
	forageSpecs(t)
	d.m.campCfg.Forage["Road"] = []forageFind{{ItemID: 29, Weight: 1}}
	cargo := newFakeCargo(100000)
	useCargo(t, cargo)
	d.m.specialist = specialistsAt(t, 100, map[string]archetypes.Specialist{archetypes.UtilityForage: {Name: "Mira", Level: 4}})
	d.m.memberLevel = func(_ int, companionID int, _ string) int { return map[int]int{1: 4}[companionID] }
	d.duty(t, "mira", "forage")
	d.rest(t)
	d.finishAndGrant()
	d.m.onNewRound(events.NewRound{})
	events.ProcessEvents()
	assert.Equal(t, 3, cargo.stacks[29], "Mira's own forage (1+4/2), not a second one for her duty")
}

// The company fighting as the rest ends leaves the work duties undone.
func TestDutiesLeftUndoneWhenTheCompanyIsFighting(t *testing.T) {
	d := newDutyWorld(t, 0)
	d.duty(t, "bran", "tend")
	d.duty(t, "mira", "watch")
	d.rest(t)
	messages := captureMessages(t)
	d.m.inBattle = func(int) bool { return true }
	d.finishAndGrant()
	text := strings.Join(*messages, "\n")
	assert.Contains(t, text, "Mira kept watch through the night.")
	assert.Contains(t, text, "left undone")
	assert.NotContains(t, text, "Bran finds nothing to tend")
}
