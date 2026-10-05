package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSkillOverHPHealthTable (Phase 35a2): HP before Vitality at levels
// 1/10/20/30/60 matches the design table, and level 60 stays within 1.6×
// level 1 for every class.
func TestSkillOverHPHealthTable(t *testing.T) {
	p := ProgressionConfig{HPBase: 48, HPFullLevels: 20, HPAfterFull: 0.2, DefaultHPPerLevel: 0.8, HPPerVitality: 0.5}
	for class, tc := range map[string]struct {
		rate  float64
		start int
		want  [5]int
	}{
		"warrior": {1.0, 10, [5]int{59, 68, 78, 80, 88}},
		"ranger":  {0.8, 6, [5]int{54, 62, 70, 72, 78}},
		"rogue":   {0.7, 4, [5]int{52, 59, 66, 67, 73}},
		"cleric":  {0.6, 2, [5]int{50, 56, 62, 63, 68}},
		"wizard":  {0.5, 0, [5]int{48, 53, 58, 59, 63}},
	} {
		var got [5]int
		for i, level := range []int{1, 10, 20, 30, 60} {
			got[i] = p.HealthAtLevel(level, 0, tc.rate, tc.start)
		}
		assert.Equal(t, tc.want, got, class)
		assert.LessOrEqual(t, float64(got[4]), 1.6*float64(got[0]), class)
	}
	assert.Equal(t, p.HealthAtLevel(10, 0, 1, 0), p.HealthAtLevel(10, 0, 1, -5), "a negative head start counts as none")
	assert.Equal(t, p.HealthAtLevel(10, 0, 1, 10)+5, p.HealthAtLevel(10, 10, 1, 10), "Vitality adds on top")
}
