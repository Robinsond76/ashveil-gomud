package characters

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"math"
	"testing"
)

func TestProgressionLevelUpStepPointsAndDeath(t *testing.T) {
	previous := configs.Flatten(configs.GetOverrides())
	t.Cleanup(func() { require.NoError(t, configs.RestoreOverrides(previous)) })
	flat := configs.Flatten(configs.GetOverrides())
	flat["GamePlay.Progression.SmoothStatGrowth"] = false
	flat["GamePlay.Progression.StatPointsEveryNLevels"] = 5
	flat["GamePlay.Progression.TrainingPointsPerLevel"] = 1
	flat["GamePlay.Progression.StatPointsPerLevel"] = 1
	require.NoError(t, configs.RestoreOverrides(flat))
	c := levelledCharacter(3)
	c.Stats.Strength.Base = 10
	c.Stats.Vitality.Base = 3
	c.Stats.Strength.Training = 9
	c.Validate()
	before := c.Stats.Strength.Value
	hp := c.HealthMax.Value
	c.Experience = c.XPTNL()
	ok, delta := c.LevelUp()
	require.True(t, ok)
	assert.Equal(t, before, c.Stats.Strength.Value)
	assert.Zero(t, delta.Strength.Value)
	assert.Greater(t, c.HealthMax.Value, hp)
	assert.Zero(t, c.StatPoints)
	assert.Equal(t, 1, c.TrainingPoints)
	c.Experience = c.XPTNL()
	ok, delta = c.LevelUp()
	require.True(t, ok)
	assert.Positive(t, delta.Strength.Value)
	assert.Equal(t, 1, c.StatPoints)
	assert.Equal(t, 2, c.TrainingPoints)
	c.LoseLevel()
	c.Experience = c.XPTNL()
	ok, _ = c.LevelUp()
	require.True(t, ok)
	assert.Equal(t, 1, c.StatPoints, "regaining a step cannot mint points")
	assert.Equal(t, 2, c.TrainingPoints)
	assert.Equal(t, 9, c.Stats.Strength.Training)
	assert.Equal(t, 0, StatPointsAtLevel(4))
	assert.Equal(t, 1, StatPointsAtLevel(5))
	assert.Equal(t, 2, StatPointsAtLevel(10))
}

func TestProgressionLegacyCharacterKeepsEarnedState(t *testing.T) {
	var c Character
	require.NoError(t, yaml.Unmarshal([]byte("name: Veteran\nlevel: 17\npeaklevel: 22\nexperience: 220000\nstatpoints: 7\ntrainingpoints: 4\nhealth: 9999\nmana: 9999\nstats:\n  strength:\n    training: 11\n  vitality:\n    training: 9\n"), &c))
	require.NoError(t, c.Validate(true))
	assert.Equal(t, 17, c.Level)
	assert.Equal(t, 22, c.PeakLevel)
	assert.Equal(t, 220000, c.Experience)
	assert.Equal(t, 7, c.StatPoints)
	assert.Equal(t, 4, c.TrainingPoints)
	assert.Equal(t, 11, c.Stats.Strength.Training)
	assert.Equal(t, c.Stats.Strength.GainsForLevel(17), c.Stats.Strength.Racial)
	assert.Equal(t, configs.GetProgressionConfig().HealthAtLevel(17, c.Stats.Vitality.ValueAdj, 5), c.HealthMax.Value)
	assert.Equal(t, c.HealthMax.Value, c.Health)
	assert.Equal(t, c.ManaMax.Value, c.Mana)
	c.Health = 3
	require.NoError(t, c.Validate(true))
	assert.Equal(t, 3, c.Health, "migration must not refill saved health")
}

func TestProgressionSaturatedXPCannotMintLevels(t *testing.T) {
	c := New()
	c.Level = 10000
	c.Experience = math.MaxInt
	assert.Equal(t, math.MaxInt, c.XPTNL())
	ok, _ := c.LevelUp()
	assert.False(t, ok)
	assert.Equal(t, 10000, c.Level)
	c.Level = 100
	c.Experience = c.XPTNL()
	ok, _ = c.LevelUp()
	assert.True(t, ok, "MaxLevel is only an editor display range")
	assert.Equal(t, 101, c.Level)
}

func TestProgressionSaturatedHPWithTraining(t *testing.T) {
	c := New()
	c.Level = math.MaxInt
	c.HealthMax.Training = 1
	c.RecalculateStats()
	assert.Equal(t, math.MaxInt, c.HealthMax.Value)
	assert.Equal(t, math.MaxInt, c.HealthMax.ValueAdj)
	c = New()
	c.HealthMax.Training = -1000
	c.RecalculateStats()
	assert.Equal(t, 1, c.HealthMax.Value)
	assert.Equal(t, 1, c.HealthMax.ValueAdj)
}

func TestPhase35LevelUpPeakProtectsStatPoints(t *testing.T) {
	g := configs.GetGamePlayConfig()
	g.Progression.StatPointsEveryNLevels = 2
	g.Progression.StatPointsPerLevel = 1
	g.Progression.SmoothStatGrowth = true
	t.Cleanup(configs.SetTestGamePlayConfig(g))
	c := levelledCharacter(1)
	for level := 2; level <= 10; level++ {
		c.Experience = c.XPTNL()
		ok, _ := c.LevelUp()
		require.True(t, ok)
		assert.Equal(t, g.Progression.StatPointsAt(level), c.StatPoints)
	}
	c.LoseLevel()
	c.Experience = c.XPTNL()
	ok, _ := c.LevelUp()
	require.True(t, ok)
	assert.Equal(t, 5, c.StatPoints)
	assert.Equal(t, 5, StatPointsAtLevel(10))
}
