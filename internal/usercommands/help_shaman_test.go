package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39c: the Shaman pages render through help, are indexed under
// character, answer to their aliases and the pages the class touched name
// it.
func TestShamanHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var listed []string
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "character" && !topic.AdminOnly {
			listed = append(listed, topic.Command)
		}
	}
	for topic, wants := range map[string][]string{
		"shaman":        {"Call Fog", "Chill Wind", "Rain", "Lightning", "Gust", "Long Weather", "help shaman-routes", "help battlescreen"},
		"shaman-routes": {"Stormcaller", "Chain lightning", "Mistweaver", "Veil of mist", "Earthspeaker", "Stoneskin", "Mountain Speaker"},
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
		"weather-caller": "shaman", "callfog": "shaman", "chillwind": "shaman", "lightning": "shaman", "windchilled": "shaman",
		"stormcaller": "shaman-routes", "mistweaver": "shaman-routes", "earthspeaker": "shaman-routes", "stoneskin": "shaman-routes",
	} {
		want, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help %s", alias, topic)
	}
	for topic, want := range map[string]string{
		"archetype": "Shaman", "classes": "shaman-routes", "promotion": "Shaman", "talents": "Long Weather",
		"combat": "help shaman", "strategy": "help shaman", "statuses": "Fogbound", "progression": "a shaman the same",
		"specialists": "shamans", "weather": "shaman", "mana": "shamans", "battlescreen": "help shaman",
	} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), want, "help %s mentions %s", topic, want)
	}
}
