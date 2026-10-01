package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLogisticsHelpTopics: Phase 32f's pages render, the new company pages
// are in the help index, answer to their multi-word aliases, and the
// changed pages carry the new rules.
func TestLogisticsHelpTopics(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var company []string
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "company" && !topic.AdminOnly {
			company = append(company, topic.Command)
		}
	}
	assert.Contains(t, company, "company-inventory")
	assert.Contains(t, company, "company-meal")

	want := map[string][]string{
		"cargo":             {"pet's", "20 kg each", "satchel adds 5 kg", "pack horse carries 40 kg", "keeps its uses"},
		"mount":             {"one riding horse and one pack horse", "120 gold", "mount saddle [horse] [saddle]", "Dunmar West Gate"},
		"encumbrance":       {"one carrying limit", "never blocked"},
		"inventory":         {"company inventory", "no limit on how many"},
		"get":               {"can't pick anything more up"},
		"buy":               {"keeps it and your gold"},
		"give":              {"can't carry any more", "own companion"},
		"market":            {"refused before any gold"},
		"eat":               {"company eat"},
		"drink":             {"company drink"},
		"set-prompt":        {"Company capacity in kg"},
		"company":           {"company inventory", "company meal"},
		"company-inventory": {"waterskin (3 of 5)", "company inv"},
		"company-meal":      {"company cargo", "smallest portion", "potion"},
	}
	for topic, lines := range want {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		plain := tagPattern.ReplaceAllString(text, "")
		for _, line := range lines {
			assert.Contains(t, plain, line, "help %s", topic)
		}
	}

	aliases := map[string]string{
		"company inventory": "company-inventory", "company inv": "company-inventory",
		"company meal": "company-meal", "company eat": "company-meal", "company drink": "company-meal",
		"horse": "mount", "saddle": "mount", "satchel": "cargo", "capacity": "cargo",
	}
	for alias, topic := range aliases {
		want, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help %s", alias, topic)
	}
}
