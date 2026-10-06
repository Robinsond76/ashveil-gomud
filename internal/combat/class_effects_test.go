package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 38b: a class's effects reach the combat formulas.

func classed(class string, level int) *characters.Character {
	c := characters.New()
	c.Level, c.Health = level, 100
	c.HealthMax.Value = 100
	c.SetClassState(class, nil)
	return c
}

func TestClassBlowDamageIsUntouchedWithoutAClass(t *testing.T) {
	defenseSpecs(t)
	foe := classed("", 10)
	assert.Equal(t, 10, classBlowDamage(classed("", 30), foe, 10))
}

func TestFinisherAddsAnOpeningStrikeAgainstAWoundedFoe(t *testing.T) {
	defenseSpecs(t)
	assassin := classed("assassin", 12)
	foe := classed("", 12)
	foe.Health = 90
	assert.Equal(t, 10, classBlowDamage(assassin, foe, 10), "a healthy foe is not finished")
	foe.Health = 40
	assert.Greater(t, classBlowDamage(assassin, foe, 10), 10, "at 40% it is")
	foe.Health = 50
	assert.Equal(t, 10, classBlowDamage(assassin, foe, 10), "50% needs Killing eye")
	killer := classed("assassin", 20)
	assert.Greater(t, classBlowDamage(killer, foe, 10), 10)
}

func TestPursuitRaisesDamageAgainstAFoeAtHalfHealth(t *testing.T) {
	defenseSpecs(t)
	hunter := classed("stalker", 12)
	foe := classed("", 12)
	foe.Health = 50
	assert.Equal(t, 12, classBlowDamage(hunter, foe, 10), "+20%")
	foe.Health = 51
	assert.Equal(t, 10, classBlowDamage(hunter, foe, 10))
}

func TestHexedFoesTakeMoreFromAHag(t *testing.T) {
	defenseSpecs(t)
	hag := classed("hag", 12)
	foe := classed("", 12)
	assert.Equal(t, 10, classBlowDamage(hag, foe, 10))
	foe.AddBuff(status.Asleep, true)
	assert.Equal(t, 12, classBlowDamage(hag, foe, 10), "+15%, rounded")
}

func TestBlockAndParryAddTheClassPoints(t *testing.T) {
	defenseSpecs(t)
	stockDefenses(t)
	plain, drilled := classed("", 20), classed("duelist", 20)
	atk := classed("", 20)
	assert.Equal(t, parryChance(plain, atk, 5)+4, parryChance(drilled, atk, 5), "Parrying drill is +4")
	assert.Equal(t, blockChance(plain, atk), blockChance(drilled, atk), "a Duelist's block is its own")
}

func TestDreadSpoilsTheAttackersAim(t *testing.T) {
	defenseSpecs(t)
	stockDefenses(t)
	dread := classed("dread-knight", 40)
	plain := classed("", 40)
	atk := classed("", 40)
	assert.Less(t, attackEdge(atk, dread), attackEdge(atk, plain))
}

func TestAurasAddEvasionAndArmor(t *testing.T) {
	defenseSpecs(t)
	c := classed("", 20)
	base, armor := c.Evasion(), c.GetDefense()
	c.Aura = characters.ClassAura{Evasion: 3, Resolve: 10}
	assert.Equal(t, base+3, c.Evasion())
	assert.Equal(t, armor+10, c.GetDefense())
}

func TestAWardAbsorbsABlowUpToItsSizeAndIsSpent(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 0, 0, 0)
	base := strikeAt(armed(edgeSwordID), armed(0))
	require.True(t, base.Hit)

	target := armed(0)
	target.RTState().Ward, target.RTState().WardCap = 1, 1000
	r := strikeAt(armed(edgeSwordID), target)
	assert.True(t, r.Hit)
	assert.Zero(t, r.DamageToTarget, "a big ward takes the whole blow")
	assert.Zero(t, target.RT.Ward, "and is spent")
	assert.Equal(t, base.DamageToTarget, strikeAt(armed(edgeSwordID), target).DamageToTarget, "the next blow lands whole")

	small := armed(0)
	small.RTState().Ward, small.RTState().WardCap = 2, 1
	r = strikeAt(armed(edgeSwordID), small)
	assert.Equal(t, max(0, base.DamageToTarget-1), r.DamageToTarget, "a small ward takes its size")
	assert.Equal(t, 1, small.RT.Ward, "a two-blow ward has one left")
}

func TestDivineShieldIgnoresTheFirstBlowOfABattleOnly(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 0, 0, 0)
	paladin := armed(0)
	paladin.Level = 60
	paladin.SetClassState("paladin", nil)
	paladin.RTState()
	assert.Zero(t, strikeAt(armed(edgeSwordID), paladin).DamageToTarget)
	assert.True(t, paladin.RT.ShieldUsed)
	assert.Positive(t, strikeAt(armed(edgeSwordID), paladin).DamageToTarget, "only once a battle")
	paladin.EndFightRT()
	assert.False(t, paladin.RT.ShieldUsed, "a new battle renews it")
}
