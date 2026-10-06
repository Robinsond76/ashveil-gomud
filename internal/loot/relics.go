package loot

import (
	"sort"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// Phase 36d: relic drops. A relic names its boss (its relic block's mob)
// and a percent chance per kill by one company; a boss kill makes at most
// one relic roll. Bad-luck protection (loot design, N = 20): each leader
// keeps a count of kills of each boss that dropped no relic, and the Nth
// such kill is guaranteed one.

// BadLuckKills is the kill on which a boss's relic is guaranteed.
const BadLuckKills = 20

// RelicsOf lists the authored relics a boss mob drops, ascending by id.
func RelicsOf(mobID int) []items.ItemSpec {
	var out []items.ItemSpec
	for _, spec := range items.RelicSpecs() {
		if spec.Relic.Mob == mobID {
			out = append(out, spec)
		}
	}
	return out
}

// RelicBosses lists the mob ids that drop relics, ascending.
func RelicBosses() []int {
	seen := map[int]bool{}
	var out []int
	for _, spec := range items.RelicSpecs() {
		if !seen[spec.Relic.Mob] {
			seen[spec.Relic.Mob] = true
			out = append(out, spec.Relic.Mob)
		}
	}
	sort.Ints(out)
	return out
}

// RelicRoll rolls a boss's relic for one company's kill. missed is the
// number of that leader's earlier kills of this boss that dropped none. It
// returns the relic dropped (nil for none) and the new missed count, which
// resets on a drop.
func RelicRoll(mobID, missed int, source string, rng Source) (*items.Item, int, error) {
	relics := RelicsOf(mobID)
	if len(relics) == 0 {
		return nil, missed, nil
	}
	total := 0
	for _, r := range relics {
		total += r.Relic.Chance
	}
	pick := -1
	if n := rng.Intn(100); n < min(total, 100) {
		pick = n
	} else if missed+1 >= BadLuckKills {
		pick = rng.Intn(total)
	}
	if pick < 0 {
		return nil, missed + 1, nil
	}
	chosen := relics[len(relics)-1]
	for _, r := range relics {
		if pick < r.Relic.Chance {
			chosen = r
			break
		}
		pick -= r.Relic.Chance
	}
	itm, err := Roll(chosen.ItemId, Options{Source: source}, rng)
	if err != nil {
		return nil, missed, err
	}
	return &itm, 0, nil
}
