package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompanyTrainHelp: Phase 35c's page renders, answers to its aliases,
// sits in the company category, and the pages it changes say so.
func TestCompanyTrainHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	expected := map[string][]string{
		"company-train": {"Help for company train", "Class skills stay", "a level-6 companion has earned 6",
			"Ranks 1 and\n2 together cost 3, so that level-6 companion has 3 left", "never gives any back", "owes the points",
			"Ranks 1 and 2: at your own camp", "Ranks 3 and 4: only from a trainer", "Waymark",
			"company train [member] [skill] [rank] confirm", "trains nothing more", "company inspect [member]", "help cooking"},
		"company":     {"help company train", "company train [member] [skill]", "already trained in an optional skill", "15% more per\nrank"},
		"skills":      {"Class skills are automatic", "company train", "help company train"},
		"cooking":     {"the best cook with you cooks", "you win a tie", "lowest\nnumber", "Brannoc cooks", "uses\nyour own Cooking only", "company train"},
		"growth":      {"training points", "help company train"},
		"specialists": {"never trained", "help company train"},
	}
	for topic, wants := range expected {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		plain := tagPattern.ReplaceAllString(text, "")
		for _, want := range wants {
			assert.Contains(t, plain, want, topic)
		}
	}
	for _, alias := range []string{"company train", "train companion", "companion training", "optional skills"} {
		text, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "Help for company train", alias)
	}
	var company []string
	for _, info := range keywords.GetAllHelpTopicInfo() {
		if info.Category == "company" {
			company = append(company, info.Command)
		}
	}
	assert.Contains(t, company, "company-train")
}
