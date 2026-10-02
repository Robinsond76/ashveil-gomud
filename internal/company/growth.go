package company

import "strings"

// Phase 33h1 companion growth. A companion's trained stats are never
// accumulated: they are its level's stat points dealt by its growth
// weights, recomputed whenever the level or the weights change, so a level
// lost and earned again cannot mint points.

// GrowthStats are the six stats in the order GoMud's AutoTrain uses.
var GrowthStats = [6]string{"strength", "speed", "smarts", "vitality", "mysticism", "perception"}

// FocusWeight is what a companion's chosen focus stat adds to its weights.
const FocusWeight = 2

// GrowthWeights are relative weights over GrowthStats.
type GrowthWeights [6]int

// EvenGrowth is the spread for a companion without an archetype.
var EvenGrowth = GrowthWeights{1, 1, 1, 1, 1, 1}

// GrowthStatIndex resolves a stat name or a unique prefix of one ("str",
// "myst") to its index in GrowthStats.
func GrowthStatIndex(name string) (int, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return 0, false
	}
	found := -1
	for i, s := range GrowthStats {
		if s == name {
			return i, true
		}
		if strings.HasPrefix(s, name) {
			if found >= 0 {
				return 0, false
			}
			found = i
		}
	}
	return found, found >= 0
}

// GrowthWeightsFrom builds weights from stat names; unknown names and
// negative weights are ignored. ok is false when nothing positive is left.
func GrowthWeightsFrom(byStat map[string]int) (GrowthWeights, bool) {
	var w GrowthWeights
	total := 0
	for name, weight := range byStat {
		i, ok := GrowthStatIndex(name)
		if !ok || weight <= 0 || !strings.EqualFold(GrowthStats[i], strings.TrimSpace(name)) {
			continue
		}
		w[i] += weight
		total += weight
	}
	return w, total > 0
}

// WithFocus adds FocusWeight to the focus stat. An empty or unknown focus
// leaves the weights alone; empty weights become EvenGrowth first.
func (w GrowthWeights) WithFocus(focus string) GrowthWeights {
	if w.Total() <= 0 {
		w = EvenGrowth
	}
	if focus == "" {
		return w
	}
	for i, s := range GrowthStats {
		if s == focus {
			w[i] += FocusWeight
		}
	}
	return w
}

// Total is the sum of the positive weights.
func (w GrowthWeights) Total() int {
	t := 0
	for _, v := range w {
		if v > 0 {
			t += v
		}
	}
	return t
}

// Deal shares points among the stats by weight in a fixed rotation (smooth
// weighted round-robin). It is prefix-stable: Deal(n+1) adds one point to
// Deal(n) and takes none away, so a level-up never lowers a stat. Empty
// weights deal evenly.
func Deal(points int, w GrowthWeights) [6]int {
	var out [6]int
	if points <= 0 {
		return out
	}
	total := w.Total()
	if total <= 0 {
		w, total = EvenGrowth, EvenGrowth.Total()
	}
	var current [6]int
	for p := 0; p < points; p++ {
		best := -1
		for i, v := range w {
			if v <= 0 {
				continue
			}
			current[i] += v
			if best < 0 || current[i] > current[best] {
				best = i
			}
		}
		current[best] -= total
		out[best]++
	}
	return out
}
