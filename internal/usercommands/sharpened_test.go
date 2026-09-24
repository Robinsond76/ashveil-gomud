package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// Phase 23b surfaces: a whetstone's uses and a weapon's edge.
const (
	sharpTestStone = 9231
	sharpTestSword = 9232
)

func sharpenedSpecs(t *testing.T) {
	t.Helper()
	for _, spec := range []items.ItemSpec{
		{ItemId: sharpTestStone, Name: "whetstone", Type: items.Object, Subtype: items.Mundane, Uses: 10},
		{ItemId: sharpTestSword, Name: "sword", Type: items.Weapon, Subtype: items.Slashing, Hands: 1},
	} {
		spec := spec
		items.SetTestItemSpec(&spec)
		t.Cleanup(func() { items.RemoveTestItemSpec(spec.ItemId) })
	}
}

func TestInventoryShowsWhetstoneUsesAndEdge(t *testing.T) {
	sharpenedSpecs(t)
	stone := items.New(sharpTestStone)
	stone.Uses = 7
	assert.Contains(t, formatInventoryItemName(stone), `(7)`)

	sword := items.New(sharpTestSword)
	assert.NotContains(t, formatInventoryItemName(sword), `sharp`)
	sword.Sharpen(1, 13)
	assert.Contains(t, formatInventoryItemName(sword), `(sharp: 13)`)
}

func TestConditionsShowsSharpened(t *testing.T) {
	sharpenedSpecs(t)
	user := users.NewUserRecord(7, 1)
	assert.Contains(t, buildConditionsPanel(user), `None`)

	user.Character.Equipment.Weapon = items.New(sharpTestSword)
	user.Character.Equipment.Weapon.Sharpen(1, 9)
	panel := buildConditionsPanel(user)
	assert.Contains(t, panel, `Sharpened`)
	assert.Contains(t, panel, `sword: +1 damage for 9 more strikes`)
	assert.NotContains(t, panel, `None`)
}
