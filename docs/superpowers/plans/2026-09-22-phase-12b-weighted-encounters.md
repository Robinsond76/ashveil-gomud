# Phase 12b: Weighted Encounter Tables — implementation plan

See the companion design doc
(`docs/superpowers/specs/2026-09-22-phase-12b-weighted-encounters-design.md`).
Open decisions (no new route content this pass, roll resolved at the
module edge not inside `Interrupt`, weight-modulo selection) resolved by
recommendation per the standing instruction.

## Tasks

- [ ] **`internal/expedition`: `WeightedInterruptionKind`, `Kinds`,
      `Validate`, `ResolveKind`.**
  - Tests first, in `internal/expedition/expedition_test.go`:
    - `TestInterruptionProfileValidateAcceptsWeightedKindsTable`
    - `TestInterruptionProfileValidateRejectsBothKindAndKinds`
    - `TestInterruptionProfileValidateRejectsEmptyKindsTable`
    - `TestInterruptionProfileValidateRejectsZeroWeightEntry`
    - `TestInterruptionProfileValidateRejectsInvalidKindInsideTable`
    - `TestResolveKindSingularFormIgnoresRoll` (any roll value returns the
      one configured `Kind`)
    - `TestResolveKindWeightedTableSelectsProportionally` (e.g. weights
      3/1 for two kinds; rolls 0,1,2 hit the first, roll 3 hits the
      second, roll 4 wraps back via modulo to the first — assert each
      boundary)
    - `TestResolveKindRejectsInvalidProfile` (an invalid profile's
      `ResolveKind` returns the same error `Validate` would)
  - Then implement.
- [ ] **`modules/expedition`: `rollUint64` seam + resolve-before-`Interrupt`
      in `interruptLocked`.**
  - Check the module's existing test harness for how `clock`/`scheduler`
    are overridden in tests (mirror that exact pattern for `rollUint64`).
  - Test: a profile with a `Kinds` table and an injected fixed
    `rollUint64` fires the expected resolved kind (assert via
    `session.Interruption.Kind` on the interrupted session, or the
    `sendToLeader` text if that becomes assertable — otherwise assert on
    the persisted session state directly, which the existing interrupt
    tests already do).
  - Implement: add `rollUint64 func() uint64` field, default
    `rand.Uint64` in `init()`; in `interruptLocked`, when
    `profile.Interruption.Kinds` is non-empty, resolve via `ResolveKind`
    into a copied, singular-form `InterruptionProfile` before calling
    `session.Interrupt`.
- [ ] `gofmt -l`, `go vet ./internal/expedition/... ./modules/expedition/...`,
      `go build ./...`, `go test -race ./...` after each task.
- [ ] `make generate`, `make validate`.
- [ ] Update `docs/PROJECT_STATUS.md`: header, Current position/Next, phase
      table row 12b (from "Not started" to complete), new work-log entry.
