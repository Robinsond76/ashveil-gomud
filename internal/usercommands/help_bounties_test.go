package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/bounty"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 76: the bounties page renders through help, is reachable by its
// aliases, is linked from the hubs players start from, and states the
// numbers the rules use.
func TestBountiesHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	text, err := GetHelpContents("bounties")
	require.NoError(t, err)
	assert.Contains(t, text, "Help for")
	assert.Contains(t, text, "bounty take [number]")
	assert.Contains(t, text, "bounty claim [number]")
	assert.Contains(t, text, "bounty drop [number]")
	assert.Contains(t, text, "A bounty never changes a foe")
	assert.Contains(t, text, "Gold only")
	assert.Contains(t, text, "20 gold for each level")
	assert.Contains(t, text, "6 gold per level")
	assert.Contains(t, text, "up to five bounties")
	assert.Contains(t, text, "every six hours")
	assert.Contains(t, text, "three bounties at once")
	assert.NotContains(t, text, "[number]]")
	// The page's numbers are the rules' numbers.
	assert.Equal(t, 20, bounty.BossPerLevel)
	assert.Equal(t, 6, bounty.GroupPerLevel)
	assert.Equal(t, 5, bounty.PostingsPerBoard)
	assert.Equal(t, 6*60*60, bounty.WindowSeconds)
	assert.Equal(t, 3, bounty.MaxHeld)
	assert.Equal(t, 3, bounty.GroupCount)
	assert.Equal(t, 24*60*60, bounty.TermSeconds)

	for _, alias := range []string{"bounty", "bounty board", "notice board", "wanted marks"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Contains(t, got, "bounty take", alias)
	}
	for _, hub := range []string{"adventure", "company", "chronicle", "webclient", "encounters"} {
		got, err := GetHelpContents(hub)
		require.NoError(t, err, hub)
		assert.Contains(t, got, "bounties", hub)
	}
}
