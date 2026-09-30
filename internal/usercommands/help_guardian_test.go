package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 30c2: help guardian renders line for line, answers to its aliases,
// sits under combat, and the pages it changed point to it.
func TestGuardianHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	listed := false
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "guardian" {
			listed = topic.Category == "combat"
		}
	}
	assert.True(t, listed, "help index lists guardian under combat")

	text, err := GetHelpContents("guardian")
	require.NoError(t, err)
	plainText := tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{
		"Help for guardian",
		"Tamsin Reed steps in front of you. (guard, 1 left)",
		"2 guards", "2 combat rounds", "knocked down (2 rounds)", "stunned (1 round)",
		"most hurt", "column, or the next one",
		"strategy [who] guard [other]", "strategy tamsin guard me",
		"Only weapon blows are guarded",
	} {
		assert.Contains(t, plainText, want)
	}
	for _, alias := range []string{"guardians", "guard", "guards", "ward", "wards"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, text, got, "help %s is help guardian", alias)
	}

	for topic, want := range map[string]string{
		"combat":         "help guardian",
		"strategy":       "guardian  fights, and steps in front of its ward",
		"tactics":        "a guardian decides who gets\nstruck",
		"formation":      "Out of reach: Tamsin Reed can't step in for you from there.",
		"webclient":      "guards you, 1 guard left",
		"battle-summary": "Guards (when there were any)",
	} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), want, topic)
	}
}
