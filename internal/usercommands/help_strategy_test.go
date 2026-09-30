package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 32d: help strategy renders line for line, answers to its aliases,
// sits under combat, and the pages it changed say what a battle allows.
func TestStrategyHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	listed := false
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "strategy" {
			listed = topic.Category == "combat"
		}
	}
	assert.True(t, listed, "help index lists strategy under combat")

	text, err := GetHelpContents("strategy")
	require.NoError(t, err)
	plainText := tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{
		"Help for strategy", "fighter", "healer", "caster",
		"weakest", "strongest", "wounded", "nearest", "furthest", "leader", "assist", "defend",
		"below the healing threshold", "Magic Missile", "Minor Heal", "casters",
		"help tactics",
		"strategy [who] target [rule]", "strategy [who] default",
		"can be read but not\nchanged",
	} {
		assert.Contains(t, plainText, want)
	}
	for _, alias := range []string{"strategies", "gambits", "roles"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, text, got, "help %s is help strategy", alias)
	}

	for topic, want := range map[string]string{
		"combat":    "help strategy",
		"targeting": "Only flee takes you out of a battle",
		"attack":    "Only flee takes you out.",
		"flee":      "the only way out",
		"break":     "is refused",
		"cast":      "by their strategy",
		"mana":      "each of your companions",
		"formation": "can't be changed during a battle",
		"archetype": "Wizards start with Magic Missile",
	} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), want, topic)
	}
}
