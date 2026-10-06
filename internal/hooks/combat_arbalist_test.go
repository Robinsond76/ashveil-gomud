package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/stretchr/testify/assert"
)

// Phase 39h: Steady Aim gives the next bolt +Attack unless a blow has landed
// on the Arbalist since its last one.
func TestSteadyAimEndsWhenABlowLandsOnTheArbalist(t *testing.T) {
	c := characters.New()
	c.HPArchetype = "arbalist"
	c.Level = 3
	rt := c.RTState()
	fx := c.ClassEffects()
	assert.Equal(t, 10, steadyAim(rt, fx), "level 3: +10 Attack while unhurt")

	// A miss, or a blow that did nothing, does not end it.
	arbalistBlow(statusHolder{char: c}, combat.AttackResult{Hit: false})
	arbalistBlow(statusHolder{char: c}, combat.AttackResult{Hit: true, DamageToTarget: 0})
	assert.Equal(t, 10, steadyAim(rt, fx))

	arbalistBlow(statusHolder{char: c}, combat.AttackResult{Hit: true, DamageToTarget: 4})
	assert.Zero(t, steadyAim(rt, fx), "struck since the last bolt: no bonus")

	// Another lineage never carries the flag.
	other := characters.New()
	other.HPArchetype = "warrior"
	other.RTState()
	arbalistBlow(statusHolder{char: other}, combat.AttackResult{Hit: true, DamageToTarget: 4})
	assert.False(t, other.RT.AimStruck)

	// Steady hands: +20.
	c.HPClass, c.Level = "sharpshooter", 15
	assert.Equal(t, 20, classes.EffectsForLineage("arbalist", "sharpshooter", 15, nil).Int(classes.SteadyAim))
}
