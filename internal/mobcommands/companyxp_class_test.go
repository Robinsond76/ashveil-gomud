package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

func TestCompanionLevelLineNamesItsNextClassMilestone(t *testing.T) {
	before := characters.Character{Name: "Tamsin", Level: 4}
	after := characters.Character{Name: "Tamsin", Level: 5}
	assert.Contains(t, companionLevelLine(before, after), "Next: your class promotion at level 10.")
}

// 39b review: a companion's level-up names the ranks it just reached, not
// only the next one.
func TestCompanionLevelLineNamesTheRanksItReached(t *testing.T) {
	before := characters.Character{Name: "Kaede", Level: 2, HPArchetype: "samurai"}
	after := characters.Character{Name: "Kaede", Level: 8, HPArchetype: "samurai"}
	line := companionLevelLine(before, after)
	assert.Contains(t, line, "New rank: Focus, +3% critical chance")
	assert.Contains(t, line, "New rank: Zanshin, when it fells a foe")
	assert.NotContains(t, line, "Iaijutsu", "a rank already held is not repeated")
	assert.Contains(t, line, "Next: your class promotion at level 10.")
}
