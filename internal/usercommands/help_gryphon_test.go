package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39f: the Gryphon Rider pages render through help, are indexed under
// character, answer to their aliases and the pages the class touched name it.
func TestGryphonRiderHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var listed []string
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "character" && !topic.AdminOnly {
			listed = append(listed, topic.Command)
		}
	}
	for topic, wants := range map[string][]string{
		"gryphon-rider":        {"Dive", "Talons", "Power dive", "healers", "narrow ground", "help gryphon-rider-routes", "-10 Evasion"},
		"gryphon-rider-routes": {"Gryphon Knight", "Lance charge", "Skyscout", "Eagle eye", "Hawk's mark", "Wyvern Rider", "Venom stoop", "Gryphon Lord", "asks anything of"},
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
		"gryphon rider": "gryphon-rider", "dive": "gryphon-rider", "talons": "gryphon-rider", "skirmisher": "gryphon-rider",
		"gryphon knight": "gryphon-rider-routes", "skyscout": "gryphon-rider-routes", "wyvern rider": "gryphon-rider-routes",
		"falcon marshal": "gryphon-rider-routes",
	} {
		want, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help %s", alias, topic)
	}
	for topic, want := range map[string]string{
		"archetype": "Gryphon Rider", "health": "Gryphon Rider", "growth": "Gryphon Rider", "armor": "Gryphon Rider",
		"shields": "Gryphon Rider", "evasion": "Gryphon Rider", "classes": "gryphon-rider-routes", "promotion": "Gryphon Rider",
		"talents": "Wing Drill", "combat": "help gryphon-rider", "strategy": "dive", "abilities": "Dive",
		"progression": "gryphon rider 5", "specialists": "Keen Eye",
	} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), want, "help %s mentions %s", topic, want)
	}
}
