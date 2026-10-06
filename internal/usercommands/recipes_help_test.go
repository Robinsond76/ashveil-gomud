package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 56: `help recipes` renders, is indexed beside cooking, answers to
// its aliases (it took recipe and recipes from cooking), and the pages the
// feature touches point at it.
func TestRecipesHelpRendersAndIsIndexed(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var listed bool
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "recipes" {
			listed = true
			assert.Equal(t, "road", topic.Category)
		}
	}
	assert.True(t, listed, "help index lists recipes")

	want, err := GetHelpContents("recipes")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(want, "")
	for _, phrase := range []string{"cook [ingredient]", "makeshift meal", "at most 4", "learn it", "recipe book",
		"camp cook", "use hearth", "common knowledge", "Recipe pages", "never buy a page", "Camp tab"} {
		assert.Contains(t, plain, phrase)
	}
	for _, alias := range []string{"recipe", "recipe book", "recipe pages", "experiment"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, alias)
	}
	for _, topic := range []string{"cooking", "camp", "campduties"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "recipes", "help %s points at the recipe book", topic)
	}
	cooking, err := GetHelpContents("cooking")
	require.NoError(t, err)
	assert.Contains(t, tagPattern.ReplaceAllString(cooking, ""), "help recipes")
}
