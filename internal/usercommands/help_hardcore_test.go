package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 77: help hardcore and help blessings render, answer to their
// aliases, are indexed on the road, state the numbers the code enforces,
// and are linked from the pages a player reaches them from.
func TestHardcoreAndBlessingsHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var road []string
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "road" && !topic.AdminOnly {
			road = append(road, topic.Command)
		}
	}
	assert.Contains(t, road, "hardcore", "help index lists hardcore under the road")
	assert.Contains(t, road, "blessings", "help index lists blessings under the road")

	pages := map[string]struct {
		phrases []string
		aliases []string
	}{
		"hardcore": {
			phrases: []string{"Iron", "two levels", "never ends in a rescue", "Take the Iron option?", "(Iron)", "foes are exactly as strong"},
			aliases: []string{"iron", "iron option", "trial of iron"},
		},
		"blessings": {
			phrases: []string{"starting item", "recruit discount", "never come to", "10%", "Iron blessings", "waits"},
			aliases: []string{"blessing", "account blessings"},
		},
	}
	for topic, want := range pages {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		plain := tagPattern.ReplaceAllString(text, "")
		assert.Contains(t, plain, "Help for", topic)
		for _, phrase := range want.phrases {
			assert.Contains(t, plain, phrase, "help %s says %q", topic, phrase)
		}
		for _, alias := range want.aliases {
			got, err := GetHelpContents(alias)
			require.NoError(t, err, alias)
			assert.Equal(t, text, got, "help %s is help %s", alias, topic)
		}
	}

	// The numbers in the pages are the numbers in the code.
	assert.Equal(t, 2, characters.IronLevelsLost, `help hardcore says "two levels"`)

	for hub, topic := range map[string]string{
		"adventure": "hardcore", "death": "hardcore", "defeat": "hardcore",
		"delete": "blessings", "lifestory": "hardcore", "company": "blessings", "webclient": "Iron badge",
	} {
		text, err := GetHelpContents(hub)
		require.NoError(t, err, hub)
		assert.Contains(t, text, topic, "help %s mentions %s", hub, topic)
	}
	adventure, err := GetHelpContents("adventure")
	require.NoError(t, err)
	assert.Contains(t, adventure, "help blessings")
}
