package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorldMapHelp (Phase 40b): `help worldmap` renders, answers to its
// aliases, is listed in the index, and the pages about what the map shows
// point to it.
func TestWorldMapHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	text, err := GetHelpContents("worldmap")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{"Help for worldmap", "adventurer", "gold ring", "shield", "pennant", "tent", "never see the camp", "Sprites", "Camps", "Terrain", "Landmarks", "no exit between them", "fog", "Style", "Classic", "regrows", "three quarters", "help camp", "embers", "bedrolls", "help resources"} {
		assert.Contains(t, plain, want)
	}
	assert.NotContains(t, text, "</ ", "no broken tags")

	for _, alias := range []string{"map window", "tile map", "world map"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, text, got, "help %s is help worldmap", alias)
	}

	var listed bool
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "worldmap" && !topic.AdminOnly {
			listed = true
		}
	}
	assert.True(t, listed, "the help index lists worldmap")

	for _, hub := range []string{"webclient", "camp", "map"} {
		hubText, err := GetHelpContents(hub)
		require.NoError(t, err, hub)
		assert.Contains(t, tagPattern.ReplaceAllString(hubText, ""), "help worldmap", "help %s points at help worldmap", hub)
	}
}
