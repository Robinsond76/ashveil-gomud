package formationcombat

import (
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/survival"
)

// Fold derives narrow-ground positions without changing saved placements.
// A member keeps its row if possible, then spills rearward, then forward.
// Column 2 is a visible reserve: those members cannot make melee attacks.
// Recomputing from surviving members brings reserves into vacated cells.
func Fold(f company.Formation, alive map[company.MemberKey]bool) company.Formation {
	var out company.Formation
	var reserve []company.MemberKey
	for r := 0; r < company.FormationRows; r++ {
		for c := 0; c < company.FormationCols; c++ {
			key := f.At(r, c)
			if key == "" || (alive != nil && !alive[key]) {
				continue
			}
			placed := false
			rows := []int{r}
			for next := r + 1; next < 3; next++ {
				rows = append(rows, next)
			}
			for next := 0; next < r; next++ {
				rows = append(rows, next)
			}
			for _, row := range rows {
				for col := 0; col < 2; col++ {
					if !placed && out.At(row, col) == "" {
						out[row][col] = key
						placed = true
					}
				}
				if placed {
					break
				}
			}
			if !placed {
				reserve = append(reserve, key)
			}
		}
	}
	for r, key := range reserve {
		out[r][2] = key
	}
	return out
}

// Cluster includes only the center and its orthogonal neighbors, never
// diagonals or a transitive chain. Reserve cells aren't physical neighbors.
func Cluster(f company.Formation, center company.MemberKey, alive map[company.MemberKey]bool, narrow bool) []company.MemberKey {
	r, c, ok := f.Find(center)
	if !ok {
		return nil
	}
	var out []company.MemberKey
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			key := f.At(row, col)
			if key == "" || (alive != nil && !alive[key]) {
				continue
			}
			dr, dc := row-r, col-c
			if dr < 0 {
				dr = -dr
			}
			if dc < 0 {
				dc = -dc
			}
			if key == center || (dr+dc == 1 && (!narrow || (c < 2 && col < 2))) {
				out = append(out, key)
			}
		}
	}
	return out
}

// Flanked is a rear cell whose own and an adjacent front column are open.
func Flanked(f company.Formation, key company.MemberKey, standing map[company.MemberKey]bool, narrow bool) bool {
	r, c, ok := f.Find(key)
	if !ok || r == 0 || (narrow && c == 2) {
		return false
	}
	covered := func(col int) bool {
		k := f.At(0, col)
		return k != "" && (standing == nil || standing[k])
	}
	if covered(c) {
		return false
	}
	cols := 3
	if narrow {
		cols = 2
	}
	return (c > 0 && !covered(c-1)) || (c+1 < cols && !covered(c+1))
}

// LegalGround combines ordinary reach with open flanks and narrow reserves.
func LegalGround(col int, f company.Formation, key company.MemberKey, alive, standing map[company.MemberKey]bool, reach Reach, narrow bool) bool {
	r, tc, placed := f.Find(key)
	if !placed || key == "" || (alive != nil && !alive[key]) {
		return false
	}
	if narrow && (col == 2 || tc == 2) {
		return reach == ReachAny
	}
	if !InLateralRange(col, tc) {
		return false
	}
	if reach == ReachAny || (r > 0 && Flanked(f, key, standing, narrow)) {
		return true
	}
	return Legal(col, f, key, alive, reach)
}

// Detection uses a zero-based percentage roll. Advantage is -1 for enemy,
// 0 for neither side, +1 for the company.
func Detection(perception, stealth, visibility, concealment, roll int, watch bool) (chance, advantage int) {
	light := 0
	if visibility == 1 {
		light = 10
	}
	if visibility <= 0 {
		light = 25
	}
	chance = max(5, min(95, 50+perception-stealth-light-concealment))
	if roll >= chance {
		if watch {
			return chance, 0
		}
		return chance, -1
	}
	if chance-roll >= 30 {
		return chance, 1
	}
	return chance, 0
}

// FatiguePenalty is the hit-chance points fatigue costs (survival owns the
// numbers, Phase 50, so the battle condition shows the same ones).
func FatiguePenalty(rest int) int { return survival.FatigueHitPenalty(rest) }

func GuardGround(f company.Formation, guardian, ward company.MemberKey, narrow bool) bool {
	if narrow {
		_, gc, gok := f.Find(guardian)
		_, wc, wok := f.Find(ward)
		if (gok && gc == 2) || (wok && wc == 2) {
			return false
		}
	}
	return GuardReach(f, guardian, ward)
}
