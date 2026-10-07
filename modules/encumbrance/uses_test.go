package encumbrance

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 32f: a half-drunk waterskin comes back out half-drunk.
func TestCargoPutTakeKeepsUses(t *testing.T) {
	user := testUser(t, 7)
	used := testItem(waterId)
	used.Uses = 2
	full := testItem(waterId)
	full.Uses = 3
	user.Character.Items = []items.Item{used, full}
	store := &fakeStore{}
	module := newTestModule(store, user)

	module.put(user, "waterskin")
	module.put(user, "waterskin")
	assert.ElementsMatch(t, []encumbrance.CargoStack{{ItemId: waterId, Count: 1, Uses: 2}, {ItemId: waterId, Count: 1}}, store.saved.Cargo[7].Stacks)
	assert.Contains(t, module.status(7), "waterskin x1 (2 uses left)")

	module.take(user, "waterskin")
	require.Len(t, user.Character.Items, 1)
	assert.Equal(t, 2, user.Character.Items[0].Uses, "the partly used one first, with its uses")
	module.take(user, "waterskin")
	assert.Equal(t, 3, user.Character.Items[1].Uses)
}

func TestConsumeCargoUse(t *testing.T) {
	user := testUser(t, 7)
	store := &fakeStore{}
	module := newTestModule(store, user)
	module.cargo[7] = encumbrance.Cargo{LeaderUserID: 7, Stacks: []encumbrance.CargoStack{{ItemId: waterId, Count: 1}}}

	require.NoError(t, module.ConsumeCargoUse(7, waterId))
	assert.Equal(t, []encumbrance.CargoStack{{ItemId: waterId, Count: 1, Uses: 2}}, store.saved.Cargo[7].Stacks, "saved")
	assert.Equal(t, module.CargoContents(7), store.saved.Cargo[7].Stacks)

	store.saveErr = assert.AnError
	assert.Error(t, module.ConsumeCargoUse(7, waterId))
	assert.Equal(t, 2, module.CargoContents(7)[0].Uses, "a failed save rolls back")

	store.saveErr = nil
	assert.ErrorIs(t, module.ConsumeCargoUse(7, rockId), encumbrance.ErrInsufficientCargo)
	assert.ErrorIs(t, module.ConsumeCargoUse(8, waterId), encumbrance.ErrInsufficientCargo)
	assert.Nil(t, module.CargoContents(8))
}

// Cargo saved before Phase 32f has no uses: it loads as full.
func TestStoredCargoWithoutUsesLoadsFull(t *testing.T) {
	var registry Registry
	require.NoError(t, decodeRegistry([]byte("cargo:\n  7:\n    leaderuserid: 7\n    stacks:\n    - itemid: 300\n      count: 2\n"), &registry))
	assert.Equal(t, []encumbrance.CargoStack{{ItemId: waterId, Count: 2}}, registry.Cargo[7].Stacks)
}

// Phase 36a: a cargo stack holds only an item id, so a rolled item would
// lose its quality and affixes in it. It stays in the pack instead.
func TestCargoPutRefusesRolledGear(t *testing.T) {
	user := testUser(t, 7)
	rolled := testItem(waterId)
	rolled.Loot = items.Rolled{Version: items.RollVersion, Quality: items.QualityFine, Rarity: items.RarityRare, Identified: true}
	user.Character.Items = []items.Item{rolled}
	store := &fakeStore{}
	module := newTestModule(store, user)

	assert.Contains(t, module.put(user, "waterskin"), "individually crafted gear")
	assert.Len(t, user.Character.Items, 1, "still in the pack")
	assert.Empty(t, store.saved.Cargo[7].Stacks, "nothing was stowed")
}

// Phase 71 review: a trophy enchant and a relic's awakening progress live on
// the item, so cargo (an item id and a count) would wipe them. The piece
// stays in the pack.
func TestCargoPutRefusesAnEnchantedOrWakingPiece(t *testing.T) {
	const trophyId = 98871
	items.SetTestItemSpec(&items.ItemSpec{ItemId: trophyId, Name: "test heart", Type: items.Commodity,
		Trophy: &items.TrophySpec{Part: items.TrophyHeart, Races: []string{"ogre"}, Chance: 20, Effects: map[string]int{"damage": 1}}})
	t.Cleanup(func() { items.RemoveTestItemSpec(trophyId) })
	for name, edit := range map[string]func(*items.Item){
		"enchanted": func(i *items.Item) { i.Trophy = trophyId },
		"waking":    func(i *items.Item) { i.Awaken = []int{2} },
	} {
		user := testUser(t, 7)
		piece := testItem(waterId)
		edit(&piece)
		user.Character.Items = []items.Item{piece}
		store := &fakeStore{}
		module := newTestModule(store, user)

		assert.Contains(t, module.put(user, "waterskin"), "cargo would lose", name)
		assert.Len(t, user.Character.Items, 1, name+": still in the pack")
		assert.Empty(t, store.saved.Cargo[7].Stacks, name+": nothing was stowed")
	}
}
