package items

// Phase 36c: selling and salvaging rolled gear, and marking junk.

// UnidentifiedPricePct is the share of its likely worth a merchant pays for
// gear whose properties are not known: the gamble is theirs, so they pay
// less than it would fetch once read (help identify).
const UnidentifiedPricePct = 60

// unidentifiedRarityPct stands in for the hidden affixes: an unread item's
// Spec holds only its quality, so the merchant prices it as the average
// item of its rarity. It reads the rarity, never the affixes, so a price
// never gives away what a Scribe would find.
var unidentifiedRarityPct = map[Rarity]int{
	RarityRare:      180,
	RarityEpic:      260,
	RarityLegendary: 400,
	RaritySet:       400,
}

// MakeshiftMealItemId (Phase 56) is what a failed cooking experiment makes:
// plain food from the ingredients, never bought back.
const MakeshiftMealItemId = 30062

// MasterworkInstrumentTier is the instrument tier that is looted, never
// bought or crafted, and so may be sold for profit.
const MasterworkInstrumentTier = 4

// IsSpecialForSale is IsSpecial for merchants: a blob, spent uses and a
// hand-edited spec still keep an item from sale, but a rolled item's own
// override and roll do not (36a left rolled gear unsellable). A cooked meal
// is never bought (Phase 50 review): it is cooked from gathered or bought
// ingredients, so buying it back would turn cooking into profit. A recipe
// page (Phase 56) is knowledge, not goods, and is not bought either.
func (i *Item) IsSpecialForSale() bool {
	if i.GetSpec().Meal != "" || i.GetSpec().Recipe > 0 || i.ItemId == MakeshiftMealItemId {
		return true
	}
	// Camp music: a crude, common or fine instrument is bought or crafted,
	// so no merchant buys it back; only a looted masterwork may be sold.
	if spec := i.GetSpec(); spec.Instrument != "" && spec.InstrumentTier < MasterworkInstrumentTier {
		return true
	}
	if len(i.Blob) > 0 {
		return true
	}
	if spec := i.GetSpec(); spec.Uses > 0 && spec.Uses != i.Uses {
		return true
	}
	if i.Spec != nil && !i.IsRolled() {
		return true
	}
	return false
}

// SaleBaseValue is the value a merchant prices the item from, before the
// merchant's own share. A plain item is its spec's value; rolled gear is
// its quality-and-affix value once identified (the Spec already holds
// both); an unidentified Rare or better is priced by rarity at a discount.
func (i *Item) SaleBaseValue() int {
	value := i.GetSpec().Value
	if !i.IsRolled() || i.Loot.Identified {
		return value
	}
	if pct, ok := unidentifiedRarityPct[i.Loot.Rarity]; ok {
		value = value * pct / 100
	}
	return value * UnidentifiedPricePct / 100
}

// IsJunkMarked reports whether the player marked the item as junk.
func (i *Item) IsJunkMarked() bool { return i.Junk }

// IsAutoJunk reports an item that `sell junk` takes without being marked:
// a junk-type item that is not rolled, quest bound or otherwise special.
func (i *Item) IsAutoJunk() bool {
	spec := i.GetSpec()
	return spec.Type == Junk && spec.QuestToken == `` && !i.IsSpecial()
}
