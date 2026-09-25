# Phase 27c: Tutorial Practice Fight — Plan

Design: [27c spec](../specs/2026-09-25-phase-27c-tutorial-practice-fight-design.md).

## Task 1: Practice mobs in the engine

- [x] Tests first (`internal/mobcommands`): a practice mob's `Suicide`
  grants no XP or gold, drops nothing, fires no `MobDeath`, fires
  `OnPracticeBeaten`, and removes the mob; an ordinary mob still gets its
  rewards.
- [x] `mobs.Mob.Practice` (`yaml:"practice"`); `mobcommands.PracticeBeaten`,
  `OnPracticeBeaten`; the practice branch in `Suicide`.

## Task 2: The Combat stage

- [x] Tests first (`stages_test.go`, `tutorial_test.go`): order and rooms
  (Combat at index 6 before Departure); the squad spawns once on entering
  or being placed; beaten foes pass the gate, and another player's don't
  count; resume replaces the squad; leave and skip remove it; the waiver;
  the view.
- [x] `StageCombat`; seams `spawnFoe`, `removeFoe`; `foes` in memory;
  `onPracticeBeaten`; config `PracticeSquad`.

## Task 3: Content

- [x] Tests first (`shipped_test.go`): seven rooms; the Practice Yard;
  mobs 67 and 68 practice, `dummy` race, `practice-squad`; help mentions
  Combat.
- [x] Room 906; 903's back exit to 906; map; `TutorialRooms`; mobs 67
  and 68; help template; config overlay.

## Task 4: Wiring test

- [x] Through `plugins.Load` and the real combat round: the party
  formation, interception at the archer, no XP or gold, re-targeting, the
  gate; the clock unchanged.

## Task 5: Docs, verification, review

- [x] `modules/tutorial/AGENTS.md`; `internal/mobcommands` note.
- [x] `go test -race ./...`, `make generate`, `make validate`.
- [x] Independent review; verify findings; record in
  `docs/PROJECT_STATUS.md` with a **Review:** line.
