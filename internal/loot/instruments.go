package loot

import (
	"sort"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// Camp music: masterwork instruments are looted, never bought or crafted.
// A masterwork names its boss (instrumentmob) and a percent chance per kill
// by one company, like a relic; it is a plain, named item with one quirk,
// not a rolled one.

// InstrumentDropsOf lists the masterwork instruments a boss mob may drop,
// ascending by item id.
func InstrumentDropsOf(mobID int) []items.ItemSpec {
	var out []items.ItemSpec
	for _, spec := range items.GetAllItemSpecs() {
		if spec.Instrument != "" && spec.InstrumentMob == mobID && spec.InstrumentChance > 0 {
			out = append(out, spec)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ItemId < out[b].ItemId })
	return out
}

// InstrumentRoll rolls a boss's masterwork instruments for one company's
// kill: each is its own chance. It returns the instruments dropped.
func InstrumentRoll(mobID int, rng Source) []items.Item {
	var out []items.Item
	for _, spec := range InstrumentDropsOf(mobID) {
		if rng.Intn(100) < spec.InstrumentChance {
			out = append(out, items.New(spec.ItemId))
		}
	}
	return out
}
