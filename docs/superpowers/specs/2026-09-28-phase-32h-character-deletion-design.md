# Phase 32h: Character Deletion — Design

From the owner's play-test notes of 2026-09-28
([roadmap](2026-09-28-playtest-feedback-roadmap.md)).

## The owner's notes

> "How do I delete a character?"

The roadmap's scope for 32h: **confirmed by password, purging every
module's state**, reusing 32b's per-user purge.

## Prior-art check

- **One login, one character.** A GoMud user record
  (`internal/users/userrecord.go`) is the account (username, password,
  role, settings, macros) and holds exactly one `Character`. The username
  is the login; the character's name is separate. There is no
  alt-character module in this fork (`users.go` and `index.go` skip
  `.alts.yaml` files for one, but nothing writes them).
- **A new account** is a user record whose character is `characters.New()`
  in room -1. The Void's script (`rooms/nowhere/-1.js`) runs `start`
  (`internal/usercommands/start.go`): race, name, archetype, starter kit,
  then "skip the tutorial?".
- **`users.DeleteUser`** removes the user file, the username index entry,
  and the character-name index entry, and refuses an online user. Nothing
  calls it. It leaves every module's state behind.
- **32b's purge** (`events.UserPurged{UserId}`): every module that keys
  state by user id drops it in its own listener (company, survival,
  expedition, camping, encumbrance, mount, walking, exposure, archetype,
  death, tutorial, GMCP), and the last listener
  (`internal/hooks/UserPurged_HandlePurge.go`) removes the user file with
  `users.RemoveUserFile`, which refuses an online user and doesn't touch
  the indexes (a replay was never indexed). The purge is idempotent.
  `modules/purge_coverage_test.go` fails when a module that saves state
  doesn't listen for it. Standing is computed from the company's
  alignment, so it has no state of its own.
- **32b's hand-off:** `PlayerDespawn{HandOff: true}` takes a character out
  of the world as `quit` does but keeps the connection;
  `events.UserHandOff{ConnectionId, FromUserId, ToUserId}` then logs a
  user in on it and places them in their saved room.
- **GoMud's permadeath** (off in Ashveil; `suicide.go`) already does an
  in-place reset: drop everything, move to room -1, `user.Character =
  characters.New()`. The account stays and the player creates again.
- **Password checks:** `UserRecord.PasswordMatches` (used by `password`).
  In-game prompts (`user.StartPrompt(...).Ask`) are **not masked**: the
  `password` command echoes what's typed. Only the login prompts mask
  (`MaskInput` in `internal/inputhandlers/login_prompt_handler.go`).
- **`quit`** refuses while the player is in a fight (`Aggro`).
- **Admin tools:** neither the web admin nor any admin command deletes
  users.

## Decisions

Open questions are marked **(open)** with a recommendation; everything
else is a default this design picks.

### A. What is deleted **(open)**

- **1. The character, keeping the login (recommended).** The character
  and everything about it goes; the account (username, password, role,
  settings, macros) stays. The player is put straight into character
  creation on the same connection, as a new account would be. This is
  GoMud's permadeath reset, done properly: every module's state goes too.
  It suits play-testing: no new login, and the old character's name is
  free to reuse at once.
- **2. The whole account.** The user record goes too; the player is
  disconnected and must register again to play. Frees the username as
  well as the character name.

The rest of this design assumes option 1. Option 2 changes only D's last
step (remove the file and both index entries; say goodbye and close the
connection instead of a hand-off).

### B. The command

- **`delete character`**, typed anywhere. It says what will be lost and
  asks for the password:

  ```
  This deletes Dain for good: their level, gear, gold, company, camp,
  and everything else. Your login stays, and you'll make a new
  character. This can't be undone.
  Type your password to delete Dain, or anything else to cancel:
  ```

- **The password is the confirmation.** A right password deletes at once;
  anything else cancels ("Nothing was deleted."). No second "type the
  name" step. **(open; recommendation: password only)**
- **The password is masked** on the prompt, as at login. In-game prompts
  get a `Masked` question option that does the same `WILL ECHO` masking
  as the login handler and keeps the answer out of input history; the
  web client gets the same password field the login uses. `password`
  starts using it too (a free fix: it echoes today).
- **Refused, with a reason, when:**
  - in a fight (as `quit`);
  - down and waiting to be returned (Ashveil death, 25a);
  - in a tutorial replay ("That's a practice character. Leave the course
    first."), since the practice character isn't the real one;
  - the character is still being created (room -1): nothing to delete.
- **Allowed** while travelling, camped, at an inn, mounted, or with
  companions out: each module already handles its player logging out in
  those states, and the purge then drops the state.
- **Wrong passwords** are logged (user id, no password). Three wrong in a
  row lock the command until the next login. **(open; recommendation: yes)**

### C. What the player sees

- On a right password: "Dain is gone." Anyone in the room sees Dain leave
  as on `quit` (no special "deleted" line).
- Then the Void and character creation, exactly as for a new account.
- `help delete` explains it.

### D. How it runs (the purge with the account kept)

On the game loop, in order:

1. **Leave the world** with `PlayerDespawn{HandOff: true}`: companions
   despawn, travel and camps are left as on logout, and everything saves.
2. **Mark and reset the record:** the user file is saved with a durable
   `Deleting` flag and the old character's name, and the character
   replaced with `characters.New()` in room -1. The old name is removed
   from the character index.
3. **Purge:** `UserPurged{UserId, KeepAccount: true}`. Every module drops
   the user's state as in 32b. The final listener, seeing `KeepAccount`,
   clears the `Deleting` flag and saves instead of removing the file.
4. **Back in:** `UserHandOff{From: id, To: id}` logs the same user in on
   the same connection; they land in the Void and creation begins.

Items, gold, and the bank balance go with the character: nothing is
dropped in the room. **(open; recommendation: destroyed, not dropped, so
deletion can't be used to hand gear to another character.)**

Kept, as account data: username, password, role, config options (prompt,
colours, screen settings), macros. Everything on the character and in
every module goes.

### E. Restart, copyover, and crashes

- **A crash mid-deletion** leaves the `Deleting` flag on the record. At
  boot, a sweep finds flagged records and re-runs steps 2–3 (the purge is
  idempotent; the reset is too). The next login goes to creation.
- **Copyover** mid-sequence: the flag is on the saved record, so the
  sweep after copyover finishes the purge; a player whose connection
  survived stays in the Void, in creation. (The plan checks whether
  copyover can land between these queued events at all.)
- **The player hangs up** between steps 1 and 4: steps 2–3 still run (the
  user is offline); step 4 finds the connection gone and the user simply
  stays offline, as 32b's hand-off already does.
- **The same login from another client** during the sequence: login
  refuses a user who is already online (`users.LoginUser`), and between
  steps 1 and 4 the record is flagged, so login waits for the sweep
  rather than loading a half-reset record.

## Module

- `internal/usercommands`: `delete` (registered as `delete`, taking
  `character`), refusals, password prompt.
- `internal/users`: the masked prompt question; the `Deleting` flag, the
  reset, the index update, and the boot sweep.
- `internal/events`: `UserPurged.KeepAccount`.
- `internal/hooks`: `HandlePurge` honours `KeepAccount`.
- `internal/inputhandlers` / the web client: masking for in-game
  prompts, reusing the login's.
- No module changes: every module already handles `UserPurged`. A module
  must not treat the purge as "the file is gone"; the coverage test's doc
  says so.

## Invariants

- **The clock:** nothing here advances time.
- **Restart and copyover:** see E. The flag is durable; the reset and the
  purge are idempotent.
- **Locks and ordering:** the whole sequence runs on the game loop as
  queued events, like 32b's hand-off. Modules drop state in their own
  listeners under their own locks.
- **Nothing leaks:** after deletion, no module holds state for the user
  id from before, and nothing of the old character (items, gold, XP,
  company, standing, claims, tutorial progress) reaches the new one.
- **Scope:** only 32h's files; 32a2, 32c and 32f are in flight in other
  branches.

## Acceptance criteria

- **Unit:** the reset (account fields kept, character new, name out of
  the index); `HandlePurge` with `KeepAccount` keeps the file; the boot
  sweep; the masked question; each refusal.
- **Wiring** (shipped config, real commands, a real connection):
  - a character with a company out, a camp, a mount, cargo, gear, and
    gold types `delete character` and the right password: they leave
    the world, land in the Void, and `start` runs; the new character has
    none of the old one's items, gold, XP, company, camp, mount, or
    cargo;
  - after deletion, no module holds state for the user from before (the
    32b purge checks, run on a kept account);
  - the old name can be taken by the new character or anyone else;
  - the same username and password log in afterwards;
  - a wrong password deletes nothing; refusals in a fight, while down,
    in a replay, and in the Void;
  - the password isn't echoed (telnet: `WILL ECHO` around the prompt);
  - a crash between the reset and the purge: the boot sweep finishes it;
  - the world clock doesn't move.
- **Player help:** a `help delete` page (aliases `delete character`,
  `deletion`), listed in `keywords.yaml`, linked from `help password`;
  the Departure lesson mentions it; `TestTutorialHelpPointersExist`
  passes.
- `go test -race ./...`, `make generate`, and `make validate` pass. The
  independent review is recorded.

## Deferred

- **Admin deletion** of another player's character or account.
- **A grace period or undo** (a stored copy restorable for some days).
- **Several characters per login** (GoMud's alt characters).
