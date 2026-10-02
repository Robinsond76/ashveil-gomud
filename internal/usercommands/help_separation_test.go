package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSeparationHelp: Phase 33h3's page renders, answers to its aliases,
// sits in the company category, and the pages it changes link it.
func TestSeparationHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	expected := map[string][]string{
		"separation": {"Your company comes with you.", "Not while your company is fighting.", "was not with you and is separated",
			"for two rounds", "off the map", "about a", "15 rounds", "logging out pauses it", "never walks into a fight",
			"never touches a fallen companion", "wherever you are once the", "always go with you", "company status"},
		"company":   {"help separation", "separated (and how"},
		"travel":    {"is separated", "help separation", "horses"},
		"death":     {"who were with you when you fell", "help separation"},
		"morale":    {"wherever you are", "help separation"},
		"mount":     {"never left", "help separation"},
		"cargo":     {"separated", "always goes with you"},
		"readiness": {"without recovering while away", "help separation"},
	}
	for topic, wants := range expected {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		plain := tagPattern.ReplaceAllString(text, "")
		for _, want := range wants {
			assert.Contains(t, plain, want, topic)
		}
	}
	travel, err := GetHelpContents("travel")
	require.NoError(t, err)
	assert.NotContains(t, tagPattern.ReplaceAllString(travel, ""), "exactly as if you'd\nwalked there", "arrival no longer walks the last exit")

	for _, alias := range []string{"separated", "left behind", "rejoin", "portal", "relocation"} {
		text, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "Help for separation", alias)
	}
	var category []string
	for _, info := range keywords.GetAllHelpTopicInfo() {
		if info.Category == "company" {
			category = append(category, info.Command)
		}
	}
	assert.Contains(t, category, "separation")
}
