package loot

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// goodsValue is the shipped value of each material salvage can yield.
var goodsValue = map[int]int{
	MaterialScrapIron: 7, MaterialIronOre: 18, MaterialSteelIngot: 45, MaterialRunestone: 60,
	MaterialAshwood: 14, MaterialSilk: 38, MaterialTannedHide: 16,
}

func yieldWorth(parts []SalvagePart) int {
	worth := 0
	for _, p := range parts {
		worth += goodsValue[p.ItemID] * p.Count
	}
	return worth
}

func specItem(spec items.ItemSpec) items.Item { return items.Item{ItemId: spec.ItemId, Spec: &spec} }

// Every catalog piece salvages into something the matching line makes, a
// piece is never worth more in materials than its own value (so buying gear
// to salvage never pays), and gear nobody can salvage gives nothing.
func TestSalvageCatalogYieldsBelowItemValue(t *testing.T) {
	salvaged, nothing := 0, 0
	for _, spec := range shippedCatalog(t) {
		parts := SalvageYield(specItem(spec))
		if len(parts) == 0 {
			nothing++
			if line := SalvageLine(spec); line != "" {
				assert.Equal(t, "cloth", line, "%s yields nothing only when it is scraps of cloth", spec.Name)
				assert.Less(t, spec.Weight, 700, spec.Name)
			}
			continue
		}
		salvaged++
		assert.LessOrEqual(t, yieldWorth(parts), spec.Value, "%s (tier %d) is worth %d but salvages into %d", spec.Name, spec.Tier, spec.Value, yieldWorth(parts))
		for _, p := range parts {
			assert.Contains(t, goodsValue, p.ItemID, spec.Name)
			assert.Positive(t, p.Count, spec.Name)
		}
	}
	assert.Greater(t, salvaged, 80, "most of the catalog salvages")
	assert.Greater(t, nothing, 0, "cloth scraps give nothing")
}

func TestSalvageLinesAndTiers(t *testing.T) {
	for _, tc := range []struct {
		name     string
		spec     items.ItemSpec
		line     string
		material int
	}{
		{"t1 sword", items.ItemSpec{ItemId: 1, Type: items.Weapon, Family: "sword", Tier: 1, Weight: 1200}, "metal", MaterialScrapIron},
		{"t2 axe", items.ItemSpec{ItemId: 1, Type: items.Weapon, Family: "axe", Tier: 2, Weight: 1500}, "metal", MaterialIronOre},
		{"t3 glaive", items.ItemSpec{ItemId: 1, Type: items.Weapon, Family: "glaive", Tier: 3, Weight: 3400}, "metal", MaterialSteelIngot},
		{"bow", items.ItemSpec{ItemId: 1, Type: items.Weapon, Family: "bow", Tier: 2, Weight: 1000}, "wood", MaterialAshwood},
		{"staff", items.ItemSpec{ItemId: 1, Type: items.Weapon, Family: "staff", Tier: 1, Weight: 1900}, "wood", MaterialAshwood},
		{"leather jerkin", items.ItemSpec{ItemId: 1, Type: items.Body, Subtype: items.Wearable, Family: "leather", Tier: 2, Weight: 2000}, "leather", MaterialTannedHide},
		{"cloth robe", items.ItemSpec{ItemId: 1, Type: items.Body, Subtype: items.Wearable, Family: "cloth", Tier: 2, Weight: 700}, "cloth", MaterialSilk},
		{"heavy helm", items.ItemSpec{ItemId: 1, Type: items.Head, Subtype: items.Wearable, Family: "heavy", Tier: 3, Weight: 2000}, "metal", MaterialSteelIngot},
		{"shield", items.ItemSpec{ItemId: 1, Type: items.Offhand, Subtype: items.Wearable, Family: "kite shield", Tier: 1, Weight: 4500, DamageReduction: 7}, "metal", MaterialScrapIron},
		{"ring", items.ItemSpec{ItemId: 1, Type: items.Ring, Subtype: items.Wearable, Tier: 1, Weight: 10}, "", 0},
		{"unaudited armor", items.ItemSpec{ItemId: 1, Type: items.Body, Subtype: items.Wearable, Weight: 4000}, "", 0},
	} {
		assert.Equal(t, tc.line, SalvageLine(tc.spec), tc.name)
		parts := SalvageYield(specItem(tc.spec))
		if tc.line == "" {
			assert.Empty(t, parts, tc.name)
			continue
		}
		require.NotEmpty(t, parts, tc.name)
		assert.Equal(t, tc.material, parts[0].ItemID, tc.name)
	}

	scraps := items.ItemSpec{ItemId: 1, Type: items.Gloves, Subtype: items.Wearable, Family: "cloth", Tier: 1, Weight: 150}
	assert.Empty(t, SalvageYield(specItem(scraps)), "a cloth glove is scraps")
}

func TestSalvageQualityRarityAndRunestoneBonuses(t *testing.T) {
	spec := items.ItemSpec{ItemId: 1, Name: "sword", Type: items.Weapon, Family: "sword", Tier: 2, Weight: 1300, Value: 100}
	count := func(q items.Quality, r items.Rarity) int {
		it := specItem(spec)
		it.Loot = items.Rolled{Version: items.RollVersion, Quality: q, Rarity: r, Tier: 2}
		parts := SalvageYield(it)
		require.NotEmpty(t, parts)
		return parts[0].Count
	}
	base := count(items.QualityStandard, items.RarityCommon)
	assert.Equal(t, 1, base)
	assert.Equal(t, base+1, count(items.QualitySuperior, items.RarityCommon))
	assert.Equal(t, base+2, count(items.QualityExquisite, items.RarityCommon))
	assert.Equal(t, base+1, count(items.QualityStandard, items.RarityRare))
	assert.Equal(t, base+2, count(items.QualityStandard, items.RarityEpic))
	assert.Equal(t, base+3, count(items.QualityStandard, items.RarityLegendary))
	assert.Equal(t, base+3, count(items.QualityStandard, items.RaritySet))
	assert.LessOrEqual(t, count(items.QualityExquisite, items.RarityLegendary), maxSalvagePerMaterial)

	has := func(parts []SalvagePart, id int) bool {
		for _, p := range parts {
			if p.ItemID == id {
				return true
			}
		}
		return false
	}
	rare := specItem(spec)
	rare.Loot = items.Rolled{Version: items.RollVersion, Quality: items.QualityStandard, Rarity: items.RarityRare, Tier: 2}
	assert.False(t, has(SalvageYield(rare), MaterialRunestone), "a Rare gives no runestone")
	epic := specItem(spec)
	epic.Loot = items.Rolled{Version: items.RollVersion, Quality: items.QualityStandard, Rarity: items.RarityEpic, Tier: 2}
	assert.True(t, has(SalvageYield(epic), MaterialRunestone), "an Epic gives a runestone shard")
	tier4 := spec
	tier4.Tier = 4
	assert.True(t, has(SalvageYield(specItem(tier4)), MaterialRunestone), "a tier 4 piece gives a runestone shard")
}
