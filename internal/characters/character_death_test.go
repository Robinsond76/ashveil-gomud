package characters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func levelledCharacter(level int) *Character {
	c := New()
	c.Level = level
	c.Experience = c.XPTL(level-1) + 5
	if level == 1 {
		c.Experience = 5
	}
	c.Validate()
	c.Health = c.HealthMax.Value
	c.Mana = c.ManaMax.Value
	return c
}

func TestLoseLevelDropsOneLevelToFloor(t *testing.T) {
	c := levelledCharacter(10)
	c.TrainingPoints, c.StatPoints = 3, 2

	from, to := c.LoseLevel()

	assert.Equal(t, 10, from)
	assert.Equal(t, 9, to)
	assert.Equal(t, 9, c.Level)
	assert.Equal(t, c.XPTL(8), c.Experience, "the floor of level 9")
	assert.Equal(t, 10, c.PeakLevel)
	assert.Equal(t, 3, c.TrainingPoints, "points are untouched")
	assert.Equal(t, 2, c.StatPoints)
	levelled, _ := c.LevelUp()
	assert.False(t, levelled, "no immediate level up at the floor")
}

func TestLoseLevelAtLevelTwo(t *testing.T) {
	c := levelledCharacter(2)

	from, to := c.LoseLevel()

	assert.Equal(t, 2, from)
	assert.Equal(t, 1, to)
	assert.Equal(t, 1, c.Experience, "Validate keeps experience at 1 or more")
}

func TestLoseLevelAtLevelOneResetsProgress(t *testing.T) {
	c := levelledCharacter(1)
	assert.Equal(t, 5, c.Experience)

	from, to := c.LoseLevel()

	assert.Equal(t, 1, from)
	assert.Equal(t, 1, to)
	assert.Equal(t, 1, c.Level)
	assert.Equal(t, 1, c.Experience, "the engine's minimum")
	assert.Equal(t, 1, c.PeakLevel)
}

func TestLoseLevelKeepsAHigherPeak(t *testing.T) {
	c := levelledCharacter(4)
	c.PeakLevel = 7

	c.LoseLevel()

	assert.Equal(t, 7, c.PeakLevel)
}

func TestLoseLevelRecalculatesAndClamps(t *testing.T) {
	c := levelledCharacter(30)
	healthMax, manaMax := c.HealthMax.Value, c.ManaMax.Value

	c.LoseLevel()

	assert.Less(t, c.HealthMax.Value, healthMax, "derived stats follow the level")
	assert.LessOrEqual(t, c.Health, c.HealthMax.Value)
	assert.LessOrEqual(t, c.Mana, c.ManaMax.Value)
	assert.LessOrEqual(t, c.ManaMax.Value, manaMax)
}

// levelUpGrant levels c up once from its floor and returns the training and
// stat points gained.
func levelUpGrant(t *testing.T, c *Character) (training, stat int) {
	t.Helper()
	training, stat = c.TrainingPoints, c.StatPoints
	c.Experience = c.XPTNL()
	levelled, _ := c.LevelUp()
	assert.True(t, levelled)
	return c.TrainingPoints - training, c.StatPoints - stat
}

func TestLevelUpGrantsPointsOnlyAbovePeak(t *testing.T) {
	c := levelledCharacter(5)
	firstTraining, firstStat := levelUpGrant(t, c)
	assert.Equal(t, 6, c.Level)
	assert.Equal(t, 6, c.PeakLevel)

	c.LoseLevel()
	assert.Equal(t, 5, c.Level)
	training, stat := levelUpGrant(t, c)
	assert.Equal(t, 6, c.Level)
	assert.Zero(t, training, "re-earning a lost level grants nothing")
	assert.Zero(t, stat)

	training, stat = levelUpGrant(t, c)
	assert.Equal(t, 7, c.Level)
	assert.Equal(t, firstTraining, training, "a new level grants as usual")
	assert.Equal(t, firstStat, stat)
	assert.Equal(t, 7, c.PeakLevel)
}

func TestLevelUpLegacyPeakZero(t *testing.T) {
	c := levelledCharacter(8)
	assert.Zero(t, c.PeakLevel, "a character saved before Phase 25a")
	fresh := levelledCharacter(8)
	fresh.PeakLevel = 8

	training, stat := levelUpGrant(t, c)
	freshTraining, freshStat := levelUpGrant(t, fresh)

	assert.Equal(t, freshTraining, training)
	assert.Equal(t, freshStat, stat)
	assert.Equal(t, 9, c.PeakLevel)
}
