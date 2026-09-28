# Phase 32b: Tutorial Replay — Plan

Design: [32b design](../specs/2026-09-28-phase-32b-tutorial-replay-design.md).
The owner's decisions are recorded there (2026-09-28); nothing is left open.

## Implementation decisions (lead, 2026-09-28)

- **The hand-off** is two queued events on the game loop: the leaving
  user's `PlayerDespawn` with `HandOff` set (every module's logout path
  runs; the engine's final leave handler logs the user out but keeps the
  connection and says no goodbye), then `UserHandOff{ConnectionId,
  FromUserId, ToUserId}`, whose `internal/hooks` handler loads the next
  user from their file, logs them in on the same connection, raises
  `PlayerSpawn`, and places them, as `enterWorld` does.
- **Connection loops** (telnet, SSH, WebSocket, copyover resume) re-read
  the connection's user after every read, so input, prompts, and a
  dropped link follow the switch.
- **Replay ids** come from a reserved range (`users.ReplayUserIdBase`,
  900,000,000 up), never from the user index, so a new account can't be
  handed a throwaway's id. The username is `replay:<id>`, which no
  account name can be.
- **`quit` or a lost connection** ends a replay the ordinary way: the
  throwaway leaves and is purged, and the real character, already saved
  and out of the world, is exactly as it was at the next login. The
  connection isn't handed back (there is none, or the player chose to
  go).
- **The purge** is `events.UserPurged{UserId}`; its final listener in
  `internal/hooks` removes the user file (`users.RemoveUserFile`, which
  refuses an online user and never touches the indexes of an unindexed
  user).
- **Boot sweep:** the tutorial module's load purges every replay user
  file whose user isn't online (copyover restores the online ones before
  plugins load).
- **Coverage:** a test reads every module's source: one that saves state
  (`SetOnSave`) must listen for `UserPurged`, unless it's on a short,
  reasoned list of world-only modules (market, weather, gmcp's Mudlet
  config).

## Task 1: The hand-off spike (`internal/events`, `internal/hooks`, `internal/users`, `main.go`)

- [ ] Tests first (`internal/hooks/userhandoff_test.go`): a user on a
  connection leaves with `HandOff`: logged out, saved, connection kept,
  no goodbye; `UserHandOff` logs the next user in on that connection and
  raises `PlayerSpawn`; a vanished connection or a next user already
  online is refused cleanly (the connection is closed with a goodbye).
- [ ] `PlayerDespawn.HandOff`, `UserHandOff`, `UserPurged` events.
- [ ] `HandleLeave` honours `HandOff`; `HandleUserHandOff`; `HandlePurge`.
- [ ] `users.LoadUserFile` (by id, no index), `users.RemoveUserFile`.
- [ ] `main.go`: every connection loop re-reads its user after a read.

## Task 2: Replay users (`internal/users`)

- [ ] Tests first (`internal/users/replay_test.go`): `NewReplayUser`
  copies name and race at level 1 with nothing, takes a reserved id and
  username, isn't in the user or character index; the `replayof` flag
  round-trips through the file; `OfflineReplayUserIds` lists replay files
  but not online replays or ordinary users; `RemoveUserFile` refuses an
  online user and is idempotent.
- [ ] `UserRecord.ReplayOf`, `NewReplayUser`, `ReplayUserIdBase`,
  `OfflineReplayUserIds`, `OnlineReplayOf`.

## Task 3: The purge in every module

- [ ] Tests first, one per module: state for a user is dropped and saved
  on `UserPurged`, other users' state kept: archetype, camping, company,
  encumbrance, expedition, exposure, mount, survival, walking, death, gmcp
  (Company, Tutorial, Mudlet caches), tutorial.
- [ ] The listeners, each under its own lock, calling no world function
  while holding it.
- [ ] `modules/purge_coverage_test.go`: every module that saves state
  handles `UserPurged` or is listed as world-only with a reason.

## Task 4: `tutorial replay` (`modules/tutorial`)

- [ ] Tests first (unit): refused mid-fight, inside a replay, with no
  connection, and with the course closed; asks for `yes`; the end paths
  (graduate, skip, death, closed course) call the hand-back for a replay
  and never for a real character; `online` notes a replay.
- [ ] `tutorial replay [yes]`: build the throwaway (archetype remembered
  in its `MiscData`), save it, queue the hand-off.
- [ ] On the throwaway's first spawn: choose the archetype at creation
  (its kit) and `Begin` the course.
- [ ] The hand-back: "You set the practice character aside.", queue the
  hand-off to the real user and the purge.
- [ ] The throwaway leaving without a hand-off (quit, link-dead expiry)
  queues its purge.
- [ ] The real character logging in while its replay is online ends and
  purges the replay.
- [ ] The boot sweep.
- [ ] `online` shows "(replaying the tutorial)".

## Task 5: Wiring through the real modules (`modules/tutorial/wiring_replay_test.go`)

- [ ] A real character with a company, a camp, gear, gold, and XP types
  `tutorial replay yes` on a connection: the real one leaves the world,
  a level-1 character with nothing is in the first room, same name.
- [ ] The replay recruits Tamsin and Oswin and camps; the real company
  and camp are untouched.
- [ ] Graduating, `tutorial skip yes`, and dying each hand back: the real
  character in the same room with the same company, camp, inventory,
  gold, and XP; no graduation cap.
- [ ] `quit` (the logoff path): the throwaway is purged and the real
  character's next login is unchanged.
- [ ] After every exit: no module holds state for the throwaway id and
  its file is gone.
- [ ] A restart mid-replay: the boot sweep purges it; the next login is
  the real character.
- [ ] Refused mid-fight and inside a replay; two replays in a row work.
- [ ] The clock never moves.

## Task 6: Player help and tutorial

- [ ] `help tutorial` covers `tutorial replay` (what it does, that
  nothing is kept, how it ends).
- [ ] The first lesson mentions the course can be replayed later.
- [ ] Tests: the page renders through `help`; `TestTutorialHelpPointersExist`.

## Task 7: Docs, verification, review

- [ ] `go test -race ./...`, `make generate`, `make validate`.
- [ ] Independent review; verify and fix findings with regression tests.
- [ ] `docs/PROJECT_STATUS.md`: the 32b row and a work-log entry with
  **Review:**; `modules/tutorial/AGENTS.md` updated.
