package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

const awakeningBladeID = 99611

// Phase 67: a relic that wakes while it is worn changes its wearer's
// effects at once, through the gear cache, and the change goes with the
// item when it comes off and is worn again.
func TestAnAwakeningWornRelicChangesTheWearersEffects(t *testing.T) {
	items.SetTestItemSpec(&items.ItemSpec{ItemId: awakeningBladeID, Name: "waking blade", Type: items.Weapon, Subtype: items.Slashing, Hands: 1, Tier: 6,
		Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 6},
		Relic: &items.RelicSpec{Signature: "Waking", Effects: map[string]int{classes.Wounded: 10}, ILvl: 20, Mob: 1, Chance: 5,
			Awakenings: []items.AwakeningSpec{
				{Name: "Edge", Kind: items.AwakenSlay, Races: []string{"ogre"}, Count: 2, Target: "ogres", Effects: map[string]int{classes.Damage: 1, classes.Armor: 2}},
			}}})
	t.Cleanup(func() { items.RemoveTestItemSpec(awakeningBladeID) })

	c := New()
	c.Level = 30
	wear(t, c, awakeningBladeID)
	assert.Equal(t, 10, c.ClassEffects().Int(classes.Wounded))
	assert.Zero(t, c.ClassEffects().Int(classes.Damage), "asleep: only the signature")
	defense := c.GetDefense()

	assert.False(t, c.Equipment.Weapon.AdvanceAwakening(0, 1))
	assert.Zero(t, c.ClassEffects().Int(classes.Damage), "half way is not awake")
	assert.True(t, c.Equipment.Weapon.AdvanceAwakening(0, 1))
	fx := c.ClassEffects()
	assert.Equal(t, 1, fx.Int(classes.Damage), "the woken power is in force without re-wearing")
	assert.Equal(t, 10, fx.Int(classes.Wounded), "and the signature stays")
	assert.Equal(t, defense+2, c.GetDefense(), "an awakened armor bonus reaches the real defense")

	// It comes off and goes back on with its progress.
	worn := c.Equipment.Weapon
	c.Equipment.Weapon = items.Item{}
	assert.Zero(t, c.ClassEffects().Int(classes.Damage))
	c.Equipment.Weapon = worn
	assert.Equal(t, 1, c.ClassEffects().Int(classes.Damage))
}
