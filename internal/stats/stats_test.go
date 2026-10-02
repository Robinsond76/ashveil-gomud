package stats

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultProgression sets all ProgressionConfig fields to their documented
// defaults so tests run independently of any loaded config file.
func defaultProgression(t *testing.T) {
	t.Helper()
	previous := configs.Flatten(configs.GetOverrides())
	t.Cleanup(func() { require.NoError(t, configs.RestoreOverrides(previous)) })
	flat := configs.Flatten(configs.GetOverrides())
	for key, value := range map[string]any{
		"StatStepLevels": 5, "BaseModFactor": 0.3333333334, "BaseModExponent": 1.0,
		"NaturalGainsModFactor": 0.5, "NaturalGainsExponent": 1.0,
		"StatCapThreshold": 105, "StatCapAnchor": 100, "StatCapExponent": 0.5, "StatCapScale": 2.0, "StatCapExemptBonus": false,
	} {
		flat["GamePlay.Progression."+key] = value
	}
	require.NoError(t, configs.RestoreOverrides(flat))
}

func TestStatInfo_GainsForLevel(t *testing.T) {
	defaultProgression(t)

	// With exponent=1.0 (linear), the formula reduces to the previous constants:
	//   basePoints = int((step-1) * 0.3333 * base)
	//   freePoints = int(step * 0.5)
	tests := []struct {
		name     string
		base     int
		level    int
		expected int
	}{
		{name: "level 1 base 10", base: 10, level: 1, expected: 0},
		{name: "level 1 base 0", base: 0, level: 1, expected: 0},
		{name: "level 0 clamps to 1", base: 10, level: 0, expected: 0},
		{name: "negative level clamps to 1", base: 10, level: -5, expected: 0},
		// Level 2 is still in the first step, with no automatic gains.
		{name: "level 2 base 10", base: 10, level: 2, expected: 0},
		// Level 10 is step 3: basePoints=6, freePoints=1 => 7.
		{name: "level 10 base 10", base: 10, level: 10, expected: 7},
		// Level 10, base 0: basePoints=0, freePoints=1.
		{name: "level 10 base 0", base: 0, level: 10, expected: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			si := StatInfo{Base: tt.base}
			assert.Equal(t, tt.expected, si.GainsForLevel(tt.level))
		})
	}
}

func TestStatInfo_Recalculate_BelowCap(t *testing.T) {
	defaultProgression(t)
	si := StatInfo{Base: 10, Training: 5}
	si.Recalculate(10)

	expectedRacial := si.GainsForLevel(10)
	assert.Equal(t, expectedRacial, si.Racial)
	assert.Equal(t, expectedRacial+5, si.Value)
	assert.Equal(t, si.Value, si.ValueAdj, "ValueAdj should equal Value when below cap")
}

func TestStatInfo_Recalculate_WithMods(t *testing.T) {
	defaultProgression(t)
	si := StatInfo{Base: 10, Training: 3}
	si.SetMod(7)
	si.Recalculate(5)

	assert.Equal(t, si.Racial+3+7, si.Value)
}

func TestStatInfo_Recalculate_AboveCap(t *testing.T) {
	defaultProgression(t)
	// Force a Value above the cap threshold by using high training.
	// With defaults: threshold=105, anchor=100, exponent=0.5, scale=2.0
	// Value = 200, overage = 200-100 = 100, ValueAdj = 100 + round(sqrt(100)*2) = 120
	si := StatInfo{Base: 10, Training: 200}
	si.Recalculate(1)

	// Value = Racial + 200; Racial at level 1 = 0, so Value = 200
	assert.Equal(t, 200, si.Value)

	// Use the actual config defaults to compute expected ValueAdj.
	cfg := configs.GetProgressionConfig()
	overage := 200 - int(cfg.StatCapAnchor)
	expectedAdj := int(cfg.StatCapAnchor) + int(math.Round(math.Pow(float64(overage), float64(cfg.StatCapExponent))*float64(cfg.StatCapScale)))
	assert.Equal(t, expectedAdj, si.ValueAdj)
	assert.True(t, si.ValueAdj < si.Value, "ValueAdj should be less than Value when above cap")
}

func TestStatInfo_Recalculate_ExactlyAtCap(t *testing.T) {
	defaultProgression(t)
	// Value == threshold triggers the cap branch.
	// With defaults: threshold=105, anchor=100, exponent=0.5, scale=2.0
	// Value=105, overage=5, ValueAdj = 100 + round(sqrt(5)*2) = 100 + round(4.47) = 104
	si := StatInfo{Base: 0, Training: 105}
	si.Recalculate(1)

	assert.Equal(t, 105, si.Value)
	cfg := configs.GetProgressionConfig()
	overage := 105 - int(cfg.StatCapAnchor)
	expectedAdj := int(cfg.StatCapAnchor) + int(math.Round(math.Pow(float64(overage), float64(cfg.StatCapExponent))*float64(cfg.StatCapScale)))
	assert.Equal(t, expectedAdj, si.ValueAdj)
}

func TestStatInfo_SetMod(t *testing.T) {
	t.Run("single mod", func(t *testing.T) {
		si := StatInfo{}
		si.SetMod(5)
		assert.Equal(t, 5, si.Mods)
	})

	t.Run("multiple mods are summed", func(t *testing.T) {
		si := StatInfo{}
		si.SetMod(3, 7, -2)
		assert.Equal(t, 8, si.Mods)
	})

	t.Run("no args resets to zero", func(t *testing.T) {
		si := StatInfo{Mods: 99}
		si.SetMod()
		assert.Equal(t, 0, si.Mods)
	})
}
