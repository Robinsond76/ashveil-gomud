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
	assert.Contains(t, w.cmd("class", "paths tamsin"), "Warlord (planned: not open yet)")
}

func TestPlayerPromotionPreviewConfirmAndRepeat(t *testing.T) {
	w, store := classBrawl(t, 10, 45)
	preview := w.cmd("class", "promote priest")
	assert.Contains(t, preview, "You can become a Priest")
	assert.Contains(t, preview, "Rank 10, Ward")
	assert.Contains(t, preview, "Routes are final")
	assert.Contains(t, preview, "class promote priest confirm")
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
	assert.Contains(t, w.cmd("class", "promote priest"), "You can become a Priest")
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
	assert.Contains(t, preview, "Brother Oswin can become a Priest")
	assert.Contains(t, preview, "Type class promote #2 priest confirm")
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

func TestPlannedEliteIsNotOpenYet(t *testing.T) {
	w, store := classBrawl(t, 30, 100)
	w.withArchetypes("warrior")
	store.state.Class = "mercenary"
	assert.Contains(t, w.cmd("class", "promote warlord confirm"), "not open yet")
}

func TestUnpromotedHighLevelCharacterKeepsItsBase(t *testing.T) {
	w, _ := classBrawl(t, 40, 0)
	view := w.cmd("class", "")
	assert.Contains(t, view, "level 40 Cleric with no promotion yet")
	assert.Contains(t, view, "Ready to promote: Druid.")
	// Elite needs an advanced class first.
	assert.Contains(t, w.cmd("class", "promote hierarch confirm"), "can't become a Hierarch")
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
