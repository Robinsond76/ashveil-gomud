package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 40d: `help walkto` renders, is indexed under the road hub and
// answers to its aliases, and the pages it touches point at it.
func TestWalktoHelpRendersAndIsIndexed(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var listed bool
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "walkto" {
			listed = true
			assert.Equal(t, "road", topic.Category)
		}
	}
	assert.True(t, listed, "help index lists walkto")

	want, err := GetHelpContents("walkto")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(want, "")
	for _, phrase := range []string{"walkto [place]", "walkto stop", "Walk to [room]", "1.5 seconds", "at most 60 steps",
		"visited", "secret exit", "locked", "strain", "random encounter", "clock", "not saved"} {
		assert.Contains(t, plain, phrase)
	}
	assert.NotContains(t, plain, "<place>", "placeholders are written [place]")
	for _, alias := range []string{"autowalk", "click-to-walk", "click to walk", "walk to", "tile-ready", "alderbrook"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, alias)
	}
	for _, topic := range []string{"worldmap", "webclient", "travel"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, text, "help walkto", "help %s points at help walkto", topic)
	}
	worldmap, err := GetHelpContents("worldmap")
	require.NoError(t, err)
	assert.Contains(t, tagPattern.ReplaceAllString(worldmap, ""), "Day/night")
	assert.Contains(t, tagPattern.ReplaceAllString(worldmap, ""), "icon")

	// The `walkto` and `autowalk` commands themselves are aliased.
	assert.Equal(t, "walkto", keywords.TryCommandAlias("autowalk"))
}
