package loot

import (
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// Phase 71: trophy drops. A creature's race decides which trophies it may
// carry (a trophy's `races`); each company that fought makes its own roll,
// like every other drop. An ordinary kill drops one with the trophy's own
// chance, an elite twice that, a boss always drops one. The roll draws on
// an injected Source so every outcome is testable.

// Trophies lists the trophy items a creature of the race may drop,
// ascending by item id.
func Trophies(race string) []items.ItemSpec {
	race = strings.ToLower(race)
	var out []items.ItemSpec
	for _, spec := range items.GetAllItemSpecs() {
		if spec.Trophy != nil && spec.Trophy.DropsFrom(race) {
			out = append(out, spec)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ItemId < out[b].ItemId })
	return out
}

// TrophyChance is the percent chance a kill of the kind drops this trophy.
func TrophyChance(spec items.ItemSpec, kind Kind) int {
	switch kind {
	case BossRoll:
		return 100
	case Elite:
		return min(100, spec.Trophy.Chance*2)
	}
	return spec.Trophy.Chance
}

// TrophyRoll rolls one company's trophy for a kill: it picks one of the
// race's trophies and drops it on that trophy's chance. Nil for none.
func TrophyRoll(kind Kind, race string, rng Source) *items.Item {
	options := Trophies(race)
	if len(options) == 0 {
		return nil
	}
	pick := options[rng.Intn(len(options))]
	if rng.Intn(100) >= TrophyChance(pick, kind) {
		return nil
	}
	itm := items.New(pick.ItemId)
	if itm.ItemId == 0 {
		return nil
	}
	return &itm
}
