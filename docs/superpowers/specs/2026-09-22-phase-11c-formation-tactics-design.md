# Phase 11c Formation Tactics Design

**Status:** Confirmed via collaborative design session, 2026-09-22. See
the Phase 11 overview doc for shared prior-art and cross-sub-phase
decisions. Depends on 11a (enemy formation must exist) and provides the
target-legality predicate 11b's `AssignTarget` needs.

## Goal

Implement the handoff's original five first-tactical-rules (§36, §15)
against real formations on both sides: front-row interception, melee
reach (column-occupancy based), polearm/innate reach, ranged full-column
access, lateral column range, and read-only adjacency queries.

## The reach model (resolved in detail during design)

This took several rounds of back-and-forth to pin down precisely, because
the first two framings (attacker's-own-row-vs-fixed-distance, then
"self-heals via ally repositioning") were both wrong — formation is locked
once combat starts, so neither an attacker's own row nor mid-fight
rearrangement can change reach; only enemy attrition can.

**The correct model is column-occupancy, not row-distance:**

- Within one *targetable* column (see lateral range below) of the
  defending formation, a plain melee attack can only land on that
  column's current **frontmost occupied slot** — whoever is standing in
  front, blocking the lane. A back-row occupant behind a living front-row
  occupant in the *same column* is not reachable by plain melee, even
  though a different column's back-row occupant (with nothing in front of
  it in its own column) is fully reachable.
- **Reach** — granted by a polearm-class weapon, or as an innate
  characteristic on some mobs/monsters (not exclusively tied to wielding a
  specific item; see Durable Model) — extends that to the column's
  frontmost occupant *or* the one directly behind them (front + middle of
  that column). It does not reach the back slot of a column while the
  front or middle slot in that column is occupied.
- **Ranged** ignores column depth/blocking entirely — ranged can target
  any occupied slot in a targetable column.
- **This is genuinely self-healing without clearing a stored target:** if
  a column's blocking front occupant dies, whoever was behind them becomes
  the new frontmost occupant and becomes reachable next round automatically
  — the underlying mechanism the design session confirmed is enemy
  attrition thinning a column, not the attacker moving or the defending
  formation being rearranged (which cannot happen mid-fight).
- **Lateral range**, independent of weapon/reach: a combatant can only
  target enemy columns within its own column index ±1 (a middle-column
  combatant reaches all three enemy columns; an edge-column combatant
  reaches two). This combines with the reach/column-occupancy rule — both
  must hold for a target to be legal.

Worked example directly from the design session: enemy grid has A at
(front, col 1), B at (back, col 1), C at (back, col 2). A plain-melee
attacker in lateral range of both columns can hit A (frontmost of col 1)
and C (frontmost — the *only* occupant — of col 2), but not B (blocked by
A in the same column). If A dies, B becomes col 1's new frontmost and
becomes reachable next round with no player action.

## Prior-art check

See the Phase 11 overview doc's shared section. Specific to this
sub-phase: there is no existing "Reach" trait anywhere in `items.ItemSpec`
or `internal/mobs` today — this phase adds it as a new boolean-ish
characteristic, sourced from either an equipped weapon's subtype or an
innate mob flag (see Durable Model), explicitly designed to leave room for
other mob characteristics later without redesigning this one (per the
design session: "there could be other characteristics too... another
time" — not built now).

## Scope

Included:

- Row/column adjacency and legality as pure, GoMud-free functions,
  operating on `company.Formation` (reused directly, both sides).
- Front-row interception: while any front-row member of the defending
  formation is alive, an attack aimed at a back-row (or middle-row,
  matching the same "blocked by whoever's in front in that column" logic
  as reach) member of that formation redirects to the front-row member in
  the *same column* instead — strict, no chance roll, no guard stance.
- The `legal(attacker, defender) bool` predicate 11b's `AssignTarget`
  needs, covering lateral range + reach/column-occupancy together.
- Adjacency queries exposed read-only (e.g. via `formation status` or a
  `formation` command addition) — satisfies "adjacency exists and can be
  queried" without any attack-legality change of its own.

Excluded (deferred; see Constraints and Deferrals): guard stance/chance-
based interception, row/column AoE, formation buffs, flanking/exposure,
movement-in-combat, mid-fight rearrangement, and full AI personality
(11d).

## Durable Model

`internal/formationcombat` (or added directly to `internal/company` — a
plan-time call given the small size of this logic; either keeps the
functions pure and GoMud-free):

```go
type Reach int
const (
    ReachNone Reach = iota // plain melee: frontmost occupant of column only
    ReachExtended            // polearm or innate: frontmost or one behind
    ReachAny                 // ranged: any depth
)

// FrontmostOccupant returns the row of the nearest-to-front living
// occupant of a column, or false if the column is empty.
func FrontmostOccupant(f Formation, col int, alive map[MemberKey]bool) (row int, ok bool)

// InLateralRange reports whether defenderCol is within attackerCol's ±1.
func InLateralRange(attackerCol, defenderCol int) bool

// InReachDepth reports whether targetRow is reachable from the column's
// frontmost occupied row, given a reach class.
func InReachDepth(frontmostRow, targetRow int, reach Reach) bool

// Legal combines both checks — the predicate 11b consumes.
func Legal(attackerCol int, f Formation, targetKey MemberKey, alive map[MemberKey]bool, reach Reach) bool

// InterceptFrontRow returns the front-row member in the same column as an
// attack aimed at a non-front-row member, if one is alive; ok=false means
// no interception applies (attack proceeds against the original target).
func InterceptFrontRow(f Formation, targetKey MemberKey, alive map[MemberKey]bool) (interceptor MemberKey, ok bool)
```

A combatant's `Reach` value is computed at the point of use (equipped
weapon subtype, or a mob's innate flag) rather than stored redundantly —
avoids a second source of truth alongside the equipped item / mob spec.

## Module / Integration

Two integration points in `NewRound_DoCombat.go`'s existing per-round
target-revalidation blocks:

- Before a party-side attack resolves, call `InterceptFrontRow` on the
  defending formation; if it returns an interceptor, the attack lands on
  them instead of the original target.
- `Legal` is exposed as the predicate 11b's `AssignTarget` requires (see
  11b's spec) and is also re-checked each round for an already-assigned
  target — a failure skips that round's attack without clearing `Aggro`,
  per the self-healing reasoning above.

## Constraints and Deferrals

- Never advances `gametime`, round count, or damage math beyond
  redirecting/gating *who* a strike lands on.
- Formation is read-only during combat in this phase — no rearrangement
  mid-fight, matching the design session's explicit confirmation.
- A member with no resolvable formation row (shouldn't happen given
  Phase 3's/11a's invariants, but defensively) fails closed: not treated
  as front row, not treated as reachable, rather than guessed.
- The `Reach` trait's exact per-item/per-mob-spec source field (a new
  `items.ItemSpec` field for polearm subtype, a new `mobs.Mob`/mob-spec
  flag for innate reach) is a plan-time schema decision, not re-litigated
  here — the design-level contract is just "Reach is derivable from either
  source, uniformly."
- Deferred: guard-stance/chance-based interception, row/column AoE,
  formation buffs, flanking/exposure, movement-in-combat skills, other mob
  characteristics beyond Reach, and full AI targeting personality (11d).

## Acceptance Criteria

- The worked example above (A front col 1, B back col 1, C back col 2)
  passes as a table-driven test: plain melee legal for A and C, illegal
  for B; B becomes legal once A is removed.
- A polearm/Reach-flagged attacker can additionally hit a column's middle
  occupant when the front slot is also occupied, still blocked from the
  back slot.
- Ranged ignores column depth entirely within lateral range.
- An attack aimed at a non-front-row defender redirects to a living
  front-row member in the same column; with no living front-row member in
  that column, the original target stands.
- A reach-illegal round skips the attack without clearing the attacker's
  stored target, and the same target becomes legal automatically once the
  blocking occupant dies (a two-round test: fails, kill the blocker,
  passes).
- Focused domain tests cover all of the above plus lateral-range edge
  columns (col 0 and col 2 each reach only 2 of 3 enemy columns) and
  adjacency query read-only correctness. `make generate`, `make validate`,
  and `go test -race ./...` provide verification.
