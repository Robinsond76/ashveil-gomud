package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The skill schedule (2026-10-12): help skill-levels renders, is indexed
// under character, answers to its aliases, and names the level each
// signature ability comes at as the code does.
func TestSkillLevelsHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	indexed := false
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "skill-levels" {
			indexed = topic.Category == "character"
		}
	}
	assert.True(t, indexed, "help index lists skill-levels under character")

	text, err := GetHelpContents("skill-levels")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	assert.Contains(t, plain, "Help for skill-levels")
	for _, alias := range []string{"unlocks", "skill-schedule", "learning"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, text, got, "help %s is help skill-levels", alias)
	}

	// Each signature that comes at level 2 is named in its class's row.
	rows := map[string]string{"warrior": "Warrior", "rogue": "Rogue", "ranger": "Ranger", "halberdier": "Halberdier", "gryphon-rider": "Gryphon Rider", "arbalist": "Arbalist", "beasttamer": "Beast Tamer"}
	for _, spec := range strategy.Abilities {
		row, ok := rows[spec.Archetype]
		if !ok || spec.MinLevel != 2 {
			continue
		}
		line := ""
		for _, l := range strings.Split(plain, "\n") {
			if strings.HasPrefix(strings.TrimSpace(l), row+" ") {
				line = l
			}
		}
		require.NotEmpty(t, line, "a row for %s", row)
		assert.Contains(t, line, spec.Name, "%s row names %s", row, spec.Name)
	}
}
