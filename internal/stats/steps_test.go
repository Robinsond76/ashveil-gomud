package stats

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func TestRacialStatsOnlyGrowOnSteps(t *testing.T) {
	cfg := configs.GetProgressionConfig()
	assert.Equal(t, 5, int(cfg.StatStepLevels))
	for _, base := range []int{1, 3, 5, 10} {
		s := StatInfo{Base: base, Training: 7, Mods: 2}
		for _, span := range [][2]int{{1, 4}, {5, 9}, {10, 14}, {55, 59}} {
			for level := span[0]; level <= span[1]; level++ {
				s.Recalculate(level)
				assert.Equal(t, s.GainsForLevel(span[0]), s.Racial)
				assert.Equal(t, s.Racial+9, s.Value)
			}
			assert.GreaterOrEqual(t, s.GainsForLevel(span[1]+1), s.GainsForLevel(span[1]))
		}
	}
}

func TestProgressionSaturatedRacialStatWithBonuses(t *testing.T) {
	cfg := configs.GetGamePlayConfig()
	cfg.Progression.BaseModExponent = 5
	t.Cleanup(configs.SetTestGamePlayConfig(cfg))
	s := StatInfo{Base: 10, Training: 1, Mods: 1, NoCap: true}
	s.Recalculate(math.MaxInt)
	assert.Equal(t, math.MaxInt, s.Racial)
	assert.Equal(t, math.MaxInt, s.Value)
	assert.Equal(t, math.MaxInt, s.ValueAdj)
	for _, exempt := range []bool{false, true} {
		cfg.Progression.StatCapExemptBonus = configs.ConfigBool(exempt)
		undo := configs.SetTestGamePlayConfig(cfg)
		s.NoCap = false
		s.Recalculate(math.MaxInt)
		assert.Positive(t, s.ValueAdj)
		undo()
	}
	cfg.Progression.StatCapExponent = 1
	cfg.Progression.StatCapScale = 2
	restore := configs.SetTestGamePlayConfig(cfg)
	s.Recalculate(math.MaxInt)
	assert.Equal(t, math.MaxInt, s.ValueAdj, "linear compression also saturates")
	restore()
	assert.Equal(t, math.MinInt, SaturatingSum(math.MinInt, -1))
	assert.Equal(t, math.MaxInt, SaturatingSum(math.MaxInt, 1))
	assert.Equal(t, 6, SaturatingSum(3, 5, -2))
}
