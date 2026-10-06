package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	relicBladeID = 99601
	relicHelmID  = 99602
	relicMailID  = 99603
	relicBootsID = 99604
	relicPlainID = 99605
)

func relicGear(t *testing.T) {
	t.Helper()
	specs := []*items.ItemSpec{
		{ItemId: relicBladeID, Name: "relic blade", Type: items.Weapon, Subtype: items.Slashing, Hands: 1, Tier: 6, Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 6},
			Relic: &items.RelicSpec{Signature: "Reaping", Effects: map[string]int{classes.Wounded: 20, classes.Armor: 3, classes.Bargain: 1}, ILvl: 20, Mob: 1, Chance: 5}},
		{ItemId: relicHelmID, Name: "relic helm", Type: items.Head, Subtype: items.Wearable, Tier: 5, DamageReduction: 4, Relic: &items.RelicSpec{Set: "chartest", ILvl: 20, Mob: 1, Chance: 5}},
		{ItemId: relicMailID, Name: "relic mail", Type: items.Body, Subtype: items.Wearable, Tier: 5, DamageReduction: 9, Relic: &items.RelicSpec{Set: "chartest", ILvl: 20, Mob: 1, Chance: 5}},
		{ItemId: relicBootsID, Name: "relic boots", Type: items.Feet, Subtype: items.Wearable, Tier: 5, DamageReduction: 3, Relic: &items.RelicSpec{Set: "chartest", ILvl: 20, Mob: 1, Chance: 5}},
		{ItemId: relicPlainID, Name: "plain cap", Type: items.Head, Subtype: items.Wearable, DamageReduction: 2},
	}
	for _, s := range specs {
		items.SetTestItemSpec(s)
	}
	items.SetTestSet(&items.SetSpec{SetId: "chartest", Name: "Char Test", Bonuses: []items.SetBonus{
		{Pieces: 2, Effects: map[string]int{classes.Evasion: 4}},
		{Pieces: 3, Effects: map[string]int{classes.Attack: 5, classes.Evasion: 2}},
	}})
	t.Cleanup(func() {
		for _, s := range specs {
			items.RemoveTestItemSpec(s.ItemId)
		}
		items.RemoveTestSet("chartest")
	})
}

func wear(t *testing.T, c *Character, id int) {
	t.Helper()
	_, worn, reason := c.Wear(items.New(id))
	require.True(t, worn, reason)
}

// Phase 36d: a worn relic's signature and an active set's bonuses reach the
// wearer's class effects, through the real Wear and removal, for a
// character with no class at all, and fall away when the piece comes off.
func TestWornRelicsGrantClassEffects(t *testing.T) {
	relicGear(t)
	c := New()
	c.Level = 30
	assert.Nil(t, c.ClassEffects(), "plain gear and no class: no effects")

	wear(t, c, relicPlainID)
	assert.Nil(t, c.ClassEffects())
	gear, sets := c.WornGear()
	assert.Empty(t, gear)
	assert.Empty(t, sets)

	wear(t, c, relicBladeID)
	fx := c.ClassEffects()
	assert.Equal(t, 20, fx.Int(classes.Wounded))
	assert.Equal(t, 3, fx.Int(classes.Armor))
	assert.True(t, fx.Has(classes.Bargain))
	assert.Equal(t, c.Equipment.Weapon.GetSpec().DamageReduction+c.Equipment.Head.GetSpec().DamageReduction+3, c.GetDefense(),
		"the signature's armor adds to worn armor in the real defense")

	// A set builds piece by piece.
	wear(t, c, relicHelmID)
	assert.Zero(t, c.ClassEffects().Int(classes.Evasion), "one piece: no bonus")
	wear(t, c, relicMailID)
	assert.Equal(t, 4, c.ClassEffects().Int(classes.Evasion), "two pieces")
	wear(t, c, relicBootsID)
	fx = c.ClassEffects()
	assert.Equal(t, 6, fx.Int(classes.Evasion), "three pieces: both bonuses")
	assert.Equal(t, 5, fx.Int(classes.Attack))
	assert.Equal(t, 20, fx.Int(classes.Wounded), "and the signature still")
	_, sets = c.WornGear()
	require.Len(t, sets, 1)
	assert.Equal(t, 3, sets[0].Worn)

	// Taking a piece off drops what it gave, at once.
	c.Equipment.Feet = items.Item{}
	fx = c.ClassEffects()
	assert.Equal(t, 4, fx.Int(classes.Evasion))
	assert.Zero(t, fx.Int(classes.Attack))
	c.Equipment.Weapon = items.Item{}
	assert.Zero(t, c.ClassEffects().Int(classes.Wounded))
	c.Equipment.Head, c.Equipment.Body = items.Item{}, items.Item{}
	assert.Nil(t, c.ClassEffects())
}

// Gear adds to a real class's effects without changing what the class has.
func TestWornRelicsAddToAClassAndNeverChangeIt(t *testing.T) {
	relicGear(t)
	c := New()
	c.Level = 30
	c.SetClassState("warlord", nil)
	before := c.ClassEffects().Int(classes.Wounded)
	own := c.ClassEffects().Clone()
	wear(t, c, relicBladeID)
	assert.Equal(t, before+20, c.ClassEffects().Int(classes.Wounded))
	c.Equipment.Weapon = items.Item{}
	assert.Equal(t, own, c.ClassEffects(), "the class's own effects are what they were")
}

// Bargain from gear works through the real once-a-battle check.
func TestRelicBargainLeavesOneHealthOnce(t *testing.T) {
	relicGear(t)
	c := New()
	c.Level = 30
	dmg, saved := c.GuardFall(50, 10, false)
	assert.Equal(t, []any{50, ""}, []any{dmg, saved}, "without the relic nothing saves it")
	wear(t, c, relicBladeID)
	dmg, saved = c.GuardFall(50, 10, false)
	assert.Equal(t, []any{9, "bargain"}, []any{dmg, saved})
	dmg, saved = c.GuardFall(50, 10, false)
	assert.Equal(t, []any{50, ""}, []any{dmg, saved}, "once a battle")
}
