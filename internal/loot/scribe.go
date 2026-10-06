package loot

import "github.com/GoMudEngine/GoMud/internal/items"

// Phase 36a Scribe: the trainable caster skill (ranks 1-4) that reads a
// Rare or better item's hidden affixes.
const (
	ScribeSkill   = "scribe"
	ScribeMaxRank = 4
)

// ScribeCovers reports whether a Scribe of the rank can identify an item of
// the rarity: rank 1 Rare, rank 2 Epic, rank 3 Legendary and Set. Rank 4
// covers the same and also shows each affix's tier and range, and the
// item's source.
func ScribeCovers(rank int, r items.Rarity) bool {
	switch r {
	case items.RarityRare:
		return rank >= 1
	case items.RarityEpic:
		return rank >= 2
	case items.RarityLegendary, items.RaritySet:
		return rank >= 3
	}
	return false
}

// ScribeRankFor is the lowest Scribe rank that identifies the rarity.
func ScribeRankFor(r items.Rarity) int {
	for rank := 1; rank <= ScribeMaxRank; rank++ {
		if ScribeCovers(rank, r) {
			return rank
		}
	}
	return 0
}

// ScribeManaCost is the mana a field identification spends: about 8 for a
// Rare, 15 for an Epic, 25 for a Legendary or Set.
func ScribeManaCost(r items.Rarity) int {
	switch r {
	case items.RarityRare:
		return 8
	case items.RarityEpic:
		return 15
	case items.RarityLegendary, items.RaritySet:
		return 25
	}
	return 0
}

// IdentifyFee is what a merchant charges to read an unidentified item
// (Phase 36c): 60 gold for a Rare, 150 for an Epic, 400 for a Legendary or
// Set. Cheaper than the gold a good find is worth, dearer than a camp rest.
func IdentifyFee(r items.Rarity) int {
	switch r {
	case items.RarityRare:
		return 60
	case items.RarityEpic:
		return 150
	case items.RarityLegendary, items.RaritySet:
		return 400
	}
	return 20
}
