# Phase 10 Mounts Design

**Status:** Confirmed and implemented, 2026-09-22. Per the owner's standing
"continue implementation with whatever you recommend, no need to ask"
instruction, the recommended decisions below were taken directly.

## Goal

Add a minimal, durable, leader-owned mount that delivers the handoff's
(§35) "MVP mount effects: increases travel speed and/or increases cargo
capacity" — at least one real, wired gameplay effect, not just display
(unlike weather/encumbrance, where Option A — engine and display only —
was the right call because those phases had substantial non-wiring value
on their own; a mount with zero effect has none).

## Prior-art check (handoff requirement: inspect before building)

No native mount/horse/ride concept exists anywhere in GoMud or Ashveil
(`grep` for `mount`, `horse`, `ride`, `MountedOn`, `type Mount` across
`internal/`/`modules/` turns up nothing but unrelated substring matches
like "amount"). Fully greenfield — build `internal/mount` + `modules/mount`,
following the same domain/module split as expedition/camping/weather/
encumbrance.

## Scope decision: which MVP effect to wire

The handoff says "and/or," so only one effect is required. Applying the
same risk reasoning weather's and encumbrance's design docs already used
twice:

- **Cargo capacity bonus — low risk, wire it now.** `modules/encumbrance`'s
  `CurrentLoad` already computes a capacity from config; adding a mount's
  bonus to that computed (never persisted) value touches no schema on an
  already-shipped, tested phase. This is exactly the same shape Phase 8's
  design doc called "lower-risk" for camping rest recovery (multiplying a
  literal constant, no schema change) — here it's additive instead of
  multiplicative, same risk class.
- **Travel speed bonus — same schema risk flagged twice already, defer.**
  `expedition.TravelSession` only stores `ProfileName` and re-resolves the
  static profile by name every time; an effective-duration multiplier
  needs a new persisted field on an already-shipped phase's durable
  session, the same reviewed-follow-up territory weather's and
  encumbrance's `TravelDurationPct` already sit in.

**Decision:** wire cargo capacity now (real effect, zero schema risk to an
existing phase); travel speed stays computed-but-unwired data on the mount
spec, exactly like weather's/encumbrance's `TravelDurationPct`, deferred to
the same reviewed follow-up.

## Durable Model

`internal/mount` is GoMud-free, matching `internal/expedition`,
`internal/camping`, `internal/weather`, and `internal/encumbrance`. It is
the simplest of the five: "Later: mount fatigue, health, feed, terrain
suitability, individual assignment" (handoff §35) means the MVP mount
carries no decaying state at all — just a stable type assignment.

```go
type MountSpec struct {
    Type                    string
    Description             string
    CargoCapacityBonusGrams int // wired: added to modules/encumbrance's capacity
    TravelDurationPct       int // computed, not yet wired (same shape as weather/encumbrance)
}

type Mount struct {
    LeaderUserID int
    Type         string // validated against the configured spec table at apply time
}
```

- One mount per leader (leader-keyed, like every other Ashveil registry).
  No per-member assignment — "individual assignment" is explicitly a
  "Later" item in the handoff.
- No acquisition economy this phase — "recruitment economics" is already a
  standing deferred item from Phase 2 onward; `mount stable <type>` is
  free and instant, same simplicity Phase 2's `company summon` shipped
  with.
- A read-only `Provider` seam in `internal/mount`
  (`CapacityBonusGrams(leaderUserID) int`), the same shape as
  `weather.Provider`/`camping.ViewProvider`, so `modules/encumbrance` can
  consult it without importing `modules/mount`.

## Module

`modules/mount` owns:

- A mount-type table as config (embedded default + on-disk overlay), same
  pattern as `weather`'s biome table / `encumbrance`'s load bands. A
  malformed entry is rejected and logged, never guessed.
- A durable, leader-keyed `Mount` registry, YAML-persisted.
- `mount` / `mount stable <type>` / `mount release` commands.
- Registers itself as `internal/mount`'s `Provider` in `init()`.

## Integration

`modules/encumbrance`'s `CurrentLoad` adds `mount.CapacityBonus(leaderUserID)`
(0 without a tracked mount) to its configured base capacity. This is the
only change to the already-shipped Phase 9 module, and it changes a
computed value only — no persisted schema touched.

## Constraints and Deferrals

- Never advances `gametime`, the round counter, or moves any player.
- Never mutates combat, health, or the native
  `Character.CarryCapacity()`/`go.go` per-move throttle (same boundary
  Phase 9 drew).
- A mount type with no configured spec is rejected at config parse time,
  never guessed; a mount record whose type no longer exists in the table
  is retained untouched for operator repair, never silently dropped or
  defaulted.
- Deferred: mount fatigue/health/feed, terrain suitability, individual
  (per-member) assignment, an acquisition/cost economy, and — per the
  scope decision above — actually wiring `TravelDurationPct` into
  `modules/expedition`.

## Acceptance Criteria (for this phase's slice)

- `mount stable <type>` durably assigns a configured mount type to the
  leader; `mount release` removes it; `mount`/`mount status` shows the
  current assignment.
- A leader with a mount sees `modules/encumbrance`'s computed capacity
  increase by the mount's `CargoCapacityBonusGrams` — a real, observable
  gameplay effect.
- Persists across restart/copyover like every other Ashveil registry.
- A malformed mount-type config entry, or a mount record referencing a
  type no longer configured, is rejected/retained and logged, never
  guessed.
- Focused domain/module tests, `make generate`, `make validate`, and
  `go test -race ./...` provide verification.
