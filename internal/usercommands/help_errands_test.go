package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 70: the errands page renders through help, is reachable by its
// aliases, and is linked from the hubs players start from.
func TestErrandsHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	text, err := GetHelpContents("errands")
	require.NoError(t, err)
	assert.Contains(t, text, "Help for")
	assert.Contains(t, text, "errand send [member] [job] [length]")
	assert.Contains(t, text, "errand recall [member]")
	assert.Contains(t, text, "escort")
	assert.Contains(t, text, "Never more")
	assert.NotContains(t, text, "[member]]")
	for _, alias := range []string{"errand", "company errands", "idle companions"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Contains(t, got, "errand send", alias)
	}
	for _, hub := range []string{"adventure", "company", "chronicle", "webclient"} {
		got, err := GetHelpContents(hub)
		require.NoError(t, err, hub)
		assert.Contains(t, got, "errands", hub)
	}
}
