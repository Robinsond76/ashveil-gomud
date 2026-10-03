package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTempoHelpRendersAndIsLinked(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	assert.Contains(t, keywords.GetAllHelpTopics(), "tempo")
	text, err := GetHelpContents("tempo")
	require.NoError(t, err)
	for _, want := range []string{"one opening turn", "at most 99", "at most 2", "0.6", "1.5", "tackle spends one turn", "once per combat round", "separate rounds"} {
		assert.Contains(t, text, want)
	}
	for _, alias := range []string{"actions", "turns"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err)
		assert.Equal(t, text, got)
	}
	for _, topic := range []string{"combat", "encumbrance", "burden", "speed", "abilities", "interrupts", "statuses"} {
		got, err := GetHelpContents(topic)
		require.NoError(t, err)
		assert.Contains(t, got, "help tempo")
	}
	speed, err := GetHelpContents("speed")
	require.NoError(t, err)
	assert.NotContains(t, speed, "extra attacks beyond")
}
