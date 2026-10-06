package classes

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var baseLineages = []string{"cleric", "ranger", "rogue", "warrior", "witch", "wizard"}

// TestEveryLineageHasThreeAdvancedRoutes: one good, one unrestricted and one
// evil route a lineage, each with an elite continuation.
func TestEveryLineageHasThreeAdvancedRoutes(t *testing.T) {
	assert.Subset(t, Lineages(), baseLineages) // the neutral lineages (39a+) have their own tests
	for _, l := range baseLineages {
		adv := Advanced(l)
		require.Len(t, adv, 3, l)
		gates := map[Gate]bool{}
		for _, c := range adv {
			gates[c.Gate] = true
			assert.Equal(t, TierAdvanced, c.Tier)
			assert.Equal(t, l, c.Lineage)
			assert.NotEmpty(t, c.Ranks, "%s has ranks", c.ID)
			elite, ok := Elite(c.ID)
			require.True(t, ok, "%s has an elite class", c.ID)
			assert.Equal(t, c.ID, elite.Parent)
			assert.Equal(t, l, elite.Lineage)
			assert.Equal(t, c.Gate, elite.Gate, "%s keeps its parent's gate", elite.ID)
		}
		assert.Len(t, gates, 3, "%s covers good, any and evil", l)
	}
}

// TestRankTablesAreOrdered: ranks sit on 5-level steps inside their tier and
// every elite class that is open has the full ladder to level 60.
func TestRankTablesAreOrdered(t *testing.T) {
	for _, c := range All() {
		prev := 0
		for _, r := range c.Ranks {
			assert.Greater(t, r.Level, prev, "%s ranks ascend", c.ID)
			assert.Zero(t, r.Level%5, "%s rank %d is on a 5-level step", c.ID, r.Level)
			assert.NotEmpty(t, r.Name)
			assert.NotEmpty(t, r.Text)
			assert.NotEmpty(t, r.Set, "%s rank %d sets an effect", c.ID, r.Level)
			prev = r.Level
			switch c.Tier {
			case TierAdvanced:
				assert.GreaterOrEqual(t, r.Level, AdvancedLevel)
				assert.Less(t, r.Level, EliteLevel)
			case TierElite:
				assert.GreaterOrEqual(t, r.Level, EliteLevel)
				assert.LessOrEqual(t, r.Level, MaxRankLevel)
			}
		}
		if c.Tier == TierAdvanced {
			var levels []int
			for _, r := range c.Ranks {
				levels = append(levels, r.Level)
			}
			assert.Equal(t, []int{10, 15, 20, 25}, levels, c.ID)
		}
		if c.Tier == TierElite && !c.Planned {
			var levels []int
			for _, r := range c.Ranks {
				levels = append(levels, r.Level)
			}
			assert.Equal(t, []int{30, 35, 40, 45, 50, 55, 60}, levels, c.ID)
		}
	}
}

// TestFaithRoutesAreOpenAndTheRestPlanned: the cleric and warrior elite
// routes of the faith design are selectable; the other lineages' elite
// routes are listed as planned (38c).
func TestFaithRoutesAreOpenAndTheRestPlanned(t *testing.T) {
	open := map[string]bool{"paladin": true, "dread-knight": true, "hierarch": true, "elder-druid": true, "demonologist": true}
	for _, c := range All() {
		if c.Tier != TierElite {
			continue
		}
		assert.Equal(t, !open[c.ID], c.Planned, c.ID)
	}
}

func TestGateBoundaries(t *testing.T) {
	cases := []struct {
		gate Gate
		al   int
		want bool
	}{
		{GateGood, 30, true}, {GateGood, 29, false}, {GateGood, 100, true}, {GateGood, -50, false},
		{GateEvil, -30, true}, {GateEvil, -29, false}, {GateEvil, -100, true}, {GateEvil, 50, false},
		{GateAny, -100, true}, {GateAny, 0, true}, {GateAny, 100, true},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, c.gate.Allows(c.al), "%v at %d", c.gate, c.al)
	}
}

func TestOptionsAtLevelTen(t *testing.T) {
	opts := Options("cleric", "", 10, 45)
	require.Len(t, opts, 3)
	byID := map[string]Option{}
	for _, o := range opts {
		byID[o.Class.ID] = o
	}
	assert.True(t, byID["priest"].Eligible)
	assert.True(t, byID["druid"].Eligible)
	assert.False(t, byID["blood-priest"].Eligible)
	assert.Contains(t, byID["blood-priest"].Reason, "-30 or lower")
	assert.Contains(t, byID["blood-priest"].Reason, "+45")

	for _, o := range Options("cleric", "", 9, 45) {
		assert.False(t, o.Eligible, o.Class.ID)
		assert.Contains(t, o.Reason, "level 10")
	}
}

// TestEliteWaitsForItsGate: an advanced character below its elite gate at
// level 30 waits, keeping its route, and the option flips once alignment
// recovers (the gate's boundary values).
func TestEliteWaitsForItsGate(t *testing.T) {
	for _, tc := range []struct {
		adv, elite string
		low, ok    int
	}{
		{"priest", "hierarch", 29, 30},
		{"knight", "paladin", 29, 30},
		{"blood-priest", "demonologist", -29, -30},
		{"blackguard", "dread-knight", -29, -30},
		{"druid", "elder-druid", -100, 100},
	} {
		if tc.adv == "druid" {
			o := Options("cleric", tc.adv, 30, tc.low)
			require.Len(t, o, 1)
			assert.True(t, o[0].Eligible, "an unrestricted route never waits")
			continue
		}
		o := Options("cleric", tc.adv, 30, tc.low)
		require.Len(t, o, 1, tc.adv)
		assert.Equal(t, tc.elite, o[0].Class.ID)
		assert.False(t, o[0].Eligible, tc.adv)
		assert.True(t, o[0].Waiting, tc.adv)
		_, err := Check("cleric", tc.adv, tc.elite, 30, tc.low)
		assert.Error(t, err)
		c, err := Check("cleric", tc.adv, tc.elite, 30, tc.ok)
		require.NoError(t, err, tc.adv)
		assert.Equal(t, tc.elite, c.ID)
	}
	_, err := Check("cleric", "priest", "hierarch", 29, 100)
	assert.Error(t, err, "level 30 is required")
}

func TestPlannedEliteIsNotSelectable(t *testing.T) {
	o := Options("rogue", "scout", 30, 100)
	require.Len(t, o, 1)
	assert.False(t, o[0].Eligible)
	assert.False(t, o[0].Waiting)
	_, err := Check("rogue", "scout", "pathfinder", 40, 100)
	assert.Error(t, err)
}

// TestRoutesAreFinal: no sibling, no cross-lineage and no second promotion.
func TestRoutesAreFinal(t *testing.T) {
	_, err := Check("cleric", "priest", "druid", 40, 100)
	assert.ErrorIs(t, err, ErrNotAvailable)
	_, err = Check("cleric", "priest", "blood-priest", 40, -100)
	assert.ErrorIs(t, err, ErrNotAvailable)
	_, err = Check("cleric", "", "knight", 10, 100)
	assert.ErrorIs(t, err, ErrNotAvailable)
	_, err = Check("cleric", "hierarch", "druid", 60, 100)
	assert.ErrorIs(t, err, ErrFinalRoute)
	_, err = Check("cleric", "", "nonsense", 10, 100)
	assert.ErrorIs(t, err, ErrNoSuchClass)
	assert.Empty(t, Options("cleric", "hierarch", 60, 100))
}

func TestOnlyUnpromotedCanTakeAdvanced(t *testing.T) {
	_, err := Check("warrior", "knight", "mercenary", 20, 100)
	assert.ErrorIs(t, err, ErrNotAvailable)
	c, err := Check("warrior", "", "knight", 10, 30)
	require.NoError(t, err)
	assert.Equal(t, "knight", c.ID)
}

// TestRanksApplyFromTheirLevelAndNeverEarlier, and a level lost removes
// them (they are derived, never saved).
func TestRanksApplyFromTheirLevel(t *testing.T) {
	assert.Equal(t, 0, EffectsFor("knight", 9, nil).Int(LayHands))
	assert.Equal(t, 2, EffectsFor("knight", 10, nil).Int(LayHands))
	assert.Equal(t, 2, EffectsFor("knight", 14, nil).Int(LayHands))
	assert.Equal(t, 3, EffectsFor("knight", 15, nil).Int(LayHands))
	assert.Equal(t, 0, EffectsFor("knight", 19, nil).Int(FaithBlock))
	assert.Equal(t, 5, EffectsFor("knight", 20, nil).Int(FaithBlock))
	assert.Equal(t, 3, EffectsFor("knight", 29, nil).Int(LayHands))

	// An elite class carries its advanced class's ranks.
	fx := EffectsFor("paladin", 30, nil)
	assert.Equal(t, 3, fx.Int(LayHands))
	assert.Equal(t, 1, fx.Int(LayFull))
	assert.Equal(t, 0, fx.Int(AuraResolv))
	assert.Equal(t, 10, EffectsFor("paladin", 35, nil).Int(AuraResolv))
	assert.Equal(t, 5, EffectsFor("paladin", 60, nil).Int(LayHands))
	assert.Equal(t, 1, EffectsFor("paladin", 60, nil).Int(DivineShield))

	// Losing levels to death removes the rank until it is regained.
	assert.Equal(t, 0, EffectsFor("paladin", 29, nil).Int(LayFull))
	assert.Nil(t, EffectsFor("knight", 9, nil))
	assert.Nil(t, EffectsFor("", 60, nil))
}

func TestSummonLaddersBuildOverRanks(t *testing.T) {
	assert.Equal(t, 1, EffectsFor("hierarch", 30, nil).Int(Summon))
	assert.Equal(t, 2, EffectsFor("demonologist", 30, nil).Int(Summon))
	fx := EffectsFor("hierarch", 60, nil)
	assert.Equal(t, 40, fx.Int(SummonArmor))
	assert.Equal(t, 2, fx.Int(AngelMercy))
	assert.Equal(t, 3, fx.Int(AngelGuards))
	assert.Equal(t, 10, fx.Int(AngelBlade))
	assert.Equal(t, 1, fx.Int(SummonSooner))
	assert.Equal(t, 8, EffectsFor("hierarch", 49, nil).Int(AngelBlade))
	assert.Equal(t, 6, EffectsFor("demonologist", 49, nil).Int(DemonClaws))
	assert.Equal(t, 8, EffectsFor("demonologist", 50, nil).Int(DemonClaws))
	// A Priest below 30 has no Angel.
	assert.Equal(t, 0, EffectsFor("priest", 29, nil).Int(Summon))
}

func TestTalentSlots(t *testing.T) {
	for level, want := range map[int]int{1: 0, 4: 0, 5: 1, 14: 1, 15: 2, 25: 3, 34: 3, 35: 4, 45: 5, 55: 6, 60: 6} {
		assert.Equal(t, want, TalentSlots(level), "level %d", level)
	}
	l, ok := NextTalentLevel(5)
	assert.True(t, ok)
	assert.Equal(t, 15, l)
	_, ok = NextTalentLevel(55)
	assert.False(t, ok)
}

func TestPickingTalents(t *testing.T) {
	assert.ErrorIs(t, CanPick("cleric", nil, 4, "mending-hands"), ErrNoTalentOwed)
	assert.NoError(t, CanPick("cleric", nil, 5, "mending-hands"))
	assert.ErrorIs(t, CanPick("cleric", nil, 5, "toughness"), ErrUnknownTalent, "a talent of another lineage")
	assert.ErrorIs(t, CanPick("cleric", []string{"deep-well"}, 5, "mending-hands"), ErrNoTalentOwed)
	assert.NoError(t, CanPick("cleric", []string{"deep-well"}, 15, "mending-hands"))
	// A talent can be taken twice, not three times.
	picked := []string{"deep-well", "deep-well"}
	assert.NoError(t, CanPick("cleric", picked, 25, "mending-hands"))
}

func TestTalentMaxIsEnforced(t *testing.T) {
	err := CanPick("warrior", []string{"tackle-drill"}, 15, "tackle-drill")
	assert.True(t, errors.Is(err, ErrTalentMaxed))
	err = CanPick("cleric", []string{"deep-well", "deep-well"}, 25, "deep-well")
	assert.True(t, errors.Is(err, ErrTalentMaxed))
}

func TestEveryLineageOffersFiveTalentsAndTheyAreDefined(t *testing.T) {
	for _, l := range baseLineages {
		ts := TalentsFor(l)
		assert.Len(t, ts, 5, l)
		for _, tl := range ts {
			assert.NotEmpty(t, tl.Add, tl.ID)
			assert.NotEmpty(t, tl.Text, tl.ID)
		}
	}
}

func TestTalentsAddToRankEffectsAndSwitchOffWithLevel(t *testing.T) {
	// A Priest's Prayer of Mending (20) and Gentle Rest (15) stack.
	fx := EffectsFor("priest", 25, []string{"gentle-rest", "mending-hands", "deep-well"})
	assert.Equal(t, 35, fx.Int(PatchCost))
	assert.Equal(t, 10, fx.Int(HealPct))
	assert.Equal(t, 10, fx.Int(ManaPct))

	// Only the talents the level has earned count.
	fx = EffectsFor("priest", 14, []string{"gentle-rest", "mending-hands"})
	assert.Equal(t, 15, fx.Int(PatchCost))
	assert.Equal(t, 0, fx.Int(HealPct))
	// A talent with no class: the lineage's base effects.
	fx = EffectsFor("", 5, []string{"deep-well"})
	assert.Equal(t, 10, fx.Int(ManaPct))
}

func TestMilestone(t *testing.T) {
	assert.Equal(t, "Next: a talent at level 5.", Milestone("", 3))
	assert.Equal(t, "Next: a talent at level 15.", Milestone("", 12))
	assert.Equal(t, "Next: your class promotion at level 10.", Milestone("", 5))
	assert.Equal(t, "Next: a talent and a rank (Greater Heal) at level 15.", Milestone("priest", 10))
	assert.Equal(t, "Next: a rank (Double ward) at level 20.", Milestone("priest", 15))
	assert.Equal(t, "Next: a talent and a rank (Prayer of Mending) at level 25.", Milestone("priest", 20))
	assert.Equal(t, "Next: your elite promotion at level 30.", Milestone("priest", 25))
	// A planned elite is not promised.
	assert.Equal(t, "Next: a talent at level 35.", Milestone("scout", 25))
	assert.Equal(t, "", Milestone("hierarch", 60))
}

// Phase 38b review: a rank spell can't be cast below its rank's level (a
// player keeps it in the spellbook after a death costs the level).
func TestSpellLockedBelowItsRank(t *testing.T) {
	assert.True(t, SpellLocked("priest", 14, "greaterheal"))
	assert.False(t, SpellLocked("priest", 15, "greaterheal"))
	assert.False(t, SpellLocked("hierarch", 35, "greaterheal"), "an elite keeps its advanced spells")
	assert.True(t, SpellLocked("hierarch", 29, "callhost"))
	assert.False(t, SpellLocked("priest", 1, "heal"), "a spell no rank teaches is never locked")
	assert.False(t, SpellLocked("", 1, "greaterheal"))
}
