package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Phase 30g3: the agility keys fall back to their defaults when unset or
// out of range, and keep a valid value.
func TestAgilityConfigValidation(t *testing.T) {
	cases := []struct {
		name                  string
		base, perStr, free    ConfigFloat
		wantBase, wantStr, wf ConfigFloat
	}{
		{"unset", 0, 0, 0, 15, 0.5, 0.35},
		{"negative", -1, -1, -0.1, 15, 0.5, 0.35},
		{"free load too high", 20, 1, 1, 20, 1, 0.35},
		{"valid", 10, 0.25, 0.5, 10, 0.25, 0.5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := CombatConfig{AgilityBaseKg: tc.base, AgilityStrengthKg: tc.perStr, AgilityFreeLoad: tc.free}
			c.validate()
			assert.Equal(t, tc.wantBase, c.AgilityBaseKg)
			assert.Equal(t, tc.wantStr, c.AgilityStrengthKg)
			assert.Equal(t, tc.wf, c.AgilityFreeLoad)
		})
	}
}
