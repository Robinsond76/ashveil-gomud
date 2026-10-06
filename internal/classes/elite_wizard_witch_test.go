package classes

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 38c3: the wizard and witch elites' ranks, talents and catch-up.

func TestWizardAndWitchEliteRanksApplyFromTheirLevel(t *testing.T) {
	for _, c := range []struct {
		class string
		level int
		key   string
		want  int
	}{
		{"archon", 29, Counter, 0},
		{"archon", 30, Counter, 12},
		{"archon", 35, WardExtra, 1},
		{"archon", 40, SpellShield, 15},
		{"archon", 45, CounterBonus, 10},
		{"archon", 45, CounterDrain, 10},
		{"archon", 50, WardBlows, 3},
		{"archon", 55, Reflect, 1},
		{"archon", 60, Aegis, 1},
		{"archon", 59, Aegis, 0},
		{"archmage", 30, Overchannel, 4},
		{"archmage", 35, ChantTrim, 1},
		{"archmage", 40, SpellCost, 15}, // Thrifty casting replaces Efficient casting's 10
		{"archmage", 45, Barrage, 1},
		{"archmage", 50, Overchannel, 3},
		{"archmage", 55, ChantBreak, 50}, // Steady casting replaces Steady chant's 25
		{"archmage", 60, Storm, 1},
		{"necromancer", 29, Raise, 0},
		{"necromancer", 30, Raise, 1},
		{"necromancer", 30, RaiseHP, 60},
		{"necromancer", 10, Siphon, 1},
		{"necromancer", 35, DrainPct, 150},
		{"necromancer", 40, GraveChill, 1},
		{"necromancer", 45, RaiseHP, 85},
		{"necromancer", 50, Raise, 2},
		{"necromancer", 55, Harvest, 3},
		{"necromancer", 60, Bargain, 1},
		{"wise-one", 30, HexWard, 3},
		{"wise-one", 30, HexWardCap, 200},
		{"wise-one", 35, SlumberLong, 2},
		{"wise-one", 40, WardMend, 1},
		{"wise-one", 45, WardCleanse, 1},
		{"wise-one", 50, WardPeace, 5},
		{"wise-one", 55, HexWard, 4},
		{"wise-one", 60, WardLife, 1},
		{"coven-mother", 30, HexLand, 5},
		{"coven-mother", 30, HexChant, 2},
		{"coven-mother", 35, HexLong, 1},
		{"coven-mother", 40, HexCost, 20},
		{"coven-mother", 45, BossHalf, 1},
		{"coven-mother", 50, TwinHex, 1},
		{"coven-mother", 55, HexLand, 10},
		{"coven-mother", 60, Circle, 1},
		{"crone-of-ash", 30, HexedDamage, 40},
		{"crone-of-ash", 30, CurseAtk, 8},
		{"crone-of-ash", 35, PoisonX2, 1},
		{"crone-of-ash", 40, DreadAll, 1},
		{"crone-of-ash", 45, Linger, 1},
		{"crone-of-ash", 50, HexedDamage, 55},
		{"crone-of-ash", 55, SoulRot, 1},
		{"crone-of-ash", 60, Doom, 1},
	} {
		assert.Equal(t, c.want, EffectsFor(c.class, c.level, nil).Int(c.key), "%s %s at %d", c.class, c.key, c.level)
	}
}

func TestOnlyTheNecromancerAndItsAdvancedClassTeachTheirSpells(t *testing.T) {
	for _, tc := range []struct {
		class, spell string
		level        int
		want         bool
	}{
		{"warlock", "siphon", 9, false},
		{"warlock", "siphon", 10, true},
		{"necromancer", "siphon", 10, true},
		{"necromancer", "raisefallen", 29, false},
		{"necromancer", "raisefallen", 30, true},
		{"warlock", "raisefallen", 60, false},
		{"archon", "raisefallen", 60, false},
	} {
		assert.Equal(t, tc.want, slices.Contains(SpellsAt(tc.class, tc.level), tc.spell), "%s %s at %d", tc.class, tc.spell, tc.level)
	}
}

func TestWizardAndWitchEliteTalentsAreForTheirElitesFromThirtyFive(t *testing.T) {
	for _, tc := range []struct {
		lineage, class, id string
		level              int
		err                error
	}{
		{"wizard", "archon", "spell-edge", 34, ErrEliteTalent},
		{"wizard", "archon", "spell-edge", 35, nil},
		{"wizard", "necromancer", "focused-will", 40, nil},
		{"wizard", "arcanist", "spell-edge", 45, ErrEliteTalent},
		{"wizard", "", "deep-reserves", 45, ErrEliteTalent},
		{"wizard", "archmage", "hex-reach", 45, ErrUnknownTalent},
		{"witch", "wise-one", "hex-reach", 35, nil},
		{"witch", "crone-of-ash", "iron-will", 45, nil},
		{"witch", "coven-mother", "deep-reserves", 55, nil},
		{"witch", "hag", "hex-reach", 45, ErrEliteTalent},
		{"witch", "wise-one", "spell-edge", 55, ErrUnknownTalent},
	} {
		// Enough slots owed, none taken yet.
		err := CanPick(tc.lineage, tc.class, nil, tc.level, tc.id)
		if tc.err == nil {
			assert.NoError(t, err, "%s %s %d %s", tc.lineage, tc.class, tc.level, tc.id)
		} else {
			assert.ErrorIs(t, err, tc.err, "%s %s %d %s", tc.lineage, tc.class, tc.level, tc.id)
		}
	}
	assert.Len(t, EliteTalentsFor("wizard"), 3)
	assert.Len(t, EliteTalentsFor("witch"), 3)
	assert.Len(t, MenuFor("wizard", "archmage", 34), 5)
	assert.Len(t, MenuFor("witch", "wise-one", 35), 8)
}

func TestWizardAndWitchEliteTalentEffects(t *testing.T) {
	base := EffectsFor("archmage", 35, []string{"deep-well", "focus"})
	edge := EffectsFor("archmage", 35, []string{"deep-well", "focus", "spell-edge"})
	assert.Equal(t, 10, edge.Int(SpellPct)-base.Int(SpellPct), "Spell Edge")
	reserves := EffectsFor("archmage", 35, []string{"deep-well", "focus", "spell-edge", "deep-reserves"})
	assert.Equal(t, 10, reserves.Int(ManaPct)-edge.Int(ManaPct), "Deep Reserves")
	assert.Equal(t, 10, EffectsFor("coven-mother", 35, []string{"hex-reach"}).Int(HexLand), "Coven's Will 5 and Hex Reach 5")
	assert.Equal(t, 25, EffectsFor("crone-of-ash", 35, []string{"iron-will"}).Int(ChantBreak)-EffectsFor("crone-of-ash", 35, nil).Int(ChantBreak), "Iron Will")
}

func TestLateWizardElitePromotionCatchesUpEveryRank(t *testing.T) {
	lines := RankLines("wizard", "archon", 29, 60)
	require.Len(t, lines, 7, "every elite rank, once")
	assert.Contains(t, lines[0], "Counterspell")
	assert.Contains(t, lines[6], "Archon's Aegis")
	assert.Equal(t, "Elite promotion ready: Warlock -> Necromancer. Visit a camp or town and type class promote #3 necromancer.",
		ReadinessNote("wizard", "warlock", 30, -40, "#3", ""))
	assert.Equal(t, "ready", PromotionState("witch", "hag", 30, -30))
	assert.Equal(t, "waiting-gate", PromotionState("witch", "hedge-witch", 30, 29))
}
