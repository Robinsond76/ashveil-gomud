# Phase 12b: Weighted Encounter Tables

## Prior-art check

12a (`docs/superpowers/specs/2026-09-22-phase-12a-encounter-kinds-design.md`)
widened `internal/expedition.InterruptionKind` from one hardcoded value to
three (`fallen-tree`, `discovery`, `tracks`), each with its own text via
`InterruptionText`. A `TravelProfile` still names exactly one `Kind` at
config time — `oak-road` is still hardwired to `fallen-tree`. 12a's design
doc named "weighted/random encounter tables (a profile rolling among
several possible kinds instead of naming exactly one)" as the next slice,
deferred because it needs a random source and a weight schema this pass
didn't need to make yet.

## Scope

**In scope:** let an `InterruptionProfile` configure a weighted table of
kinds instead of exactly one, and roll among them once, at fire time, the
result persisted into `TravelInterruption` exactly like today's single-kind
case. No new session states, commands, or encounter subsystem — still the
same interrupt/resume/return mechanism 12a reused.

**Explicitly deferred:**
- Shipping new route content that actually uses a weighted table (e.g. a
  new named profile wired to room exits). 12a already named this as
  12b's decision to make and this pass keeps deferring it: proving the
  mechanism through unit and module-level tests is enough for now, and
  real new routes are world-content work independent of the engine
  change — better done once there's a reason to add a new route (new
  zone/exit work), not invented just to exercise this feature.
- Anything from Phase 12's subsystem-backed encounter types (combat,
  merchant, dialogue, branching choice, camp-opportunity) — unchanged
  from 12a's deferral list.
- Re-rolling per view or per session (a fired interruption's kind is
  fixed and persisted at the moment it fires, matching today's exact
  semantics for `TravelInterruption` — restart/copyover must render the
  same choice).

## Durable model

`InterruptionProfile` gains an alternate form:

```go
// WeightedInterruptionKind is one entry in a weighted interruption roll:
// Kind fires with probability proportional to Weight among its table.
type WeightedInterruptionKind struct {
    Kind   InterruptionKind `yaml:"kind"`
    Weight uint             `yaml:"weight"`
}

type InterruptionProfile struct {
    Kind       InterruptionKind           `yaml:"kind,omitempty"`
    Kinds      []WeightedInterruptionKind `yaml:"kinds,omitempty"`
    Checkpoint uint8                      `yaml:"checkpoint"`
}
```

Exactly one of `Kind` (today's singular form, unchanged) or `Kinds` (new)
must be set — `Validate()` rejects both empty and both populated. A `Kinds`
table must be non-empty, every entry a `Valid()` kind with `Weight > 0`.

A new pure method resolves the roll:

```go
// ResolveKind picks which kind fires. A singular-Kind profile returns Kind
// unchanged and ignores roll entirely (today's exact behavior, still
// deterministic). A Kinds table selects proportionally to Weight: roll is
// reduced modulo the total weight, so any caller-supplied uint64 source
// works — the caller owns where randomness comes from, this stays pure.
func (i InterruptionProfile) ResolveKind(roll uint64) (InterruptionKind, error)
```

## Integration

`TravelSession.Interrupt(now, p)` is **not changed** — it still reads
`p.Interruption.Kind` directly (unchanged pure state-transition function,
all 14 existing tests keep passing unmodified). Instead,
`modules/expedition.interruptLocked` resolves the roll *before* calling
`Interrupt`: when `profile.Interruption.Kinds` is set, it calls
`profile.Interruption.ResolveKind(m.rollUint64())` and builds a resolved
copy of the profile (`Kind` set to the rolled result, `Kinds` cleared) to
pass into the existing, untouched `session.Interrupt(now, resolvedProfile)`.
This keeps the random roll at the engine-facing edge — the same place
`m.clock()` already lives — while the durable domain package stays pure and
its existing test suite untouched.

`ExpeditionModule` gains a `rollUint64 func() uint64` seam (mirroring
`clock func() time.Time`), defaulting to `rand.Uint64()` in `init()` and
overridable by tests for deterministic assertions on which kind fires from
a weighted table.

## Constraints and deferrals

- Never advances GoMud's global clock/round count — untouched.
- Must survive restart/copyover — untouched: the roll happens once, at
  fire time, and only the *resolved* singular `Kind` is ever persisted
  into `TravelInterruption`. A restart mid-interruption re-renders the
  same already-resolved kind exactly as today; it never re-rolls.
- `oak-road`'s shipped config stays on plain `Kind: fallen-tree` — no
  config file changes in this pass (see Scope).

## Acceptance criteria

- `InterruptionProfile.Validate()` accepts a well-formed `Kinds` table and
  rejects: both `Kind` and `Kinds` set, an empty `Kinds` table, a
  zero-weight entry, and an invalid kind inside the table.
- `ResolveKind` returns `Kind` unchanged (ignoring `roll`) for a
  singular-form profile, and proportionally selects among `Kinds` for a
  weighted-table profile (deterministic for a fixed `roll`, verified by
  edge-of-band assertions).
- `modules/expedition`'s `interruptLocked` fires with the resolved kind's
  `InterruptionText`, provable via `rollUint64` injection.
- `go test -race ./...`, `make generate`, `make validate` all pass.
