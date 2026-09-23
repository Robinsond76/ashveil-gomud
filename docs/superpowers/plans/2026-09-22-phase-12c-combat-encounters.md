# Phase 12c: Combat Encounters During Travel — implementation plan

See the companion design doc
(`docs/superpowers/specs/2026-09-22-phase-12c-combat-encounters-design.md`).
Open decisions (mob commanded to attack immediately rather than relying on
idle-tick AI, resume/return gated on live-instance check rather than a new
session state, one mob not a party, no new route config this pass) resolved
by recommendation per the standing instruction — this session's explicit
direction was "just do combat encounters, keep the rest as future ideas."

## Tasks

- [ ] **`internal/expedition`: `Combat` kind, `CombatMobID`,
      `CombatMobInstanceId`.**
  - Tests first, in `internal/expedition/expedition_test.go`:
    - `TestInterruptionKindValidAcceptsCombat`
    - extend `TestInterruptionTextReturnsDistinctNonEmptyTextPerKind` (or
      add a sibling) to cover `Combat` too
    - `TestInterruptionProfileValidateRejectsCombatKindWithoutMobID`
    - `TestInterruptionProfileValidateRejectsCombatInKindsTableWithoutMobID`
    - `TestInterruptionProfileValidateAcceptsCombatKindWithMobID`
    - `TestInterruptionProfileValidateAllowsZeroMobIDForNonCombatKind`
      (regression guard: a `fallen-tree`/`discovery`/`tracks` profile still
      needs no `CombatMobID`)
  - Then implement: add the constant, extend `Valid()`/`InterruptionText`,
    add the `CombatMobID`/`CombatMobInstanceId` fields, extend `Validate()`.
- [ ] **`modules/expedition`: `MobSpawner` seam + spawn-on-fire +
      gate-on-active.**
  - Tests first, in `modules/expedition/expedition_test.go` (new
    `fakeMobSpawner` implementing `MobSpawner`, tracking calls and
    returning canned results, mirroring the existing `fakeMover`/
    `fakeSurvival` style):
    - `TestInterruptionCombatKindSpawnsHostileMobAndPersistsInstanceId`
    - `TestInterruptionCombatSpawnFailureStillFiresPlainPause` (fails
      open: session still transitions to `Interrupted`,
      `CombatMobInstanceId` stays 0)
    - `TestResumeRefusedWhileCombatEncounterStillActive`
    - `TestResumeAllowedOnceCombatEncounterResolved`
    - `TestReturnRefusedWhileCombatEncounterStillActive`
  - Then implement: `MobSpawner` interface, `nativeMobSpawner` (via
    `mobs.NewMobById`/`room.AddMob`/`mob.Command`/`mobs.GetInstance`,
    mirroring `modules/company/runtime.go`'s `Spawn`/`IsLive`), the
    `mobSpawner` field + `init()` default, the spawn-on-fire block in
    `interruptLocked`, and the active-encounter gate in `resume` and
    `returnToOrigin`.
- [ ] `gofmt -l`, `go vet ./internal/expedition/... ./modules/expedition/...`,
      `go build ./...`, `go test -race ./...` after each task.
- [ ] `make generate`, `make validate`.
- [ ] Update `docs/PROJECT_STATUS.md`: header, Current position/Next
      (record that combat encounters are the only Phase 12 encounter
      subsystem being built now; the rest are future ideas, not planned
      work), phase table row 12c, new work-log entry.
