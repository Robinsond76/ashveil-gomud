package users

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/strategy"

	"github.com/stretchr/testify/assert"
)

// Phase 35b: the level-up report names each scaling spell and ability
// that grew, and a spell first owned at this level by its size alone.
func TestPowerLines(t *testing.T) {
	before := []powerEntry{{"Magic Missile", "8-13"}, {"Minor Heal", "10-16"}, {"Opening Strike", "+2 damage"}}
	after := []powerEntry{{"Magic Missile", "9-14"}, {"Minor Heal", "10-16"}, {"Opening Strike", "+3 damage"}, {"Shower of Sparks", "5-8"}}
	assert.Equal(t, []string{"Magic Missile 8-13 -> 9-14", "Opening Strike +2 damage -> +3 damage", "Shower of Sparks 5-8"}, powerLines(before, after))
	assert.Empty(t, powerLines(after, after))
}

// Phase 39a review: a halberdier's level-up report names Brace when it
// comes (level 3), Hook (6), the Sweep that reaches the whole row (8), and
// Hook's better chance (20).
func TestPowerSnapshotNamesTheHalberdiersLevels(t *testing.T) {
	c := characters.New()
	c.HPArchetype = "halberdier"
	at := func(level int) []powerEntry {
		c.Level = level
		return abilityPower(c, strategy.CompanionAbilities("halberdier"))
	}
	assert.Empty(t, powerLines(at(1), at(2)))
	assert.Equal(t, []string{"Brace held blow at 125%"}, powerLines(at(2), at(3)))
	assert.Equal(t, []string{"Hook 20% to trip a leaper"}, powerLines(at(5), at(6)))
	assert.Equal(t, []string{"Sweep one foe beside -> the whole row"}, powerLines(at(7), at(8)))
	assert.Equal(t, []string{"Hook 20% to trip a leaper -> 40% to trip a leaper"}, powerLines(at(19), at(20)))
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
