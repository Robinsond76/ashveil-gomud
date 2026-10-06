package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

// Phase 36d: a Legendary's signature reaches the combat formulas through
// the wearer's class effects, with no class at all.
func TestLegendaryReapingRaisesBlowsAgainstAWoundedFoe(t *testing.T) {
	defenseSpecs(t)
	const reaperID = 99701
	items.SetTestItemSpec(&items.ItemSpec{ItemId: reaperID, Name: "test reaper", Type: items.Weapon, Subtype: items.Slashing, Hands: 2, Tier: 6,
		Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 12},
		Relic:  &items.RelicSpec{Signature: "Reaping", Effects: map[string]int{classes.Wounded: 20}, ILvl: 30, Mob: 1, Chance: 5}})
	t.Cleanup(func() { items.RemoveTestItemSpec(reaperID) })

	reaper := classed("", 30)
	foe := classed("", 30)
	foe.Health = 50
	assert.Equal(t, 10, classBlowDamage(reaper, foe, 10), "without the relic nothing changes")

	reaper.Equipment.Weapon = items.New(reaperID)
	assert.Equal(t, 12, classBlowDamage(reaper, foe, 10), "+20% against a foe at half health")
	foe.Health = 51
	assert.Equal(t, 10, classBlowDamage(reaper, foe, 10), "not above half")

	reaper.Equipment.Weapon = items.Item{}
	foe.Health = 50
	assert.Equal(t, 10, classBlowDamage(reaper, foe, 10), "and gone when it is put away")
}
