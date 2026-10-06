package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 30c1: help tactics renders line for line, answers to its aliases,
// sits under combat, and the pages it changed point to it.
func TestTacticsHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	listed := false
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "tactics" {
			listed = topic.Category == "combat"
		}
	}
	assert.True(t, listed, "help index lists tactics under combat")

	text, err := GetHelpContents("tactics")
	require.NoError(t, err)
	plainText := tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{
		"Help for tactics",
		"none", "leader", "casters", "healers", "nearest", "weakest", "strongest", "wounded",
		"company tactics focus [rule]", "company tactics focus default",
		"company tactics healing [percent]", "company tactics default",
		"You call the company onto the bandit captain.",
		"Your company is still\nturning; try again next round.",
		"10 to 90 percent", "It is 50, half health",
		"wolves go for the\nmost badly hurt",
		"tactics is short for company tactics",
		"From level 5, a healer on the other side changes the default.",
		"Your company marks the goblin shaman as a healer and goes for it first.",
	} {
		assert.Contains(t, plainText, want)
	}
	for _, alias := range []string{"focus", "company-tactics", "personalities", "personality"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, text, got, "help %s is help tactics", alias)
	}

	for topic, want := range map[string]string{
		"combat":    "help tactics",
		"strategy":  "A company focus (help tactics) overrides every target rule",
		"targeting": "Enemy personalities",
		"webclient": "the Focus buttons call a new focus",
		"company":   "company tactics",
		"attack":    "but a new company focus",
	} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), want, topic)
	}
}
