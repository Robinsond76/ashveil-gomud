# Phase 29f: Paced Combat Output — Implementation Plan

Design: [Phase 29f](../specs/2026-09-29-phase-29f-paced-combat-design.md).
Base: `135b72e`.

Implement in the worktree `.worktrees/phase-29f-paced-combat` on branch
`claude/next-phase-planning-9u98ae`. Pacing is presentation only: do not
change combat odds, damage, events, buffs, the world clock, or the round
count. Only the `DoCombat` listener's cadence changes. As each task lands,
run the focused tests for the packages it touches. Run the full
verification once, after the independent review's fixes.

## Task 1 — Combat cadence

Files:
- `internal/configs/config.timing.go`
- `_datafiles/config.yaml`
- `internal/hooks/combat_pace.go` (new)
- `internal/hooks/hooks.go`
- `internal/hooks/combat_pace_test.go`

- [ ] Write failing tests:
  - `Timing.CombatEveryRounds` defaults to 2 and a value below 1 becomes 2;
  - `combatRoundDue(n)` is true only on multiples of it, and on every round
    when it is 1;
  - the registered `NewRound` wrapper `CombatOnCadence` calls combat only
    on due rounds (a counting stub stands in for `DoCombat`).
- [ ] Add the setting, with a comment in `config.yaml`, and add
  `CombatRoundDuration()`.
- [ ] Add the wrapper and register it in place of `DoCombat`. Tests that
  call `hooks.DoCombat` directly are unaffected.
- [ ] Wiring test: over several `NewRound` events queued through the real
  event queue with every usual listener registered, survival drain,
  alignment drift, and `util.GetRoundCount()` advance exactly as before.
  Only combat skips rounds. It lives in `modules/company`, whose harness
  registers those listeners.

## Task 2 — Causal combat tagging in `internal/events`

Files:
- `internal/events/events.go`
- `internal/events/cause.go` (new)
- `internal/events/cause_test.go`

- [ ] Write failing tests:
  - `WithCause(r, fn)` makes events queued inside `fn` carry `r`;
  - while an event carrying `r` is dispatched, `Cause()` reports `r`, and
    events its listeners queue inherit it;
  - after dispatch, `Cause()` is 0 again;
  - an event requeued with `CancelAndRequeue` keeps its cause;
  - `Input` events never inherit a cause;
  - untagged events stay 0.
- [ ] Implement:
  - an unexported `cause` on `prioritizedEvent` and on requeues;
  - a current cause held in an atomic, set around `DoListeners` in
    `ProcessEvents` and by `WithCause`;
  - `Cause()`.

  `events.Message` gains no field and no event contract changes. Document
  the ordering assumption in `internal/events/AGENTS.md`.

## Task 3 — The pacer (`internal/combatpace`)

Files:
- `internal/combatpace/pace.go`
- `internal/combatpace/pacer.go`
- `internal/combatpace/pacer_test.go`
- `internal/combatpace/AGENTS.md`

- [ ] Write failing table tests, all on an explicit clock:
  - pace parsing and defaults (`normal`, or `off` for a screen reader);
  - gaps per pace;
  - the first line due at once, the rest in order;
  - gaps compressed to the window, clamped to 90% of the combat round;
  - a longer gap before a marked (dramatic) line;
  - a short gap before an indented line;
  - `Flush` and `FlushAll` return every held line in order;
  - `Busy`;
  - marks cleared by `StartRound`;
  - lines from an older round flushed ahead of a newer round's.
- [ ] Implement:
  - `Pace` and `Spec`;
  - `Parse` and `For(option, screenReader)`;
  - a `Pacer` with `Hold`, `Due(now)`, `Flush`, `FlushAll`, `Busy`,
    `Mark`, and `StartRound`;
  - `Default()` and `UseForTest`.

  Keep it pure. It has one mutex, never calls out while holding it, and
  returns lines instead of sending them.

## Task 4 — Delivery wiring in `internal/hooks`

Files:
- `internal/hooks/Message_SendMessages.go`
- `internal/hooks/RedrawPrompt_SendRedraw.go`
- `internal/hooks/combat_pace.go`
- `internal/hooks/hooks.go`
- `internal/events/eventtypes.go` (the new `CombatPaceDrained` event)

- [ ] Write failing tests for the helpers:
  - the pace lookup reads `ConfigOptions["combatpace"]`, with defaults;
  - a delivery seam (`deliver`) that tests replace;
  - a clock seam (`paceNow`).
- [ ] Implement:
  - `Message_SendMessage` holds a recipient's text when `events.Cause()`
    is non-zero and that recipient's pace isn't `off`, and otherwise
    delivers at once (quiet-message checks come first);
  - `CombatOnCadence` starts each round: it sends every held line
    (leftovers), clears marks, and snapshots prompts (D7);
  - a `NewTurn` listener, `ReleasePacedCombat`, sends due lines and
    redraws, and when a queue drains, clears the snapshot, redraws the
    live prompt, and queues `CombatPaceDrained`;
  - `RedrawPrompt` shows the snapshot while the player is busy;
  - an untagged `RoomChange`, `PlayerDespawn`, or copyover (a flush-only
    copyover contributor registered with the hooks) sends that player's
    held lines at once.
- [ ] **Wiring test (real entry points)** in
  `modules/company/wiring_pace_test.go`, driving the brawl harness through
  `CombatOnCadence` → `ProcessEvents` → `Message_SendMessage` → `NewTurn`
  releases on a fake clock:
  - the lines come out in the order they were produced, over the window;
  - leftovers flush before the next round's first line;
  - a `say` during the round arrives at once;
  - `off` and the screen-reader default arrive at once;
  - the companion death notice follows its death line, and the battle
    summary follows the closing line;
  - a walk out of the room flushes;
  - the prompt shows the snapshot until the lines drain.

## Task 5 — Beat marks for crit follow-ups

Files:
- `internal/combat/pain_reactions.go` (or where 29e appends reactions)
- `internal/hooks/combat_narration.go` (death lines)
- tests beside each

- [ ] Write failing tests: a surviving critical's pain lines (the victim
  and room variants) and a critical kill's death lines are marked with the
  pacer. Normal hits and misses are not.
- [ ] Mark the rendered strings when they are appended. Extend the wiring
  test: the pain line's gap is the longer one.

## Task 6 — `set combatpace`

Files:
- `internal/usercommands/set.go`
- `internal/usercommands/set_combatpace_test.go`

- [ ] Write failing tests:
  - `set combatpace slow` stores `slow`, and it survives a user save and
    load;
  - a bad value is rejected with the valid values;
  - `set` with no arguments lists it, with the default shown;
  - switching to `off` flushes held lines.
- [ ] Implement.

## Task 7 — Web client holds (D4, D7)

Files:
- `modules/gmcp/gmcp.Company.go`
- `modules/gmcp/gmcp.Char.go`
- `modules/gmcp/gmcp_pace_test.go`

- [ ] Write failing tests. While `combatpace` is busy for a user:
  - the company feed's refresh sends nothing, the battle view included;
  - `Char.Vitals` is not sent;
  - on `CombatPaceDrained`, both are sent with the current values.
- [ ] Implement: `modules/gmcp` imports only the engine package
  `combatpace`.

## Task 8 — Player help and tutorial

Files:
- `_datafiles/world/default/templates/help/combatpace.template` (new)
- `combat.template`
- `set.md`
- `narration.template`
- `_datafiles/world/default/keywords.yaml`
- `modules/tutorial/stages.go`
- `internal/usercommands/help_combat_test.go`
- the tutorial tests

- [ ] Write failing assertions:
  - `help combatpace` renders the four paces, the screen-reader default,
    the catch-up and flush rules, and the note that chat is never delayed;
  - its aliases `pace`, `pacing`, and `combat-pace` resolve;
  - `help combat` states the 8-second combat round and why actions wait,
    and links `help combatpace`;
  - `help set` lists `combatpace`;
  - `help narration` mentions the dramatic pause;
  - the Combat lesson points to `set combatpace` / `help combatpace`, and
    `TestTutorialHelpPointersExist` passes.
- [ ] Write the content and pass `go test ./internal/usercommands
  ./modules/tutorial`.

## Task 9 — Review, verification, integration

- [ ] Get an independent reviewer report on the full diff (`git diff
  135b72e..HEAD`). Focus:
  - the causal tagging's goroutine assumptions;
  - leaks or ordering holes in held lines;
  - lock ordering (`LockMud` → the pacer);
  - clock and round-count invariants;
  - copyover;
  - missing wiring coverage;
  - inaccurate help.
- [ ] Reproduce each finding. Fix the real ones with regression tests;
  note the rejected ones.
- [ ] Run `make generate`, `make validate`, `go test -race ./...`, and
  `git diff --check` once, after the fixes.
- [ ] Record the phase, the verification, and a **Review:** line in
  `docs/PROJECT_STATUS.md`; commit; merge to `master`; push.

## Review focus

- Nothing outside a combat round's causal chain is ever held.
  - Pay particular attention to `Input` and to events queued off the game
    loop.
- Held lines are never lost:
  - on move, quit, copyover, or a pace change;
  - no text is duplicated.
- The pacer never touches the round count; the cadence gates only
  `DoCombat`.
