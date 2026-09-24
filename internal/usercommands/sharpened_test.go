package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	messages := captureLookMessages(t)
	_, err := Conditions(``, user, nil, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	panel := strings.Join(*messages, "\n")
	assert.Contains(t, panel, `Sharpened`)
	assert.Contains(t, panel, `sword: +1 damage for 9 more strikes`)
	assert.NotContains(t, panel, `None`)
}

// TestLookAtSharpenedBladeShowsEdge drives the real look command on a
// sharpened blade in the pack.
func TestLookAtSharpenedBladeShowsEdge(t *testing.T) {
	sharpenedSpecs(t)
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 1)
	user.Character.Name = "Tester"
	users.SetTestUser(user)
	carried := items.New(sharpTestSword)
	carried.Sharpen(1, 11)
	user.Character.StoreItem(carried)
	room := testRoom()
	room.Tags = append(room.Tags, rooms.TagLit)
	messages := captureLookMessages(t)

	_, err := Look("sword", user, room, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*messages, "\n"), "Its edge is honed: +1 damage for its next 11 strikes.")
}
