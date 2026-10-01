package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestFriendlyEffectHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	text, err := GetHelpContents("friendly-effects")
	require.NoError(t, err)
	for _, want := range []string{"without a player party", "6 mana", "wound limit", "Temporary charms", "-10 health", "caught up in", "another battle", "no one left to help"} {
		assert.Contains(t, text, want)
	}
	for _, alias := range []string{"friendly-scopes", "company-healing", "support-scopes"} {
		other, err := GetHelpContents(alias)
		require.NoError(t, err)
		assert.Equal(t, text, other)
	}
	for _, topic := range []string{"company", "combat", "heal", "cast", "spells"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err)
		assert.Contains(t, text, "help friendly-effects")
	}
	text, err = GetHelpContents("protection")
	require.NoError(t, err)
	assert.Contains(t, text, "another player's battle")
	// 33b review: the added paragraphs stay above each page's see-also line.
	for _, topic := range []string{"cast", "spells", "heal"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err)
		link := strings.Index(text, "help friendly-effects")
		require.GreaterOrEqual(t, link, 0, topic)
		assert.Less(t, link, strings.LastIndex(text, "ee also"), topic)
	}
}
