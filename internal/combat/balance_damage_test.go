package combat

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func TestStrengthDamageGrowthAndCap(t *testing.T) {
	cfg := configs.GetGamePlayConfig()
	cfg.Combat.DamageBonusMin = 2
	cfg.Combat.DamageBonusMax = 20
	cfg.Combat.DamagePerStrength = 1.75
	cfg.Combat.StatEdgeSpan = 10
	cfg.Combat.DamageEdgeMax = 4
	t.Cleanup(configs.SetTestGamePlayConfig(cfg))
	// 6 against 2: 2 + 6×1.75 + 0.4×4 = 14.1, rounded down.
	for _, c := range []struct{ atk, def, want int }{{0, 0, 2}, {2, 2, 5}, {6, 6, 12}, {12, 12, 20}, {2, 6, 5}, {6, 2, 14}, {-1, 0, 2}, {math.MaxInt, math.MinInt, 20}, {math.MaxInt, math.MaxInt, 20}} {
		assert.Equal(t, c.want, damageBonus(c.atk, c.def), "%d against %d", c.atk, c.def)
	}
	cfg.Combat.DamagePerStrength = 0
	restore := configs.SetTestGamePlayConfig(cfg)
	defer restore()
	assert.Equal(t, 2, damageBonus(12, 12), "zero disables absolute Strength growth")
	assert.Equal(t, 6, damageBonus(100, 0), "a full edge adds DamageEdgeMax")
	cfg.Combat.DamageEdgeMax = 40
	configs.SetTestGamePlayConfig(cfg)
	assert.Equal(t, 20, damageBonus(100, 0), "the cap holds")
}

// Armor comparison uses a stat-independent approximation of marginal damage.
func TestStrengthRankingIncludesConfiguredDamageGrowth(t *testing.T) {
	cfg := configs.GetGamePlayConfig()
	cfg.Combat.DamageBonusMin, cfg.Combat.DamageBonusMax = 2, 20
	cfg.Combat.DamagePerStrength = 0
	restore := configs.SetTestGamePlayConfig(cfg)
	t.Cleanup(restore)
	cfg.Combat.DamageEdgeMax, cfg.Combat.StatEdgeSpan = 4, 10
	configs.SetTestGamePlayConfig(cfg)
	legacy := statWeight("strength")
	assert.InDelta(t, 0.4, legacy, 0.00001, "a point of edge is a tenth of DamageEdgeMax")
	cfg.Combat.DamagePerStrength = 1.75
	configs.SetTestGamePlayConfig(cfg)
	assert.InDelta(t, legacy+1.75, statWeight("strength"), 0.00001)
}
