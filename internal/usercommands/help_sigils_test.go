package usercommands

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/sigils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 54: `help sigils` renders, is indexed under combat and answers to its
// aliases, and the pages a sigil touches point at it.
func TestSigilsHelpRendersAndIsIndexed(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	listed := false
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "sigils" {
			listed = true
			assert.Equal(t, "combat", topic.Category)
		}
	}
	assert.True(t, listed, "help index lists sigils")

	want, err := GetHelpContents("sigils")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(want, "")
	for _, phrase := range []string{"cast sigil of", "sigil chalk", "15 real", "Fire", "Ward", "Stillness", "Mending", "25%", "one sigil to a room", "never bought back", "Raiders"} {
		assert.Contains(t, plain, phrase)
	}
	for _, alias := range []string{"sigil", "cast sigil", "chalk", "fire sigil", "stillness sigil"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, alias)
	}
	for _, topic := range []string{"cast", "spells", "battlefield", "combat", "spell"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, text, "sigils", "help %s points at help sigils", topic)
	}
	// The numbers the page states are the rules' own.
	assert.Contains(t, plain, "15 real")
	assert.Equal(t, 15, sigils.Minutes)
	assert.Contains(t, plain, fmt.Sprintf("hit %d%%", sigils.FirePct))
	assert.Contains(t, plain, fmt.Sprintf("land %d%% stronger", sigils.MendingPct))
	assert.Contains(t, plain, "Each member of the company")
	assert.Equal(t, 3, sigils.StillRounds)
	for _, k := range sigils.Kinds {
		assert.Contains(t, plain, fmt.Sprintf("%d mana", k.ManaCost()), k)
	}
}

func TestLookLinesNameTheSigilAndWhoseItIs(t *testing.T) {
	assert.Empty(t, sigilLines(99999, 1), "no sigil in the room, no line")
}
