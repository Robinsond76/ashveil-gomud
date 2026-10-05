package configs

import (
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func TestProgressionStepsAndHP(t *testing.T) {
	p := ProgressionConfig{StatStepLevels: 5, HPBase: 5, HPFullLevels: 20, HPAfterFull: 1, HPPerVitality: 1}
	p.Validate()
	for _, tc := range []struct{ level, step, next int }{{1, 1, 5}, {4, 1, 5}, {5, 2, 10}, {9, 2, 10}, {10, 3, 15}, {60, 13, 65}} {
		assert.Equal(t, tc.step, p.StatStep(tc.level))
		assert.Equal(t, tc.next, p.NextStatStep(tc.level))
	}
	assert.Equal(t, 135, p.HealthAtLevel(20, 10, 6, 0))
	assert.Equal(t, 136, p.HealthAtLevel(21, 10, 6, 0))
	assert.Equal(t, 75, p.HealthAtLevel(20, 10, 3, 0))
}

func TestProgressionXPIncrementalKnee(t *testing.T) {
	p := ProgressionConfig{XPBase: 1000, XPLevelFactor: 0.75, XPLevelPower: 2, XPKneeLevel: 60, XPKneeGrowth: 1.1}
	p.Validate()
	for level := 1; level <= 60; level++ {
		assert.Equal(t, 1000+750*level*level, p.XPThreshold(level, 1))
	}
	knee := p.XPThreshold(60, 1)
	cost := float64(knee - p.XPThreshold(59, 1))
	assert.Equal(t, knee+int(cost*1.1), p.XPThreshold(61, 1))
	assert.Equal(t, int((float64(knee)+cost*1.1+cost*1.1*1.1)*1.5), p.XPThreshold(62, 1.5))
	assert.Equal(t, math.MaxInt, p.XPThreshold(math.MaxInt, 1))
	assert.Equal(t, math.MaxInt, p.NextStatStep(math.MaxInt))
	assert.Equal(t, p.XPThreshold(1, 1), p.XPThreshold(0, 1))
}

func TestProgressionDerivedValuesSaturate(t *testing.T) {
	p := ProgressionConfig{}
	p.Validate()
	assert.Equal(t, math.MaxInt, p.HealthAtLevel(math.MaxInt, math.MaxInt, 6, 0))
	p.BaseModExponent = 5
	assert.Equal(t, math.MaxInt, p.RacialForLevel(math.MaxInt, 10))
}
