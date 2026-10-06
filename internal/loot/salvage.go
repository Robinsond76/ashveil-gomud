package loot

import "github.com/GoMudEngine/GoMud/internal/items"

// Phase 36c salvage: a smith turns unwanted gear into the materials it was
// made of. The tier picks the material, weight picks how much of it, and
// quality and rarity add pieces. Salvaged gear is gone, so a smith is also
// an economy sink for gear nobody wants.

// Material item ids (the 36b goods).
const (
	MaterialScrapIron     = 212
	MaterialIronOre       = 220
	MaterialSteelIngot    = 221
	MaterialRunestone     = 222
	MaterialAshwood       = 223
	MaterialSilk          = 224
	MaterialTannedHide    = 225
	maxSalvagePerMaterial = 6
)

// SalvagePart is some number of one material.
type SalvagePart struct {
	ItemID int
	Count  int
}

// SalvageLine names the material line an item is made of: "metal", "wood",
// "leather", "cloth", or "" when a smith has nothing to take from it
// (jewellery, packs, lanterns, and unaudited legacy armor).
func SalvageLine(spec items.ItemSpec) string {
	switch {
	case spec.Type == items.Weapon:
		switch spec.Family {
		case "staff", "bow", "crossbow":
			return "wood"
		}
		return "metal"
	case spec.IsShield():
		return "metal"
	case spec.IsArmor():
		switch spec.Family {
		case "cloth":
			return "cloth"
		case "leather":
			return "leather"
		case "medium", "heavy":
			return "metal"
		}
	}
	return ""
}

// SalvageYield is what a smith takes out of the item, or nil when it
// holds nothing salvageable. It reads the item's roll even when it is
// unidentified: quality and rarity are visible on the item, affixes are not
// part of the price.
func SalvageYield(item items.Item) []SalvagePart {
	spec := item.GetSpec()
	line := SalvageLine(spec)
	if line == "" {
		return nil
	}
	tier := max(1, spec.Tier)
	if item.IsRolled() && item.Loot.Tier > 0 {
		tier = item.Loot.Tier
	}

	count := 1
	switch w := spec.Weight; {
	case w >= 4000:
		count = 3
	case w >= 1500:
		count = 2
	}

	var material int
	switch line {
	case "metal":
		switch {
		case tier >= 3:
			material = MaterialSteelIngot
			count = max(1, count-1) // an ingot is dense
		case tier == 2:
			material = MaterialIronOre
		default:
			material = MaterialScrapIron
		}
	case "wood":
		material = MaterialAshwood
	case "leather":
		material = MaterialTannedHide
	case "cloth":
		// Only a robe or tunic holds a bolt's worth; a cap or glove is scraps.
		if spec.Weight < 700 {
			return nil
		}
		material = MaterialSilk
		count = 1
	}

	extra := 0
	switch item.Loot.Quality {
	case items.QualitySuperior:
		extra++
	case items.QualityExquisite:
		extra += 2
	}
	if item.IsRolled() {
		switch item.Loot.Rarity {
		case items.RarityRare:
			extra++
		case items.RarityEpic:
			extra += 2
		case items.RarityLegendary, items.RaritySet:
			extra += 3
		}
	}
	count = min(count+extra, maxSalvagePerMaterial)

	parts := []SalvagePart{{ItemID: material, Count: count}}
	if tier >= 4 || (item.IsRolled() && item.Loot.Rarity.Rank() >= items.RarityEpic.Rank()) {
		parts = append(parts, SalvagePart{ItemID: MaterialRunestone, Count: 1})
	}
	// Phase 36d: a plain piece never breaks down into more than it is worth,
	// so the runestone shard of a tier 4+ piece goes to pieces that can bear
	// it (a masterwork glove is worth less than the shard).
	if !item.IsRolled() && len(parts) == 2 && partsWorth(parts) > spec.Value {
		parts = parts[:1]
	}
	return parts
}

// materialWorth is the shipped value of each salvage material; a test holds
// it equal to the item files.
var materialWorth = map[int]int{
	MaterialScrapIron: 7, MaterialIronOre: 18, MaterialSteelIngot: 45, MaterialRunestone: 60,
	MaterialAshwood: 14, MaterialSilk: 38, MaterialTannedHide: 16,
}

func partsWorth(parts []SalvagePart) int {
	worth := 0
	for _, p := range parts {
		worth += materialWorth[p.ItemID] * p.Count
	}
	return worth
}
