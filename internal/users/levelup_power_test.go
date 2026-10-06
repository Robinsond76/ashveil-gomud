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

// Phase 39f review: a gryphon rider's level-up report names Dive's damage
// when Power dive raises it (level 8).
func TestPowerSnapshotNamesTheGryphonRidersPowerDive(t *testing.T) {
	c := characters.New()
	c.HPArchetype = "gryphon-rider"
	at := func(level int) []powerEntry {
		c.Level = level
		return abilityPower(c, strategy.CompanionAbilities("gryphon-rider"))
	}
	assert.Equal(t, []powerEntry{{"Dive", "100% of a blow"}}, at(1))
	assert.Empty(t, powerLines(at(2), at(3)))
	assert.Equal(t, []string{"Dive 100% of a blow -> 125% of a blow"}, powerLines(at(7), at(8)))
}

// Phase 39h: an arbalist's level-up report names Piercing Bolt's damage when
// a route rank (Heavy stock) raises it.
func TestPowerSnapshotNamesTheArbalistsPiercingBolt(t *testing.T) {
	c := characters.New()
	c.HPArchetype = "arbalist"
	at := func(level int) []powerEntry {
		c.Level = level
		return abilityPower(c, strategy.CompanionAbilities("arbalist"))
	}
	assert.Equal(t, []powerEntry{{"Piercing Bolt", "140% of a shot"}}, at(1))
	assert.Empty(t, powerLines(at(7), at(8)), "Armor-breaker is a rank line, not a size")
	c.HPClass = "siegebreaker"
	assert.Equal(t, []string{"Piercing Bolt 140% of a shot -> 160% of a shot"}, powerLines(at(14), at(15)))
}
