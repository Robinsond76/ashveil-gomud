package gmcp

import (
	"encoding/json"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 36c: GMCP item lists carry a rolled item's name as players see it
// (the 36a deferral), while Name stays the plain name commands match, and a
// plain item's payload is unchanged.
func TestGMCPItemListsShowRolledNames(t *testing.T) {
	testItemSpecs(t, items.ItemSpec{ItemId: 989301, Name: "short sword", NameSimple: "sword", Type: items.Weapon, Subtype: items.Slashing, Hands: 1,
		Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 6}, Value: 100, Weight: 1000})

	rolled := items.New(989301)
	rolled.ApplyRoll(items.Rolled{Version: items.RollVersion, Tier: 1, ILvl: 5, Quality: items.QualityFine, Rarity: items.RarityRare, Identified: true, Name: "Gloomfang",
		Affixes: []items.RolledAffix{{ID: "keen", Label: "Keen", Mechanic: "statmod:damage", Value: 2}}})
	hidden := items.New(989301)
	hidden.ApplyRoll(items.Rolled{Version: items.RollVersion, Tier: 1, ILvl: 5, Quality: items.QualityStandard, Rarity: items.RarityEpic, Identified: false, Name: "Hidden"})
	plain := items.New(989301)

	got := newInventory_Item(rolled)
	assert.Equal(t, "short sword", got.Name, "Name stays the plain command name")
	assert.Equal(t, "Gloomfang, a fine short sword (rare)", got.Label)
	assert.Equal(t, "rare", got.Rarity)
	assert.False(t, got.Unidentified)

	got = newInventory_Item(hidden)
	assert.Equal(t, "epic", got.Rarity)
	assert.True(t, got.Unidentified)
	assert.Equal(t, "short sword (epic, unidentified)", got.Label, "an unread item shows no generated name")

	data, err := json.Marshal(newInventory_Item(plain))
	require.NoError(t, err)
	assert.NotContains(t, string(data), "label", "a plain item's payload is unchanged")
	assert.NotContains(t, string(data), "rarity")

	var eq characters.Worn
	eq.Weapon = rolled
	worn := buildWornPayload(eq)
	assert.Equal(t, "Gloomfang, a fine short sword (rare)", worn["weapon"].Label, "worn items carry it too")

	room := roomContentsItem(rolled)
	assert.Equal(t, "short sword", room.Name)
	assert.Equal(t, "Gloomfang, a fine short sword (rare)", room.Label)
	assert.Equal(t, "rare", room.Rarity)
	data, err = json.Marshal(roomContentsItem(plain))
	require.NoError(t, err)
	assert.NotContains(t, string(data), "label")

	// 36c review: a junk-marked item says so in its details.
	assert.NotContains(t, newInventory_Item(plain).Details, "junk")
	plain.Junk = true
	assert.Contains(t, newInventory_Item(plain).Details, "junk")

}

// Phase 36d: a relic's GMCP entry carries its signature (or set and bonuses)
// in words and its lore for the gear tooltip, and an ordinary item's payload
// is unchanged.
func TestGMCPItemListsCarryRelicText(t *testing.T) {
	testItemSpecs(t,
		items.ItemSpec{ItemId: 989401, Name: "Test Reaper", NameSimple: "reaper", Type: items.Weapon, Subtype: items.Slashing, Hands: 1, Tier: 6,
			Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 6}, Value: 100, Weight: 1000,
			Relic: &items.RelicSpec{Signature: "Reaping", Effects: map[string]int{classes.Wounded: 20}, Lore: "Old and grey.", ILvl: 30, Mob: 1, Chance: 5}},
		items.ItemSpec{ItemId: 989402, Name: "Test Helm", NameSimple: "helm", Type: items.Head, Subtype: items.Wearable, Tier: 5, DamageReduction: 3, Value: 100, Weight: 1000,
			Relic: &items.RelicSpec{Set: "gmcpset", ILvl: 30, Mob: 1, Chance: 5}},
		items.ItemSpec{ItemId: 989403, Name: "plain cap", Type: items.Head, Subtype: items.Wearable, Value: 10, Weight: 100},
	)
	items.SetTestSet(&items.SetSpec{SetId: "gmcpset", Name: "Gmcp Regalia", Bonuses: []items.SetBonus{
		{Pieces: 2, Effects: map[string]int{classes.Armor: 3}},
		{Pieces: 3, Effects: map[string]int{classes.Evasion: 4}},
	}})
	t.Cleanup(func() { items.RemoveTestSet("gmcpset") })

	blade := newInventory_Item(items.New(989401))
	assert.Equal(t, []string{"Reaping (while worn): blows deal 20% more to a foe at or below half health."}, blade.Relic)
	assert.Equal(t, "Old and grey.", blade.RelicLore)
	assert.Equal(t, "legendary", blade.Rarity, "a relic shows its rarity even before it is rolled")

	helm := newInventory_Item(items.New(989402))
	require.GreaterOrEqual(t, len(helm.Relic), 3)
	assert.Contains(t, helm.Relic[0], "Gmcp Regalia")
	assert.Contains(t, helm.Relic[1], "2 worn: +3% damage reduction")
	assert.Equal(t, "set", helm.Rarity)

	data, err := json.Marshal(newInventory_Item(items.New(989403)))
	require.NoError(t, err)
	assert.NotContains(t, string(data), "relic")

	// 36d review: the Company window's gear and cargo rows carry it too,
	// so a relic handed to a companion still says what it does.
	var eq characters.Worn
	eq.Weapon = items.New(989401)
	member := company.InventoryMemberOf("ysolde", "Ysolde", company.MemberState{Equipment: eq})
	worn := inventoryMemberOf(member).Worn
	require.NotEmpty(t, worn)
	assert.Equal(t, blade.Relic, worn[0].Relic)
	assert.Equal(t, helm.Relic, cargoItem(encumbrance.CargoStack{ItemId: 989402, Count: 1}).Relic)
	data, err = json.Marshal(cargoItem(encumbrance.CargoStack{ItemId: 989403, Count: 1}))
	require.NoError(t, err)
	assert.NotContains(t, string(data), "relic")
}

// Phase 67: a relic's awakenings and their progress ride the same relic
// lines, in the gear tooltip and the Company window's rows, so the web
// client shows them at any width the tooltip does.
func TestGMCPRelicLinesCarryAwakeningProgress(t *testing.T) {
	testItemSpecs(t,
		items.ItemSpec{ItemId: 989411, Name: "Test Waker", NameSimple: "waker", Type: items.Weapon, Subtype: items.Slashing, Hands: 1, Tier: 6,
			Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 6}, Value: 100, Weight: 1000,
			Relic: &items.RelicSpec{Signature: "Waking", Effects: map[string]int{classes.Wounded: 10}, ILvl: 30, Mob: 1, Chance: 5,
				Awakenings: []items.AwakeningSpec{
					{Name: "First Rite", Kind: items.AwakenSlay, Races: []string{"ogre"}, Count: 4, Target: "ogres", Effects: map[string]int{classes.Damage: 1}},
					{Name: "Last Rite", Kind: items.AwakenLair, Mob: 1, Target: "the warden", Effects: map[string]int{classes.Armor: 3}},
				}}},
	)
	itm := items.New(989411)
	itm.AdvanceAwakening(0, 3)
	got := newInventory_Item(itm).Relic
	require.Len(t, got, 3)
	assert.Contains(t, got[1], "Sleeping, First Rite (3 of 4): slay 4 ogres while it is worn")
	assert.Contains(t, got[2], "Sleeping, Last Rite (0 of 1): defeat the warden while it is worn")

	itm.AdvanceAwakening(0, 1)
	got = newInventory_Item(itm).Relic
	assert.Equal(t, "Awakened, First Rite: +1 damage on every landed blow.", got[1])

	var eq characters.Worn
	eq.Weapon = itm
	member := company.InventoryMemberOf("ysolde", "Ysolde", company.MemberState{Equipment: eq})
	worn := inventoryMemberOf(member).Worn
	require.NotEmpty(t, worn)
	assert.Equal(t, got, worn[0].Relic, "a companion's relic shows its own progress")
	assert.Len(t, cargoItem(encumbrance.CargoStack{ItemId: 989411, Count: 1}).Relic, 3, "a stack in cargo shows what it wakes")
}

// Phase 71: a trophy enchant on an ordinary (non-relic) piece rides the same
// relic lines to the web client, in the gear tooltip and the Company
// window's rows, and a plain piece carries none.
func TestGMCPEnchantedPlainPieceCarriesItsEnchantLine(t *testing.T) {
	testItemSpecs(t,
		items.ItemSpec{ItemId: 989421, Name: "plain blade", NameSimple: "blade", Type: items.Weapon, Subtype: items.Slashing, Hands: 1, Tier: 1,
			Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 6}, Value: 50, Weight: 1000},
		items.ItemSpec{ItemId: 989422, Name: "brute's heart", NameSimple: "heart", Type: items.Commodity, Value: 12,
			Trophy: &items.TrophySpec{Part: items.TrophyHeart, Races: []string{"ogre"}, Chance: 20, Effects: map[string]int{classes.Damage: 1}}},
	)
	itm := items.New(989421)
	assert.Empty(t, newInventory_Item(itm).Relic)
	require.NoError(t, itm.EnchantWithTrophy(989422))
	got := newInventory_Item(itm).Relic
	require.Len(t, got, 1)
	assert.Equal(t, "Enchanted with brute's heart (while worn): +1 damage on every landed blow.", got[0])
	assert.Contains(t, newInventory_Item(itm).Label, "(enchanted: brute's heart)", "review: lists tag a plain enchanted piece")

	var eq characters.Worn
	eq.Weapon = itm
	member := company.InventoryMemberOf("ysolde", "Ysolde", company.MemberState{Equipment: eq})
	worn := inventoryMemberOf(member).Worn
	require.NotEmpty(t, worn)
	assert.Equal(t, got, worn[0].Relic)
}
