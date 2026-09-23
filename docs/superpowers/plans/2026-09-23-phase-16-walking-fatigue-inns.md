# Phase 16: Walking Fatigue, Inns, and Travel/Rest Modifiers — implementation plan

> **For agentic workers:** use `superpowers:subagent-driven-development` or
> `superpowers:executing-plans` when they are installed. Otherwise follow the
> manual fallback in `CLAUDE.md`. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Wire the deferred weather, load, and mount travel and rest
modifiers. Add per-step walking fatigue with penalty bands. Add paid inn
rest that grants *Well Rested*.

**Spec:** `docs/superpowers/specs/2026-09-23-phase-16-walking-fatigue-inns-design.md`

**Status:** Planned, not started. The spec's open decisions 1–8 must be
confirmed, or replaced by the user's standing "proceed with your
recommendation", before Task 1. Record which happened in the spec.

**Delegation:** Tasks 1, 3, and 5 are pure and self-contained, so they can
go to the cheap implementer tier with a narrow brief. Tasks 2, 4, 6, 7, and 8
involve persisted-state recovery, timers, game-loop buff mutation, or lock
ordering. The lead implements those directly, per `CLAUDE.md`.

## Global constraints

- Never advance or fast-forward the world clock or round count. Travel and
  rest stay real-time UTC sessions. `NewRound` listeners only read the
  clock.
- Every new persisted field has an upgrade default that reproduces
  pre-phase behaviour: nil `Modifiers` is 100/100/100, and zero `Recovery`
  is 20.
- Survival is a leaf lock. Module mutexes never call another Ashveil
  module while held. Resolve provider reads outside `expedition.mu`.
- Mutate `Character` (buffs, gold) only on the game loop, never in timer
  callbacks.
- Invalid persisted records are kept and logged, never guessed or deleted.
- All balance numbers go in module config overlays.

## Review focus

- A pre-upgrade `TravelSession` or `RestSession` completes exactly as
  before.
- Exertion and rest replays after a crash never raise `ErrExertionConflict`
  or `ErrRestConflict`: modified amounts are fixed and persisted.
- An inn stay completes and recovers exactly once across a crash, restart,
  or copyover. A stale timer does nothing. A failed save refunds gold.
- Well Rested is granted only from `NewRound`, never from a timer
  goroutine.
- Walking charges only successful ordinary `go` steps, and only the
  members who actually moved.
- The shipped buffs leave max HP unchanged.

### Task 1: Pure travel modifiers (delegable)

**Files:** Modify `internal/expedition/expedition.go` and
`internal/expedition/expedition_test.go`.

- [ ] Write failing tests:
  - `TestTravelModifiersValidate`: 25–300 each, and nil is valid.
  - `TestWithModifiers` (table):
    - nil is the identity
    - duration scaling
    - hunger and thirst scale by `ExertionPct`
    - fatigue scales by `ExertionPct × FatiguePct`
    - round half-up, and a positive base never reaches 0
    - an interruption checkpoint is unchanged
  - `TestSessionValidateRejectsBadModifiers`.
  - `TestExertionTelescopesWithModifiers`: a completed modified route
    charges exactly the modified total.
- [ ] Run `go test ./internal/expedition -count=1` and confirm it fails.
- [ ] Implement `TravelModifiers{DurationPct, ExertionPct, FatiguePct}`
  (yaml `duration_pct` / `exertion_pct` / `fatigue_pct`),
  `TravelSession.Modifiers *TravelModifiers` (`modifiers,omitempty`),
  `TravelProfile.WithModifiers`, and the extended `Validate`.
- [ ] Run `go test -race ./internal/expedition -count=1`.

### Task 2: Wire travel modifiers into `modules/expedition`

**Files:**
- modify `internal/encumbrance/provider.go`, `modules/encumbrance/encumbrance.go`
- modify `internal/mount/mount.go`, `internal/mount/provider.go`, `modules/mount/mount.go`, `modules/mount/files/data-overlays/config.yaml`
- modify `modules/expedition/expedition.go`, `modules/expedition/expedition_test.go`
- modify the encumbrance, mount, and weather module tests

- [ ] Write failing tests:
  - encumbrance `CurrentModifiers` returns the resolved band, and returns
    100/100 with no load.
  - mount `FatiguePct` parses, validates (0 or 25–300), and `Modifiers`
    returns it; no mount gives 100/100.
  - expedition, with fake modifier sources:
    - `StartTravel` persists the resolved `Modifiers`
    - progress, checkpoint, and completion timing use the modified
      duration
    - checkpoint exertion totals match the modified profile
    - an interruption fires at the same route fraction
    - reload keeps the modifiers and finishes a journey that was overdue
      when the server restarted
    - a legacy session with nil `Modifiers` completes with the original
      duration and exertion
    - an untracked zone or no provider gives 100
    - departure and status text name the modifiers
- [ ] Run the focused tests and confirm they fail.
- [ ] Implement:
  - the provider queries
  - a `modifierSources` injection seam on `ExpeditionModule` (native:
    weather, encumbrance, mount), resolved *before* `m.mu` in `StartTravel`
  - the single `sessionProfileLocked(session)` helper, replacing every
    `m.profile(session.ProfileName)` lookup that feeds duration or
    exertion math
  - the updated texts, and removal of the "not yet applied" wording in the
    encumbrance and mount status and config comments
- [ ] **Wiring test:** register the real `modules/weather`,
  `modules/encumbrance`, and `modules/mount` providers with a raining
  tracked zone, a loaded company, and a pack-horse. `StartTravel` through
  `expedition.Start` persists the expected product.
- [ ] Run `go test -race ./internal/expedition ./internal/encumbrance ./internal/mount ./modules/expedition ./modules/encumbrance ./modules/mount -count=1`.
- [ ] Commit: `feat(expedition): apply weather, load, and mount travel modifiers`.

### Task 3: Pure rest recovery and inn stay model (delegable)

**Files:** Modify `internal/camping/camping.go` and
`internal/camping/camping_test.go`.

- [ ] Write failing tests:
  - `RestSession.Recovery`: `RecoveryFor` returns 20 for 0, and the value
    otherwise; negative values are rejected.
  - `Camp.StartRestWithRecovery` persists the amount.
  - `InnStay` validation, `StartInnStay(leader, room, price, recovery,
    now)`, `ProgressAt`/`Due` with a configurable duration, and
    `Complete`. A completed stay can't complete again, and
    `MarkRecoveryApplied` is idempotent.
- [ ] Run `go test ./internal/camping -count=1` and confirm it fails.
- [ ] Implement the fields and pure transitions. Keep `StartRest` working
  for existing callers.
- [ ] Run `go test -race ./internal/camping -count=1`.

### Task 4: Weather-scaled camp rest, then inns in `modules/camping`

**Files:**
- modify `modules/camping/camping.go`, `modules/camping/camping_test.go`, `modules/camping/files/data-overlays/config.yaml`
- create `modules/camping/files/datafiles/buffs/1030-well_rested.yaml`
- modify `_datafiles/world/default/rooms/frostfang/61.yaml` to add the `inn` tag

- [ ] Write failing tests for camp rest:
  - `startRest` fixes `Recovery` from a fake weather condition
    (`RestRecoveryPct` 75 gives 15)
  - completion passes 15, and a replay after reload passes 15 again with
    no conflict
  - a legacy record with `Recovery` 0 passes 20
- [ ] Write failing tests for inns, with fake store, clock, scheduler,
  survival, gold, and weather:
  - `inn status` in an inn and outside one
  - `inn rest` refusals: not an inn, combat, downed, travelling (via an
    injected expedition check), camp mid-rest, already staying, can't pay
  - payment is deducted and the stay persisted; a save failure refunds and
    leaves no record
  - the timer completes and recovers exactly once
    (`inn-rest-<leader>-<room>-<start>`)
  - a stale generation does nothing
  - crash after Completed and before recovery: retry recovers once
  - an overdue stay after `load()` completes once
  - `MovementBlocked` and `RenderCampView` while staying
  - Well Rested is **not** applied in the timer path; the `NewRound`
    listener grants buff 1030 to an online leader and live companions,
    then deletes the record
  - an offline leader keeps the record until they are online
- [ ] Run `go test ./modules/camping -count=1` and confirm it fails.
- [ ] Implement:
  - weather-scaled `startRest` (the zone condition is read before `m.mu`)
  - the `Stays` registry with strict decode
  - stay timers and generations kept separate from camp timers
  - the `inn` subcommands
  - gold deduction and refund
  - recovery through the existing idempotent path
  - the movement block and view
  - the `NewRound` grant listener
  - config parsing with defaults: `InnRoomTag` inn, `InnPrice` 10,
    `InnRecovery` 50, `InnRestDuration` 60s, `InnBuffID` 1030
  - the Well Rested buff file: flag `well-rested`, xpscale 5, +1
    speed/perception, 450 rounds, non-permanent, no vitality
- [ ] **Wiring tests:**
  - `inn rest` through the module's registered command handler on a real
    room tagged `inn`
  - movement is then refused through the real `usercommands.Go`
  - buff 1030 loads from the shipped file with real config, and max HP is
    unchanged after it is applied
- [ ] Run `go test -race ./internal/camping ./modules/camping ./internal/usercommands -count=1`.
- [ ] Commit: `feat(camping): weather-scaled camp rest and paid inn stays`.

### Task 5: Pure walking-fatigue math (delegable)

**Files:** Create `internal/fatigue/fatigue.go`,
`internal/fatigue/fatigue_test.go`, and `internal/fatigue/provider.go`.

- [ ] Write failing tests:
  - `TestStepCostCenti` (table): each factor alone (terrain, weather, load,
    mount, cold band, rested), all combined, clamping at 0% and 400%, an
    `indoor` room costing 0, and a city costing 0.
  - `TestAccumulate`: remainder carry across steps, whole-point output, and
    no negative values.
  - `TestBandBuffFor`: 100–51 none, 50–26 Weary, 25–1 Exhausted, 0 Spent.
  - `TestStepTakenNoProvider` does nothing.
- [ ] Run `go test ./internal/fatigue -count=1` and confirm it fails.
- [ ] Implement `StepInputs`, `StepCostCenti`, `Accumulate(remainder,
  cost) (points, newRemainder)`, `BandBuff`, and the `StepProvider` seam
  with `SetStepProvider` and `StepTaken(userID, fromRoomID, toRoomID)`.
- [ ] Run `go test -race ./internal/fatigue -count=1`.

### Task 6: `modules/fatigue`: step charge, remainders, band buffs

**Files:**
- create `modules/fatigue/fatigue.go`, `modules/fatigue/fatigue_test.go`
- create `modules/fatigue/files/data-overlays/config.yaml`
- create `modules/fatigue/files/datafiles/buffs/1040-weary.yaml`, `1041-exhausted.yaml`, `1042-spent.yaml`
- add a read-only cold-band query seam to `internal/climate` and implement it in `modules/exposure`, with a test

- [ ] Write failing tests, with fake store, survival drain, roster, and
  providers:
  - config parsing and defaults (the terrain table as a list, not a map,
    per Phase 15's loader note)
  - a step drains the leader and only the companions that were in the
    origin room
  - remainders carry, persist, and reload
  - roster pruning
  - the exposure cold band and Well Rested each change the cost
  - `NewRound` every `TickRounds` syncs band buffs: add, swap, remove, and
    refresh against permabuff reconciliation
  - unspawned companions are skipped and keep their remainder
  - the save is dirty-flag flushed
  - a drain error is logged and doesn't panic
- [ ] Run `go test ./modules/fatigue -count=1` and confirm it fails.
- [ ] Implement the module (init, store, lock, tick, step provider
  registration) and ship the buffs: no vitality, non-permanent, refreshed
  each tick.
- [ ] Run `make generate` and check that `modules/all-modules.go` gained
  exactly `modules/fatigue`.
- [ ] Run `go test -race ./internal/fatigue ./internal/climate ./modules/fatigue ./modules/exposure -count=1`.

### Task 7: Hook walking into `usercommands.Go`

**Files:** Modify `internal/usercommands/go.go`; create or modify
`internal/usercommands/go_fatigue_test.go`.

- [ ] Write failing wiring tests with the real `modules/fatigue` provider,
  real rooms, and a real user:
  - a successful forest step drains
  - a city step doesn't drain
  - a move refused by combat, a lock, or action points doesn't drain
  - a profiled travel exit doesn't drain
  - a companion in the origin room is charged, and one elsewhere isn't
  - the Well Rested buff halves the drain
- [ ] Run the focused test and confirm it fails.
- [ ] Add `fatigue.StepTaken(user.UserId, originRoomId, destRoom.RoomId)`
  right after `rooms.MoveToRoom` succeeds.
- [ ] Run `go test -race ./internal/usercommands ./modules/fatigue -count=1`.
- [ ] **Wiring test:** buffs 1040–1042 load from the shipped files with
  real config, and max HP is unchanged.
- [ ] Commit: `feat(fatigue): per-step walking fatigue and fatigue penalty bands`.

### Task 8: Verify, review, and record

**Files:** Modify `docs/PROJECT_STATUS.md`, this plan's checkboxes, and
the spec status line.

- [ ] Run `gofmt`, `go vet`, `go build ./...`, `go test -race ./...`,
  `make generate` (check for no unexpected drift), and `make validate`.
- [ ] Boot the server once and check the new buffs load with no new
  errors.
- [ ] If an interactive Telnet client is available, run live acceptance:
  1. walk the forest and watch fatigue fall
  2. `inn rest` at Frostfire Inn
  3. confirm no world-time skip, one recovery, and Well Rested
  4. confirm the walking drain is halved

  If not, record that it wasn't run.
- [ ] **Review gate:** dispatch an independent reviewer subagent
  (`general-purpose`, most capable tier) over `git diff <base>..HEAD`.
  Give it the spec, the clock, restart, and lock-ordering invariants, and
  the review focus above. It reports findings only.
- [ ] Reproduce each finding. Fix the real ones, each with a regression
  test. Note the rejected ones and why. Rerun the full verification.
- [ ] Update `docs/PROJECT_STATUS.md`:
  - the Phase 16 row
  - a work-log entry with a **Review:** line
  - the upstream room-rental overlap noted for builders
  - Phase 17 (archetypes) as next

  Then merge and push.
