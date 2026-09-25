package usercommands

import (
	"strings"
	"testing"

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
