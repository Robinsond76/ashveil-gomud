package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGrowthAndContractsHelp: Phase 33h1's pages render, answer to their
// aliases, sit in the company category, and the pages they touch link them.
func TestGrowthAndContractsHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	expected := map[string][]string{
		"growth":     {"company growth [member] [stat]", "balanced", "strength 4, vitality 3, speed 2, perception 1", "adds 2", "never an extra one", "help contracts"},
		"contracts":  {"Rodric's Rats", "15000", "standing in your room", "experience scale", "help growth"},
		"company":    {"help growth", "help contracts", "company growth"},
		"experience": {"help growth", "help contracts"},
		"quests":     {"help contracts"},
	}
	for topic, wants := range expected {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		plain := tagPattern.ReplaceAllString(text, "")
		for _, want := range wants {
			assert.Contains(t, plain, want, topic)
		}
	}
	for alias, topic := range map[string]string{"specialize": "growth", "company growth": "growth", "contract": "contracts", "company contracts": "contracts"} {
		text, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "Help for "+topic, alias)
	}
	var company []string
	for _, info := range keywords.GetAllHelpTopicInfo() {
		if info.Category == "company" {
			company = append(company, info.Command)
		}
	}
	assert.Contains(t, company, "growth")
	assert.Contains(t, company, "contracts")
}
