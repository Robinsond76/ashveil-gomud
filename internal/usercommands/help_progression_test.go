package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestProgressionHelp(t *testing.T) {
	useWorld(t, "default")
	g := configs.GetGamePlayConfig()
	g.Progression.SmoothStatGrowth = true
	g.Progression.StatPointsEveryNLevels = 2
	g.Progression.StatPointsPerLevel = 1
	g.Progression.HPFullLevels = 20
	t.Cleanup(configs.SetTestGamePlayConfig(g))
	keywords.LoadAliases()
	text, err := GetHelpContents("progression")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{"Smooth automatic stats", "every 2 levels", "through level 20", "class promotion (coming)", "second class option (coming)", "Warrior 10 to start, 1 HP per level", "wizard none and 0.5", "help evasion", "Fractional gains accumulate", "no level cap", "training", "60", "1.1"} {
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

func TestProgressionStatusNamesNextMilestone(t *testing.T) {
	useWorld(t, "default")
	u := users.NewUserRecord(7, 0)
	u.Character.Level = 9
	u.Character.Validate()
	text := tagPattern.ReplaceAllString(buildStatusPanel(u), "")
	assert.True(t, strings.Contains(text, "Next milestone:") || strings.Contains(text, "Next:"))
	assert.Contains(t, text, "level 10")
	assert.Contains(t, text, "class")
	assert.Contains(t, text, "promotion (coming)")
	assert.NotContains(t, text, "stat step")
}
