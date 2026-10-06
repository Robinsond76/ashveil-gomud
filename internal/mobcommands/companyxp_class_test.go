package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

func TestCompanionLevelLineNamesItsNextClassMilestone(t *testing.T) {
	before := characters.Character{Name: "Tamsin", Level: 4}
	after := characters.Character{Name: "Tamsin", Level: 5}
	assert.Contains(t, companionLevelLine(before, after, "warrior", "#2"), "Next: your class promotion at level 10.")
}

// Phase 38c1: a companion's level-up report names the ranks it earned and an
// elite promotion that is ready or waiting on its gate.
func TestCompanionLevelLineNamesRanksAndElitePromotion(t *testing.T) {
	cases := []struct {
		name      string
		class     string
		from, to  int
		alignment int8
		want      []string
		not       []string
	}{
		{"ready", "mercenary", 29, 30, 0, []string{"Elite promotion ready: Mercenary -> Warlord.", "class promote #2 warlord"}, nil},
		{"waiting", "knight", 29, 30, 22, []string{"Paladin needs alignment +30 (theirs: +22).", "Tamsin keeps their Knight ranks"}, []string{"promotion ready"}},
		{"rank up", "warlord", 34, 35, 0, []string{"New rank: Battle Cry, at the start of each battle"}, nil},
		{"below 30", "knight", 28, 29, 90, nil, []string{"Elite promotion", "needs alignment"}},
	}
	for _, tc := range cases {
		before := characters.Character{Name: "Tamsin", Level: tc.from}
		after := characters.Character{Name: "Tamsin", Level: tc.to, Alignment: tc.alignment}
		before.SetClassState(tc.class, nil)
		after.SetClassState(tc.class, nil)
		line := companionLevelLine(before, after, "warrior", "#2")
		for _, w := range tc.want {
			assert.Contains(t, line, w, tc.name)
		}
		for _, n := range tc.not {
			assert.NotContains(t, line, n, tc.name)
		}
	}
}

// 39b review: a companion's level-up names the ranks it just reached, not
// only the next one.
func TestCompanionLevelLineNamesTheRanksItReached(t *testing.T) {
	before := characters.Character{Name: "Kaede", Level: 2, HPArchetype: "samurai"}
	after := characters.Character{Name: "Kaede", Level: 8, HPArchetype: "samurai"}
	line := companionLevelLine(before, after, "samurai", "#2")
	assert.Contains(t, line, "New rank: Focus, +3% critical chance")
	assert.Contains(t, line, "New rank: Zanshin, when it fells a foe")
	assert.NotContains(t, line, "Iaijutsu", "a rank already held is not repeated")
	assert.Contains(t, line, "Next: your class promotion at level 10.")
}
