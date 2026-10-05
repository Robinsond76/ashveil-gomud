package configs

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrainingPointsAt(t *testing.T) {
	p := ProgressionConfig{TrainingPointsEveryNLevels: 1, TrainingPointsPerLevel: 1}
	for _, tc := range []struct{ level, points int }{{-1, 0}, {0, 0}, {1, 1}, {2, 2}, {6, 6}, {20, 20}} {
		assert.Equal(t, tc.points, p.TrainingPointsAt(tc.level), "level %d", tc.level)
	}
	p.TrainingPointsEveryNLevels = 2
	for _, tc := range []struct{ level, points int }{{1, 0}, {2, 1}, {5, 2}, {6, 3}} {
		assert.Equal(t, tc.points, p.TrainingPointsAt(tc.level), "every 2, level %d", tc.level)
	}
	p.TrainingPointsEveryNLevels = 1
	p.TrainingPointsPerLevel = 0
	assert.Zero(t, p.TrainingPointsAt(10))
	p.TrainingPointsPerLevel = 10
	assert.Equal(t, math.MaxInt, p.TrainingPointsAt(math.MaxInt))
}
