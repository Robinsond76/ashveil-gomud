// Package engagement picks per-combatant targets within an enemy party
// (Phase 11a's mobparty.Party) by a minimal weakest/strongest/random
// preference, restricted to whatever a caller-supplied legality predicate
// (Phase 11c's lateral-range/reach rule) currently allows.
//
// This package is deliberately GoMud-free: it takes plain Combatant values
// and an injected LegalFunc rather than importing internal/mobs,
// internal/characters, or internal/mobparty directly, so assignment logic
// is testable in isolation and stays usable once 11c's real legality
// predicate exists to inject in place of a test stub.
//
// Engagement itself is never persisted — it mirrors characters.Aggro's own
// non-persistence: which company members are engaged with which party is
// derived each round from live Aggro state, not stored separately.
package engagement

import (
	"fmt"
	"math/rand"
	"sort"
)

// Combatant is the minimal view of one participant (player or mob) needed
// for target assignment: identity, current HP (for weakest/strongest
// selection), and formation position (for the legality predicate).
type Combatant struct {
	ID  int // UserId for a player combatant, MobInstanceId for a mob combatant
	HP  int
	Row int
	Col int
}

// Preference selects how AssignTarget picks among legal, living candidates.
type Preference int

const (
	Weakest Preference = iota
	Strongest
	Random
)

// LegalFunc reports whether attacker may currently target defender —
// Phase 11c's lateral-range/reach predicate, injected by the caller.
type LegalFunc func(attacker, defender Combatant) bool

// Engagement is a thin, unpersisted marker: leaderUserID's company is
// currently engaged with the party identified by PartyID. It carries no
// behavior of its own — per-member targets remain each combatant's own
// characters.Aggro, revalidated every round by the (future) hooks
// integration, not tracked here.
type Engagement struct {
	LeaderUserID int
	PartyID      string
}

// AssignTarget picks a target for attacker from candidates, restricted to
// those with HP > 0 and for which legal(attacker, candidate) is true (a
// nil legal treats every living candidate as legal), then applies pref.
// It returns ok=false if no candidate qualifies — the caller does nothing
// that round, matching today's "no target, skip" behavior; it never
// panics or loops on an empty or fully-illegal candidate set.
func AssignTarget(attacker Combatant, candidates []Combatant, pref Preference, legal LegalFunc) (targetID int, ok bool) {
	eligible := make([]Combatant, 0, len(candidates))
	for _, c := range candidates {
		if c.HP <= 0 {
			continue
		}
		if legal != nil && !legal(attacker, c) {
			continue
		}
		eligible = append(eligible, c)
	}

	if len(eligible) == 0 {
		return 0, false
	}

	switch pref {
	case Strongest:
		sort.SliceStable(eligible, func(i, j int) bool { return eligible[i].HP > eligible[j].HP })
		return eligible[0].ID, true
	case Random:
		return eligible[rand.Intn(len(eligible))].ID, true
	case Weakest:
		fallthrough
	default:
		sort.SliceStable(eligible, func(i, j int) bool { return eligible[i].HP < eligible[j].HP })
		return eligible[0].ID, true
	}
}

// PartyAlive reports whether any candidate is still alive (HP > 0). Once
// this is false for a party, any Engagement against it has ended: the
// (future) hooks integration clears every remaining company member's
// Aggro pointed at that party rather than calling AssignTarget again.
func PartyAlive(candidates []Combatant) bool {
	for _, c := range candidates {
		if c.HP > 0 {
			return true
		}
	}
	return false
}

// String renders an Engagement for logging/debugging.
func (e Engagement) String() string {
	return fmt.Sprintf("engagement(leader=%d, party=%s)", e.LeaderUserID, e.PartyID)
}
