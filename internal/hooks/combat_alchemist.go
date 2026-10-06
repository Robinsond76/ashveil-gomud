package hooks

import (
	"slices"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/strategy"
)

// Phase 39g: the Alchemist's Fire Flask. Its other flasks are ordinary
// automatic spells (internal/strategy, the spell files); this is the one
// choice they need that no spell had: which foes the fire reaches.

// flameTargets are the foes a Fire Flask reaches: the foe aimed at, and then
// the foes nearest it, the rest of its row first, up to two in all (one more
// for each Wide throw rank).
func flameTargets(a actor, g enemyparty.Group, foes []int, primary int) []int {
	att := a.att
	att.Spell = true
	var first strategy.Foe
	var rest []strategy.Foe
	found := false
	for _, f := range enemyparty.Foes(g, att) {
		switch {
		case !slices.Contains(foes, f.ID):
		case f.ID == primary:
			first, found = f, true
		default:
			rest = append(rest, f)
		}
	}
	out := []int{primary}
	if !found {
		// Not in the group's formation: the others in the order they stand.
		for _, id := range foes {
			if id != primary && len(out) < 2+a.char.ClassEffects().Int(classes.FlaskReach) {
				out = append(out, id)
			}
		}
		return out
	}
	dist := func(f strategy.Foe) [3]int {
		return [3]int{abs(f.Row - first.Row), abs(f.Col - first.Col), f.Col}
	}
	sort.SliceStable(rest, func(i, j int) bool {
		di, dj := dist(rest[i]), dist(rest[j])
		for k := range di {
			if di[k] != dj[k] {
				return di[k] < dj[k]
			}
		}
		return false
	})
	reach := 2 + a.char.ClassEffects().Int(classes.FlaskReach)
	for _, f := range rest {
		if len(out) >= reach {
			break
		}
		out = append(out, f.ID)
	}
	return out
}
