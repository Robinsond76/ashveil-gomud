package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebClientHelp (Phase 32g): `help webclient` renders, answers to its
// aliases, names each tab and what its buttons send, and the pages about
// what the dock shows point to it.
func TestWebClientHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	text, err := GetHelpContents("webclient")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	assert.Contains(t, plain, "Help for webclient")
	for _, want := range []string{"Character", "Company", "Combat", "Comm", "Inventory", "Camp", "Who", "Kills",
		"cargo put", "cargo take", "company meal", "strategy", "formation", "scout", "Reset Layout"} {
		assert.Contains(t, plain, want)
	}
	assert.NotContains(t, text, "</ ", "no broken tags")

	for _, alias := range []string{"web client", "dock", "panels", "tabs", "layout"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, text, got, "help %s is help webclient", alias)
	}

	var listed bool
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "webclient" && !topic.AdminOnly {
			listed = true
		}
	}
	assert.True(t, listed, "the help index lists webclient")

	for _, page := range []string{"company", "cargo", "company-inventory", "camp", "strategy", "formation"} {
		got, err := GetHelpContents(page)
		require.NoError(t, err, page)
		assert.Contains(t, tagPattern.ReplaceAllString(got, ""), "help webclient", page)
	}
}
