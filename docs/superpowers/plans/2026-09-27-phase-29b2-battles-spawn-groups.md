# Phase 29b2: One Battle at a Time, and Spawn Groups — Plan

Design: [29b2 design](../specs/2026-09-27-phase-29b2-battles-spawn-groups-design.md).

## Task 1: The battle registry (`internal/battle`)

- [x] Tests first (`internal/battle/battle_test.go`): `Begin`/`Current`/
  `End`; first-set order kept per player and group, ties by room order;
  `Next` picks the earliest group set on a player; a group matched by a
  shared enemy; `Reset` for tests.
- [x] Implement `battle.go`.

## Task 2: Spawn groups (`internal/mobs`, `internal/mobparty`, `internal/rooms`)

- [x] Tests first:
  - `mobparty`: a mob with a spawn group is grouped by it; even chunks (6
    gives 3+3, 7 gives 4+3);
  - `rooms` (pure planner): hostile, non-solitary spawns form one group;
    a group of one is topped up from the list in order; more than five
    split evenly; a solitary or non-hostile mob is left alone; a
    respawned mob joins a group with room that isn't fighting.
- [x] `mobs.Mob.SpawnGroup` (runtime) and `Solitary` (`solitary:`).
- [x] `Room.Prepare` forms groups and tops up after its spawn pass;
  grouped mobs don't wander.

## Task 3: Battles in the round (`internal/hooks`, `internal/usercommands`)

- [x] Tests first (pure): who is set on a player; the hold decision.
- [x] The battle pass at the top of `DoCombat`: end battles that are
  over, begin the next, turn a waiting group onto a free player.
- [x] The hold check before a mob strikes a player or a companion.
- [x] 29a's upkeep limited to the battle's group; in-turn reassignment
  from the battle's group.
- [x] The stream: fights opened and ended from battles, solo players
  included; 29b's merge-by-room removed.
- [x] `attack` refuses a waiting group.

## Task 4: Wiring through the real round

- [x] `modules/company/wiring_battles_test.go`: three hostile groups one
  at a time (a fight and summary each; no blow from a waiting group; the
  next begins the round after); a second player takes the next group;
  `attack` on a waiting group refused; the clock unchanged.
- [x] Spawning through `Room.Prepare` with real room and mob files: one
  hostile entry gives two; two kinds give one mixed group; `solitary`
  alone.
- [x] 27c, 29a, and 29b tests pass (29b's one-fight-per-room and solo
  tests updated to battles).

## Task 5: Player help and tutorial

- [x] `help combat`, `help targeting`, `help battle-summary` updated;
  `solitary` noted for builders only if a builder help page exists.
- [x] The tutorial's Combat lesson: a group fights as one, others wait.
- [x] `TestCombatHelpTopics` and `TestTutorialHelpPointersExist` pass.

## Task 6: Docs, verification, review

- [x] `go test -race ./...`, `make generate`, `make validate`.
- [x] Independent review; verify and fix findings with regression tests.
- [x] `docs/PROJECT_STATUS.md`: the 29b2 row and a work-log entry with
  **Review:**.
