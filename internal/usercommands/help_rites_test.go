package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 74: the rites page renders through help, is reachable by its
// aliases, and is linked from the hubs players start from.
func TestRitesHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	text, err := GetHelpContents("rites")
	require.NoError(t, err)
	assert.Contains(t, text, "Help for")
	assert.Contains(t, text, "rite hold [member]")
	assert.Contains(t, text, "rite skip [member]")
	assert.Contains(t, text, "loyalty up 3 for each close")
	assert.Contains(t, text, "down 5 for each close")
	assert.Contains(t, text, "no more than 80")
	assert.Contains(t, text, "no less than 30")
	assert.Contains(t, text, "They cost no gold and give none")
	assert.NotContains(t, text, "[member]]")
	for _, alias := range []string{"rite", "funeral", "mourning", "hold rites"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Contains(t, got, "rite hold", alias)
	}
	for _, hub := range []string{"adventure", "company", "chronicle", "webclient", "bonds", "resurrect"} {
		got, err := GetHelpContents(hub)
		require.NoError(t, err, hub)
		assert.Contains(t, got, "rites", hub)
	}
}
