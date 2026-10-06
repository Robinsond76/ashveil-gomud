package usercommands

import (
	"regexp"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAshveilHelpTopics: every Ashveil help page renders, is listed in the
// help index, and answers to its aliases; the GoMud pages Ashveil updated
// still render. No page leaks template markup or a raw placeholder.
func TestAshveilHelpTopics(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	listed := map[string]bool{}
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		listed[topic.Command] = true
	}
	pages := []string{
		"adventure", "company", "standing", "archetype", "autoskill", "trap",
		"travel", "survival", "strain", "weather", "temperature", "cargo", "mount",
		"camp", "inn", "cooking", "market", "rumors",
		"alignment", "status", "conditions", "inventory", "experience",
		"eat", "drink", "encumbrance", "death",
	}
	leak := regexp.MustCompile(`\{\{|\}\}|~[a-z]`)
	for _, topic := range pages {
		assert.True(t, listed[topic], "help index lists %s", topic)
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		plain := tagPattern.ReplaceAllString(text, "")
		assert.Contains(t, plain, "Help for", topic)
		assert.Greater(t, len(plain), 300, "%s has a real page", topic)
		assert.False(t, leak.MatchString(plain), "%s leaks markup", topic)
	}

	aliases := map[string]string{
		"ashveil": "adventure", "recruit": "company", "reputation": "standing",
		"traps": "trap", "hunger": "survival", "fatigue": "survival",
		"walking": "strain", "cold": "temperature", "load": "cargo", "horse": "mount",
		"journey": "travel", "camping": "camp", "hearth": "cooking", "cook": "cooking",
		"markets": "market", "rumours": "rumors", "gossip": "rumors",
	}
	for alias, topic := range aliases {
		want, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help %s", alias, topic)
	}

	// The overview names every page.
	hub, err := GetHelpContents("adventure")
	require.NoError(t, err)
	for _, topic := range pages[1:] {
		if topic == "eat" || topic == "drink" || topic == "encumbrance" || topic == "inventory" {
			continue
		}
		assert.Contains(t, hub, "help "+topic, "the overview links %s", topic)
	}
}
