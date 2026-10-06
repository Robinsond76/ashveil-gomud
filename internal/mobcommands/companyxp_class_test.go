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
