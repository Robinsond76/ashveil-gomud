package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 50: the battle condition a member began a battle in (RT.Fare*).

func fared(damage, guard int) *characters.Character {
	c := characters.New()
	c.RTState().FareDamage, c.RTState().FareGuard = damage, guard
	return c
}

func TestFareDamageCutsAndRaisesBlows(t *testing.T) {
	plain := characters.New()
	assert.Equal(t, 20, fareDamage(plain, plain, 20), "no condition: no change")
	assert.Equal(t, 18, fareDamage(fared(-10, 0), plain, 20), "starving: 10% off")
	assert.Equal(t, 19, fareDamage(fared(-5, 0), plain, 20), "hungry: 5% off, rounded")
	assert.Equal(t, 21, fareDamage(fared(5, 0), plain, 20), "a strong meal: 5% on")
	assert.Equal(t, 18, fareDamage(plain, fared(0, 10), 20), "a hearty target takes 10% less")
	assert.Equal(t, 22, fareDamage(plain, fared(0, -10), 20), "a parched target takes 10% more")
	assert.Equal(t, 1, fareDamage(fared(-10, 0), fared(0, 10), 1), "a blow keeps 1")
	assert.Equal(t, 0, fareDamage(fared(-10, 0), plain, 0), "a miss stays a miss")
}

func TestFareSpellFactor(t *testing.T) {
	assert.InDelta(t, 1, FareSpellFactor(characters.New(), characters.New()), 1e-9)
	assert.InDelta(t, 0.9, FareSpellFactor(fared(-10, 0), characters.New()), 1e-9)
	assert.InDelta(t, 1.05*0.9, FareSpellFactor(fared(5, 0), fared(0, 10)), 1e-9)
	assert.InDelta(t, 1, FareSpellFactor(nil, nil), 1e-9)
}

// Through the strike loop: a starving dealer's landed blows average about
// 10% less than a fed one's.
func TestStarvingBlowsLandWeakerThroughTheStrikeLoop(t *testing.T) {
	poisonSpecs(t)
	target := edgeFighter(90231)
	mean := func(cut int) float64 {
		total, n := 0, 0
		for i := 0; i < 3000 && n < 400; i++ {
			src := edgeFighter(90231)
			src.Equipment.Weapon = items.New(poisonTenID)
			src.RTState().FareDamage = cut
			res := calculateCombat(*src, *target, User, Mob, 0, 0)
			if res.Hit && !res.Crit && res.DamageToTarget > 0 {
				total += res.DamageToTarget + res.DamageToTargetReduction
				n++
			}
		}
		require.Positive(t, n)
		return float64(total) / float64(n)
	}
	fed, starving := mean(0), mean(-10)
	assert.InDelta(t, 0.9, starving/fed, 0.07)
}
