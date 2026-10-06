package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 36a: rolled gear keeps its roll through company equip, compare and
// remove, and the level requirement refuses a member who is too low, with
// the reason, through both the equip and compare commands.
func TestCompanyEquipRefusesRolledGearAboveMemberLevelAndKeepsTheRoll(t *testing.T) {
	b := equipmentBrawl(t)
	rolled := items.New(10004)
	rolled.ApplyRoll(items.Rolled{
		Version: items.RollVersion, Tier: 1, ILvl: 25, Quality: items.QualityFine, Rarity: items.RarityRare, Identified: true,
		LevelReq: 20, BaseValue: 5, Name: "Gloomfang",
		Affixes: []items.RolledAffix{{ID: "strength", Label: "Mighty", Mechanic: "statmod:strength", Value: 3, Tier: 2, MinValue: 2, MaxValue: 4}},
	})
	b.aria.Character.Items = []items.Item{rolled}
	companion := b.companion(1)
	companion.Character.Level = 5

	assert.Contains(t, b.cmd("company", "compare #1 "+rolled.ShorthandId()), "must be level 20")
	assert.Contains(t, b.cmd("company", "equip #1 "+rolled.ShorthandId()), "must be level 20")
	assert.NotEqual(t, rolled.UUID, companion.Character.Equipment.Weapon.UUID, "nothing was assigned")
	require.Len(t, b.aria.Character.Items, 1, "the item stays in the cargo")

	companion.Character.Level = 20
	require.Contains(t, b.cmd("company", "equip #1 "+rolled.ShorthandId()), "equipment updated")
	worn := companion.Character.Equipment.Weapon
	assert.Equal(t, rolled.UUID, worn.UUID)
	assert.Equal(t, rolled.Loot, worn.Loot, "the roll moves with the instance")
	assert.Equal(t, 3, worn.GetSpec().StatMods.Get("strength"))

	require.Contains(t, b.cmd("company", "remove #1 weapon"), "equipment updated")
	var back *items.Item
	for i := range b.aria.Character.Items {
		if b.aria.Character.Items[i].UUID == rolled.UUID {
			back = &b.aria.Character.Items[i]
		}
	}
	require.NotNil(t, back, "back in the cargo")
	assert.Equal(t, rolled.Loot, back.Loot)
	assert.Equal(t, rolled.GetSpec().Damage, back.GetSpec().Damage)
}
