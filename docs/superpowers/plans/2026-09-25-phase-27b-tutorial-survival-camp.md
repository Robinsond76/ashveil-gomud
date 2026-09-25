# Phase 27b: Tutorial Survival and Camp Lessons — Plan

Design: [27b spec](../specs/2026-09-25-phase-27b-tutorial-survival-camp-design.md).

## Task 1: Engine seams

- [x] Tests first: `internal/survival` — `OnProvision` fires on a
  successful `Provision` only; `internal/usercommands` —
  `IsRegistered`; `internal/camping` — `AbandonCamp` without a provider is
  a no-op, with one it delegates.
- [x] `survival.Provisioned` and `survival.OnProvision` (fired in the
  `Provision` wrapper).
- [x] `usercommands.IsRegistered(cmd)`.
- [x] `camping.CampAbandoner` / `SetCampAbandoner` / `AbandonCamp`.

Files: `internal/survival/survival.go`, `internal/usercommands/usercommands.go`,
`internal/camping/provider.go` and tests.

## Task 2: Camping module implements AbandonCamp

- [x] Tests first (`modules/camping/abandon_test.go`): an idle camp, a
  resting camp (timer stopped, no recovery), a finished camp (recovery kept),
  no camp, a failed save keeps it; the inn stay is untouched.
- [x] `AbandonCamp`, sharing the camp half of `AbandonForDeath`; registered
  in `init`.

## Task 3: Stages as data

- [x] Tests first (`stages_test.go`): order Character, Company, Formation,
  Survival, Camp, Departure; rooms 0, 1, 2, 4, 5, 3; `required` filters by
  registration; `survivalDone` needs fed, watered, and the inspections.
- [x] `Stage.Inspections`; Survival and Camp stages; `required`;
  `inspected`; `survivalDone`.

## Task 4: Module behaviour

- [x] Tests first (`tutorial_test.go`): inspections count in their own
  stage; a provision counts only in Survival; supplies once and only for
  what's missing; Camp passes on Rested; the camp is struck on pass, skip,
  leave, and place; Survival and Camp waivers; the view's checklist.
- [x] Seams `restTier`, `restReporting`, `abandonCamp`, `registered`,
  `carries`; `onProvision`; `enterStage` (supplies); config
  `RationItemId`, `WaterItemId`.

## Task 5: Content

- [x] Tests first (`shipped_test.go`): six rooms, one per stage, no
  scripts or forward exits; tags (`indoor` hall, `outdoor` yard,
  `camping` campground); supply items exist; help mentions Survival and
  Camp.
- [x] Rooms 904, 905; 900 tagged `indoor`; 903's back exit to 905; map
  coordinates; `TutorialRooms` in `_datafiles/config.yaml`; help template;
  tutorial config overlay.

## Task 6: Wiring test

- [x] Extend `wiring_test.go`: load the survival-side modules; walk
  Survival with the real `eat`/`drink` and inspections; Camp with the real
  `camp`, `camp fire`, `camp rest` in the copy; Rested passes and the camp
  is struck; a second player's `tutorial skip yes` mid-rest leaves no camp;
  the clock is unchanged.

## Task 7: Docs, verification, review

- [x] `modules/tutorial/AGENTS.md`, `modules/camping` notes.
- [x] `go test -race ./...`, `make generate`, `make validate`.
- [x] Independent review; verify findings; record in
  `docs/PROJECT_STATUS.md` with a **Review:** line.
