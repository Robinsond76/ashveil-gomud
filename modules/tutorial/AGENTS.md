# Tutorial Module Guide

Phase 27a. The Ashveil tutorial: a short course of rooms, one per stage, that a
new character walks through in their own ephemeral copies. See the
[27a design](../../docs/superpowers/specs/2026-09-25-phase-27a-tutorial-framework-design.md).

- **Stages** (`stages.go`) are data, in order: Character, Company,
  Formation, Survival, Camp (27b), Combat (27c), Alignment (27d),
  Departure. Each names its room by index into
  `SpecialRooms.TutorialRooms` (900–907). New rooms are appended to that
  list, so existing indexes never shift: Survival is 904 (index 4), Camp
  905 (index 5), Combat 906 (index 6), Alignment 907 (index 7), and the
  Gate stays 903 (index 3). New stages go before Departure; keep Departure
  last and one room per stage (`TestShippedTutorialRooms`, which checks
  exits in stage order).
- **Gates check results, never typed text.** A stage's `Inspections` are
  counted through `usercommands.OnCommandDone` (aliases resolved), only in
  that stage, and only for registered commands (`usercommands.IsRegistered`),
  so a missing module never traps anyone. Company and Formation read the
  company seams; Survival also needs a real meal and drink
  (`survival.OnProvision`, any member fed); Camp needs a rest tier held
  (`camping.RestTierOf`), which only a completed rest grants. Gates run on
  `companyview.OnRefresh`, so after every command and every round, on the
  game loop.
- **The practice fight (27c):** Combat (room 906) raises a squad
  (`PracticeSquad`: three straw footmen, 67, and a straw archer, 68) in the
  player's copy when they walk in or are placed there. Practice mobs
  (`practice: true`, the harmless `dummy` race) are beaten without rewards
  and fire `mobcommands.OnPracticeBeaten`; the gate is every foe of the
  player's squad beaten. Squads are in memory (`fights`): resume raises a
  fresh one and removes the old; leave, skip, and passing remove what's
  left. The leader must be placed in the grid for formation rules to cover
  their blows.
- **Inspections can name a subcommand** (27d): "company alignment" counts
  `company` with `alignment` as the first word of the rest
  (`inspectionMatches`); registration is checked by the command.
- **The view** (27d): `TutorialView` implements `internal/tutorial.Viewer`
  with the same checklist the terminal shows (`checklist`), in plain text;
  the gmcp module sends it as the `Tutorial` package. Fire
  `domain.Changed` whenever a player's place in the course changes, so the
  panel resends at once. A course whose `TutorialRooms` lacks a stage's
  room, listed or loadable, is closed (`available`, `closedCourse`): no gates
  run, the player's place is kept for when it's fixed, and `tutorial skip
  yes` still lets them go.
- **Death in the course** is an ordinary death that ends the course as a
  skip (27c decision): nothing in the course can kill quickly.
- **Supplies (27b):** reaching Survival (walking in, or being placed there)
  gives a ration and water once per character (`tutorial-supplied`), and
  only what the pack lacks (`RationItemId`, `WaterItemId`).
- **Course camps are struck** (`camping.AbandonCamp`) when Camp passes, on
  skip, leave, and logout, and on every placement: a camp in a room copy
  can't outlive the copy (whose ID a later copy may reuse), and a stale one
  would block the player's next camp. A failed strike sets
  `tutorial-strike` and is retried on every refresh until it succeeds.
- **Progress** is `MiscData` on the character (`tutorial-state`,
  `tutorial-stage`, `tutorial-seen`, `tutorial-supplied`, `tutorial-strike`),
  saved with the user file. The room copies (`copies`) are in memory only:
  `PlayerSpawn` queues a resume that makes fresh copies and opens the way up
  to the saved stage.
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
- `wiring_test.go` calls `plugins.Load` with the company, survival,
  camping, weather, exposure, walking, and encumbrance modules; it restores
  plugin state with `plugins.SnapshotLoadStateForTest`. A camp rest takes a
  real minute, so the test starts a real one and then gives the Rested buff
  it would grant.
