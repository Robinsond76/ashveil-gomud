package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 30d1: help interrupts renders line for line, answers to its
// aliases, sits under combat, and the pages it changed point to it.
func TestInterruptsHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	listed := false
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "interrupts" {
			listed = topic.Category == "combat"
		}
	}
	assert.True(t, listed, "help index lists interrupts under combat")

	text, err := GetHelpContents("interrupts")
	require.NoError(t, err)
	plainText := tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{
		"Help for interrupts",
		"Any weapon blow that draws blood\nbreaks it",
		"The goblin hexer's chant breaks off under the blow. (Withering Hex interrupted)",
		"(Minor Heal interrupted, 1 mana back)",
		"half its mana comes back",
		"from the first word",
		"back row", "casters",
		"(shield bash, 3 damage, stunned)",
		"50%", "1 to 4 damage", "one counter a round",
	} {
		assert.Contains(t, plainText, want)
	}
	for _, alias := range []string{"interrupt", "interrupted", "chant", "chants", "concentration", "shield-bash", "shieldbash", "counter", "counters", "hexer"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, text, got, "help %s is help interrupts", alias)
	}

	for _, topic := range []string{"combat", "cast", "strategy", "tactics", "statuses", "guardian", "battle-summary", "formation"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "help interrupts", topic)
	}
}
