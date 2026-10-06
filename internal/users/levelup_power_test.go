package users

import (
	"testing"

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
