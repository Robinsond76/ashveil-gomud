package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInventoryMemberOf (Phase 32g): a member's worn and carried items
// as data, with the reference a command resolves to exactly that item,
// weights, uses, and the pack that counts.
func TestInventoryMemberOf(t *testing.T) {
	for _, s := range []items.ItemSpec{
		{ItemId: 989101, Name: "iron sword", Weight: 1500, Type: items.Weapon},
		{ItemId: 989102, Name: "satchel", Weight: 600, CarryBonus: 5000},
		{ItemId: 989103, Name: "waterskin", Weight: 1000, Uses: 5, Subtype: items.Drinkable},
	} {
		spec := s
		items.SetTestItemSpec(&spec)
		t.Cleanup(func() { items.RemoveTestItemSpec(spec.ItemId) })
	}
	sword := items.New(989101)
	satchel := items.New(989102)
	water := items.New(989103)
	water.Uses = 3
	state := MemberState{Items: []items.Item{satchel, water}}
	state.Equipment.Weapon = sword

	m := InventoryMemberOf(CompanionMemberKey(2), "Tamsin", state)
	assert.Equal(t, CompanionMemberKey(2), m.Key)
	assert.Equal(t, "Tamsin", m.Name)
	assert.Equal(t, 1500+600+1000, m.Grams)
	assert.Equal(t, "satchel", m.Pack)
	assert.Equal(t, 5000, m.PackBonusGrams)
	require.Len(t, m.Worn, 1)
	assert.Equal(t, InventoryItem{Ref: sword.ShorthandId(), Name: "iron sword", Grams: 1500, Count: 1, Type: "weapon", Slot: "weapon"}, m.Worn[0])
	require.Len(t, m.Carried, 2)
	assert.Equal(t, water.ShorthandId(), m.Carried[1].Ref, "a reference to exactly this waterskin")
	assert.Equal(t, 3, m.Carried[1].Uses)
	assert.Equal(t, 5, m.Carried[1].UsesMax)
	assert.Equal(t, "drinkable", m.Carried[1].Subtype)

	empty := InventoryMemberOf(LeaderMemberKey, "Dain", MemberState{})
	assert.Empty(t, empty.Pack)
	assert.Equal(t, []InventoryItem{}, empty.Worn, "empty lists, never null")
	assert.Equal(t, []InventoryItem{}, empty.Carried)
}
