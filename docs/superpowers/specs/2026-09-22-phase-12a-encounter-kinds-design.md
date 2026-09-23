# Phase 12a: Encounter Kinds

## Prior-art check

Phase 12 ("Rich Expedition Encounters", `docs/ASHVEIL_GOMUD_AGENT_HANDOFF.md`
section 37) is deliberately unscoped in the handoff doc — a bullet list of
example encounter types (combat, merchants, discoveries, route choices,
injured NPC, weather hazard, camp opportunity, ruined site, tracks,
resources, social encounters) and the line "This becomes a content system
rather than engine plumbing." No durable model, integration point, or
acceptance criteria exists yet, and no prior design/plan doc for it does
either.

The only existing encounter-shaped mechanism is Phase 6's travel
interruption (`internal/expedition.InterruptionKind`/`InterruptionProfile`,
wired in `modules/expedition/expedition.go`'s `syncAndCompleteLocked` →
`interruptLocked`): a `TravelProfile` may name exactly one
`InterruptionKind` (today only `fallen-tree` is valid) to fire, once, at a
configured checkpoint (1..9 of 10). Firing pauses the session
(`Interrupted`); the leader resolves it with `travel resume` (continue) or
`travel return` (turn back). The Phase 6 design doc explicitly named this
as the seam a later encounter system should reuse ("A later handler may use
the same interruption identity and resolve/resume contract" —
`docs/superpowers/specs/2026-09-21-phase-6-travel-interruptions-design.md`).

## Scope

Phase 12's full example list spans several genuinely different subsystems:
combat encounters need mob spawning and Phase 11 formation-combat wiring;
merchants need a shop/trade flow; injured NPCs and social encounters need
dialogue; route choices need branching UX beyond resume/return; camp
opportunities cross into `internal/camping`. Building all of that in one
pass would repeat Phase 11's mistake of scoping too broadly at once (that
phase took five separate branches: 11a/11b/11c domain layers plus two more
wiring passes). This pass, 12a, is the first and narrowest slice:

**Generalize `InterruptionKind` from a single hardcoded value into a small,
extensible, data-driven set of non-combat, non-branching flavor kinds**,
proving the "kind is real content, not just an enum with one member"
abstraction before building anything that needs new subsystems.

**In scope:**
- Two new kinds, `discovery` and `tracks`, alongside the existing
  `fallen-tree`. Each fires and resolves through the *exact* existing
  interrupt/resume/return mechanism — no new session states, no new
  commands, no new persistence shape beyond widening the `Kind` enum.
- Per-kind display text, extracted from the single hardcoded string in
  `interruptionTextLocked` into a per-kind lookup, so adding a fourth kind
  later is a data/text change, not a new code path.
- A route may still only configure one `Kind` (today's `InterruptionProfile`
  shape, unchanged) — `oak-road` keeps using `fallen-tree`; a route wanting
  `discovery` or `tracks` instead just names it in config.

**Explicitly deferred (not this pass):**
- **Weighted/random encounter tables** (a profile rolling among several
  possible kinds instead of naming exactly one at config time). This is
  the natural 12b: it needs a random source, a weight schema, and
  determinism/testability decisions this pass doesn't need to make yet.
- **Combat, merchant, injured-NPC, route-choice, camp-opportunity, ruined-
  site, resource, and social encounters** — each needs a subsystem this
  pass doesn't touch (mob spawning + Phase 11 formation combat, a shop
  flow, dialogue, branching choice UX, or `internal/camping` integration).
  Future 12-series passes, one subsystem at a time, same shape as 11a-11c.
- **Non-travel encounters** (room-based, camp-based). Phase 12a stays
  inside the existing travel-interruption trigger point.

## Durable model

`internal/expedition.InterruptionKind` gains two more valid values:

```go
const (
    FallenTree InterruptionKind = "fallen-tree"
    Discovery  InterruptionKind = "discovery"
    Tracks     InterruptionKind = "tracks"
)
```

`Valid()` accepts all three. `InterruptionProfile`, `TravelInterruption`,
and `TravelSession` are structurally unchanged — only the set of legal
`Kind` values widens, so existing persisted sessions and the `oak-road`
config keep decoding exactly as they do today (backward compatible, no
migration).

A new package-level lookup in `internal/expedition` maps kind to its
player-facing text, so `modules/expedition` doesn't need a switch statement
per call site:

```go
// InterruptionText returns the player-facing description for a fired
// interruption's kind. Every valid kind has an entry; an unknown kind
// (which Validate already rejects everywhere a session is constructed)
// falls back to a generic message rather than panicking.
func InterruptionText(kind InterruptionKind, profileName string) string
```

This keeps content (the strings) inside the domain package `internal/`
prefers for GoMud-free logic, alongside the enum it describes, rather than
leaving it in the GoMud-facing module — matching the project's existing
"pure package owns the data-driven content" pattern (e.g. `survival`,
`weather`).

## Integration

`modules/expedition/expedition.go`'s `interruptionTextLocked` becomes a
thin call to `expedition.InterruptionText(candidate.Interruption.Kind,
candidate.ProfileName)`, replacing its one hardcoded sentence. No other
call site changes: `interruptLocked`, `syncAndCompleteLocked`, and the
`travel resume`/`travel return` commands are all kind-agnostic already
(they operate on `session.Interruption` generically), which is exactly why
this pass doesn't need to touch them.

## Constraints and deferrals

- Never advances GoMud's global clock/round count — untouched, this pass
  only widens an enum and a text lookup.
- Must survive restart/copyover — untouched: `TravelInterruption`'s
  `yaml:"kind"` field already round-trips any string, and `Validate()`
  already rejects unknown kinds on load, so no migration is needed for
  existing persisted `fallen-tree` sessions.
- `oak-road`'s shipped config is left on `fallen-tree` (no behavior change
  for existing content); a second example profile is *not* added in data
  in this pass — proving the abstraction via unit tests is enough, and
  adding real additional route content is better sequenced with 12b's
  weighted-table work so a route can be authored once against the final
  shape.

## Acceptance criteria

- `InterruptionKind.Valid()` accepts `fallen-tree`, `discovery`, and
  `tracks`, and rejects anything else (existing test coverage extended,
  not replaced).
- `InterruptionText` returns distinct, non-empty text for each of the
  three kinds, and a safe fallback for an invalid kind.
- `modules/expedition`'s interruption message for a firing session matches
  `InterruptionText`'s output for that session's kind — proven by a
  focused test if the module's existing test harness supports asserting
  on `sendToLeader`'s text (check before writing; a pure-domain test of
  `InterruptionText` itself is required either way).
- `go test -race ./...`, `make generate`, `make validate` all pass.
