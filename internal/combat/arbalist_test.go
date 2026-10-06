package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	arbalistBowID   = 90250
	arbalistPlateID = 90251
)

// Phase 39h: a Piercing Bolt ignores part of the target's armor, only for the
// blow that carries it, and a foe's armor wears down by what its bolts shred.
func TestPiercingBoltIgnoresArmorForOneBlow(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 0, 0, 0)
	items.SetTestItemSpec(&items.ItemSpec{ItemId: arbalistBowID, Name: "test crossbow", Type: items.Weapon, Subtype: items.Shooting, Hands: 2,
		Damage: items.Damage{DiceRoll: "20d1", Attacks: 1, DiceCount: 20, SideCount: 1}})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: arbalistPlateID, Name: "test plate", Type: items.Body, DamageReduction: 60})
	t.Cleanup(func() { items.RemoveTestItemSpec(arbalistBowID); items.RemoveTestItemSpec(arbalistPlateID) })
	armored := func() *characters.Character { c := armed(0); c.Equipment.Body = items.New(arbalistPlateID); return c }
	shooter := func() *characters.Character { c := armed(arbalistBowID); c.RTState(); return c }
	require.Equal(t, 60, armored().GetDefense())

	total := func(pierce int) int {
		sum := 0
		for i := 0; i < 400; i++ {
			s := shooter()
			s.RT.BlowPierce = pierce
			sum += strikeAt(s, armored()).DamageToTarget
		}
		return sum
	}
	plain, half, all := total(0), total(50), total(100)
	assert.Greater(t, half, plain+400*2, "half the armor ignored: a harder blow")
	assert.Greater(t, all, half+400*2, "all of it: harder still")
	assert.Equal(t, 400*20, all, "no armor left to turn the blow aside")
}

func TestShreddedArmorWearsDownThenFloorsAtNothing(t *testing.T) {
	defenseSpecs(t)
	items.SetTestItemSpec(&items.ItemSpec{ItemId: arbalistPlateID, Name: "test plate", Type: items.Body, DamageReduction: 60})
	t.Cleanup(func() { items.RemoveTestItemSpec(arbalistPlateID) })
	foe := armed(0)
	foe.Equipment.Body = items.New(arbalistPlateID)
	assert.Equal(t, 60, foe.GetDefense())
	foe.RTState().Shred = 30
	assert.Equal(t, 30, foe.GetDefense())
	foe.RT.Shred = 45
	assert.Equal(t, 15, foe.GetDefense())
	foe.RT.Shred = 90
	assert.Zero(t, foe.GetDefense(), "never below no armor")
	foe.EndFightRT()
	assert.Equal(t, 60, foe.GetDefense(), "the battle's end restores it")
}
