# Phase 32b: Tutorial Replay — Design

From the owner's play-test notes of 2026-09-28
([roadmap](2026-09-28-playtest-feedback-roadmap.md)).

## The owner's rules (2026-09-28)

> "I don't want to have to keep creating new characters every time I want
> to test the game. Let's create a way for a character to redo the
> tutorial indefinitely."

Asked how a replay should treat the character (2026-09-28):

> "Create an instance of a level 1 player character with nothing, and flow
> through the tutorial as nothing. Once I leave the tutorial, I go back to
> exactly where I was and I keep nothing."

So:

1. **A fresh character.** Level 1, no gear, gold, company, standing, or
   progress, as a brand-new player would start the course.
2. **Unlimited.** A player can replay as often as they like.
3. **Exactly back.** Leaving the replay, by graduating, skipping, logging
   out, or dying, returns the player to their real character exactly as
   it was: room, company, camp, travel, everything.
4. **Nothing kept.** Nothing from the replay carries over: not items,
   XP, recruits, the graduation reward, or kit claims.

## Prior-art check

- **Per-user state lives in each module's registry**, keyed by user id:
  company (`companies`), survival, expedition, camping, encumbrance,
  mount, walking, exposure, archetype. Tutorial progress is in the
  character's `MiscData`. Standing and alignment are on the character or
  the company record.
- **There is no in-game character switch.** A connection gets its user at
  login (`internal/inputhandlers/login.go` → `users.LoginUser`). Leaving
  the world raises `PlayerDespawn`, and `world.enterWorld` raises
  `PlayerSpawn` and places the character. Every module already handles a
  player logging out mid-travel, mid-camp, or with companions out, and
  handles them logging back in.
- **The course** runs in per-player copies of rooms 900–907
  (`modules/tutorial`). Finishing or skipping it is final today
  (`tutorial.go`: `stateSkipped`, `stateGraduated` send the player to the
  start room). Practice mobs give nothing. The course can't be left on
  foot except through the Gate, which ends it.
- **Creation** (`internal/usercommands/start.go`): race, name,
  archetype (22a), the owed starter kit, then "skip the tutorial?".
- **Deletion:** `users.DeleteUser` removes the user file and index
  entries, refuses an online user, and leaves module state behind.
  Nothing calls it.
- **`quit`** refuses while the player is in a fight (`Aggro`).

## Approach: a throwaway user, not a stashed character

Two ways to meet "exactly back, nothing kept":

- **A. A throwaway user record (chosen).** The replay is a separate,
  temporary user with its own id. The player's real character leaves the
  world as if they had logged out, and the connection is switched to the
  throwaway. On leaving the replay, the throwaway is purged and the
  connection switched back, which is a login of the real character.
  - Isolation comes for free: every module keys state by user id, so the
    replay's company, survival, and camps never touch the real ones.
  - "Exactly back" reuses logout and login, which every system already
    survives, including travel and camps.
- **B. Stash and restore in place.** Snapshot the real character and
  every module's state for the user, wipe them, replay, then restore.
  Every module would need snapshot/restore, a crash mid-replay would need
  a durable stash, and a missed module would leak state both ways.
  Rejected.

## Decisions

### A. Starting a replay

- **`tutorial replay`**, typed anywhere, asks for confirmation ("Your
  character will step out of the world until you leave the course. Type
  `tutorial replay yes`").
- **Refused** when `quit` would be (in a fight), and when the player is
  already in a replay.
- **Anyone can use it**, not only admins: nothing is kept, so nothing can
  be farmed. **(recommendation applied)**
- **The real character leaves the world** exactly as on `quit`: the same
  despawn, saves, and companion handling. Anyone in the room sees them
  leave.
- **The throwaway** is created with a new user id, flagged `Replay` with
  the real user's id (durable, on the user record). It isn't in the user
  or character name indexes and can't be logged into.
- **Creation (owner, 2026-09-28): copy the real character and go
  straight in.** The throwaway takes the real character's name, race, and
  archetype, at level 1 with nothing, and no creation step runs.
- It then enters the course at stage 1 exactly as a new player would.

### B. During a replay

- It's an ordinary new character: starter kit, the free recruits, the
  practice fight, and camps, in its own course copy.
- **`who` and tells:** the replay shows the player's name with a
  "(replaying the tutorial)" note, and tells to the name reach them.
- **Chat channels** work as usual.

### C. Leaving a replay

Any end of the course ends the replay:

- **graduating** (walking out of the Gate): the graduation reward is
  given to the throwaway and then discarded with it, so it's never kept;
- **`tutorial skip yes`**;
- **dying** in the course;
- **logging out or losing the connection** (after the usual link-dead
  grace);
- **the course closing** (rooms missing).

On leaving, the player is told "You set the practice character aside."
The throwaway is purged (D), and the real character is logged back in on
the same connection, in the room they left, with their company, camp,
and travel as they were.

### D. The purge (shared with 32h)

- A new event, **`UserPurged{UserId}`**. Every module that keys state by
  user id drops that user's entry and saves: company, survival,
  expedition, camping, encumbrance, mount, walking, exposure, archetype,
  tutorial runtime, and the GMCP caches.
- Companion mobs, camps, and course copies are despawned or struck
  first, through each module's existing logout/abandon paths.
- Then the user file is removed (`users.DeleteUser`).
- A test enumerates every registered module that persists per-user state
  and asserts it handles `UserPurged`, so a future module can't forget.

### E. Restart, copyover, and crashes

- **Copyover** mid-replay: connections survive and the throwaway is
  restored as the online user. The replay carries on.
- **A restart or crash** mid-replay drops the connection. At boot, every
  user record flagged `Replay` is purged (D). The real character was
  logged out cleanly when the replay started, so the next login is
  exactly back.
- **Logging in to the real account while its replay is still online**
  (another client): the replay ends and is purged first.

## Module

- `modules/tutorial`: `tutorial replay`, the end-of-course hand-back.
- `internal/users`: the `Replay` flag, creating an unindexed user, and
  the boot sweep.
- A connection-switch helper (world plus `internal/users`): log one user
  out of a connection and another in, raising the despawn and spawn
  events in order. **This is the riskiest part.** Plan task 1 is a spike
  that proves the switch on a real connection before the rest is built.
- `internal/events`: `UserPurged`. Each per-user module listens for it.
- `internal/users` / `internal/characters`: a level-1 character built
  from the real one's name, race, and archetype (no creation prompts).

## Invariants

- **The clock:** nothing here advances time. The real character's
  offline time runs like any logout, including the online-time-only
  rescue allowance (25b).
- **Restart and copyover:** see E. The `Replay` flag is durable; the
  purge is idempotent, so running it twice (a crash mid-purge) is safe.
- **Locks and ordering:** the switch and the purge run on the game loop.
  Modules drop state in their own event handlers, under their own locks,
  never holding one across a world call.
- **Nothing leaks:** no item, gold, XP, claim, or record reaches the
  real character, and nothing of the real character is visible to the
  replay.

## Acceptance criteria

- **Unit:** the `Replay` flag and boot sweep; each module's
  `UserPurged` handler; the purge coverage test.
- **Wiring** (shipped config, real commands, a real connection):
  - a real character with a company, a camp, and gear types
    `tutorial replay yes`: they leave the world, and a level-1
    character with nothing is in the course's first room;
  - the replay recruits Tamsin and Oswin and pitches a camp; the real
    company and camp are untouched;
  - graduating, `tutorial skip yes`, dying in the course, and logging
    out each return the player to the real character, in the same room,
    with the same company, camp, inventory, gold, and XP;
  - after any exit, no module holds state for the throwaway id and its
    user file is gone;
  - a restart mid-replay: the boot sweep purges the throwaway, and the
    next login is the real character, unchanged;
  - `tutorial replay` refused mid-fight and inside a replay;
  - a replay twice in a row works (no leftover claims or progress).
- **Player help:** `help tutorial` covers `tutorial replay`; the course's
  first lesson mentions that the course can be replayed later;
  `TestTutorialHelpPointersExist` passes.
- `go test -race ./...`, `make generate`, and `make validate` pass. The
  independent review is recorded.

## Deferred

- Replaying a **single lesson** (jump to a stage).
- **Character deletion** (32h) reuses the purge.
- An admin-only setting to disable replays (not needed while nothing is
  kept).
