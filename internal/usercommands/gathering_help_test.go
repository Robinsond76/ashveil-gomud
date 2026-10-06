package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 40a2: `help gathering` renders, is indexed under the road hub and
// answers to its aliases, and the pages it touches point at it.
func TestGatheringHelpRendersAndIsIndexed(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var listed bool
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "gathering" {
			listed = true
			assert.Equal(t, "road", topic.Category)
		}
	}
	assert.True(t, listed, "help index lists gathering")

	want, err := GetHelpContents("gathering")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(want, "")
	for _, phrase := range []string{"gather herbs", "gather firewood", "fish", "hunt", "picked clean", "bitter weed", "camp fire", "snares", "20 minutes", "Doing line", "Room Info window"} {
		assert.Contains(t, plain, phrase)
	}
	for _, alias := range []string{"gather", "herbs", "firewood", "fish", "fishing", "hunt", "hunting", "snares"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, alias)
	}
	for _, topic := range []string{"resources", "camp", "forage", "cooking", "survival", "webclient"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, text, "help gathering", "help %s points at help gathering", topic)
	}
	cooking, err := GetHelpContents("cooking")
	require.NoError(t, err)
	assert.Contains(t, cooking, "Grilled fish")
}
