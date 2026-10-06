package classes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Phase 38c2: the rogue and ranger elites' ranks and talents.

func TestRogueAndRangerEliteTalentsAreOfferedToTheirElitesFromThirtyFive(t *testing.T) {
	assert.Len(t, EliteTalentsFor("rogue"), 3)
	assert.Len(t, EliteTalentsFor("ranger"), 3)
	assert.Len(t, MenuFor("rogue", "duelist", 45), 5, "an advanced class sees only the lineage's five")
	assert.Len(t, MenuFor("rogue", "swordmaster", 34), 5)
	assert.Len(t, MenuFor("rogue", "swordmaster", 35), 8)
	assert.Len(t, MenuFor("ranger", "ravager", 35), 8)
	picked := []string{"keen-edge", "footwork", "toughness"}
	assert.NoError(t, CanPick("rogue", "nightblade", picked, 35, "razors-edge"))
	assert.NoError(t, CanPick("ranger", "marksman", picked, 35, "long-draw"))
	assert.ErrorIs(t, CanPick("rogue", "assassin", picked, 35, "shadow-footing"), ErrEliteTalent)
	assert.ErrorIs(t, CanPick("ranger", "sentinel", picked, 35, "shadow-footing"), ErrUnknownTalent, "another lineage's")
	assert.Equal(t, 5, EffectsFor("pathfinder", 35, []string{"shadow-footing"}).Int(Evasion)-EffectsFor("pathfinder", 35, nil).Int(Evasion))
	assert.Equal(t, 10, EffectsFor("sentinel", 35, []string{"long-draw"}).Int(RangedPct))
	assert.Equal(t, 15, EffectsFor("ravager", 35, []string{"quick-nock"}).Int(OpenMeter))
}

func TestRogueAndRangerEliteSignatureRanksAppearAtTheirLevels(t *testing.T) {
	for _, tc := range []struct {
		class string
		level int
		key   string
		want  int
	}{
		{"pathfinder", 29, PathEye, 0}, {"pathfinder", 30, PathEye, 1}, {"pathfinder", 49, PathOpens, 1}, {"pathfinder", 50, PathOpens, 2},
		{"pathfinder", 60, AmbushFlip, 1}, {"pathfinder", 59, AmbushFlip, 0},
		{"swordmaster", 45, RiposteRound, 2}, {"swordmaster", 44, RiposteRound, 0}, {"swordmaster", 60, RiposteFree, 1},
		{"nightblade", 30, DeathMark, 40}, {"nightblade", 40, Finisher, 3}, {"nightblade", 60, Coup, 20},
		{"sentinel", 30, Overwatch, 1}, {"sentinel", 54, OverwatchMax, 1}, {"sentinel", 55, OverwatchMax, 2}, {"sentinel", 60, GuardArrow, 1},
		{"marksman", 30, RangedCrit, 15}, {"marksman", 35, AimCD, 2}, {"marksman", 60, PerfectShot, 1},
		{"ravager", 30, HuntDown, 75}, {"ravager", 54, FleePenalty, 25}, {"ravager", 55, FleePenalty, 40}, {"ravager", 60, Apex, 1},
	} {
		assert.Equal(t, tc.want, EffectsFor(tc.class, tc.level, nil).Int(tc.key), "%s at %d: %s", tc.class, tc.level, tc.key)
	}
}
