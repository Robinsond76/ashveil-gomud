package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeleteHelp (Phase 32h): help delete renders the command and what is
// kept, answers to its aliases, and help password links it.
func TestDeleteHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	text, err := GetHelpContents("delete")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	assert.Contains(t, plain, "Help for delete")
	assert.Contains(t, plain, "delete character")
	assert.Contains(t, plain, "Your login stays")
	assert.Contains(t, plain, "Three wrong passwords")
	for _, alias := range []string{"deletion", "delete-character", "reroll"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, text, got, alias)
	}

	pw, err := GetHelpContents("password")
	require.NoError(t, err)
	assert.Contains(t, tagPattern.ReplaceAllString(pw, ""), "help delete")

	var listed bool
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "delete" {
			listed = true
		}
	}
	assert.True(t, listed, "in the help index")
}
