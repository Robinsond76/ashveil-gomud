package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReadinessHelp: Phase 33h2's page renders, answers to its aliases,
// sits in the company category, and the pages it changes link it and no
// longer promise a free refill.
func TestReadinessHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	expected := map[string][]string{
		"readiness": {"still on 8 when you next log in", "last", "automatic save", "keeps the health and mana it had", "never comes back below 1", "starts full once",
			"Nobody recovers while you're logged out", "full health and mana", "doesn't restore health or mana itself", "half its", "as hurt as when it ran"},
		"company":   {"Readiness", "help readiness", "keeps the health and mana it had"},
		"health":    {"help readiness", "inn restores you all to"},
		"inn":       {"full health and mana", "help readiness"},
		"camp":      {"doesn't restore health or mana itself", "help readiness"},
		"resurrect": {"half their health and half their mana", "help readiness"},
		"wounds":    {"at half health", "help readiness"},
		"heal":      {"help readiness"},
		"quit":      {"help readiness"},
	}
	for topic, wants := range expected {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		plain := tagPattern.ReplaceAllString(text, "")
		for _, want := range wants {
			assert.Contains(t, plain, want, topic)
		}
	}
	company, err := GetHelpContents("company")
	require.NoError(t, err)
	assert.NotContains(t, tagPattern.ReplaceAllString(company, ""), "the companion is healed", "a level no longer heals")
	wounds, err := GetHelpContents("wounds")
	require.NoError(t, err)
	assert.NotContains(t, tagPattern.ReplaceAllString(wounds, ""), "comes back whole")

	for _, alias := range []string{"recovery", "vitals", "companion health", "regen"} {
		text, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "Help for readiness", alias)
	}
	var category []string
	for _, info := range keywords.GetAllHelpTopicInfo() {
		if info.Category == "company" {
			category = append(category, info.Command)
		}
	}
	assert.Contains(t, category, "readiness")
}
