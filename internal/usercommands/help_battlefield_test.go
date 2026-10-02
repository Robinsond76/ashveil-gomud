package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBattlefieldHelpRendersAndIsLinked(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	text, err := GetHelpContents("battlefield")
	require.NoError(t, err)
	for _, want := range []string{"orthogonal", "two columns", "26-50", "1-25", "three combat rounds", "Frostbitten", "nobody is struck twice", "costs none", "first applies or changes"} {
		assert.Contains(t, text, want)
	}
	for _, alias := range []string{"battlefield-conditions", "clusters", "flanking", "narrow-ground"} {
		rendered, err := GetHelpContents(alias)
		require.NoError(t, err)
		assert.Equal(t, text, rendered)
	}
	ambush, err := GetHelpContents("ambush")
	require.NoError(t, err)
	assert.Contains(t, ambush, "opening combat round")
	assert.Contains(t, ambush, "5-95")
	for _, alias := range []string{"surprise", "surprised"} {
		rendered, err := GetHelpContents(alias)
		require.NoError(t, err)
		assert.Equal(t, ambush, rendered)
	}
	hub, err := GetHelpContents("combat")
	require.NoError(t, err)
	assert.Contains(t, hub, "help battlefield")
	assert.Contains(t, hub, "help ambush")
}
