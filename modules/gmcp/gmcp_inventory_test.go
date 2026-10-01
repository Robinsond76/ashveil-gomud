package gmcp

import (
	"encoding/json"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mount"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testItemSpecs(t *testing.T, specs ...items.ItemSpec) {
	t.Helper()
	for _, s := range specs {
		spec := s
		items.SetTestItemSpec(&spec)
		t.Cleanup(func() { items.RemoveTestItemSpec(spec.ItemId) })
	}
}

func inventoryTestSources(stacks []encumbrance.CargoStack) inventorySources {
	return inventorySources{
		companions: func(int) ([]company.InventoryMember, bool) {
			return []company.InventoryMember{
				{Key: company.CompanionMemberKey(1), Name: "<i>Tamsin</i>", Grams: 2000, Worn: []company.InventoryItem{}, Carried: []company.InventoryItem{}},
				{Key: company.CompanionMemberKey(2), Name: "Ysolde", Fallen: true, Worn: []company.InventoryItem{}, Carried: []company.InventoryItem{}},
			}, true
		},
		herd: func(int) []mount.HorseView {
			return []mount.HorseView{
				{ID: 1, Name: "pack horse", Kind: mount.KindPack, Saddle: "pack saddle", CapacityGrams: 100000},
				{ID: 2, Name: "riding horse", Kind: mount.KindRiding, Saddle: "riding saddle", CapacityGrams: 10000},
			}
		},
		cargo: func(int) []encumbrance.CargoStack { return stacks },
		load: func(int) (encumbrance.Load, bool) {
			return encumbrance.Load{PersonalGrams: 4000, CompanionGrams: 2000, CargoGrams: 2800, CapacityGrams: 145000, MemberCapacityGrams: 45000, MountCapacityGrams: 100000}, true
		},
	}
}

func inventoryUser(t *testing.T) *users.UserRecord {
	testItemSpecs(t,
		items.ItemSpec{ItemId: 989201, Name: "waterskin", Weight: 1000, Uses: 5, Subtype: items.Drinkable},
		items.ItemSpec{ItemId: 989202, Name: "seared meat", Weight: 300, Subtype: items.Edible},
	)
	u := users.NewUserRecord(7, 1)
	u.Character.Name = "Dain"
	water := items.New(989201)
	water.Uses = 3
	u.Character.Items = []items.Item{water}
	return u
}

// TestCompanyInventoryPayload (Phase 32g): the Inventory tab's data: the
// load split, the player first then each companion, horses, and cargo
// stacks with command references and uses; names travel as data.
func TestCompanyInventoryPayload(t *testing.T) {
	u := inventoryUser(t)
	src := inventoryTestSources([]encumbrance.CargoStack{{ItemId: 989202, Count: 6}, {ItemId: 989201, Count: 1, Uses: 2}})
	data, err := json.Marshal(buildInventoryPayload(u, src))
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, map[string]any{"total_g": 8800.0, "capacity_g": 145000.0, "member_capacity_g": 45000.0, "mount_capacity_g": 100000.0, "cargo_g": 2800.0}, got["load"])
	assert.Equal(t, true, got["companions_known"])
	members := got["members"].([]any)
	require.Len(t, members, 3)
	you := members[0].(map[string]any)
	assert.Equal(t, "leader", you["key"])
	assert.Equal(t, "Dain", you["name"])
	carried := you["carried"].([]any)
	require.Len(t, carried, 1)
	water := carried[0].(map[string]any)
	assert.Equal(t, u.Character.Items[0].ShorthandId(), water["ref"])
	assert.Equal(t, 3.0, water["uses"])
	assert.Equal(t, 5.0, water["uses_max"])
	assert.Equal(t, "<i>Tamsin</i>", members[1].(map[string]any)["name"])
	assert.Equal(t, true, members[2].(map[string]any)["fallen"])

	horses := got["horses"].([]any)
	require.Len(t, horses, 2)
	assert.Equal(t, map[string]any{"id": 1.0, "name": "pack horse", "kind": "pack", "saddle": "pack saddle", "capacity_g": 100000.0, "rides": false}, horses[0])
	assert.Equal(t, true, horses[1].(map[string]any)["rides"])

	cargo := got["cargo"].([]any)
	require.Len(t, cargo, 2)
	meat := cargo[0].(map[string]any)
	assert.Equal(t, "!989202", meat["ref"], "cargo take matches a stack by item id")
	assert.Equal(t, 6.0, meat["count"])
	assert.Equal(t, 300.0, meat["grams"])
	assert.Equal(t, 2.0, cargo[1].(map[string]any)["uses"])
}

// TestCompanyInventoryUnknowns: without a company provider the player's
// own share is still sent, marked so.
func TestCompanyInventoryUnknowns(t *testing.T) {
	u := inventoryUser(t)
	src := inventorySources{
		companions: func(int) ([]company.InventoryMember, bool) { return nil, false },
		herd:       func(int) []mount.HorseView { return nil },
		cargo:      func(int) []encumbrance.CargoStack { return nil },
		load:       func(int) (encumbrance.Load, bool) { return encumbrance.Load{}, false },
	}
	data, _ := json.Marshal(buildInventoryPayload(u, src))
	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Nil(t, got["load"])
	assert.Equal(t, false, got["companions_known"])
	assert.Len(t, got["members"].([]any), 1)
	assert.Equal(t, []any{}, got["horses"])
	assert.Equal(t, []any{}, got["cargo"])
}

// TestCompanyExtrasSendOnlyOnChange: an extra message goes once, again
// only when it changes, on its own, and again after forget.
func TestCompanyExtrasSendOnlyOnChange(t *testing.T) {
	f, out := testFeed()
	body := `{"a":1}`
	f.extras = []companyExtra{{module: "Company.Test", build: func(*users.UserRecord) []byte { return []byte(body) }}}
	u := users.NewUserRecord(7, 1)
	f.updateExtras(u)
	f.updateExtras(u)
	require.Len(t, *out, 1)
	assert.Equal(t, "Company.Test", (*out)[0].module)
	body = `{"a":2}`
	f.updateExtras(u)
	require.Len(t, *out, 2)
	f.forget(7)
	f.updateExtras(u)
	assert.Len(t, *out, 3, "re-sent after login or copyover")

	f.accepting = func(int) bool { return false }
	body = `{"a":3}`
	f.updateExtras(u)
	assert.Len(t, *out, 3, "nothing to a connection without GMCP")
}

// TestBackpackSummaryWeights (Phase 32g): the gear window's header reads
// the player's own gear weight against the company's load and capacity,
// which 32f's weight limit replaced the item count with.
func TestBackpackSummaryWeights(t *testing.T) {
	u := inventoryUser(t) // a 1 kg waterskin
	s := backpackSummary(u, func(int) (encumbrance.Load, bool) {
		return encumbrance.Load{PersonalGrams: 1000, CargoGrams: 3000, CapacityGrams: 25000}, true
	})
	assert.Equal(t, GMCPCharModule_Payload_Inventory_Backpack_Summary{Count: 1, WeightG: 1000, LoadG: 4000, CapacityG: 25000, Burden: "unburdened"}, s)

	unknown := backpackSummary(u, func(int) (encumbrance.Load, bool) { return encumbrance.Load{}, false })
	assert.Equal(t, 1000, unknown.WeightG)
	assert.Zero(t, unknown.CapacityG, "omitted when the load can't be read")
}

// TestBackpackSummaryBurden (Phase 30g3): the summary names how burdened
// the player's own load leaves them, a word the web Overview shows; it
// rides on every Char.Inventory, Char.Inventory.Backpack, and
// Char.Inventory.Backpack.Summary update.
func TestBackpackSummaryBurden(t *testing.T) {
	u := inventoryUser(t)
	testItemSpecs(t, items.ItemSpec{ItemId: 989203, Name: "anvil", Weight: 40000})
	g := &GMCPCharModule{}
	summary := func() GMCPCharModule_Payload_Inventory_Backpack_Summary {
		t.Helper()
		data, name := g.GetCharNode(u, `Char.Inventory.Backpack.Summary`)
		require.Equal(t, `Char.Inventory.Backpack.Summary`, name)
		s, ok := data.(GMCPCharModule_Payload_Inventory_Backpack_Summary)
		require.True(t, ok)
		return s
	}
	assert.Equal(t, "unburdened", summary().Burden)
	u.Character.Items = append(u.Character.Items, items.New(989203))
	assert.Equal(t, "heavily burdened", summary().Burden)
	data, _ := g.GetCharNode(u, `Char.Inventory.Backpack`)
	backpack, ok := data.(*GMCPCharModule_Payload_Inventory_Backpack)
	require.True(t, ok)
	assert.Equal(t, "heavily burdened", backpack.Summary.Burden)
}

// TestExtrasFollowASnapshot (32g review finding 1): the client stores the
// extras under Company, which a new snapshot replaces, so every extra is
// sent again after one, unchanged or not, and after it.
func TestExtrasFollowASnapshot(t *testing.T) {
	f, out := testFeed()
	f.extras = []companyExtra{{module: "Company.Inventory", build: func(*users.UserRecord) []byte { return []byte(`{"a":1}`) }}}
	u := users.NewUserRecord(7, 1)
	s := sampleCompany()
	refresh := func() { f.update(7, s); f.updateExtras(u) }
	refresh()
	refresh()
	require.Len(t, *out, 2, "nothing new: nothing sent")
	s.Checkpoint = "Elsewhere"
	refresh()
	require.Len(t, *out, 4)
	assert.Equal(t, "Company", (*out)[2].module)
	assert.Equal(t, "Company.Inventory", (*out)[3].module, "the unchanged extra follows the snapshot")
	s.Leader.HP = 1
	refresh()
	require.Len(t, *out, 5)
	assert.Equal(t, "Company.Vitals", (*out)[4].module, "a vitals update replaces nothing: no extras")
}
