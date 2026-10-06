package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 49: the banter help page renders through `help`, is indexed, and is
// linked from the camp and combat hubs and the set page.
func TestBanterHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	found := false
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "banter" && !topic.AdminOnly {
			found = true
		}
	}
	assert.True(t, found, "help index lists banter")

	page, err := GetHelpContents("banter")
	require.NoError(t, err)
	text := tagPattern.ReplaceAllString(page, "")
	for _, want := range []string{"Help for banter", "set banter off", "stoic, cheerful, grim, boastful, wry or devout", "Around the fire", "never in the middle of one"} {
		assert.Contains(t, text, want)
	}
	for _, alias := range []string{"camp-talk", "chatter", "companion-talk"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, page, got, "help %s is help banter", alias)
	}
	for _, hub := range []string{"camp", "combat", "set", "webclient"} {
		hubPage, err := GetHelpContents(hub)
		require.NoError(t, err, hub)
		assert.Contains(t, hubPage, "banter", "help %s points to banter", hub)
	}
}
