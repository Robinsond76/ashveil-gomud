package strategy

import "sort"

// HexTargets are the foes a hex reaches (Phase 38a), the primary first: a
// foe winding up a blow, else a chanting caster, else the nearest (the
// front-most row, then the left-most column). Then, up to reach foes in all,
// the rest of the primary's row and the nearest others. A row hex (Miasma)
// covers the primary's row only. foes are the ones worth the hex;
// windingUp reports a foe mid wind-up. It returns nil for no foes.
func HexTargets(foes []Foe, row bool, reach int, windingUp func(id int) bool) []int {
	if len(foes) == 0 || reach < 1 {
		return nil
	}
	pool := append([]Foe(nil), foes...)
	rank := func(f Foe) int {
		switch {
		case windingUp != nil && windingUp(f.ID):
			return 0
		case f.Chanting && f.Caster:
			return 1
		}
		return 2
	}
	sort.SliceStable(pool, func(i, j int) bool {
		if ri, rj := rank(pool[i]), rank(pool[j]); ri != rj {
			return ri < rj
		}
		if pool[i].Row != pool[j].Row {
			return pool[i].Row < pool[j].Row
		}
		return pool[i].Col < pool[j].Col
	})
	primary, rest := pool[0], pool[1:]
	sort.SliceStable(rest, func(i, j int) bool {
		di, dj := abs(rest[i].Row-primary.Row), abs(rest[j].Row-primary.Row)
		if di != dj {
			return di < dj
		}
		if rest[i].Row != rest[j].Row {
			return rest[i].Row < rest[j].Row
		}
		return rest[i].Col < rest[j].Col
	})
	out := []int{primary.ID}
	for _, f := range rest {
		if len(out) >= reach {
			break
		}
		if row && f.Row != primary.Row {
			continue
		}
		out = append(out, f.ID)
	}
	return out
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
