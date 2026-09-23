# Phase 16: Walking Fatigue, Inns, and Travel/Rest Multipliers — implementation plan

See `docs/superpowers/specs/2026-09-23-phase-16-walking-fatigue-inns-design.md`.
The user confirmed the decisions on 2026-09-23 and they are recorded in the
design doc. Decision 3 changed: mount fatigue relief covers at most two
riders and applies to walking only. Implementation has not started; wait for
the user's go-ahead.

Implement directly, without delegating to Haiku. Almost every task touches
timers, persisted-state recovery, or lock ordering (the `CLAUDE.md`
escalation threshold). Task 1 alone may be delegated if wanted: it is pure,
one package, and has a clear spec.

Invariants for every task:

- Never advance the world clock or round count.
- Every new durable field survives restart and copyover, and a legacy
  record loads neutral.
- Lock order is walking → {encumbrance, mount, weather, exposure} →
  survival, and expedition → {encumbrance, mount, weather} → survival.
- Off-loop timers never touch characters or buffs.

## Tasks

- [ ] **0. Baseline.**
  - The decisions are confirmed and recorded (done while planning).
  - `go test -race ./...` is green on the branch head.
  - Check whether Dunmar is weather-tracked (decision 6 assumes it isn't).
- [ ] **1. `internal/walking` (pure).** Files: `internal/walking/walking.go`,
      `walking_test.go`, `provider.go`.
  - Tests first:
    - `TestTerrainCost`: settlement biomes, a lit biome, and an `indoor`
      tag cost 0; the biome table; a `strain:<n>` tag overrides; a malformed
      tag is ignored
    - `TestStepCost`: each multiplier alone, all combined, the
      [25, 400] clamp, half-up rounding, and the design's worked-example
      table
    - `TestAccrue`: the carry keeps the remainder; drain is whole points;
      a large cost drains several points; the carry stays in 0..99
    - `TestColdPct`: a band-to-multiplier table, with heat bands at 100
    - `TestSteppedProvider`: nil provider is a no-op; a registered provider
      receives the arguments
  - Then implement.
- [ ] **2. Read seams.** Files: `internal/encumbrance/provider.go`,
      `internal/mount/{mount,provider}.go`, `internal/climate/climate.go`,
      plus the modules implementing them (`modules/encumbrance`,
      `modules/mount`, `modules/exposure`).
  - Tests first:
    - `encumbrance.CurrentBand` resolves the configured band for a real
      load, and is neutral without a provider
    - `MountSpec.FatiguePct` validation (0 means 100, range 25–300),
      `MountSpec.Riders` (0 means 2, negative rejected), and config parsing
    - `mount.Relief` returns (100, 0) without a mount and (spec pct, riders)
      with one; `mount.TravelDurationPct` is 100 without a mount
    - `climate.ExposureOf` reads the exposure registry under its lock, and
      is absent without a provider
  - Data: pack-horse `FatiguePct: 75`, `Riders: 2`; config comments no longer say
    "informational".
- [ ] **3. Travel multipliers (`internal/expedition` + `modules/expedition`).**
  - Tests first, domain:
    - `TestEffectiveProfileLegacy`: all pcts 0 leaves the profile unchanged
    - `TestEffectiveProfileScales`: duration, all needs, and fatigue-only
    - `TestValidatePctRange`
    - `TestInterruptionCheckpointScales`
  - Tests first, module:
    - multipliers locked at `StartTravel` from a fake weather, load, and
      mount; the mount changes duration only, never `FatiguePct`
    - they persist and reload; a weather change mid-journey doesn't change
      the session
    - **a legacy session YAML without the fields reloads and completes on
      the original schedule and cost**
    - the Collapsed refusal (a member at 0 fatigue) with no state change
    - the departure line names the factors
  - Implement `m.sessionProfile(session)` and replace every
    `m.profile(session.ProfileName)` read of `Duration`/`Exertion`.
- [ ] **4. Camp rest multiplier (`internal/camping` + `modules/camping`).**
  - Tests first:
    - `RestSession.Recovery` 0 → `FatigueRecovery` (legacy)
    - `ProgressFor(now, duration)` matches the old `ProgressAt` at 60 s
    - `camp rest` in a tracked zone stores the scaled recovery
    - completion applies the scaled amount
    - a legacy camp YAML completes with 20
- [ ] **5. Inns (`internal/camping` + `modules/camping`).** New file
      `modules/camping/inn.go`; `InnStay` in `internal/camping/inn.go`.
  - Tests first, domain: `InnStay` start, due, and complete transitions and
    validation.
  - Tests first, module:
    - `inn` in a non-inn room refuses
    - `inn` shows the price for leader + companions
    - `inn rest` without gold refuses with no change
    - `inn rest` takes gold and persists the stay
    - a failed save refunds the gold and leaves no stay
    - movement is blocked during a stay
    - an active camp rest and an inn stay exclude each other
    - the timer completes the stay, applies `InnFatigueRecovery` once (a
      replay is idempotent), and sets `WellRestedPending`
    - **the timer callback never calls the buff seam** (a fake buff applier
      asserts it is only called from the `NewRound` handler)
    - the `NewRound` handler grants buff 1030 to the leader and spawned
      companions, then clears the pending flag
    - reload mid-stay reschedules; reload after it's overdue completes once
  - Config: `InnRoomTag`, `PricePerMember`, `InnRestDuration`,
    `InnFatigueRecovery`, `WellRestedBuffId`.
- [ ] **6. `modules/walking`.** Files: `modules/walking/walking.go`,
      `walking_test.go`, `files/data-overlays/config.yaml`,
      `files/datafiles/buffs/1030-*.yaml`, `1031-*.yaml`, `1032-*.yaml`.
  - Tests first:
    - config parsing and defaults (terrain table, multipliers, `TickRounds`)
    - `Stepped` drains each company member by the accrued whole points and
      keeps the carries
    - the carry is persisted by the save callback and reloaded
    - Well Rested (the `well-rested` flag) halves it
    - mount relief goes to the leader and the lowest companion ID only; a
      third member pays full cost; an unspawned companion doesn't take a
      seat
    - a cold exposure band raises it
    - a settlement step costs nothing and touches no survival state
    - pruning drops members who have left the roster
    - the tick applies and swaps the Exhausted and Collapsed buffs, never
      with vitality mods (a shipped-buff test loads the real files, as in
      Phase 15)
    - `strain` lists the cost here and each factor
  - Then `make generate`.
- [ ] **7. `Go` wiring.** File: `internal/usercommands/go.go`. Add
      `walking.Stepped(user.UserId, originRoomId, destRoom.RoomId)` after a
      successful `MoveToRoom`, on the instant path only.
- [ ] **8. Wiring tests (real entry points).**
  - `usercommands.Go` from a real forest room with a real user and a
    charmed companion charges both members; the same in a `city` room
    charges nothing
  - a profiled exit starts a session and the arrival adds no walking charge
  - `go` on a travel exit with a heavy real inventory and a tracked zone
    starts a longer session
  - `camp rest` through the camping user command in a tracked zone
  - `inn rest` through the user command in real room 2003 with real gold
    and a spawned companion, through to Well Rested on the next round
  - a real exposure registry value feeds the walking cold multiplier
- [ ] **9. Concurrency test.** Under `-race`, run concurrently: `Stepped`
      calls, a camping and inn timer completion, an expedition checkpoint,
      and an exposure tick drain. Run it with `-count=20`.
- [ ] **10. Content.**
  - Dunmar room 2003, *The Waymark Inn* (biome `city`, tags `inn`,
    `indoor`), with an exit from 2001 and back
  - the `inn` tag on Frostfire Inn 61
  - per decision 6: move 2002 into a new zone, *Old King's Road*
    (`defaultbiome: forest`), with a `zone-config.yaml`, keeping 2002's
    room ID, tags, and the 2001 `travel_profile` exit unchanged
  - Test: the world loads, and 2001→2002 travel and camping still work in
    the new zone.
- [ ] **11. Verify.** `gofmt`, `go vet`, `go build ./...`,
      `go test -race ./...`, `make generate`, `make validate`. Boot the
      server and check the buff count, the new room, and that the logs have
      no new errors.
- [ ] **12. Review gate.** Dispatch an independent reviewer subagent over
      `git diff <base>..HEAD` with this design, the invariants, and the lock
      order. Verify every finding yourself, fix the real ones with
      regression tests, and note the rejected ones. Re-run Task 11.
- [ ] **13. Record.** Add a Phase 16 entry to `docs/PROJECT_STATUS.md` (with
      the **Review:** line), update the phase table and "Next", tick this
      plan, then merge and push.
