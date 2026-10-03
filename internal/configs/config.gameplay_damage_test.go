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
