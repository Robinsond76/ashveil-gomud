package configs

import (
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func TestTempoConfigValidation(t *testing.T) {
	for _, c := range []CombatConfig{{}, {TempoMin: ConfigFloat(math.NaN()), TempoMax: ConfigFloat(math.Inf(1)), TempoSpeedRef: ConfigFloat(math.NaN()), TempoSpeedSpan: ConfigFloat(math.Inf(1))}, {TempoMin: -1, TempoMax: 100, TempoSpeedRef: -1, TempoSpeedSpan: -1, MaxTurnsPerRound: 99}} {
		c.validate()
		assert.Equal(t, ConfigFloat(0.6), c.TempoMin)
		assert.Equal(t, ConfigFloat(1.5), c.TempoMax)
		assert.Equal(t, ConfigFloat(10), c.TempoSpeedRef)
		assert.Equal(t, ConfigFloat(40), c.TempoSpeedSpan)
		assert.Equal(t, ConfigInt(2), c.MaxTurnsPerRound)
	}
	c := CombatConfig{TempoMin: 0.8, TempoMax: 1.2, TempoSpeedRef: 8, TempoSpeedSpan: 20, MaxTurnsPerRound: 1}
	c.validate()
	assert.Equal(t, ConfigFloat(0.8), c.TempoMin)
	assert.Equal(t, ConfigFloat(1.2), c.TempoMax)
	assert.Equal(t, ConfigInt(1), c.MaxTurnsPerRound)
}
