package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func inventoryText(t *testing.T, user *users.UserRecord, rest string) string {
	t.Helper()
	messages := captureLookMessages(t)
	_, err := Inventory(rest, user, nil, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	return tagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
}

// TestInventoryLeadsWithCompanyLoad (Phase 26a): load against capacity,
// the cargo part, supplies, and the item-count note, before the engine's
// equipment and Carrying line; a filter shows only matches, as before.
func TestInventoryLeadsWithCompanyLoad(t *testing.T) {
	useWorld(t, "default")
	useSummary(t, sampleSummary())
	for _, spec := range []items.ItemSpec{
		{ItemId: 9811, Name: "bread", Type: items.Food, Subtype: items.Edible, Nutrition: 20},
		{ItemId: 9812, Name: "waterskin", Type: items.Drink, Subtype: items.Drinkable, Hydration: 20},
	} {
		spec := spec
		items.SetTestItemSpec(&spec)
		t.Cleanup(func() { items.RemoveTestItemSpec(spec.ItemId) })
	}
	user := users.NewUserRecord(7, 1)
	user.Character.StoreItem(items.New(9811))
	user.Character.StoreItem(items.New(9811))
	user.Character.StoreItem(items.New(9812))

	text := inventoryText(t, user, "")
	assert.Contains(t, text, "Company load: Burdened, 8.0 of 10.0 kg (3.0 kg in cargo)")
	assert.Contains(t, text, "Supplies:     2 food, 1 drink")
	assert.Contains(t, text, "how many items you can hold")
	assert.Less(t, strings.Index(text, "Company load"), strings.Index(text, "Equipment"))
	assert.Contains(t, text, "Carrying:")

	found := inventoryText(t, user, "bread")
	assert.Contains(t, found, "Found in your bag")
	assert.NotContains(t, found, "Company load", "filters are unchanged")
}

// TestInventoryWithoutLoad: no encumbrance provider, no load lines.
func TestInventoryWithoutLoad(t *testing.T) {
	useWorld(t, "default")
	useSummary(t, companyview.Summary{Alive: 1})
	text := inventoryText(t, users.NewUserRecord(7, 1), "")
	assert.NotContains(t, text, "Company load")
	assert.Contains(t, text, "Equipment")
}

// Phase 36d: the inventory lists what worn relics do, and nothing for plain gear.
func TestInventoryListsWornRelicsAndSetProgress(t *testing.T) {
	useWorld(t, "default")
	useSummary(t, companyview.Summary{Alive: 1})
	specs := []*items.ItemSpec{
		{ItemId: 9821, Name: "Test Reaper", NameSimple: "reaper", Type: items.Weapon, Subtype: items.Slashing, Hands: 1, Tier: 6,
			Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 6},
			Relic:  &items.RelicSpec{Signature: "Reaping", Effects: map[string]int{classes.Wounded: 20}, ILvl: 30, Mob: 1, Chance: 5}},
		{ItemId: 9822, Name: "Test Helm", Type: items.Head, Subtype: items.Wearable, Tier: 5, DamageReduction: 3, Relic: &items.RelicSpec{Set: "invset", ILvl: 30, Mob: 1, Chance: 5}},
		{ItemId: 9823, Name: "Test Mail", Type: items.Body, Subtype: items.Wearable, Tier: 5, DamageReduction: 6, Relic: &items.RelicSpec{Set: "invset", ILvl: 30, Mob: 1, Chance: 5}},
		{ItemId: 9824, Name: "Test Boots", Type: items.Feet, Subtype: items.Wearable, Tier: 5, DamageReduction: 2, Relic: &items.RelicSpec{Set: "invset", ILvl: 30, Mob: 1, Chance: 5}},
	}
	for _, s := range specs {
		items.SetTestItemSpec(s)
	}
	items.SetTestSet(&items.SetSpec{SetId: "invset", Name: "Inventory Regalia", Bonuses: []items.SetBonus{
		{Pieces: 2, Effects: map[string]int{classes.Armor: 3}},
		{Pieces: 3, Effects: map[string]int{classes.Evasion: 4}},
	}})
	t.Cleanup(func() {
		for _, s := range specs {
			items.RemoveTestItemSpec(s.ItemId)
		}
		items.RemoveTestSet("invset")
	})
	user := users.NewUserRecord(7, 1)
	user.Character.Level = 30

	assert.NotContains(t, inventoryText(t, user, ""), "Relics worn", "plain gear lists nothing")

	user.Character.Equipment.Weapon = items.New(9821)
	user.Character.Equipment.Head = items.New(9822)
	user.Character.Equipment.Body = items.New(9823)
	text := inventoryText(t, user, "")
	assert.Contains(t, text, "Relics worn:")
	assert.Contains(t, text, "Test Reaper: Reaping (while worn): blows deal 20% more to a foe at or below half health.")
	assert.Contains(t, text, "Inventory Regalia: 2 of 3 pieces")
	assert.Contains(t, text, "+3% damage reduction on top of worn armor", "the bonus in force")
	assert.NotContains(t, text, "+4 Evasion", "the third piece's bonus is not yet")

	user.Character.Equipment.Feet = items.New(9824)
	assert.Contains(t, inventoryText(t, user, ""), "+4 Evasion")
	assert.NotContains(t, inventoryText(t, user, "reaper"), "Relics worn", "a filter shows only matches")
}
