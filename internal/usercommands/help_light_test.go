package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Light gear (living map design, LG): `help light` covers torches, the
// oil-burning lantern and its commands, and answers to their names.
func TestLightHelpCoversTorchesAndLanterns(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	want, err := GetHelpContents("light")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(want, "")
	for _, phrase := range []string{"Torches and lanterns", "light torch", "can't be put out", "light lantern", "off hand",
		"douse lantern", "fill lantern", "flask of lamp oil", "about a game day of oil", "help camp gear", "a companion holding one gets no light"} {
		assert.Contains(t, plain, phrase)
	}
	for _, alias := range []string{"lantern", "torch", "lamp oil", "douse", "darkness"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, alias)
	}
}
