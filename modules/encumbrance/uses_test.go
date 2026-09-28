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
