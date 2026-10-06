package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39g: the Alchemist pages render through help, are indexed under
// character, answer to their aliases and the pages the class touched name it.
func TestAlchemistHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var listed []string
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "character" && !topic.AdminOnly {
			listed = append(listed, topic.Command)
		}
	}
	for topic, wants := range map[string][]string{
		"alchemist":        {"Healing Draught", "Antidote", "Fire Flask", "Bracing Tonic", "satchel", "help brew", "help alchemist-routes", "lands the moment"},
		"alchemist-routes": {"Apothecary", "Potent draughts", "Bombardier", "Pitch and tar", "Mutagenist", "Pure mutagen", "Panacean", "any alignment"},
		"brew":             {"one reagent makes one flask", "Reagents carried", "camp"},
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
		"flask": "alchemist", "flasks": "alchemist", "healing-draught": "alchemist", "reagents": "alchemist",
		"bombardier": "alchemist-routes", "apothecary": "alchemist-routes", "mutagenist": "alchemist-routes",
		"brewing-flasks": "brew",
	} {
		want, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help %s", alias, topic)
	}
	for topic, want := range map[string]string{
		"archetype": "Alchemist", "health": "Alchemist", "growth": "Alchemist", "armor": "Alchemist",
		"shields": "Alchemist", "evasion": "Alchemist", "classes": "Alchemist", "promotion": "Alchemist",
		"talents": "Deep Satchel", "combat": "help alchemist", "strategy": "Fire Flask", "camp": "help brew",
		"progression": "alchemist 3", "specialists": "alchemist",
	} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), want, "help %s mentions %s", topic, want)
	}
}
