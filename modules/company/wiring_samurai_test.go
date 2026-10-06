package company

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39b wiring: the Samurai's lineage through the real strategy pass,
// guard rules and combat round (shipped config, DoCombat), in the brawl
// world.

// hatamotoGuardBrawl is a guard brawl whose Tamsin is a level-level Hatamoto
// (a Samurai route) fighting by the ordinary fighter strategy.
func hatamotoGuardBrawl(t *testing.T, level int) *brawl {
	t.Helper()
	b := guardBrawl(t, "tamsin fighter")
	tamsin := b.companion(1)
	tamsin.Character.HPArchetype = "samurai"
	tamsin.Character.Level = level
	tamsin.Character.SetClassState("hatamoto", nil)
	require.Equal(t, 2, tamsin.Character.ClassEffects().Int(classes.Bodyguard))
	return b
}

func TestHatamotoBodyguardStepsInForTheLeaderTwiceABattle(t *testing.T) {
	b := hatamotoGuardBrawl(t, 10)
	got := b.listen()
	tamsin := b.companion(1)

	b.strike(0, false)
	out := b.fight()
	assert.Contains(t, out, "Tamsin Reed steps in front of you. (guard, 1 left)")
	used := guardEvents(*got, combatstream.GuardUsed)
	require.Len(t, used, 1)
	assert.Equal(t, tamsin.InstanceId, used[0].Source.MobInstanceId)
	assert.Equal(t, 7, used[0].Target.UserId)
	assert.Zero(t, blowsOn(*got, "u:7"), "the blow went to the Hatamoto")
	assert.Equal(t, battle.MaxGuards, battle.GuardsLeft(7, "companion:1"), "its strategy guards are untouched")

	b.toughen()
	b.hold(nil)
	b.strike(0, false)
	assert.Contains(t, b.fight(), "Tamsin Reed steps in front of you. (guard, none left)")

	b.toughen()
	b.hold(nil)
	b.strike(0, false)
	before := len(guardEvents(*got, combatstream.GuardUsed))
	b.fight()
	assert.Len(t, guardEvents(*got, combatstream.GuardUsed), before, "two a battle, no more")
	assert.GreaterOrEqual(t, blowsOn(*got, "u:7"), 1, "the third blow reached the leader")
}

func TestBodyguardOnlyGuardsTheLeaderAndOnlyForAHatamoto(t *testing.T) {
	b := hatamotoGuardBrawl(t, 10)
	got := b.listen()
	b.strike(3, false)
	b.fight()
	assert.Empty(t, guardEvents(*got, combatstream.GuardUsed), "a blow on Garrick is not the leader's")

	plain := guardBrawl(t, "tamsin fighter")
	plain.companion(1).Character.HPArchetype = "samurai"
	plain.companion(1).Character.SetClassState("kensai", nil)
	more := plain.listen()
	plain.strike(0, false)
	plain.fight()
	assert.Empty(t, guardEvents(*more, combatstream.GuardUsed), "a Kensai has no Bodyguard")
}

func TestSamuraiFightsByTheStrongestRuleUntilTold(t *testing.T) {
	assert.Equal(t, strategy.Strongest, strategy.Default("samurai").Rule)
	assert.Equal(t, strategy.Fighter, strategy.Default("samurai").Role)
	assert.Equal(t, strategy.Weakest, strategy.Default("warrior").Rule)

	b := newBrawl(t)
	b.withArchetypes("samurai")
	assert.Contains(t, b.cmd("strategy", ""), "strongest")
	b.cmd("strategy", "me weakest")
	assert.Contains(t, b.cmd("strategy", ""), "weakest")
}

// samuraiRounds is a fight begun on the captain with Aria a Samurai of the
// route and level; nobody lands a blow unless a test says so.
func samuraiRounds(t *testing.T, level int, class string) *brawl {
	t.Helper()
	b := newBrawl(t)
	store := &fakeClassStore{state: classes.State{Class: class}}
	classes.SetProvider(store)
	t.Cleanup(func() { classes.SetProvider(nil) })
	b.withArchetypes("samurai")
	b.unplaced()
	forceBlows(t, false)
	noCounters(t)
	b.aria.Character.Level = level
	b.startWitchFight()
	return b
}

func TestFocusBuildsOverQuietRoundsThroughTheRealRound(t *testing.T) {
	b := samuraiRounds(t, 3, "")
	for i := 0; i < 4; i++ {
		b.hold(nil)
		b.fight()
	}
	rt := b.aria.Character.RT
	require.NotNil(t, rt)
	assert.GreaterOrEqual(t, rt.Quiet, 2, "quiet rounds are counted")
	assert.Positive(t, b.aria.Character.ClassCrit())
	assert.True(t, rt.IaiSpent, "her first swing spent Iaijutsu")
}

func TestRoninVengeanceSeesTheFallenThroughTheRealRound(t *testing.T) {
	b := samuraiRounds(t, 10, "ronin")
	b.hold(nil)
	b.fight()
	require.NotNil(t, b.aria.Character.RT)
	assert.Equal(t, 5, b.aria.Character.RT.SidePeak, "the whole company stood")
	assert.Zero(t, b.aria.Character.Aura.Fallen)

	b.companion(2).Character.Health = 0
	b.hold(nil)
	b.fight()
	assert.Equal(t, 1, b.aria.Character.Aura.Fallen, "Oswin has fallen")
}

func TestSamuraiAbilitiesResetBetweenBattles(t *testing.T) {
	b := samuraiRounds(t, 8, "")
	b.hold(nil)
	b.fight()
	require.NotNil(t, b.aria.Character.RT)
	b.aria.Character.EndFightRT()
	assert.True(t, b.aria.Character.IaiReady(), "the next battle opens with Iaijutsu again")
	hooks.ResetTempoForTest()
}

// samuraiClassBrawl is the class brawl with Aria a Samurai of the level and
// alignment, and a fake class store.
func samuraiClassBrawl(t *testing.T, level, alignment int) (*trainWorld, *fakeClassStore) {
	t.Helper()
	w := trainingBrawl(t)
	store := &fakeClassStore{}
	classes.SetProvider(store)
	t.Cleanup(func() { classes.SetProvider(nil) })
	w.withArchetypes("samurai")
	w.aria.Character.Level = level
	w.aria.Character.Alignment = int8(alignment)
	return w, store
}

func TestClassShowsASamuraisBaseRanksBeforePromotion(t *testing.T) {
	w, _ := samuraiClassBrawl(t, 1, 0)
	view := w.cmd("class", "")
	assert.Contains(t, view, "level 1 Samurai with no promotion yet")
	assert.Contains(t, view, "Rank 1, Iaijutsu")
	assert.NotContains(t, view, "Rank 3, Focus")
	assert.Contains(t, view, "Next: a rank (Focus) at level 3.")

	w.aria.Character.Level = 8
	view = w.cmd("class", "")
	for _, want := range []string{"Rank 1, Iaijutsu", "Rank 3, Focus", "Rank 8, Zanshin"} {
		assert.Contains(t, view, want)
	}
}

func TestSamuraiPromotesAtAnyAlignmentThroughTheRealCommand(t *testing.T) {
	for _, alignment := range []int{-100, 0, 100} {
		t.Run(fmt.Sprintf("alignment %d", alignment), func(t *testing.T) {
			w, store := samuraiClassBrawl(t, 10, alignment)
			paths := w.cmd("class", "paths")
			for _, want := range []string{"Kensai", "Hatamoto", "Ronin", "any alignment", "Sword Saint"} {
				assert.Contains(t, paths, want)
			}
			view := w.cmd("class", "")
			for _, want := range []string{"Ready to promote: Kensai.", "Ready to promote: Hatamoto.", "Ready to promote: Ronin."} {
				assert.Contains(t, view, want)
			}
			assert.Contains(t, w.cmd("class", "promote hatamoto confirm"), "You are now a Hatamoto.")
			assert.Equal(t, "hatamoto", store.state.Class)
			view = w.cmd("class", "")
			assert.Contains(t, view, "a Hatamoto (advanced class)")
			assert.Contains(t, view, "Rank 10, Bodyguard")
			assert.Contains(t, view, "Rank 1, Iaijutsu", "its base ranks stay")
		})
	}
}

func TestSamuraiTalentsThroughTheRealCommand(t *testing.T) {
	w, store := samuraiClassBrawl(t, 5, 0)
	assert.Contains(t, w.cmd("talent", "list"), "Sharp Eye")
	assert.Contains(t, w.cmd("talent", "pick sharp-eye confirm"), "Sharp Eye")
	assert.Equal(t, []string{"sharp-eye"}, store.state.Talents)
}
