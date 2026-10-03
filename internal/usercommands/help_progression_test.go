package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestProgressionHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	text, err := GetHelpContents("progression")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{"levels 5, 10, 15", "Warrior 3 HP", "wizard 1.5", "Fractional gains accumulate", "no level cap", "training", "60", "1.1"} {
		assert.Contains(t, plain, want)
	}
	assert.NotContains(t, plain, "{{")
	for _, alias := range []string{"stat-steps", "leveling"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err)
		assert.Equal(t, text, got)
	}
	for _, topic := range []string{"experience", "status", "stats", "health", "vitality", "growth"} {
		got, err := GetHelpContents(topic)
		require.NoError(t, err)
		assert.Contains(t, got, "progression", topic)
	}
}

func TestProgressionStatusNamesNextStep(t *testing.T) {
	useWorld(t, "default")
	u := users.NewUserRecord(7, 0)
	u.Character.Level = 9
	u.Character.Validate()
	text := tagPattern.ReplaceAllString(buildStatusPanel(u), "")
	assert.True(t, strings.Contains(text, "Next stat step:") || strings.Contains(text, "Step:"))
	assert.Contains(t, text, "level 10")
}
