# Phase 11b Unit-vs-Unit Engagement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. **Status: fully implemented and committed on `phase-11b-unit-engagement` — every step below is checked off.**

**Goal:** Give company-vs-party combat a coordinated target-assignment rule
— when a company member has no current legal target, pick one from the
enemy party by a minimal weakest/strongest/random preference, restricted to
whatever a legality predicate (Phase 11c's lateral-range/reach rule) allows
— as a pure, independently testable domain function.

**Architecture:** A new GoMud-free domain package, `internal/engagement`,
exposes `Combatant` (the minimal identity/HP/formation-position view of one
participant), `Preference`, a `LegalFunc` type the caller injects, and
`AssignTarget` — a pure function that filters candidates to those alive and
legal, then picks one by preference. `PartyAlive` is the companion helper
for "does this party still have anyone worth being engaged with."

**Spec deviation, deliberate:** the design doc's illustrative signature is
`AssignTarget(member Combatant, party mobparty.Party, pref Preference, legal func(attacker, defender Combatant) bool) (targetID int, ok bool)`.
`mobparty.Party` (Phase 11a) only carries `Members []int` (instance IDs)
and a `Formation` — it has no HP field, and weakest/strongest selection
needs current HP. Passing `party mobparty.Party` directly would leave
`AssignTarget` with no way to compare candidates by HP without an
additional lookup the domain layer can't perform (it has no access to live
`*mobs.Mob`/`*characters.Character`). This plan instead takes
`candidates []Combatant` — the caller (the future hooks integration, not
this phase) adapts `Party.Members` plus each member's live HP and formation
row/col into `Combatant` values, the same "caller adapts real engine types
into a plain struct" shape `mobparty.MobSummary` already established in
11a. `internal/engagement` therefore doesn't import `internal/mobparty` at
all — it doesn't need to; nothing here requires the grouping logic, only
the resulting per-member data.

**Explicit scope boundary (per the spec's own sequencing constraint):**
this plan implements and tests the domain layer only.
`internal/hooks/NewRound_DoCombat.go` integration — the part that actually
triggers engagement when a company member attacks/is attacked and calls
`AssignTarget` each round — is **not** part of this plan. The spec states
this directly: *"`AssignTarget`'s `legal` parameter must be satisfiable by
11c's predicate before this phase's integration step... can land for
real... the actual `NewRound_DoCombat.go` wiring is the last task, done
together with or after 11c lands."* 11c (Formation Tactics, which owns the
legality predicate) is unplanned as of this writing. Building the hook
integration now would mean wiring real combat behavior against a `legal`
stub that gets thrown away the moment 11c ships — worse than not wiring it
yet. This plan delivers the tested, reusable building block; a follow-up
task (tracked in `docs/PROJECT_STATUS.md`, not here) wires it in
alongside 11c.

**Tech Stack:** Go, `testify` (`assert`/`require`), stdlib `sort`/`math/rand`.

**Spec:** `docs/superpowers/specs/2026-09-22-phase-11b-unit-engagement-design.md`
(and the shared prior-art/decisions in
`docs/superpowers/specs/2026-09-22-phase-11-formation-combat-design.md`).

## Global Constraints

- Never advances `gametime`, the round counter, or moves any character.
- Never mutates damage math — this phase is entirely about *who* has a
  target, not how hard an attack hits.
- `characters.Aggro` stays genuinely single-target per character; this
  phase does not turn it into a list or introduce a multi-character
  "encounter" object. `Engagement` is a thin, unpersisted marker only.
- Targeting preference defaults to one project-wide rule for this phase —
  no per-member player-configurable UI (deferred, matches the spec's own
  scoping of "AI target-selection personality" as a later item).
- No new persisted state, no new commands, no new config file — matches
  `Aggro`'s own non-persistence.
- `internal/engagement` stays GoMud-free: it takes plain `Combatant`
  structs and an injected `LegalFunc`, never `*mobs.Mob`/
  `*characters.Character`/`*users.UserRecord` directly.

---

## File Structure

- Create: `internal/engagement/engagement.go` — `Combatant`, `Preference`,
  `LegalFunc`, `AssignTarget`, `Engagement`, `PartyAlive`. Pure domain
  logic, no GoMud imports.
- Create: `internal/engagement/engagement_test.go` — domain tests for
  weakest/strongest/random selection, no-legal-target handling,
  reassignment after a target's death, and `PartyAlive`.

## Task 1: `internal/engagement` domain package

**Files:**
- Create: `internal/engagement/engagement.go`
- Test: `internal/engagement/engagement_test.go`

**Interfaces:**
- Consumes: nothing beyond the Go standard library (`sort`, `math/rand`,
  `fmt`). No dependency on `internal/mobparty`, `internal/company`,
  `internal/characters`, or `internal/mobs` — see the spec-deviation note
  above.
- Produces (for the future hooks-integration task, once 11c exists):
  - `type Combatant struct { ID int; HP int; Row int; Col int }`
  - `type Preference int` with `Weakest`, `Strongest`, `Random`
  - `type LegalFunc func(attacker, defender Combatant) bool`
  - `func AssignTarget(attacker Combatant, candidates []Combatant, pref Preference, legal LegalFunc) (targetID int, ok bool)`
  - `type Engagement struct { LeaderUserID int; PartyID string }`
  - `func PartyAlive(candidates []Combatant) bool`

- [x] **Step 1: Write the failing tests**

Create `internal/engagement/engagement_test.go`:

```go
package engagement_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/engagement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func alwaysLegal(attacker, defender engagement.Combatant) bool {
	return true
}

func TestAssignTargetWeakestPicksLowestHP(t *testing.T) {
	attacker := engagement.Combatant{ID: 100, HP: 20}
	candidates := []engagement.Combatant{
		{ID: 1, HP: 30},
		{ID: 2, HP: 5},
		{ID: 3, HP: 15},
	}

	targetID, ok := engagement.AssignTarget(attacker, candidates, engagement.Weakest, alwaysLegal)

	require.True(t, ok)
	assert.Equal(t, 2, targetID)
}

func TestAssignTargetStrongestPicksHighestHP(t *testing.T) {
	attacker := engagement.Combatant{ID: 100, HP: 20}
	candidates := []engagement.Combatant{
		{ID: 1, HP: 30},
		{ID: 2, HP: 5},
		{ID: 3, HP: 15},
	}

	targetID, ok := engagement.AssignTarget(attacker, candidates, engagement.Strongest, alwaysLegal)

	require.True(t, ok)
	assert.Equal(t, 1, targetID)
}

func TestAssignTargetRandomPicksAmongLegalCandidates(t *testing.T) {
	attacker := engagement.Combatant{ID: 100, HP: 20}
	candidates := []engagement.Combatant{
		{ID: 1, HP: 30},
		{ID: 2, HP: 5},
	}

	legalIDs := map[int]bool{1: true, 2: true}

	for i := 0; i < 20; i++ {
		targetID, ok := engagement.AssignTarget(attacker, candidates, engagement.Random, alwaysLegal)
		require.True(t, ok)
		assert.True(t, legalIDs[targetID], "target %d must be one of the candidates", targetID)
	}
}

func TestAssignTargetSkipsDeadCandidates(t *testing.T) {
	attacker := engagement.Combatant{ID: 100, HP: 20}
	candidates := []engagement.Combatant{
		{ID: 1, HP: 0},  // dead, previously the target
		{ID: 2, HP: 12}, // alive, should be picked as the only survivor
	}

	targetID, ok := engagement.AssignTarget(attacker, candidates, engagement.Weakest, alwaysLegal)

	require.True(t, ok)
	assert.Equal(t, 2, targetID)
}

func TestAssignTargetFiltersByLegalFunc(t *testing.T) {
	attacker := engagement.Combatant{ID: 100, HP: 20, Col: 1}
	candidates := []engagement.Combatant{
		{ID: 1, HP: 10, Col: 0}, // in lateral range
		{ID: 2, HP: 5, Col: 2},  // in lateral range, weaker, but marked illegal below
	}

	// Only candidate 1 is "legal" in this stub, even though candidate 2 is
	// weaker — proves AssignTarget respects the injected predicate over
	// pure HP ordering.
	onlyFirstLegal := func(attacker, defender engagement.Combatant) bool {
		return defender.ID == 1
	}

	targetID, ok := engagement.AssignTarget(attacker, candidates, engagement.Weakest, onlyFirstLegal)

	require.True(t, ok)
	assert.Equal(t, 1, targetID)
}

func TestAssignTargetNoLegalTargetReturnsFalseWithoutPanicking(t *testing.T) {
	attacker := engagement.Combatant{ID: 100, HP: 20}
	candidates := []engagement.Combatant{
		{ID: 1, HP: 10},
		{ID: 2, HP: 5},
	}

	neverLegal := func(attacker, defender engagement.Combatant) bool {
		return false
	}

	targetID, ok := engagement.AssignTarget(attacker, candidates, engagement.Weakest, neverLegal)

	assert.False(t, ok)
	assert.Equal(t, 0, targetID)
}

func TestAssignTargetEmptyCandidatesReturnsFalse(t *testing.T) {
	attacker := engagement.Combatant{ID: 100, HP: 20}

	targetID, ok := engagement.AssignTarget(attacker, nil, engagement.Weakest, alwaysLegal)

	assert.False(t, ok)
	assert.Equal(t, 0, targetID)
}

func TestPartyAliveTrueWhenAnyMemberHasPositiveHP(t *testing.T) {
	assert.True(t, engagement.PartyAlive([]engagement.Combatant{
		{ID: 1, HP: 0},
		{ID: 2, HP: 3},
	}))
}

func TestPartyAliveFalseWhenAllMembersDead(t *testing.T) {
	assert.False(t, engagement.PartyAlive([]engagement.Combatant{
		{ID: 1, HP: 0},
		{ID: 2, HP: 0},
	}))
}

func TestPartyAliveFalseWhenNoMembers(t *testing.T) {
	assert.False(t, engagement.PartyAlive(nil))
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/engagement/... -v`
Expected: FAIL — `package internal/engagement is not a package` (the
package doesn't exist yet).

- [x] **Step 3: Write the implementation**

Create `internal/engagement/engagement.go`:

```go
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
```

- [x] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/engagement/... -v`
Expected: PASS, all ten tests.

- [x] **Step 5: Run `go vet` and `gofmt` on the new package**

Run: `gofmt -l internal/engagement && go vet ./internal/engagement/...`
Expected: both commands print nothing.

- [x] **Step 6: Run the full build and race suite**

Run: `go build ./...`
Expected: builds cleanly.

Run: `go test -race ./...`
Expected: PASS, full suite, no regressions anywhere in the repo (the new
package has zero engine dependencies, so nothing else should be affected).

- [x] **Step 7: Commit**

```bash
git add internal/engagement/engagement.go internal/engagement/engagement_test.go
git commit -m "feat(engagement): add Phase 11b target-assignment domain logic"
```

## Task 2: Verification and status update

**Files:**
- Modify: `docs/PROJECT_STATUS.md`

- [x] **Step 1: Run full verification**

```bash
make generate
make validate
go test -race ./...
```

Expected: all three succeed (record the actual test/package counts from the
`go test -race ./...` output in the status update below — don't guess a
number).

- [x] **Step 2: Update `docs/PROJECT_STATUS.md`**

Add a "Phase 11b — Unit-vs-Unit Engagement (domain layer complete, hooks
integration deferred, <today's date>)" work-log entry following the
existing format (What/Why/Step completed/Key commits/Verification/Live
acceptance/Deferred). State plainly in "Deferred": the
`NewRound_DoCombat.go` hooks integration (blocked on 11c's legality
predicate, per the spec's own sequencing note — this is not an oversight,
it's the documented plan), 11c itself, 11d, real per-member preference
configuration (still a single project-wide default), and PvP/
`internal/parties` interaction. Update:
- The phase progress table row `11b | Unit-vs-unit engagement | Designed, not implemented`
  to something like `Domain layer complete; hooks integration awaits 11c`.
- The `## Current position` "Completed"/"Next" summary.
- The file header's `**HEAD:**` line.

- [x] **Step 3: Commit**

```bash
git add docs/PROJECT_STATUS.md
git commit -m "docs: record Phase 11b domain-layer completion"
```

---

## Self-Review Notes

- **Spec coverage:** weakest/strongest/random selection ✓ (Task 1, three
  dedicated tests). Filtering to legal candidates via an injected predicate
  ✓ (`TestAssignTargetFiltersByLegalFunc`). No-legal-target handling (false,
  no panic/loop) ✓ (`TestAssignTargetNoLegalTargetReturnsFalseWithoutPanicking`,
  `TestAssignTargetEmptyCandidatesReturnsFalse`). Reassignment implicitly
  covered: `AssignTarget` is stateless and always recomputes from current
  candidate HP, so "reassign after death" is the same code path as initial
  assignment with a dead entry present (`TestAssignTargetSkipsDeadCandidates`)
  — there is no separate "reassign" function to test because none is
  needed; this is called out explicitly rather than left implicit.
  Engagement-end cleanup ✓ (`PartyAlive`, three tests). `Engagement` type
  itself ✓ (defined, documented as a thin marker per the spec's "not
  persisted" model). The `NewRound_DoCombat.go` wiring and full
  "attacking a member engages the whole company" acceptance criterion are
  explicitly NOT covered by this plan — see the Architecture section's
  scope-boundary note; they require 11c and are out of scope by the spec's
  own words, not a gap in this plan.
- **Placeholder scan:** no TBD/TODO, every step has real code.
- **Type consistency:** `Combatant`, `Preference`, `LegalFunc`,
  `AssignTarget`, `PartyAlive`, `Engagement` are used identically between
  their Task 1 definition and every test.

---

**Plan complete and saved to `docs/superpowers/plans/2026-09-22-phase-11b-unit-engagement.md`.**
