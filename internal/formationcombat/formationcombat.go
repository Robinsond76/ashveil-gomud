// Package formationcombat implements the column-occupancy reach model,
// lateral column range, and front-row interception over company.Formation
// (reused directly from internal/company for both the player company side
// and, via mobparty, the enemy party side).
//
// This package is deliberately GoMud-free: every function takes a
// company.Formation plus a caller-supplied "who's still alive" map, never
// a live *mobs.Mob/*characters.Character. Formation is locked once combat
// starts (no mid-fight rearrangement anywhere in this codebase), so the
// only thing that changes reach/legality round to round is which members
// are alive — the self-healing behavior described in the Phase 11c design
// doc falls directly out of that: nothing here ever mutates a Formation or
// clears a caller's stored target, it only answers "is this legal right
// now" fresh from current alive state.
package formationcombat

import "github.com/GoMudEngine/GoMud/internal/company"

// Reach determines how deep into a column an attacker can strike.
type Reach int

const (
	ReachNone     Reach = iota // plain melee: column's frontmost occupant only
	ReachExtended              // polearm or innate: frontmost, or one behind it
	ReachAny                   // ranged: any occupied depth in the column
)

// FrontmostOccupant returns the row of the nearest-to-front living
// occupant of a column: the cell nearest row 0 whose MemberKey is present
// in the formation and alive[key] is true. ok is false if the column has
// no living occupant (empty, or every occupant is dead).
func FrontmostOccupant(f company.Formation, col int, alive map[company.MemberKey]bool) (row int, ok bool) {
	for r := 0; r < company.FormationRows; r++ {
		key := f.At(r, col)
		if key == "" {
			continue
		}
		if alive != nil && !alive[key] {
			continue
		}
		return r, true
	}
	return 0, false
}

// InLateralRange reports whether defenderCol is within attackerCol's own
// column index plus or minus one.
func InLateralRange(attackerCol, defenderCol int) bool {
	diff := attackerCol - defenderCol
	if diff < 0 {
		diff = -diff
	}
	return diff <= 1
}

// InReachDepth reports whether targetRow is reachable given the column's
// current frontmost occupied row and an attacker's Reach class.
func InReachDepth(frontmostRow, targetRow int, reach Reach) bool {
	switch reach {
	case ReachAny:
		return true
	case ReachExtended:
		return targetRow == frontmostRow || targetRow == frontmostRow+1
	default:
		return targetRow == frontmostRow
	}
}

// Legal reports whether an attacker in attackerCol may currently target
// targetKey within f, given who's alive and the attacker's Reach. This is
// the predicate Phase 11b's AssignTarget consumes as its injected
// LegalFunc (adapted to that package's Combatant shape by the caller).
func Legal(attackerCol int, f company.Formation, targetKey company.MemberKey, alive map[company.MemberKey]bool, reach Reach) bool {
	if targetKey == "" {
		return false
	}
	if alive != nil && !alive[targetKey] {
		return false
	}

	targetRow, targetCol, found := f.Find(targetKey)
	if !found {
		return false
	}
	if !InLateralRange(attackerCol, targetCol) {
		return false
	}

	frontmostRow, ok := FrontmostOccupant(f, targetCol, alive)
	if !ok {
		return false
	}
	return InReachDepth(frontmostRow, targetRow, reach)
}

// InterceptFrontRow reports the front-row member in the same column as an
// attack aimed at a non-front-row member of f, if one is alive. ok is
// false when no interception applies (targetKey is already front row, not
// found, or that column's front slot has no living occupant) — the caller
// should then proceed against the original target.
func InterceptFrontRow(f company.Formation, targetKey company.MemberKey, alive map[company.MemberKey]bool) (interceptor company.MemberKey, ok bool) {
	targetRow, targetCol, found := f.Find(targetKey)
	if !found || targetRow == 0 {
		return "", false
	}

	frontKey := f.At(0, targetCol)
	if frontKey == "" {
		return "", false
	}
	if alive != nil && !alive[frontKey] {
		return "", false
	}
	return frontKey, true
}

// LegalTargets returns every member of f that an attacker in attackerCol
// could currently legally target, given alive and reach — the read-only
// adjacency query the design doc's acceptance criteria call for.
func LegalTargets(attackerCol int, f company.Formation, alive map[company.MemberKey]bool, reach Reach) []company.MemberKey {
	var legal []company.MemberKey
	for r := 0; r < company.FormationRows; r++ {
		for c := 0; c < company.FormationCols; c++ {
			key := f.At(r, c)
			if key == "" {
				continue
			}
			if Legal(attackerCol, f, key, alive, reach) {
				legal = append(legal, key)
			}
		}
	}
	return legal
}
