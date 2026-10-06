package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

// Phase 38c2: the rogue and ranger elites' effects reach the combat formulas.

func TestCoupDeGraceFellsAFoeBelowTwentyPercentAndDoublesAgainstABoss(t *testing.T) {
	defenseSpecs(t)
	night := classed("nightblade", 60)
	foe := classed("", 30)
	assert.Equal(t, 5, coupDamage(night, foe, 5, 30), "a healthy foe takes the blow")
	assert.Equal(t, 19, coupDamage(night, foe, 5, 19), "below 20% it falls outright")
	foe.RTState().Boss = true
	assert.Equal(t, 10, coupDamage(night, foe, 5, 19), "a boss takes double instead")
	assert.Equal(t, 5, coupDamage(classed("assassin", 45), foe, 5, 19), "no Coup before the elite's last rank")
}

func TestSpoiledBlowsLandAtHalfDamage(t *testing.T) {
	defenseSpecs(t)
	foe := classed("", 10)
	attacker := classed("", 10)
	assert.Equal(t, 10, classBlowDamage(attacker, foe, 10))
	attacker.RTState().Spoil = 50
	assert.Equal(t, 5, classBlowDamage(attacker, foe, 10))
}

func TestDeathMarkAddsDamageOnlyAgainstTheMarkedFoe(t *testing.T) {
	defenseSpecs(t)
	night := classed("nightblade", 30)
	marked, other := classed("", 10), classed("", 10)
	night.RTState().DeathMark = marked.RTState()
	got := classBlowDamage(night, marked, 100)
	assert.Greater(t, got, 100, "the mark")
	assert.Equal(t, 100, classBlowDamage(night, other, 100), "the others take no extra")
}

func TestRangedEffectsNeedAShootingWeapon(t *testing.T) {
	defenseSpecs(t)
	m := classed("marksman", 60)
	assert.False(t, m.Shooting())
	assert.Zero(t, m.EliteCrit(classed("", 10)), "Called Shot is a bow's")
}

func TestStepReachOverrideComesFromTheClassState(t *testing.T) {
	c := characters.New()
	assert.False(t, c.ReachOverride())
	c.RTState().Stepping = true
	assert.True(t, c.ReachOverride())
}
