package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The quick menu help page renders through `help`, is indexed, answers to its
// aliases, and is linked from the web client and phone pages.
func TestQuickMenuHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	found := false
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "quickmenu" && !topic.AdminOnly {
			found = true
		}
	}
	assert.True(t, found, "help index lists quickmenu")

	page, err := GetHelpContents("quickmenu")
	require.NoError(t, err)
	text := tagPattern.ReplaceAllString(page, "")
	for _, want := range []string{"Help for quickmenu", "Enter on an empty command box", "Alt+Up", "Back", "Close", "Battle", "Gather"} {
		assert.Contains(t, text, want)
	}
	for _, alias := range []string{"quick-menu", "arrow keys", "keyboard"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, page, got, "help %s is help quickmenu", alias)
	}
	for _, hub := range []string{"webclient", "mobile"} {
		hubPage, err := GetHelpContents(hub)
		require.NoError(t, err, hub)
		assert.Contains(t, hubPage, "quickmenu", "help %s points to quickmenu", hub)
	}
}
