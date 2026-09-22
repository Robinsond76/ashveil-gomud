package combat_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

func TestResolveReachUnarmedNoInnateReachIsNone(t *testing.T) {
	var c characters.Character
	assert.Equal(t, formationcombat.ReachNone, combat.ResolveReach(&c, false))
}

func TestResolveReachInnateReachWithNoWeaponIsExtended(t *testing.T) {
	var c characters.Character
	assert.Equal(t, formationcombat.ReachExtended, combat.ResolveReach(&c, true))
}

func TestResolveReachShootingWeaponIsAny(t *testing.T) {
	spec := &items.ItemSpec{ItemId: 9001, Subtype: items.Shooting}

	var c characters.Character
	c.Equipment.Weapon = items.Item{ItemId: spec.ItemId, Spec: spec}

	assert.Equal(t, formationcombat.ReachAny, combat.ResolveReach(&c, false))
}

func TestResolveReachPolearmWeaponIsExtended(t *testing.T) {
	spec := &items.ItemSpec{ItemId: 9002, Subtype: items.Stabbing, Reach: true}

	var c characters.Character
	c.Equipment.Weapon = items.Item{ItemId: spec.ItemId, Spec: spec}

	assert.Equal(t, formationcombat.ReachExtended, combat.ResolveReach(&c, false))
}

func TestResolveReachOrdinaryWeaponIsNone(t *testing.T) {
	spec := &items.ItemSpec{ItemId: 9003, Subtype: items.Slashing}

	var c characters.Character
	c.Equipment.Weapon = items.Item{ItemId: spec.ItemId, Spec: spec}

	assert.Equal(t, formationcombat.ReachNone, combat.ResolveReach(&c, false))
}
