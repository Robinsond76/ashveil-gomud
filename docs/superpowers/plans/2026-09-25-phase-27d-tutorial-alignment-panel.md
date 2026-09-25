# Phase 27d: Tutorial Alignment Lesson and Browser Panel — Plan

Design: [27d spec](../specs/2026-09-25-phase-27d-tutorial-alignment-panel-design.md).

## Task 1: The Alignment stage

- [x] Tests first (`stages_test.go`, `tutorial_test.go`): order and rooms
  (Alignment at index 7 before Departure); subcommand inspections
  (`company alignment` counts only with that first word); registration by
  command; Alignment passes on the three inspections.
- [x] `StageAlignment`; inspection keys with a subcommand in
  `onCommandDone` and `required`.

## Task 2: The tutorial view

- [x] Tests first: `internal/tutorial` `ViewOf` without a provider, with a
  non-viewer, with a viewer; the module's view per stage (title, number,
  goal, checklist, plain hints) and none when not active.
- [x] `tutorial.View`, `Viewer`, `ViewOf`; `TutorialModule.TutorialView`,
  sharing the checklist with the terminal `view`.

## Task 3: The Tutorial GMCP package

- [x] Tests first (`modules/gmcp`): the payload from a view; `{}` when
  inactive; sent on change only; forget on spawn and despawn; a request;
  nothing for a connection that hasn't accepted GMCP.
- [x] `gmcp.Tutorial.go`; `!!GMCP(Tutorial)` in `gmcp.go`;
  `help gmcp-tutorial`.

## Task 4: The web client window

- [x] `window-tutorial.js`, included in `webclient-pure.html`.
- [x] `scripts/browser/tutorial-panel-harness.html` and
  `tutorial-panel-check.mjs`; run in Chromium; screenshots.

## Task 5: Content

- [x] Tests first (`shipped_test.go`): eight rooms; Corvin offered at 907
  with an alignment a new company refuses; help lists Alignment.
- [x] Room 907; 903's back exit to 907; `TutorialRooms`; mob 69; the
  recruiter; help.

## Task 6: Wiring

- [x] `modules/tutorial/wiring_test.go`: the Alignment lesson through
  `plugins.Load`.
- [x] `Tutorial` through `plugins.Load` with the gmcp module and real
  `GMCPOut` events (in the same wiring test).

## Task 7: Docs, verification, review

- [x] Module guides; `go test -race ./...`, `make generate`,
  `make validate`, `make js-lint`.
- [x] Independent review; verify findings; record in
  `docs/PROJECT_STATUS.md` with a **Review:** line.
