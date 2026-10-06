package company

import (
	"errors"
	"slices"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 38b wiring: "class" and "talent" through the real commands, the
// company store, spawn and restart in the brawl world. The leader's class
// state is held by a fake of the archetype registry (modules/archetype's own
// tests cover its persistence).

// fakeClassStore stands in for the archetype registry's class half.
type fakeClassStore struct {
	state   classes.State
	failing bool
	commits int
}

func (f *fakeClassStore) PlayerClass(int) classes.State {
	return classes.State{Class: f.state.Class, Talents: slices.Clone(f.state.Talents)}
}

func (f *fakeClassStore) PromotePlayer(_ int, class string) error {
	if f.failing {
		return errors.New("disk full")
	}
	if f.state.Class == class {
		return nil
	}
	f.state.Class = class
	f.commits++
	return nil
}

func (f *fakeClassStore) PickPlayerTalent(_, _ int, talent string) error {
	if f.failing {
		return errors.New("disk full")
	}
	f.state.Talents = append(f.state.Talents, talent)
	f.commits++
	return nil
}

// classBrawl is the training brawl with Aria a cleric of the given level
// and alignment, and Tamsin (warrior) and Oswin (cleric) companions.
func classBrawl(t *testing.T, level, alignment int) (*trainWorld, *fakeClassStore) {
	t.Helper()
	w := trainingBrawl(t)
	store := &fakeClassStore{}
	classes.SetProvider(store)
	t.Cleanup(func() { classes.SetProvider(nil) })
	w.withArchetypes("cleric")
	w.aria.Character.Level = level
	w.aria.Character.Alignment = int8(alignment)
	return w, store
}

// classFailStore loads as the real store does and fails every save.
type classFailStore struct{ Store }

func (classFailStore) Save(domain.Registry) error { return errors.New("disk full") }

func setCompanionAlignment(t *testing.T, id, alignment int) {
	t.Helper()
	require.NoError(t, module.registry.SetDisposition(7, id, domain.Disposition{Alignment: alignment, Loyalty: 70}))
}

func companionClass(t *testing.T, id int) (string, []string) {
	t.Helper()
	class, talents, ok := module.CompanionClassState(7, id)
	require.True(t, ok)
	return class, talents
}

func TestClassShowsNoPromotionBeforeAndAReadyOneAtLevelTen(t *testing.T) {
	w, _ := classBrawl(t, 9, 50)
	assert.Contains(t, w.cmd("class", ""), "level 9 Cleric with no promotion yet")
	assert.Contains(t, w.cmd("class", ""), "Next: your class promotion at level 10.")
	assert.NotContains(t, w.cmd("class", ""), "Ready to promote")

	w.aria.Character.Level = 10
	view := w.cmd("class", "")
	assert.Contains(t, view, "Ready to promote: Priest. Type class promote priest.")
	assert.Contains(t, view, "Ready to promote: Druid.")
	assert.NotContains(t, view, "Blood Priest", "the evil route isn't open at +50")
}

func TestClassPathsListsTheLineagesRoutesAndTheirGates(t *testing.T) {
	w, _ := classBrawl(t, 10, 0)
	paths := w.cmd("class", "paths")
	for _, want := range []string{"Priest", "Druid", "Blood Priest", "alignment +30 or higher", "alignment -30 or lower", "any alignment", "Hierarch", "Demonologist", "Elder Druid", "Routes are final"} {
		assert.Contains(t, paths, want)
	}
	assert.Contains(t, w.cmd("class", "paths oswin"), "Routes for a Cleric")
	assert.Contains(t, w.cmd("class", "paths tamsin"), "Knight")
	assert.Contains(t, w.cmd("class", "paths tamsin"), "then Warlord: pressure on the chosen target")
	assert.NotContains(t, w.cmd("class", "paths tamsin"), "planned")
}

func TestPlayerPromotionPreviewConfirmAndRepeat(t *testing.T) {
	w, store := classBrawl(t, 10, 45)
	preview := w.cmd("class", "promote priest")
	assert.Contains(t, preview, "Cleric -> Priest (advanced, cleric lineage)")
	assert.Contains(t, preview, "Gate: alignment +30 or higher (yours: +45)  ready")
	assert.Contains(t, preview, "Now:  rank 10 Ward")
	assert.Contains(t, preview, "Next: rank 15 (level 15): Greater Heal")
	assert.Contains(t, preview, "Routes are final")
	assert.Contains(t, preview, "class promote self priest confirm")
	assert.Empty(t, store.state.Class, "a preview changes nothing")

	assert.Contains(t, w.cmd("class", "promote priest confirm"), "You are now a Priest.")
	assert.Equal(t, "priest", store.state.Class)
	assert.Equal(t, 1, store.commits)
	assert.Contains(t, w.cmd("class", ""), "a Priest (advanced class)")

	assert.Contains(t, w.cmd("class", "promote priest confirm"), "already a Priest")
	assert.Equal(t, 1, store.commits, "a repeated confirmation changes nothing")

	// Routes are final: no sibling, no other lineage.
	assert.Contains(t, w.cmd("class", "promote druid confirm"), "Routes are final")
	assert.Equal(t, "priest", store.state.Class)
	assert.Contains(t, w.cmd("class", "promote knight confirm"), "can't become a Knight")
}

func TestPromotionNeedsLevelAndTheAlignmentGate(t *testing.T) {
	w, store := classBrawl(t, 9, 100)
	assert.Contains(t, w.cmd("class", "promote priest confirm"), "needs level 10")
	w.aria.Character.Level = 10
	// The good gate's boundary values.
	w.aria.Character.Alignment = 29
	assert.Contains(t, w.cmd("class", "promote priest confirm"), "alignment +30 or higher; yours is +29")
	assert.Empty(t, store.state.Class)
	w.aria.Character.Alignment = 30
	assert.Contains(t, w.cmd("class", "promote priest confirm"), "You are now a Priest.")
}

func TestPromotionDriftBetweenPreviewAndConfirmIsRechecked(t *testing.T) {
	w, store := classBrawl(t, 10, 40)
	assert.Contains(t, w.cmd("class", "promote priest"), "Cleric -> Priest (advanced, cleric lineage)")
	w.aria.Character.Alignment = 10 // drifted after the preview
	assert.Contains(t, w.cmd("class", "promote priest confirm"), "needs alignment +30 or higher")
	assert.Empty(t, store.state.Class)
}

func TestPromotionRefusedInBattleTravelAndRest(t *testing.T) {
	w, store := classBrawl(t, 10, 40)
	w.journey.travelling = true
	assert.Contains(t, w.cmd("class", "promote priest confirm"), "travelling")
	w.journey.travelling = false
	w.camp.resting = true
	assert.Contains(t, w.cmd("class", "promote priest confirm"), "resting")
	w.camp.resting = false
	w.cmd("attack", "#"+itoa(w.bandits["bandit captain"][0]))
	assert.Contains(t, w.cmd("class", "promote priest confirm"), "battle")
	assert.Empty(t, store.state.Class)
}

func TestPlayerPromotionSaveFailureChangesNothing(t *testing.T) {
	w, store := classBrawl(t, 10, 40)
	store.failing = true
	assert.Contains(t, w.cmd("class", "promote priest confirm"), "couldn't be saved; nothing changed")
	assert.Empty(t, store.state.Class)
	store.failing = false
	assert.Contains(t, w.cmd("class", "promote priest confirm"), "You are now a Priest.")
}

func TestCompanionPromotionThroughTheCommandSavesAndSurvivesARestart(t *testing.T) {
	w, _ := classBrawl(t, 3, 0)
	oswin := w.companion(2)
	oswin.Character.Level = 10
	setCompanionAlignment(t, 2, 40)
	w.respawn() // the logout snapshots level 10; the login spawns from it
	require.Equal(t, 10, w.companion(2).Character.Level)

	view := w.cmd("class", "oswin")
	assert.Contains(t, view, "Brother Oswin is a level 10 Cleric with no promotion yet. Alignment +40.")
	assert.Contains(t, view, "Ready to promote: Priest. Type class promote #2 priest.")

	preview := w.cmd("class", "promote oswin priest")
	assert.Contains(t, preview, "Brother Oswin: Cleric -> Priest (advanced, cleric lineage)")
	assert.Contains(t, preview, "Gate: alignment +30 or higher (theirs: +40)  ready")
	assert.Contains(t, preview, "Type: class promote #2 priest confirm")
	class, _ := companionClass(t, 2)
	assert.Empty(t, class)

	assert.Contains(t, w.cmd("class", "promote #2 priest confirm"), "Brother Oswin is now a Priest.")
	class, _ = companionClass(t, 2)
	assert.Equal(t, "priest", class)
	live, _ := w.companion(2).Character.ClassState()
	assert.Equal(t, "priest", live, "the live companion has it at once")
	assert.Contains(t, w.cmd("class", "promote oswin priest confirm"), "already a Priest")
	assert.Contains(t, w.cmd("class", "promote oswin druid confirm"), "Routes are final")

	// Saved at once, and a restart brings it back.
	stored := domain.NewRegistry()
	require.NoError(t, module.store.Load(stored))
	rec, _ := stored.Get(7)
	for _, c := range rec.Companions {
		if c.ID == 2 {
			assert.Equal(t, "priest", c.Class)
		}
	}
	w.respawn()
	live, _ = w.companion(2).Character.ClassState()
	assert.Equal(t, "priest", live)
	assert.Contains(t, w.cmd("class", "oswin"), "a Priest (advanced class)")
}

func TestCompanionPromotionRefusals(t *testing.T) {
	w, _ := classBrawl(t, 3, 0)
	// Tamsin is a level-6 warrior: too low.
	assert.Contains(t, w.cmd("class", "promote tamsin knight confirm"), "needs level 10")
	// A cleric route for a warrior is not open.
	w.companion(1).Character.Level = 12
	w.respawn()
	setCompanionAlignment(t, 1, 80)
	assert.Contains(t, w.cmd("class", "promote tamsin priest confirm"), "can't become a Priest")
	// A companion's own alignment gates it, not the leader's or the company's.
	w.aria.Character.Alignment = 100
	setCompanionAlignment(t, 1, 0)
	assert.Contains(t, w.cmd("class", "promote tamsin knight confirm"), "alignment +30 or higher; yours is +0")
	assert.Contains(t, w.cmd("class", "promote tamsin mercenary confirm"), "Tamsin Reed is now a Mercenary.")
	assert.Contains(t, w.cmd("class", "promote nobody priest confirm"), "no companion like that")
	assert.Contains(t, w.cmd("class", "promote tamsin gibberish confirm"), `no class called`)
}

func TestCompanionPromotionSaveFailureRestoresTheRecord(t *testing.T) {
	w, _ := classBrawl(t, 3, 0)
	w.companion(2).Character.Level = 10
	setCompanionAlignment(t, 2, 40)
	w.respawn()
	real := module.store
	module.store = classFailStore{real}
	t.Cleanup(func() { module.store = real })
	assert.Contains(t, w.cmd("class", "promote oswin priest confirm"), "couldn't be saved; nothing changed")
	module.store = real
	class, _ := companionClass(t, 2)
	assert.Empty(t, class)
	live, _ := w.companion(2).Character.ClassState()
	assert.Empty(t, live)
}

func TestFallenCompanionCannotPromote(t *testing.T) {
	w, _ := classBrawl(t, 3, 0)
	w.companion(2).Character.Level = 10
	setCompanionAlignment(t, 2, 40)
	w.respawn()
	require.NoError(t, module.registry.MarkDead(7, 2, domain.CompanionDeath{OpID: "test", Remaining: 100, Allowance: 100}))
	assert.Contains(t, w.cmd("class", "promote oswin priest confirm"), "fallen")
}

func TestEliteWaitsForTheGateAndPromotesWhenItRecovers(t *testing.T) {
	w, store := classBrawl(t, 30, 29)
	store.state.Class = "priest"
	view := w.cmd("class", "")
	assert.Contains(t, view, "Waiting: Hierarch needs alignment +30 or higher; yours is +29.")
	assert.Contains(t, view, "keeps your ranks")
	assert.Contains(t, w.cmd("class", "promote hierarch confirm"), "needs alignment +30 or higher")
	assert.Equal(t, "priest", store.state.Class)

	w.aria.Character.Alignment = 30
	assert.Contains(t, w.cmd("class", "promote hierarch confirm"), "You are now a Hierarch.")
	assert.Equal(t, "hierarch", store.state.Class)
	assert.Contains(t, w.cmd("class", "promote druid confirm"), "final")
	assert.Contains(t, w.cmd("class", ""), "a Hierarch (elite class)")
}

// The rogue, ranger, wizard and witch elites arrive with 38c2 and 38c3.
func TestPlannedEliteIsNotOpenYet(t *testing.T) {
	w, store := classBrawl(t, 30, 100)
	w.withArchetypes("rogue")
	store.state.Class = "scout"
	assert.Contains(t, w.cmd("class", "promote pathfinder confirm"), "not open yet")
	assert.NotContains(t, w.cmd("class", ""), "Ready to promote: Pathfinder")
}

func TestUnpromotedHighLevelCharacterKeepsItsBase(t *testing.T) {
	w, _ := classBrawl(t, 40, 0)
	view := w.cmd("class", "")
	assert.Contains(t, view, "level 40 Cleric with no promotion yet")
	assert.Contains(t, view, "Ready to promote: Druid.")
	// Elite needs an advanced class first, and says so.
	refused := w.cmd("class", "promote hierarch confirm")
	assert.Contains(t, refused, "can't become a Hierarch")
	assert.Contains(t, refused, "take the Priest first, then the Hierarch in the same visit. Type class promote priest.")
}

func TestTalentsThroughTheCommand(t *testing.T) {
	w, store := classBrawl(t, 4, 0)
	assert.Contains(t, w.cmd("talent", ""), "0 of 6 talents earned so far, 0 to choose")
	assert.Contains(t, w.cmd("talent", "pick deep-well confirm"), "no talent to choose now. The next comes at level 5.")

	w.aria.Character.Level = 5
	view := w.cmd("talent", "")
	assert.Contains(t, view, "1 to choose")
	assert.Contains(t, view, "Deep Well (deep-well): +10% maximum mana.")
	assert.Contains(t, view, "Type talent pick [talent] to choose.")
	assert.Contains(t, w.cmd("class", ""), "1 talent to choose: talent pick [talent].")

	preview := w.cmd("talent", "pick deep well")
	assert.Contains(t, preview, "You can take Deep Well")
	assert.Contains(t, preview, "talent pick deep-well confirm")
	assert.Empty(t, store.state.Talents)

	assert.Contains(t, w.cmd("talent", "pick deep-well confirm"), "You take Deep Well")
	assert.Equal(t, []string{"deep-well"}, store.state.Talents)
	assert.Contains(t, w.cmd("talent", "pick mending-hands confirm"), "no talent to choose now. The next comes at level 15.")
	assert.Contains(t, w.cmd("talent", ""), "Deep Well: +10% maximum mana.")

	// A talent of another lineage, and an unknown one.
	w.aria.Character.Level = 15
	assert.Contains(t, w.cmd("talent", "pick toughness confirm"), "can't take Toughness")
	assert.Contains(t, w.cmd("talent", "pick gibberish confirm"), "no talent called")
	// The same talent twice is allowed, a third time is not.
	assert.Contains(t, w.cmd("talent", "pick deep-well confirm"), "You take Deep Well")
	w.aria.Character.Level = 25
	assert.Contains(t, w.cmd("talent", "pick deep-well confirm"), "as many times as it can be taken")
	assert.Equal(t, []string{"deep-well", "deep-well"}, store.state.Talents)
}

func TestTalentPickSaveFailureChangesNothing(t *testing.T) {
	w, store := classBrawl(t, 5, 0)
	store.failing = true
	assert.Contains(t, w.cmd("talent", "pick deep-well confirm"), "couldn't be saved; nothing changed")
	assert.Empty(t, store.state.Talents)
}

func TestCompanionTalentThroughTheCommandWorksOnTheLiveMob(t *testing.T) {
	w, _ := classBrawl(t, 3, 0)
	oswin := w.companion(2)
	oswin.Character.Level = 5
	w.respawn()
	before := w.companion(2).Character.ManaMax.Value
	require.NotZero(t, before)

	assert.Contains(t, w.cmd("talent", "oswin"), "1 to choose")
	assert.Contains(t, w.cmd("talent", "pick oswin deep-well confirm"), "Brother Oswin takes Deep Well")
	_, talents := companionClass(t, 2)
	assert.Equal(t, []string{"deep-well"}, talents)
	after := w.companion(2).Character.ManaMax.Value
	assert.Greater(t, after, before, "the live companion's mana pool grew at once")
	assert.InDelta(t, float64(before)*1.10, float64(after), 1.0)

	// A level lost to death switches the talent off until it is regained;
	// the pick itself is never lost.
	w.companion(2).Character.Level = 4
	w.companion(2).Character.RecalculateStats()
	assert.Zero(t, w.companion(2).Character.ClassEffects().Int(classes.ManaPct), "the talent is off below its level")
	_, talents = companionClass(t, 2)
	assert.Equal(t, []string{"deep-well"}, talents)
}

func TestOldRecordsReadAsUnpromoted(t *testing.T) {
	w, _ := classBrawl(t, 3, 0)
	class, talents := companionClass(t, 1)
	assert.Empty(t, class)
	assert.Empty(t, talents)
	live, liveTalents := w.companion(1).Character.ClassState()
	assert.Empty(t, live)
	assert.Empty(t, liveTalents)
	assert.Nil(t, w.companion(1).Character.ClassEffects())
}

// Phase 38c1: the elite step through the real commands for the six open
// routes, at the gate boundaries, for the leader.
func TestEliteGatesForTheSixOpenRoutes(t *testing.T) {
	for _, tc := range []struct {
		lineage, advanced, elite, name string
		good, evil                     bool
	}{
		{"warrior", "knight", "paladin", "Paladin", true, false},
		{"warrior", "mercenary", "warlord", "Warlord", false, false},
		{"warrior", "blackguard", "dread-knight", "Dread Knight", false, true},
		{"cleric", "priest", "hierarch", "Hierarch", true, false},
		{"cleric", "druid", "elder-druid", "Elder Druid", false, false},
		{"cleric", "blood-priest", "demonologist", "Demonologist", false, true},
	} {
		t.Run(tc.elite, func(t *testing.T) {
			wait, ok := 0, 0 // the alignments one point short of the gate, and at it
			switch {
			case tc.good:
				wait, ok = 29, 30
			case tc.evil:
				wait, ok = -29, -30
			}
			w, store := classBrawl(t, 29, ok)
			w.withArchetypes(tc.lineage)
			store.state.Class = tc.advanced
			assert.Contains(t, w.cmd("class", "promote "+tc.elite+" confirm"), "needs level 30", "level 29")
			assert.Equal(t, tc.advanced, store.state.Class)
			assert.NotContains(t, w.cmd("class", ""), "Ready to promote")

			w.aria.Character.Level = 30
			if tc.good || tc.evil {
				w.aria.Character.Alignment = int8(wait)
				assert.Contains(t, w.cmd("class", ""), "Waiting: "+tc.name, "one point short of the gate waits")
				assert.Contains(t, w.cmd("class", "promote "+tc.elite+" confirm"), "needs alignment")
				assert.Equal(t, tc.advanced, store.state.Class)
			}
			w.aria.Character.Alignment = int8(ok)
			assert.Contains(t, w.cmd("class", ""), "Ready to promote: "+tc.name)
			preview := w.cmd("class", "promote "+tc.elite)
			assert.Contains(t, preview, "(elite, "+tc.lineage+" lineage)")
			assert.Contains(t, preview, "Now:  rank 30")
			assert.Contains(t, preview, "Talents from 35 add:")
			assert.Equal(t, tc.advanced, store.state.Class, "a preview changes nothing")

			assert.Contains(t, w.cmd("class", "promote "+tc.elite+" confirm"), "You are now a "+tc.name+".")
			assert.Equal(t, tc.elite, store.state.Class)
			assert.Contains(t, w.cmd("class", "promote "+tc.elite+" confirm"), "already a "+tc.name, "a repeated confirm changes nothing")
			assert.Contains(t, w.cmd("class", ""), "("+"elite class)")
			assert.Contains(t, w.cmd("class", "promote "+tc.advanced+" confirm"), "final", "routes are final")
		})
	}
}

// A late promoter takes the advanced class and the elite one in the same
// visit, and gets every rank its level has earned at once.
func TestLateElitePromotionCatchesUpEveryRank(t *testing.T) {
	w, store := classBrawl(t, 42, 0)
	w.withArchetypes("warrior")
	preview := w.cmd("class", "promote mercenary")
	assert.Contains(t, preview, "Warrior -> Mercenary (advanced, warrior lineage)")
	assert.Contains(t, w.cmd("class", "promote mercenary confirm"), "You are now a Mercenary.")

	preview = w.cmd("class", "promote warlord")
	assert.Contains(t, preview, "Mercenary -> Warlord (elite, warrior lineage)")
	for _, rank := range []string{"rank 30 Marked for Ruin", "rank 35 Battle Cry", "rank 40 Quicker tackle"} {
		assert.Contains(t, preview, rank)
	}
	assert.NotContains(t, preview, "rank 45 Sunder", "not yet earned")
	assert.Contains(t, preview, "Next: rank 45 (level 45): Sunder")
	assert.Contains(t, preview, "Talents from 35 add: Iron Hide, Second Wind, Veteran's Edge")

	confirmed := w.cmd("class", "promote warlord confirm")
	assert.Contains(t, confirmed, "You are now a Warlord. Your route is final.")
	for _, rank := range []string{"Rank 30, Marked for Ruin", "Rank 35, Battle Cry", "Rank 40, Quicker tackle"} {
		assert.Contains(t, confirmed, rank)
	}
	assert.Equal(t, "warlord", store.state.Class)
	view := w.cmd("class", "")
	assert.Contains(t, view, "a Warlord (elite class)")
	assert.Contains(t, view, "Rank 30, Marked for Ruin")
	assert.Contains(t, view, "Rank 40, Quicker tackle")
	assert.NotContains(t, view, "Rank 45")
	assert.Equal(t, 5, w.aria.Character.ClassEffects().Int(classes.MarkRuin))
	assert.Equal(t, 3, w.aria.Character.ClassEffects().Int(classes.BattleCry))
}

// An elite companion promotes by the same commands, the class is saved, and
// a restart keeps it with its ranks.
func TestEliteCompanionPromotionKeepsTheClassAcrossARestart(t *testing.T) {
	w, _ := classBrawl(t, 3, 0)
	tamsin := w.companion(1)
	tamsin.Character.Level = 31
	setCompanionAlignment(t, 1, 0)
	w.respawn()
	require.Equal(t, 31, w.companion(1).Character.Level)
	// Phase 38c1 review: a base companion asking for the elite is sent to
	// its own advanced step, with its selector.
	assert.Contains(t, w.cmd("class", "promote tamsin warlord confirm"), "take the Mercenary first, then the Warlord in the same visit. Type class promote #1 mercenary.")
	require.NoError(t, module.registry.SetCompanionClass(7, 1, "mercenary"))
	w.respawn()

	view := w.cmd("class", "tamsin")
	assert.Contains(t, view, "Ready to promote: Warlord. Type class promote #1 warlord.")
	assert.Regexp(t, `\[promote ready.*class #1\]`, w.cmd("company", "status"))
	preview := w.cmd("class", "promote tamsin warlord")
	assert.Contains(t, preview, "Tamsin Reed: Mercenary -> Warlord (elite, warrior lineage)")
	assert.Contains(t, preview, "Type: class promote #1 warlord confirm")

	assert.Contains(t, w.cmd("class", "promote #1 warlord confirm"), "Tamsin Reed is now a Warlord (elite). Their route is final.")
	class, _ := companionClass(t, 1)
	assert.Equal(t, "warlord", class)
	live, _ := w.companion(1).Character.ClassState()
	assert.Equal(t, "warlord", live)
	assert.Equal(t, 5, w.companion(1).Character.ClassEffects().Int(classes.MarkRuin), "the live companion has the rank at once")
	assert.Contains(t, w.cmd("company", "status"), "Warlord (elite warrior)")
	assert.NotContains(t, w.cmd("company", "status"), "promote ready")

	w.respawn() // copyover and restart rebuild the mob from the record
	class, _ = companionClass(t, 1)
	assert.Equal(t, "warlord", class)
	live, _ = w.companion(1).Character.ClassState()
	assert.Equal(t, "warlord", live)
	assert.Contains(t, w.cmd("class", "tamsin"), "a Warlord (elite class)")
}

func TestCompanionWaitingOnTheGateIsMarkedInTheRoster(t *testing.T) {
	w, _ := classBrawl(t, 3, 0)
	w.companion(1).Character.Level = 30
	setCompanionAlignment(t, 1, 29)
	w.respawn()
	require.NoError(t, module.registry.SetCompanionClass(7, 1, "knight"))
	w.respawn()
	roster := w.cmd("company", "status")
	assert.Contains(t, roster, "Knight (advanced warrior)")
	assert.Regexp(t, `\[waiting: alignment.*class #1\]`, roster)
	assert.Contains(t, w.cmd("class", "promote tamsin paladin confirm"), "needs alignment +30")
	setCompanionAlignment(t, 1, 30)
	assert.Regexp(t, `\[promote ready.*class #1\]`, w.cmd("company", "status"))
}

// Elite talents are for elites from level 35: refused below it, and for a
// non-elite, then taken through the command.
func TestEliteTalentsThroughTheCommand(t *testing.T) {
	w, store := classBrawl(t, 34, 0)
	w.withArchetypes("warrior")
	store.state = classes.State{Class: "warlord", Talents: []string{"toughness", "toughness", "keen-edge"}}
	assert.Contains(t, w.cmd("talent", "pick iron-hide confirm"), "Iron Hide is an elite talent: it opens to an elite class from level 35.", "level 34")

	w.aria.Character.Level = 35
	menu := w.cmd("talent", "")
	assert.Contains(t, menu, "Iron Hide (iron-hide): +5 armor.")
	assert.Contains(t, menu, "Second Wind (second-wind)")
	assert.Contains(t, menu, "Veteran's Edge (veterans-edge): +3 Attack.")
	assert.Contains(t, w.cmd("talent", "pick iron hide"), "You can take Iron Hide")
	assert.Empty(t, store.state.Talents[3:], "a preview changes nothing")
	assert.Contains(t, w.cmd("talent", "pick iron-hide confirm"), "You take Iron Hide: +5 armor.")
	assert.Equal(t, []string{"toughness", "toughness", "keen-edge", "iron-hide"}, store.state.Talents)
	assert.Equal(t, 5, w.aria.Character.ClassEffects().Int(classes.Armor))

	// Each is taken once.
	w.aria.Character.Level = 45
	assert.Contains(t, w.cmd("talent", "pick iron-hide confirm"), "as many times as it can be taken")

	// An advanced Mercenary has none of them, and another lineage's are unknown.
	store.state = classes.State{Class: "mercenary", Talents: []string{"toughness", "toughness", "keen-edge"}}
	w.aria.Character.Level = 35
	assert.NotContains(t, w.cmd("talent", ""), "Iron Hide")
	assert.Contains(t, w.cmd("talent", "pick iron-hide confirm"), "is an elite talent")
	store.state = classes.State{Class: "hierarch", Talents: []string{"deep-well", "deep-well", "mending-hands"}}
	w.withArchetypes("cleric")
	assert.Contains(t, w.cmd("talent", "pick iron-hide confirm"), "can't take Iron Hide")
	assert.Contains(t, w.cmd("talent", "pick font-of-grace confirm"), "You take Font of Grace")
}
