package users

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/strategy"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 35b: the level-up report names each scaling spell and ability
// that grew, and a spell first owned at this level by its size alone.
func TestPowerLines(t *testing.T) {
	before := []powerEntry{{"Magic Missile", "8-13"}, {"Minor Heal", "10-16"}, {"Opening Strike", "+2 damage"}}
	after := []powerEntry{{"Magic Missile", "9-14"}, {"Minor Heal", "10-16"}, {"Opening Strike", "+3 damage"}, {"Shower of Sparks", "5-8"}}
	assert.Equal(t, []string{"Magic Missile 8-13 -> 9-14", "Opening Strike +2 damage -> +3 damage", "Shower of Sparks 5-8"}, powerLines(before, after))
	assert.Empty(t, powerLines(after, after))
}

// Phase 46: a halberdier's level-up report names Brace (level 3), Hook (6),
// the Sweep that reaches the whole row (8) and Hook's better chance (20) as
// the shared "New rank" lines, and the power report no longer repeats them.
func TestHalberdierLevelsAreNewRankLines(t *testing.T) {
	rank := func(from, to int) []string { return classes.RankLines("halberdier", "", from, to) }
	assert.Empty(t, rank(1, 2))
	require.Len(t, rank(2, 3), 1)
	assert.Contains(t, rank(2, 3)[0], "New rank: Brace, ")
	assert.Contains(t, rank(5, 6)[0], "New rank: Hook, ")
	assert.Contains(t, rank(7, 8)[0], "New rank: Wide sweep, ")
	assert.Contains(t, rank(19, 20)[0], "New rank: Deep hook, ")

	c := characters.New()
	c.HPArchetype = "halberdier"
	for _, level := range []int{1, 3, 6, 8, 20} {
		c.Level = level
		assert.Empty(t, abilityPower(c, strategy.CompanionAbilities("halberdier")), "level %d", level)
	}
}
