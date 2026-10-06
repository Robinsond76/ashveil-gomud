package loot

import (
	"errors"
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 36a generator. Randomness is injected, so every roll is
// deterministic in tests. Rolls only happen in tests and through the admin
// "spawn loot" command until the drop tables (Phase 37) wire them to kills.

// Source is the random source a roll draws from: Intn returns 0..n-1.
type Source interface{ Intn(n int) int }

type gameSource struct{}

func (gameSource) Intn(n int) int { return util.Rand(n) }

// GameSource draws from the game's random source.
func GameSource() Source { return gameSource{} }

// ILvl bounds (design: item level 1-70).
const (
	MinILvl = 1
	MaxILvl = 70
)

// qualityWeights, in percent (design table).
var qualityWeights = []struct {
	q items.Quality
	w int
}{
	{items.QualityCrude, 100}, {items.QualityWorn, 200}, {items.QualityStandard, 450},
	{items.QualityFine, 170}, {items.QualitySuperior, 70}, {items.QualityExquisite, 10},
}

// rarityWeights are an ordinary foe's weights in tenths of a percent.
// Legendary and Set are authored items and never roll by chance.
var rarityWeights = []struct {
	r items.Rarity
	w int
}{
	{items.RarityCommon, 700}, {items.RarityUncommon, 220}, {items.RarityRare, 65}, {items.RarityEpic, 14},
}

// Options steer one roll. Empty fields roll by the design's weights.
type Options struct {
	ILvl    int
	Rarity  items.Rarity
	Quality items.Quality
	Source  string // where it came from, shown at Scribe rank 4
}

// ErrNotRollable is returned for an item that cannot roll (a consumable, a
// pack, a key).
var ErrNotRollable = errors.New("that item is not equipment that can roll")

// slotClass maps a spec to the affix slot class it belongs to; "" means the
// item cannot roll.
func slotClass(spec items.ItemSpec) string {
	switch spec.Type {
	case items.Weapon:
		return SlotWeapon
	case items.Offhand:
		if spec.IsShield() {
			return SlotShield
		}
		return SlotArmor
	case items.Head, items.Body, items.Belt, items.Gloves, items.Legs, items.Feet:
		if spec.Subtype == items.Wearable {
			return SlotArmor
		}
	case items.Neck, items.Ring:
		if spec.Subtype == items.Wearable {
			return SlotJewelry
		}
	}
	return ""
}

// Rollable reports whether a spec is equipment the generator can roll.
func Rollable(spec items.ItemSpec) bool { return slotClass(spec) != "" }

func (s AffixSet) eligible(spec items.ItemSpec, class string, ilvl int, major bool, used map[string]bool) []Affix {
	var out []Affix
	for _, a := range s.Affixes {
		if a.Major != major || used[a.Group] {
			continue
		}
		ok := false
		for _, sl := range a.Slots {
			if sl == class {
				ok = true
			}
		}
		if !ok || a.Tiers[0].MinILvl > ilvl {
			continue
		}
		switch {
		case a.Mechanic == "protection" && spec.DamageReduction <= 0:
			continue // protection only builds on gear that already protects
		case a.Mechanic == "weightpct" && spec.Weight <= 0:
			continue
		}
		out = append(out, a)
	}
	return out
}

func pickWeighted(rng Source, list []Affix) Affix {
	total := 0
	for _, a := range list {
		total += a.Weight
	}
	n := rng.Intn(total)
	for _, a := range list {
		if n < a.Weight {
			return a
		}
		n -= a.Weight
	}
	return list[len(list)-1]
}

// rollAffix picks the affix's highest tier the item level allows and a
// value in its range.
func rollAffix(rng Source, a Affix, ilvl int) items.RolledAffix {
	tier := 0
	for i, t := range a.Tiers {
		if t.MinILvl <= ilvl {
			tier = i
		}
	}
	t := a.Tiers[tier]
	value := t.Min
	if t.Max > t.Min {
		value += rng.Intn(t.Max - t.Min + 1)
	}
	out := items.RolledAffix{ID: a.ID, Major: a.Major, Mechanic: a.Mechanic, Value: value, Tier: tier + 1, MinValue: t.Min, MaxValue: t.Max}
	switch {
	case a.Prefix != "" && a.Suffix != "":
		if rng.Intn(2) == 0 {
			out.Label = a.Prefix
		} else {
			out.Label, out.Suffix = a.Suffix, true
		}
	case a.Suffix != "":
		out.Label, out.Suffix = a.Suffix, true
	default:
		out.Label = a.Prefix
	}
	return out
}

func weightedRarity(rng Source) items.Rarity {
	total := 0
	for _, e := range rarityWeights {
		total += e.w
	}
	n := rng.Intn(total)
	for _, e := range rarityWeights {
		if n < e.w {
			return e.r
		}
		n -= e.w
	}
	return items.RarityCommon
}

func weightedQuality(rng Source) items.Quality {
	total := 0
	for _, e := range qualityWeights {
		total += e.w
	}
	n := rng.Intn(total)
	for _, e := range qualityWeights {
		if n < e.w {
			return e.q
		}
		n -= e.w
	}
	return items.QualityStandard
}

// affixCount is how many affixes a rarity rolls, and whether one is major.
func affixCount(r items.Rarity, rng Source) (n int, major bool) {
	switch r {
	case items.RarityUncommon:
		return 1, false
	case items.RarityRare:
		return 2 + rng.Intn(2), false
	case items.RarityEpic:
		return 3 + rng.Intn(2), true
	case items.RarityLegendary:
		// The authored signature effect arrives with the legendary catalog;
		// until then its slot is one major affix.
		return 3, true
	case items.RaritySet:
		return 2, false
	}
	return 0, false
}

// LevelRequirement is the level needed to wear an item: ilvl-5 for Rare
// and above, ilvl-10 for Uncommon, none for Common (owner, 2026-10-05).
func LevelRequirement(r items.Rarity, ilvl int) int {
	switch {
	case r.Rank() >= items.RarityRare.Rank():
		return max(0, ilvl-5)
	case r == items.RarityUncommon:
		return max(0, ilvl-10)
	}
	return 0
}

func generatedName(rng Source, n Names) string {
	if len(n.Prefixes) == 0 || len(n.Suffixes) == 0 {
		return ""
	}
	return n.Prefixes[rng.Intn(len(n.Prefixes))] + strings.ToLower(n.Suffixes[rng.Intn(len(n.Suffixes))])
}

// Generate rolls the layers of one item of the base spec.
func (s AffixSet) Generate(spec items.ItemSpec, opts Options, rng Source) (items.Rolled, error) {
	class := slotClass(spec)
	if class == "" {
		return items.Rolled{}, ErrNotRollable
	}
	ilvl := min(max(opts.ILvl, MinILvl), MaxILvl)
	rarity := opts.Rarity
	if rarity == "" {
		rarity = weightedRarity(rng)
	} else if !rarity.Valid() {
		return items.Rolled{}, fmt.Errorf("unknown rarity %q", rarity)
	}
	quality := opts.Quality
	if quality == "" {
		quality = weightedQuality(rng)
	} else if !quality.Valid() {
		return items.Rolled{}, fmt.Errorf("unknown quality %q", quality)
	}

	r := items.Rolled{
		Version:    items.RollVersion,
		Tier:       max(1, spec.Tier),
		ILvl:       ilvl,
		Quality:    quality,
		Rarity:     rarity,
		Identified: !rarity.DropsUnidentified(),
		LevelReq:   LevelRequirement(rarity, ilvl),
		BaseValue:  spec.Value,
		Source:     opts.Source,
	}

	want, wantMajor := affixCount(rarity, rng)
	used := map[string]bool{}
	for i := 0; i < want; i++ {
		major := wantMajor && i == want-1
		pool := s.eligible(spec, class, ilvl, major, used)
		if len(pool) == 0 && major {
			pool = s.eligible(spec, class, ilvl, false, used)
		}
		if len(pool) == 0 {
			break // fewer affixes than wanted: the pool ran dry
		}
		a := pickWeighted(rng, pool)
		used[a.Group] = true
		r.Affixes = append(r.Affixes, rollAffix(rng, a, ilvl))
	}
	if rarity.Rank() >= items.RarityRare.Rank() {
		r.Name = generatedName(rng, s.Names)
	}
	return r, nil
}

// Roll generates an item instance of itemID with the loaded affix data.
func Roll(itemID int, opts Options, rng Source) (items.Item, error) {
	spec := items.GetItemSpec(itemID)
	if spec == nil {
		return items.Item{}, fmt.Errorf("no item %d", itemID)
	}
	rolled, err := Affixes().Generate(*spec, opts, rng)
	if err != nil {
		return items.Item{}, err
	}
	itm := items.New(itemID)
	itm.ApplyRoll(rolled)
	return itm, nil
}
