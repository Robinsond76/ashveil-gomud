package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCoordinationHelp (Phase 33i2): help coordination renders with the
// four tiers and their numbers, answers to its aliases, and the pages the
// phase changed point to it and no longer say enemies take no wounds.
func TestCoordinationHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	text, err := GetHelpContents("coordination")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{"Help for coordination", "a rabble", "a band", "a drilled company", "a veteran company",
		"levels 1-9", "levels 10-24", "levels 25-44", "levels 45 and up", "below 30%", "below 50%", "below 60%", "below 70%",
		"light ones", "5 game hours"} {
		assert.Contains(t, plain, want)
	}
	for _, alias := range []string{"coordinated", "enemy-roles", "rabble", "drilled", "enemy-wounds", "enemy-recovery"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, text, got, "help %s is help coordination", alias)
	}
	for _, topic := range []string{"assessment", "scout", "consider", "combat", "wounds", "guardian", "tactics", "webclient"} {
		page, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(page, ""), "help coordination", topic)
	}
	wounds, err := GetHelpContents("wounds")
	require.NoError(t, err)
	assert.NotContains(t, wounds, "Enemies are not.")
	assessment, err := GetHelpContents("assessment")
	require.NoError(t, err)
	assert.Contains(t, tagPattern.ReplaceAllString(assessment, ""), "They fight as a band: a healer among them.")
}
