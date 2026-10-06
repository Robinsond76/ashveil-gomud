package classes

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 38d: the Sorcerer and High Sorcerer.

func TestSorcererRanksApplyFromTheirLevel(t *testing.T) {
	for _, c := range []struct {
		class string
		level int
		key   string
		want  int
	}{
		{"sorcerer", 9, Lance, 0},
		{"sorcerer", 10, Lance, 1},
		{"sorcerer", 10, LanceCost, 18},
		{"sorcerer", 15, LancePct, 25},
		{"sorcerer", 20, ChantBreak, 25},
		{"sorcerer", 25, LanceCost, 15},
		{"high-sorcerer", 29, LanceTrim, 0},
		{"high-sorcerer", 30, LancePct, 40}, // replaces Gathered power's 25
		{"high-sorcerer", 35, LanceTrim, 1},
		{"high-sorcerer", 40, ChantBreak, 50}, // replaces Steady chant's 25
		{"high-sorcerer", 45, LanceTwin, 50},
		{"high-sorcerer", 50, LanceCost, 12},
		{"high-sorcerer", 55, ManaPct, 20},
		{"high-sorcerer", 59, LanceFree, 0},
		{"high-sorcerer", 60, LanceFree, 1},
		{"high-sorcerer", 60, Lance, 1}, // keeps the advanced class's rank
	} {
		assert.Equal(t, c.want, EffectsFor(c.class, c.level, nil).Int(c.key), "%s %s at %d", c.class, c.key, c.level)
	}
}

func TestOnlyTheSorcererPathTeachesTheLance(t *testing.T) {
	for _, tc := range []struct {
		class string
		level int
		want  bool
	}{
		{"sorcerer", 9, false},
		{"sorcerer", 10, true},
		{"high-sorcerer", 30, true},
		{"arcanist", 60, false},
		{"archmage", 60, false},
		{"warlock", 60, false},
	} {
		assert.Equal(t, tc.want, slices.Contains(SpellsAt(tc.class, tc.level), "arcanelance"), "%s at %d", tc.class, tc.level)
	}
	assert.True(t, SpellLocked("sorcerer", 9, "arcanelance"))
	assert.False(t, SpellLocked("sorcerer", 10, "arcanelance"))
	assert.False(t, SpellLocked("arcanist", 10, "arcanelance"), "a spell no rank teaches stays unlocked-by-class")
}

func TestSorcererIsAnOpenRouteWithAnOpenElite(t *testing.T) {
	opts := Options("wizard", "", 10, 0)
	var found bool
	for _, o := range opts {
		if o.Class.ID == "sorcerer" {
			found = true
			assert.True(t, o.Eligible, "any alignment")
		}
	}
	assert.True(t, found, "class paths lists the Sorcerer")
	elite, ok := Elite("sorcerer")
	require.True(t, ok)
	assert.Equal(t, "high-sorcerer", elite.ID)
	assert.False(t, elite.Planned)
	assert.Equal(t, "ready", PromotionState("wizard", "sorcerer", 30, -100))
	lines := RankLines("wizard", "high-sorcerer", 29, 60)
	require.Len(t, lines, 7, "every elite rank, once")
	assert.Contains(t, lines[6], "Instant Lance")
}
