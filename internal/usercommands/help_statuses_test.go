package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStatusesHelp (Phase 30a): the page renders through help, is listed
// under combat with its aliases, is linked from the combat and narration
// pages, and names every status the game ships.
func TestStatusesHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	listed := false
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "combat" && topic.Command == "statuses" {
			listed = true
		}
	}
	assert.True(t, listed, "help index lists statuses under combat")

	page, err := GetHelpContents("statuses")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(page, "")
	assert.Contains(t, plain, "Help for")

	for _, alias := range []string{"status", "status-effects", "bleeding", "stagger", "knockdown", "stun", "burning", "exposed", "armor-break", "hobbled", "overloaded", "crit-effects"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, page, got, "help %s is help statuses", alias)
	}

	// Every shipped status is named on the page.
	for _, id := range status.Ids() {
		word := status.Word(id)
		require.NotEmpty(t, word, "status %d", id)
		assert.Contains(t, strings.ToLower(plain), word, "the page names %s", word)
	}

	for _, topic := range []string{"combat", "narration"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, text, "help statuses", "%s links to help statuses", topic)
	}
}
