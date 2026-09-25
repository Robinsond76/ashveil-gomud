# Tutorial Module Guide

Phase 27a. The Ashveil tutorial: a short course of rooms, one per stage, that a
new character walks through in their own ephemeral copies. See the
[27a design](../../docs/superpowers/specs/2026-09-25-phase-27a-tutorial-framework-design.md).

- **Stages** (`stages.go`) are data, in order: Character, Company, Formation,
  Departure. Each names its room by index into `SpecialRooms.TutorialRooms`
  (900–903). Later phases (27b, 27c) insert theirs before Departure; keep
  Departure last and keep one room per stage (`TestShippedTutorialRooms`).
- **Gates check results, never typed text.** Character counts the four
  inspections through `usercommands.OnCommandDone` (aliases resolved); the
  others read the company seams (`company.CompanyMembers`,
  `company.FormationFor`). Gates run on `companyview.OnRefresh`, so after
  every command and every round, on the game loop.
- **Progress** is `MiscData` on the character (`tutorial-state`,
  `tutorial-stage`, `tutorial-seen`), saved with the user file. The room
  copies (`copies`) are in memory only: `PlayerSpawn` queues a resume that
  makes fresh copies and opens the way up to the saved stage.
- **Ways on** are temporary exits (`east`, and `gate` to the start room)
  opened by the module. The room files have no forward exits and no
  scripts; don't add them back, and never block commands in the course.
- **Moves** go through `travel`, which also relocates the company
  (`company.RelocateCompany`) and shows the room: companions only follow on
  foot. Leaving the course relocates them too.
- **Rewards:** the graduation item (`GraduationItemId`) is given only on
  the transition to `graduated`. Skipping gives nothing. Recruits and kits
  stay with the company claims and the archetype claim marker; the
  tutorial grants neither. `Begin` on an active course resumes it, and on a
  finished one sends the player to the start room, so the course never
  restarts.
- **Hand-off:** `internal/tutorial.Begin` is called by `start` when the
  player keeps the tutorial. Without this module, `start` falls back to the
  engine's ephemeral path.
- `wiring_test.go` calls `plugins.Load` with the company module; it
  restores plugin state with `plugins.SnapshotLoadStateForTest`.
