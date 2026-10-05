package combat

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/stretchr/testify/assert"
)

// skillConfig pins the 35a2 shipped combat numbers.
func skillConfig(t *testing.T) {
	t.Helper()
	cfg := configs.GetGamePlayConfig()
	cfg.Combat.StatEdgeSpan = 40
	cfg.Combat.SkillEdgeSpan = 20
	cfg.Combat.DefaultAttackRate, cfg.Combat.DefaultEvasionRate = 1, 1
	cfg.Combat.ToHitMin, cfg.Combat.ToHitEven, cfg.Combat.ToHitMax = 10, 60, 95
	cfg.Combat.DodgeChanceMin, cfg.Combat.DodgeChanceEven, cfg.Combat.DodgeChanceMax = 3, 12, 40
	cfg.Combat.ParryChanceMin, cfg.Combat.ParryChanceEven, cfg.Combat.ParryChanceMax = 3, 12, 40
	cfg.Combat.BlockChanceMin, cfg.Combat.BlockChanceEven, cfg.Combat.BlockChanceMax = 8, 15, 55
	cfg.Combat.CritChanceMin, cfg.Combat.CritChanceEven, cfg.Combat.CritChanceMax = 5, 15, 30
	cfg.Combat.DamageBonusMin, cfg.Combat.DamageBonusMax = 6, 12
	cfg.Combat.DamagePerStrength, cfg.Combat.DamageEdgeMax = 0.375, 2
	t.Cleanup(configs.SetTestGamePlayConfig(cfg))
}

func fighterAt(level int) *characters.Character {
	c := characters.New()
	c.Level = level
	return c
}

// Each opposed chance at rating gaps of 0, ±10 and ±20, through the
// character wrappers combat uses; crit chance doesn't move with skill.
func TestSkillGapChances(t *testing.T) {
	skillConfig(t)
	for _, tc := range []struct {
		atk, def       int // levels: ratings at the default rate of 1
		hit, dodge     int
		parry, block10 int // block with a 10-armor shield
	}{
		{20, 20, 60, 12, 12, 25},
		{30, 20, 77, 7, 7, 16},
		{20, 30, 35, 26, 26, 40},
		{40, 20, 95, 3, 3, 8},
		{20, 40, 10, 40, 40, 55},
		{60, 20, 95, 3, 3, 8},
	} {
		atk, def := fighterAt(tc.atk), fighterAt(tc.def)
		assert.Equal(t, tc.hit, hitChance(atk, def), "hit %d vs %d", tc.atk, tc.def)
		assert.Equal(t, tc.dodge, dodgeChance(def, atk), "dodge %d vs %d", tc.atk, tc.def)
		assert.Equal(t, tc.parry, parryChance(def, atk, 0), "parry %d vs %d", tc.atk, tc.def)
		assert.Equal(t, tc.block10, blockChanceForEdge(10, defenseEdge(def, atk)), "block %d vs %d", tc.atk, tc.def)
	}
	assert.Equal(t, critChance(5, 5, false, false), critChance(5, 5, false, false), "crit ignores level")
	assert.Equal(t, 15, critChance(5, 5, false, false))

	// The stat edge still adds: a faster attacker at an even skill hits more.
	quick, slow := fighterAt(20), fighterAt(20)
	quick.Stats.Speed.ValueAdj, slow.Stats.Speed.ValueAdj = 30, 10
	assert.Equal(t, 77, hitChance(quick, slow), "a 20-point Speed lead is half a stat edge")
	assert.InDelta(t, 1.0, combinedEdge(1, 0.5), 1e-9, "held to a full edge")
	assert.InDelta(t, -1.0, combinedEdge(-0.8, -0.8), 1e-9)
}

// The damage bonus at Strength 3/6/8/10/16 against an equal foe.
func TestSkillOverHPDamageBonus(t *testing.T) {
	skillConfig(t)
	for str, want := range map[int]int{3: 7, 6: 8, 8: 9, 10: 9, 16: 12} {
		assert.Equal(t, want, damageBonus(str, str), "Strength %d", str)
	}
	assert.Equal(t, 12, damageBonus(60, 0), "held to the max")
}

// TestSkillOverHPHitSize (acceptance 1): a 1d10 weapon's landed hits
// between equal unarmored warriors (Strength 3/6/8/10/16 at levels
// 1/10/20/30/60, HP before Vitality) take a steady share of HP at every
// level: the middle 80% of rolls within 13–27%, the average 17–23%.
func TestSkillOverHPHitSize(t *testing.T) {
	skillConfig(t)
	p := configs.ProgressionConfig{HPBase: 48, HPFullLevels: 20, HPAfterFull: 0.2, DefaultHPPerLevel: 0.8, HPPerVitality: 0.5}
	for level, str := range map[int]int{1: 3, 10: 6, 20: 8, 30: 10, 60: 16} {
		hp := float64(p.HealthAtLevel(level, 0, 1.0, 10))
		bonus := damageBonus(str, str)
		low, high := float64(2+bonus)/hp, float64(9+bonus)/hp // the 10th and 90th percentile rolls
		avg := (5.5 + float64(bonus)) / hp
		// In whole percents, as the design's table rounds them.
		assert.GreaterOrEqual(t, math.Round(low*100), 13.0, "level %d: low hit %.3f", level, low)
		assert.LessOrEqual(t, math.Round(high*100), 27.0, "level %d: high hit %.3f", level, high)
		assert.InDelta(t, 0.20, avg, 0.03, "level %d: average hit %.3f", level, avg)
		t.Logf("level %d: bonus %d, HP %.0f, middle 80%% %.0f–%.0f%%, average %.1f%%", level, bonus, hp, low*100, high*100, avg*100)
	}
}
