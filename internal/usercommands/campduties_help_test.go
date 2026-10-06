package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 51: `help camp duties` renders, is indexed beside camp gear and
// answers to its aliases, and the pages the duties touch point at it.
func TestCampDutiesHelpRendersAndIsIndexed(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var listed bool
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "campduties" {
			listed = true
			assert.Equal(t, "road", topic.Category)
		}
	}
	assert.True(t, listed, "help index lists camp duties")

	want, err := GetHelpContents("campduties")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(want, "")
	for _, phrase := range []string{"camp duties", "sleep", "watch", "tend", "forage", "cook", "brew",
		"two watchers spot raiders more often than", "no better than", "Ready", "Rested", "never to Rested",
		"camp duties clear", "forage cooldown", "Alchemists only", "nothing changes"} {
		assert.Contains(t, plain, phrase)
	}
	for _, alias := range []string{"camp duties", "camp-duties", "duties", "rest duties", "watch duty"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, alias)
	}
	for _, topic := range []string{"camp", "campwatch", "vigil", "forage", "cooking"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "help camp duties", "help %s points at help camp duties", topic)
	}
}
