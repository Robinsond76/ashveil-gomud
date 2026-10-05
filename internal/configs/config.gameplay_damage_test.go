package configs

import (
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func TestDamageGrowthConfigValidation(t *testing.T) {
	for _, v := range []float64{-1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		c := CombatConfig{DamagePerStrength: ConfigFloat(v)}
		c.validate()
		assert.Zero(t, c.DamagePerStrength)
	}
	c := CombatConfig{DamagePerStrength: 1.75}
	c.validate()
	assert.Equal(t, ConfigFloat(1.75), c.DamagePerStrength)
}

// 30g6 amendment: the stat edge's span and even values fall back to safe
// defaults, and the even values stay within their bounds.
func TestStatEdgeConfigValidation(t *testing.T) {
	c := CombatConfig{StatEdgeSpan: ConfigFloat(math.NaN()), DamageEdgeMax: -3}
	c.validate()
	assert.Equal(t, ConfigFloat(10), c.StatEdgeSpan)
	assert.Equal(t, ConfigInt(60), c.ToHitEven)
	assert.Equal(t, ConfigInt(15), c.CritChanceEven)
	assert.Zero(t, c.DamageEdgeMax)
	c = CombatConfig{StatEdgeSpan: -1, ToHitMin: 40, ToHitMax: 90, ToHitEven: 95, CritChanceMin: 10, CritChanceMax: 20, CritChanceEven: 3}
	c.validate()
	assert.Equal(t, ConfigFloat(10), c.StatEdgeSpan)
	assert.Equal(t, ConfigInt(90), c.ToHitEven, "held to the maximum")
	assert.Equal(t, ConfigInt(10), c.CritChanceEven, "held to the minimum")
}

// 30g6 amendment E: after HPFullLevels each archetype gains HPAfterFull
// scaled by its rate against the default (middle) rate.
func TestHealthAfterFullKeepsClassProportion(t *testing.T) {
	p := ProgressionConfig{HPBase: 10, DefaultHPPerLevel: 2, HPFullLevels: 10, HPAfterFull: 1, HPPerVitality: 1}
	assert.Equal(t, 1.0, p.HealthAfterFull(2))
	assert.Equal(t, 2.0, p.HealthAfterFull(4))
	assert.Equal(t, 0.5, p.HealthAfterFull(1))
	assert.Equal(t, 10+40+40, p.HealthAtLevel(30, 0, 4, 0), "a double-rate class doubles its later gains too")
	assert.Equal(t, 10+10+10, p.HealthAtLevel(30, 0, 1, 0))
	assert.Equal(t, 10+20+10, p.HealthAtLevel(20, 0, 0, 0), "no rate takes the default")
}
