# Phase 27a: Tutorial Framework and the First Lessons — Implementation Plan

Design: [27a spec](../specs/2026-09-25-phase-27a-tutorial-framework-design.md).
Each task is tests first.

- [ ] **Task 1: seams.** `internal/tutorial` (`Provider`, `Begin`, `Active`),
  `usercommands.OnCommandDone` fired by `TryCommand` with the resolved
  command, and recruiters resolving a copy to its template room. Tests
  first: `TestBeginWithoutProvider`, `TestOnCommandDoneResolvesAlias`,
  `TestRecruitInEphemeralCopy`.
- [ ] **Task 2: stages and progress** (`modules/tutorial/stages.go`). Tests
  first: `TestStageOrder`, `TestProgressRoundTrip`, `TestCharacterGate`,
  `TestCompanyGate`, `TestFormationGate`.
- [ ] **Task 3: the course** (`modules/tutorial/tutorial.go`): `Begin`
  (copies and placement), gate checks on `OnRefresh`, exit unlocking,
  graduation, skip, resume on `PlayerSpawn`. Tests first with fakes:
  `TestAdvanceUnlocksNextRoom`, `TestGraduateOnce`, `TestSkipGivesNothing`,
  `TestResumeRebuildsAtStage`, `TestNextOnlyWhenAllowed`.
- [ ] **Task 4: the `tutorial` command.** Tests first: status, next, and
  skip with its confirmation.
- [ ] **Task 5: content.** Rooms 900–903 rewritten, JS removed, the Muster
  Yard recruiter in the company config, and `help tutorial`. Tests first:
  `TestShippedTutorialRooms` and `TestShippedTutorialRecruiter`.
- [ ] **Task 6: `start` hand-off.** `start` calls `tutorial.Begin` when a
  provider is registered. Test: the legacy path without one.
- [ ] **Task 7: wiring** (`modules/tutorial/wiring_test.go`). Through
  `plugins.Load`: the whole course, resume, a second player's skip, and
  the clock unchanged.
- [ ] **Task 8: docs and verification.** Module guides,
  `go test -race ./...`, `make generate`, `make validate`, `make js-lint`,
  independent review, `docs/PROJECT_STATUS.md`.
