package loot

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// Phase 37 drop tables (loot slice 3). Each kill makes one roll per foe; a
// won random encounter makes one group cache roll; a boss adds its own
// roll. Rolls draw on an injected Source, so every outcome is testable.

// Range is an inclusive level range.
type Range struct {
	Low  int `yaml:"low"`
	High int `yaml:"high"`
}

// ZoneProfile is a zone's drop profile (zone-config.yaml "loot:"). An empty
// profile draws item levels from the source's own level, held to the
// engine's bounds, and the tier from that level.
type ZoneProfile struct {
	ILvl Range `yaml:"ilvl,omitempty"` // item levels a drop here may have; keeps a farmed zone from producing runaway items
	Tier int   `yaml:"tier,omitempty"` // catalog tier gear is drawn from; 0 follows the item level (tier 1 at 1-9, 2 at 10-19, ...)
}

// Source is what made a drop roll.
type Kind int

const (
	Ordinary Kind = iota // a foe's own roll
	Elite                // an elite foe
	Cache                // a won random encounter's group roll
	BossRoll             // a boss's roll, on top of the group's cache
)

func (k Kind) String() string {
	return [...]string{"ordinary", "elite", "cache", "boss"}[k]
}

// Rule is one source's numbers (design table).
type Rule struct {
	EquipChance int          // percent chance of equipment
	EquipCount  int          // items when it drops
	Boost       int          // Rare and Epic weight multiplier
	MinRarity   items.Rarity // guaranteed rarity of the first item
	GoodsMin    int          // goods rolls from the foe's category table
	GoodsMax    int
}

// Rules are the drop numbers by source. An ordinary or elite foe's goods
// come from its own template's table at death, as before this phase, so
// only the cache and the boss list goods rolls here.
var Rules = map[Kind]Rule{
	Ordinary: {EquipChance: 12, EquipCount: 1, Boost: 1},
	Elite:    {EquipChance: 30, EquipCount: 1, Boost: 2},
	Cache:    {EquipChance: 20, EquipCount: 1, Boost: 1, GoodsMin: 1, GoodsMax: 2},
	BossRoll: {EquipChance: 100, EquipCount: 2, Boost: 5, MinRarity: items.RarityRare, GoodsMin: 3, GoodsMax: 3},
}

// ItemLevel is the item level a drop from a source of the given level has,
// held to the zone's range when it sets one.
func (p ZoneProfile) ItemLevel(level int) int {
	if p.ILvl.High >= p.ILvl.Low && p.ILvl.Low >= 1 {
		level = min(max(level, p.ILvl.Low), p.ILvl.High)
	}
	return min(max(level, MinILvl), MaxILvl)
}

// BaseTier is the catalog tier gear at item level ilvl is drawn from.
func (p ZoneProfile) BaseTier(ilvl int) int {
	if p.Tier > 0 {
		return min(p.Tier, 6)
	}
	return min(ilvl/10+1, 6)
}

// bases lists the rollable catalog bases by tier, ascending by id so a
// scripted Source is deterministic.
func bases() map[int][]int {
	out := map[int][]int{}
	for _, spec := range items.GetAllItemSpecs() {
		if spec.Tier < 1 || !Rollable(spec) || spec.Relic != nil { // relics drop only from their boss
			continue
		}
		out[spec.Tier] = append(out[spec.Tier], spec.ItemId)
	}
	for _, ids := range out {
		sort.Ints(ids)
	}
	return out
}

// PickBase picks a catalog base item for the tier: the tier itself mostly,
// one tier either side sometimes (70/15/15), the nearest stocked tier when
// the choice has none. It returns 0 when no tiered equipment is loaded.
func PickBase(tier int, rng Source) int {
	all := bases()
	if len(all) == 0 {
		return 0
	}
	want := tier
	switch n := rng.Intn(100); {
	case n < 15:
		want--
	case n < 30:
		want++
	}
	best, bestDist := 0, 1<<30
	var tiers []int
	for t := range all {
		tiers = append(tiers, t)
	}
	sort.Ints(tiers)
	for _, t := range tiers {
		d := t - want
		if d < 0 {
			d = -d
		}
		if d < bestDist {
			best, bestDist = t, d
		}
	}
	ids := all[best]
	return ids[rng.Intn(len(ids))]
}

// Equipment rolls a source's equipment drop: nothing, or the rule's count
// of generated items at the zone's item level.
func Equipment(kind Kind, sourceLevel int, p ZoneProfile, label string, rng Source) ([]items.Item, error) {
	return EquipmentWith(kind, sourceLevel, p, label, rng, 0)
}

// EquipmentWith is Equipment with chanceBonus points added to the rule's
// chance of holding equipment (Phase 38c2: a Pathfinder's Trailwise).
func EquipmentWith(kind Kind, sourceLevel int, p ZoneProfile, label string, rng Source, chanceBonus int) ([]items.Item, error) {
	rule, ok := Rules[kind]
	if !ok || rule.EquipCount < 1 {
		return nil, nil
	}
	if chance := min(100, rule.EquipChance+chanceBonus); chance < 100 && rng.Intn(100) >= chance {
		return nil, nil
	}
	ilvl := p.ItemLevel(sourceLevel)
	tier := p.BaseTier(ilvl)
	var out []items.Item
	for i := 0; i < rule.EquipCount; i++ {
		id := PickBase(tier, rng)
		if id == 0 {
			return out, nil
		}
		opts := Options{ILvl: ilvl, RarityBoost: rule.Boost, Source: label}
		if i == 0 {
			opts.MinRarity = rule.MinRarity
		}
		itm, err := Roll(id, opts, rng)
		if err != nil {
			return out, fmt.Errorf("loot: roll item %d: %w", id, err)
		}
		out = append(out, itm)
	}
	return out, nil
}

// Goods rolls the cache's or boss's goods from a category table: the
// rule's number of draws, each one entry's items.
func Goods(kind Kind, table Table, rng Source) []items.Item {
	rule := Rules[kind]
	if rule.GoodsMax < 1 {
		return nil
	}
	n := rule.GoodsMin
	if rule.GoodsMax > rule.GoodsMin {
		n += rng.Intn(rule.GoodsMax - rule.GoodsMin + 1)
	}
	var out []items.Item
	for i := 0; i < n; i++ {
		entry, ok := table.Resolve(uint64(rng.Intn(1 << 30)))
		if !ok || items.GetItemSpec(entry.ItemID) == nil {
			continue
		}
		for c := entry.RollCount(uint64(rng.Intn(1 << 30))); c > 0; c-- {
			out = append(out, items.New(entry.ItemID))
		}
	}
	return out
}

// CacheGold is the gold a won random encounter's cache holds, by the
// source's level band (templates keep their own gold).
func CacheGold(level int, boss bool, rng Source) int {
	gold := level*4 + rng.Intn(level*2+1)
	if boss {
		gold += level * 10
	}
	return gold
}
