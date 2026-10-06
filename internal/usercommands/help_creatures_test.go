package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 38e: the creature pages render through help, are indexed under
// character, answer to their aliases, and the hubs point to them.
func TestCreatureHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var listed []string
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "character" && !topic.AdminOnly {
			listed = append(listed, topic.Command)
		}
	}
	for topic, wants := range map[string][]string{
		"creatures":   {"Hound", "Stone Golem", "carries nothing", "bound", "help repair"},
		"hound":       {"Run down", "Worry", "Fleet", "Savage pursuit", "wounded", "90 gold", "eats, drinks"},
		"stone-golem": {"Stone body", "Anchor", "Granite", "Bedrock", "needs no food", "180 gold", "nearest", "help repair"},
		"repair":      {"company repair", "half the golem's maximum health", "stone mortar", "fight"},
	} {
		assert.Contains(t, listed, topic, "help index lists %s", topic)
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		plain := tagPattern.ReplaceAllString(text, "")
		assert.Contains(t, plain, "Help for", topic)
		for _, want := range wants {
			assert.Contains(t, plain, want, "help %s mentions %s", topic, want)
		}
	}
	for alias, topic := range map[string]string{
		"constructs": "creatures", "brindle": "hound", "war hound": "hound",
		"cairn": "stone-golem", "stone golem": "stone-golem", "mortar": "repair",
	} {
		want, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help %s", alias, topic)
	}
	for topic, want := range map[string]string{
		"classes": "help creatures", "company": "company repair", "equipment": "creature",
	} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), want, "help %s points to the creature pages", topic)
	}
}
