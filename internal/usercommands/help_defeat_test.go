package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefeatHelp (Phase 53): the defeat page is indexed beside death, answers
// to its aliases, names the four scenarios and the reclaim command, and the
// pages that used to promise a church and a lost level point to it.
func TestDefeatHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	listed := map[string]string{}
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		listed[topic.Command] = topic.Category
	}
	require.Contains(t, listed, "defeat", "help index lists defeat")
	assert.Equal(t, listed["death"], listed["defeat"], "beside death")

	for alias, want := range map[string]string{"captured": "defeat", "robbed": "defeat", "left-for-dead": "defeat", "reclaim": "defeat"} {
		text, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "Help for "+want, alias)
	}

	defeat, err := GetHelpContents("defeat")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(defeat, "")
	for _, want := range []string{"Rescued", "Captured", "Left for dead", "Robbed", "reclaim", "No level is lost", "Hungry and Exhausted"} {
		assert.Contains(t, plain, want)
	}
	assert.NotContains(t, plain, "{{")

	death, err := GetHelpContents("death")
	require.NoError(t, err)
	plain = tagPattern.ReplaceAllString(death, "")
	assert.Contains(t, plain, "help defeat")
	assert.Contains(t, plain, "When no scenario fits")

	for _, hub := range []string{"combat", "adventure"} {
		text, err := GetHelpContents(hub)
		require.NoError(t, err, hub)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "help defeat", hub)
	}
}
