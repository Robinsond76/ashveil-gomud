package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// TestFoeDamageBonusRampsWithLevel: an enemy's flat bonus is none at level 1,
// grows evenly and is whole at DamageFullLevel.
func TestFoeDamageBonusRampsWithLevel(t *testing.T) {
	cfg := configs.GetGamePlayConfig()
	cfg.Combat.DamageFullLevel, cfg.Combat.HeroDamageLead = 5, 0
	t.Cleanup(configs.SetTestGamePlayConfig(cfg))
	for _, c := range []struct{ level, want int }{{0, 0}, {1, 0}, {2, 2}, {3, 5}, {4, 7}, {5, 10}, {30, 10}} {
		assert.Equal(t, c.want, levelDamageBonus(10, c.level, false), "level %d", c.level)
	}
	cfg.Combat.HeroDamageLead = 2
	configs.SetTestGamePlayConfig(cfg)
	assert.Equal(t, 5, levelDamageBonus(10, 1, true), "the hero reads two levels ahead")
	assert.Equal(t, 0, levelDamageBonus(10, 1, false))
	cfg.Combat.DamageFullLevel = 0
	configs.SetTestGamePlayConfig(cfg)
	assert.Equal(t, 10, levelDamageBonus(10, 1, false), "zero turns the ramp off")
}

// TestLevelOneFoeHitsForItsDice: through AttackMobVsPlayer, a level-1 enemy
// with a 1d4 weapon never beats a critical hit of its dice, a level-5 enemy
// still carries the whole flat bonus, and a charmed (company) mob of level 1
// keeps the bonus.
func TestLevelOneFoeHitsForItsDice(t *testing.T) {
	foe := woundFight(t) // 1d4 club, every blow lands as a critical hit
	cfg := configs.GetGamePlayConfig()
	cfg.Combat.DamageBonusMin, cfg.Combat.DamageBonusMax = 6, 12
	cfg.Combat.DamageFullLevel, cfg.Combat.HeroDamageLead = 5, 0
	t.Cleanup(configs.SetTestGamePlayConfig(cfg))
	user := users.NewUserRecord(90340, 90340)
	*user.Character = *edgeFighter(90231)

	hits := func(level int) (lo, hi int) {
		foe.Character.Level = level
		for i := 0; i < 60; i++ {
			user.Character.Health = user.Character.HealthMax.Value
			if d := AttackMobVsPlayer(foe, user).DamageToTarget; d > 0 {
				lo, hi = 0, max(hi, d)
			}
		}
		return lo, hi
	}
	_, hi := hits(1)
	assert.LessOrEqual(t, hi, 4, "a level-1 foe hits for its 1d4 alone")
	_, hi = hits(5)
	assert.Greater(t, hi, 4, "a full-level foe carries the whole flat bonus")
}

// TestLevelOneFoeCritStaysUnderTen: with the shipped crit multipliers and
// every blow a telling critical hit, a level-1 foe's 1d4 never tops 9, even
// against a target it far out-sees; at full level the Perception edge
// counts again (owner, 2026-10-12: a level-1 crit is at most 9).
func TestLevelOneFoeCritStaysUnderTen(t *testing.T) {
	foe := woundFight(t) // 1d4 club, every blow lands as a critical hit
	cfg := configs.GetGamePlayConfig()
	cfg.Combat.DamageBonusMin, cfg.Combat.DamageBonusMax = 0, 0
	cfg.Combat.DamageFullLevel, cfg.Combat.HeroDamageLead = 6, 2
	cfg.Combat.CritMultMin, cfg.Combat.CritMultMax = 1.5, 3.0
	cfg.Combat.TellingFactor = 1.4
	cfg.Combat.TellingEven, cfg.Combat.TellingFull, cfg.Combat.TellingLeast = 100, 100, 100
	t.Cleanup(configs.SetTestGamePlayConfig(cfg))
	foe.Character.Stats.Perception.ValueAdj = 500
	user := users.NewUserRecord(90341, 90341)
	*user.Character = *edgeFighter(90231)

	worst := func(level int) (hi int) {
		foe.Character.Level = level
		for i := 0; i < 80; i++ {
			user.Character.Health = user.Character.HealthMax.Value
			hi = max(hi, AttackMobVsPlayer(foe, user).DamageToTarget)
		}
		return hi
	}
	assert.Equal(t, 8, worst(1), "a telling 4 (6) plus half the die (2)")
	assert.Greater(t, worst(6), 9, "a full-level foe's Perception edge counts")
}
