package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAbilitiesHelp (Phase 33e): help abilities renders with its numbers
// and commands, answers to its aliases, is indexed under combat, and the
// pages it changed point to it.
func TestAbilitiesHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	text, err := GetHelpContents("abilities")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{"Help for abilities", "Tackle", "Opening Strike", "Aimed Shot",
		"between 20% and 80%", "Rest: 4 combat rounds", "Rest: 2 combat rounds", "Rest: 3 combat rounds",
		"strategy [who] abilities off", "strategy [who] reserve [percent]", "Abilities use no", "items"} {
		assert.Contains(t, plain, want)
	}
	assert.NotContains(t, plain, "<who>", "placeholders are written [who]")

	for _, alias := range []string{"ability", "opening-strike", "aimed-shot", "reserve", "class-abilities"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, text, got, "help %s is help abilities", alias)
	}

	listed := false
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "abilities" && topic.Category == "combat" {
			listed = true
		}
	}
	assert.True(t, listed, "help abilities is in the combat index")

	for _, page := range []string{"combat", "strategy", "company", "tactics", "archetype", "brawling", "skulduggery", "track"} {
		got, err := GetHelpContents(page)
		require.NoError(t, err, page)
		assert.Contains(t, tagPattern.ReplaceAllString(got, ""), "help abilities", "help %s points to help abilities", page)
	}
	strategy, err := GetHelpContents("strategy")
	require.NoError(t, err)
	assert.Contains(t, tagPattern.ReplaceAllString(strategy, ""), "strategy [who] reserve [percent]")
}
