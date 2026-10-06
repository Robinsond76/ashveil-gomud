package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 43a: `help camp supplies` renders, is indexed beside camp gear and
// answers to its aliases, and the pages the supplies touch point at it.
func TestCampSuppliesHelpRendersAndIsIndexed(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var listed bool
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "campsupplies" {
			listed = true
			assert.Equal(t, "road", topic.Category)
		}
	}
	assert.True(t, listed, "help index lists camp supplies")

	want, err := GetHelpContents("campsupplies")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(want, "")
	for _, phrase := range []string{"Fortifying broth", "Warming draught", "Cooling salve", "Watch incense", "about 5%", "15",
		"camp prepare broth", "camp prepare incense", "camp prepare status", "camp prepare clear", "camp supplies",
		"one personal benefit", "Nothing is refunded", "never above 90%", "dearer"} {
		assert.Contains(t, plain, phrase)
	}
	for _, alias := range []string{"camp supplies", "camp-supplies", "camp prepare", "broth", "fortifying broth", "warming draught", "cooling salve", "incense", "watch incense"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, alias)
	}
	for _, topic := range []string{"camp", "campwatch", "temperature"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, text, "camp supplies", "help %s points at help camp supplies", topic)
	}
	resources, err := GetHelpContents("resources")
	require.NoError(t, err)
	assert.Contains(t, tagPattern.ReplaceAllString(resources, ""), "empty")
	assert.NotContains(t, resources, "used up with its last glug")
}
