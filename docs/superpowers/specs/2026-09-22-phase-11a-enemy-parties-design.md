# Phase 11a Enemy Parties Design

**Status:** Confirmed via collaborative design session, 2026-09-22. See
`docs/superpowers/specs/2026-09-22-phase-11-formation-combat-design.md`
for the shared prior-art check and cross-sub-phase decisions this spec
depends on.

## Goal

Give mobs a first-class "party" concept — a stable group identity with its
own 3×3 formation, auto-assigned by role — so a room can contain multiple
separate hostile groups, each displayed and (in 11b) engageable as a unit,
mirroring the player company on the other side of the fight.

## Prior-art check

See the shared prior-art check in the Phase 11 overview doc. Specific to
this sub-phase: `mobs.Mob.Groups []string` already exists and is already
the signal `mobs.MakeHostile` uses for group hostility — reused here as the
signal for "these mobs are one party," rather than inventing a second
grouping mechanism. `internal/combat.MobRank` already computes `EHP` and
`DPS` per mob spec at its effective level — reused for the auto-assignment
heuristic rather than adding new stats.

## Scope

Included:

- A `Party` domain type: stable identity, member mob instance IDs, a
  `company.Formation`-typed 3×3 grid (reusing the existing type directly,
  not a parallel reimplementation).
- Formation happens automatically when mobs sharing a room and a `Groups`
  tag are first assembled into a party (on room load/mob spawn) — never
  hand-authored per spawn entry, and never player-controlled.
- Auto-assignment heuristic, applied to a party of up to 5 members
  (see Constraints for the cap): rank members by `MobRank.EHP` descending.
  The highest-`EHP` member(s) fill the front row first (up to 3), the
  next-highest fill the middle row, and the remainder (typically the
  highest-`DPS`/lowest-`EHP` members) fill the back row.
- A solo mob (no `Groups` tag shared with anything else present) is still
  a party of one, per the design session's "even a single mob is a unit"
  decision — occupying the front-row center cell by default.
- Room display: when a room has more than one hostile party present, each
  is listed as its own entry (not each mob individually), with aggregate
  messaging for a multi-mob party (e.g., "a pack of 3 goblins").

Excluded (deferred; see Constraints and Deferrals): actually engaging a
party in combat (11b), any tactical effect of the formation (11c),
mid-combat formation changes, and anything involving `internal/parties`
(the separate multiplayer grouping system, left untouched).

## Durable Model

A `Party` is **not** persisted — it is assembled fresh whenever mobs are
present in a room (matching `Aggro`'s own non-persistence, per the shared
prior-art check: combat-adjacent state does not survive restart today, and
there is no reason for enemy grouping to be the first such state that
does). `internal/mobparty` (or similar; final package name decided at plan
time) is GoMud-free:

```go
type Party struct {
    ID      string // stable within one room's lifetime, e.g. derived from the Groups tag
    Members []int  // mob InstanceIds
    Formation company.Formation
}

// Assemble groups mobInstanceIds sharing a Groups tag into parties, then
// assigns each party's formation via the EHP/DPS heuristic. A mob with no
// Groups tag becomes its own solo party.
func Assemble(mobs []MobSummary) []Party
```

`MobSummary` is a small local struct (InstanceId, Groups, EHP, DPS) so this
package stays GoMud-free and testable without importing `internal/mobs` or
`internal/combat` directly into the pure assembly logic — the module layer
adapts real mob data into `MobSummary`.

## Module / Integration

This is unusually light on "module" work compared to prior phases — there
is no new durable registry, no new commands, no new config file. The
integration point is wherever a room's mob list is displayed
(`internal/rooms`/`internal/usercommands/look.go`'s mob listing) and
wherever 11b will need to enumerate "the parties present in this room."
`Assemble` is called on demand (not cached, not event-driven) each time a
party listing is needed, since there is no persisted state to keep in
sync — this avoids inventing a cache-invalidation problem for something
that's cheap to recompute.

## Constraints and Deferrals

- Never advances `gametime`, the round counter, or moves any mob.
- Never mutates combat, health, or hostility (`Groups`/`MakeHostile`
  semantics are read, never changed).
- A party's formation is capped at 5 members, mirroring the company cap
  (`internal/company`'s `MaxCompanions`-derived limit) — a `Groups` tag
  shared by more than 5 mobs in one room splits into multiple parties
  rather than overflowing one formation; exact split rule (e.g., by spawn
  order) is a plan-time detail, not a design-level decision.
- Deferred: actual engagement (11b), tactical effects (11c), any change to
  `internal/parties`, mid-combat rearrangement, and any admin/authoring UI
  for hand-placing a specific mob's formation slot (the heuristic is
  authoritative for this phase).

## Acceptance Criteria

- A room with mobs sharing a `Groups` tag shows them as one party listing,
  not N individual mob lines.
- A solo mob (no shared `Groups` tag with anything else present) is
  treated as a party of one.
- Each assembled party has a valid, capped, heuristically-ordered
  `company.Formation` (higher-`EHP` members in front).
- Focused domain tests cover: single-mob parties, multi-mob grouping by
  `Groups` tag, the EHP-ordering heuristic, and the 5-member cap/split
  behavior. `make generate`, `make validate`, and `go test -race ./...`
  provide verification.
