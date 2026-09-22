# Phase 11b Unit-vs-Unit Engagement Design

**Status:** Confirmed via collaborative design session, 2026-09-22. See
the Phase 11 overview doc for shared prior-art and cross-sub-phase
decisions. Depends on 11a (enemy parties must exist to engage).

## Goal

Make a fight actually mean "your company vs. their party," not "one
character's `Aggro` happens to point at one mob while everyone else stands
around." Close the specific gap the prior-art check found: a companion
today only retaliates reactively if *it* personally gets attacked, with no
coordinated targeting at all.

## Prior-art check

See the Phase 11 overview doc's shared section. The load-bearing finding
for this sub-phase specifically: `characters.Aggro` is genuinely
single-target per character and that is **not** being rearchitected — this
phase adds a coordinated *initiation* routine that sets each present
company member's own individual `Aggro`, it does not turn `Aggro` itself
into a list or build a new "encounter" object spanning multiple
characters.

## Scope

Included:

- **Engagement trigger:** attacking (or being attacked by) any member of
  an enemy party engages the attacker's whole company against that party.
  A charmed companion with no target gets one assigned; the existing
  reactive retaliation snippets in `NewRound_DoCombat.go` become one case
  of the more general rule rather than the only path in.
- **Target assignment:** each company member without a current legal
  target gets one via a **minimal targeting-preference rule** — favor
  weakest (lowest current HP), strongest (highest current HP), or random,
  selectable per member (a simple default, not full player-configurable
  UI, is acceptable for this phase; see Constraints) — filtered to only
  candidates that are legal per 11c's lateral-range/reach rules. (11b
  depends on 11c's legality predicate existing as a callable function;
  11c's redirect/reach *behavior* is not required to be wired into combat
  yet for 11b to use its pure legality check. See the sequencing note in
  Constraints.)
- **Re-assignment on target loss:** when a company member's target dies
  (or otherwise becomes permanently invalid), they get a new target via
  the same preference rule, as long as the enemy party still has living,
  legal members and the company member is still engaged.
- **Engagement end:** when a party has no living members, the engagement
  ends and any company member still targeting it stops acting (mirrors
  today's dead-target handling, just applied at the party level as a
  bulk check rather than per-character).

Excluded (deferred; see Constraints and Deferrals): the tactical
consequences of formation (redirect-on-interception, reach gating — 11c
implements the *behavior*, 11b only needs 11c's target-legality
*predicate*), full AI personality (wounded-seeking, threat-awareness —
11d), and anything involving `internal/parties` or PvP.

## Durable Model

No new persisted state — matches `Aggro`'s own non-persistence (shared
prior-art finding). An **Engagement** is a runtime-only concept:

```go
// Engagement is not persisted; it is derived each round from which
// company members currently have Aggro pointed at a member of which
// party, plus which members still need a target assigned.
type Engagement struct {
    LeaderUserID int
    PartyID      string
}

// AssignTarget picks a legal target for one company member from a party,
// per the given preference, or false if no legal target exists.
func AssignTarget(member Combatant, party mobparty.Party, pref Preference, legal func(attacker, defender Combatant) bool) (targetID int, ok bool)

type Preference int
const (
    Weakest Preference = iota
    Strongest
    Random
)
```

`legal` is 11c's exported predicate (row/column legality), injected rather
than imported directly, keeping 11b's assignment logic testable without
11c's full implementation present — the two sub-phases can be planned and
tested independently even though 11c's predicate must exist before 11b's
integration lands for real.

## Module / Integration

The integration point is `NewRound_DoCombat.go`'s existing per-round
target-revalidation blocks (the same seam the original Phase 11 research
identified as the natural gate location) — before an attack resolves,
check whether the attacker has a legal target; if not, attempt
`AssignTarget` before falling through to "no target, skip this round."
This is additive to the existing dead-target-skip behavior, not a
replacement of it.

## Constraints and Deferrals

- Never advances `gametime`, round count, or moves any character.
- Never mutates damage math — this phase is entirely about *who* has a
  target, not how hard they hit.
- Targeting preference defaults to a single project-wide rule (e.g.
  "weakest") for this phase rather than shipping per-member player
  configuration — a real preference-selection command/UI is a natural
  11d-or-later extension once the base mechanic is proven, matching how
  the original handoff scoped "AI target-selection personality" as its
  own item (9) rather than bundling it into the first tactical slice.
- **Sequencing dependency on 11c:** `AssignTarget`'s `legal` parameter
  must be satisfiable by 11c's predicate before this phase's integration
  step (not its domain-layer implementation) can land for real. The plan
  for 11b should build and test `AssignTarget` against a fake/stub
  `legal` function, and the actual `NewRound_DoCombat.go` wiring is the
  last task, done together with or after 11c lands.
- Deferred: full personality-driven target selection (11d), PvP
  engagement, and any interaction with `internal/parties`.

## Acceptance Criteria

- Attacking a member of an enemy party (11a) results in every present,
  able company member acquiring a legal target within that party by the
  next round, without any additional player command.
- A company member whose target dies is reassigned within that same
  party, if one is still legal, without player action.
- An engagement ends cleanly (no lingering `Aggro` pointed at nothing)
  once a party has no living members.
- Focused domain tests cover: weakest/strongest/random selection,
  reassignment on target death, no-legal-target handling (does nothing,
  doesn't panic or loop), and engagement-end cleanup. `make generate`,
  `make validate`, and `go test -race ./...` provide verification.
