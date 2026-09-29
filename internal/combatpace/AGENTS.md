# Combat Pace Package Guide

## Scope

- Ashveil Phase 29f: this package holds each player's combat lines and
  says when each is due. Its only job is pacing what players see.
- It is pure. It does no delivery, reads no config, and never touches the
  world clock or round count. `internal/hooks/combat_pace.go` wires it:
  - which text is held (`events.Cause()`);
  - delivery;
  - the clock (`paceNow`);
  - prompts;
  - flushing on move, quit, and copyover.

## Working Rules

- One mutex. It takes no other lock and never calls out while holding it.
  Callers run on the game loop under `util.LockMud`, so the lock order is
  `LockMud` → the pacer.
- A held line is never dropped:
  - `Hold` returns an older round's leftovers for sending at once;
  - `Flush` and `FlushAll` return everything held.
- Timing values live in `specs`. `help combatpace` quotes them; update it
  whenever they change.

## Verification

- `go test ./internal/combatpace` covers the scheduling.
- The real delivery path is covered by
  `modules/company/wiring_pace_test.go`.
