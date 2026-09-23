# Phase 12a: Encounter Kinds — implementation plan

See the companion design doc
(`docs/superpowers/specs/2026-09-22-phase-12a-encounter-kinds-design.md`)
for full reasoning. Open decisions (kind set, text-lookup location, no
config change to `oak-road`, weighted rolling deferred to 12b) were
resolved by recommendation per the standing "proceed with your
recommendation, record which happened" instruction — no user round-trip
needed for a narrow, backward-compatible enum widening.

## Tasks

- [ ] **`internal/expedition`: widen `InterruptionKind` and add
      `InterruptionText`.**
  - Tests first, in `internal/expedition/expedition_test.go`:
    - `TestInterruptionKindValidAcceptsAllThreeKinds` (table over
      `fallen-tree`/`discovery`/`tracks`, all `Valid() == true`).
    - `TestInterruptionKindValidRejectsUnknown` (extend/confirm existing
      coverage still holds for e.g. `"campfire"`).
    - `TestInterruptionTextReturnsDistinctNonEmptyTextPerKind` (all three
      kinds produce different, non-empty strings, each including the
      route name).
    - `TestInterruptionTextFallsBackForUnknownKind` (a non-`Valid` kind
      still returns a non-empty generic string, never panics/empty).
  - Then implement: add `Discovery`, `Tracks` constants, extend `Valid()`,
    add `InterruptionText(kind InterruptionKind, profileName string) string`.
- [ ] **`modules/expedition`: replace the hardcoded interruption sentence.**
  - Check whether the module's existing test harness can assert on
    `sendToLeader`'s captured text for an interrupted session (grep
    existing interrupt-flow tests for a fake/spy sender). If so, add
    `TestInterruptLockedSendsKindSpecificText` (or extend an existing
    interrupt test) asserting the sent text matches
    `expedition.InterruptionText` for the session's kind. If no such spy
    exists, skip this test and note it in the work-log's verification
    section — `internal/expedition`'s own `InterruptionText` tests are the
    binding coverage either way.
  - Implement: `interruptionTextLocked` becomes
    `expedition.InterruptionText(candidate.Interruption.Kind, candidate.ProfileName)`;
    delete the old hardcoded sentence.
- [ ] `gofmt -l`, `go vet ./internal/expedition/... ./modules/expedition/...`,
      `go build ./...`, `go test -race ./...` after each of the two tasks
      above.
- [ ] `make generate`, `make validate`.
- [ ] Update `docs/PROJECT_STATUS.md`: header, Current position/Next, phase
      table row 12 (from "Not started" to "In progress: 12a encounter-kind
      abstraction shipped; weighted tables and subsystem-backed kinds
      (combat/merchant/social/etc.) still ahead"), new work-log entry,
      Known issues if anything new applies.
