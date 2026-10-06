package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39h: the Arbalist pages render through help, are indexed under
// character, answer to their aliases and the pages the class touched name it.
func TestArbalistHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var listed []string
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "character" && !topic.AdminOnly {
			listed = append(listed, topic.Command)
		}
	}
	for topic, wants := range map[string][]string{
		"arbalist":        {"Piercing Bolt", "Steady Aim", "Crippling Bolt", "Armor-breaker", "armored", "winds the crossbow", "help arbalist-routes", "140%"},
		"arbalist-routes": {"Siegebreaker", "Sundering bolts", "Sharpshooter", "Practiced loader", "Warden of the Wall", "Pavise", "Siege Master", "asks anything of"},
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
		"crossbowman": "arbalist", "piercing bolt": "arbalist", "arbalists": "arbalist", "armor-breaker": "arbalist",
		"siegebreaker": "arbalist-routes", "sharpshooter": "arbalist-routes", "warden of the wall": "arbalist-routes",
		"deadeye": "deadeye", "bastion": "bastion",
	} {
		want, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help %s", alias, topic)
	}
	for topic, want := range map[string]string{
		"archetype": "Arbalist", "health": "Arbalist", "growth": "Arbalist", "armor": "Arbalist",
		"shields": "Arbalist", "evasion": "Arbalist", "classes": "arbalist-routes", "promotion": "Arbalist",
		"talents": "Heavy Bolts", "combat": "help arbalist", "strategy": "armored", "abilities": "Piercing Bolt",
		"progression": "arbalist 4", "equipmenttiers": "help arbalist",
	} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), want, "help %s mentions %s", topic, want)
	}
}
