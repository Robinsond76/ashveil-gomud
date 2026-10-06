package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 40a3: `help camp gear` renders, is indexed beside gathering and
// answers to its aliases, and the pages the gear touches point at it.
func TestCampGearHelpRendersAndIsIndexed(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var listed bool
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "campgear" {
			listed = true
			assert.Equal(t, "road", topic.Category)
		}
	}
	assert.True(t, listed, "help index lists camp gear")

	want, err := GetHelpContents("campgear")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(want, "")
	for _, phrase := range []string{"Bedroll", "+25% Fatigue", "Oiled canvas tent", "Fire steel and tinder", "Iron cookpot",
		"Camp bells and trip lines", "Field surgeon's kit", "20% chance", "90%", "embers", "10 rests", "5 uses", "camp fire", "Thieves", "Never taken", "mean thieves never"} {
		assert.Contains(t, plain, phrase)
	}
	for _, alias := range []string{"camp gear", "camp-gear", "bedroll", "tent", "cookpot", "fire steel", "camp bells", "trip lines", "surgeon's kit", "embers", "thieves", "theft", "camp theft"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, alias)
	}
	for _, topic := range []string{"camp", "gathering", "campwatch", "cooking", "wounds"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, text, "camp gear", "help %s points at help camp gear", topic)
	}
	assert.NotContains(t, plain, "Nothing here is stolen")
	camp, err := GetHelpContents("camp")
	require.NoError(t, err)
	assert.Contains(t, camp, "embers")
	assert.NotContains(t, camp, "you can't rest again at a camp")
}
