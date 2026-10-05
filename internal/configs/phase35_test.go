package configs

import (
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func TestSmoothStatGrowth(t *testing.T) {
	p := ProgressionConfig{StatStepLevels: 5}
	p.Validate()
	old := p
	p.SmoothStatGrowth = true
	for base := 1; base <= 20; base++ {
		for _, level := range []int{5, 10, 15, 20, 60} {
			assert.Equal(t, old.RacialForLevel(level, base), p.RacialForLevel(level, base))
		}
		for level := 2; level <= 60; level++ {
			assert.GreaterOrEqual(t, p.RacialForLevel(level, base), p.RacialForLevel(level-1, base))
		}
	}
	assert.Greater(t, p.RacialForLevel(4, 10), old.RacialForLevel(4, 10))
	p.StatStepLevels = 1
	old.StatStepLevels = 1
	for level := 1; level <= 20; level++ {
		assert.Equal(t, old.RacialForLevel(level, 10), p.RacialForLevel(level, 10))
	}
}
func TestStatPointsAt(t *testing.T) {
	p := ProgressionConfig{StatPointsEveryNLevels: 2, StatPointsPerLevel: 1}
	for _, tc := range []struct{ level, points int }{{0, 0}, {1, 0}, {2, 1}, {3, 1}, {4, 2}, {10, 5}, {60, 30}} {
		assert.Equal(t, tc.points, p.StatPointsAt(tc.level))
	}
	p.StatPointsEveryNLevels = 1
	assert.Zero(t, p.StatPointsAt(1), "creation earns no level-up reward")
	assert.Equal(t, 9, p.StatPointsAt(10))
	p.StatPointsPerLevel = 10
	assert.Equal(t, math.MaxInt, p.StatPointsAt(math.MaxInt))
}
func TestPhase35HealthShape(t *testing.T) {
	p := ProgressionConfig{DefaultHPPerLevel: 2.5, HPFullLevels: 20, HPAfterFull: 1.5, HPBase: 40, HPPerVitality: 0.5}
	p.Validate()
	for _, rate := range []float64{3, 2, 1.5, 2.5, 2.5} {
		assert.InDelta(t, rate*0.6, p.HealthAfterFull(rate), 1e-12)
		for _, level := range []int{10, 20, 21, 60} {
			want := 40 + int(float64(min(level, 20))*rate) + int(float64(max(level-20, 0))*rate*0.6) + 5
			assert.Equal(t, want, p.HealthAtLevel(level, 10, rate))
		}
	}
}
