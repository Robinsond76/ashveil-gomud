package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCombatHelpTopics: every topic in the shipped help index's combat
// category renders, and the Ashveil combat pages answer to their aliases
// (help reach, help whetstone, ...).
func TestCombatHelpTopics(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var combat []string
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "combat" && !topic.AdminOnly {
			combat = append(combat, topic.Command)
		}
	}
	for _, want := range []string{"combat", "formation", "targeting", "chemistry", "sharpen", "light", "battle-summary", "resurrect"} {
		assert.Contains(t, combat, want, "help index lists %s under combat", want)
	}
	for _, topic := range combat {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "Help for", topic)
	}

	aliases := map[string]string{
		"battle": "combat", "fighting": "combat",
		"reach": "formation", "interception": "formation",
		"target": "targeting", "whetstone": "sharpen", "darkness": "light",
		"battlesummary": "battle-summary", "resurrection": "resurrect",
	}
	for alias, topic := range aliases {
		want, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help %s", alias, topic)
	}

	// The death page reflects Ashveil's death rules (Phase 25a).
	text, err := GetHelpContents("death")
	require.NoError(t, err)
	assert.Contains(t, text, "one level")
	assert.Contains(t, text, "church")
}
