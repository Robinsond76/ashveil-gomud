# Phase 32h: Character Deletion — Plan

Design: [32h design](../specs/2026-09-28-phase-32h-character-deletion-design.md).
The owner's answers are recorded there (2026-09-28); nothing is left open.

## Implementation decisions (lead, 2026-09-28)

- **The sequence** is 32b's hand-off with the same user on both ends.
  `delete character` sets the durable `Deleting` flag on the online
  record, saves it, and queues `PlayerDespawn{HandOff: true}`. Every
  module's logout path runs. `HandleLeave` (the engine's final leave
  listener), seeing a hand-off of a user flagged `Deleting`, queues
  `UserPurged{UserId, KeepAccount: true}` and then
  `UserHandOff{conn, id, id}`. Queuing them from the final leave
  listener puts them behind anything the other despawn listeners queued.
- **The reset** is `HandlePurge` with `KeepAccount`: after every module
  has dropped the user's state, `users.ResetDeletedCharacter` loads the
  file, removes the old name from the character index, replaces the
  character with a new one (room -1, the Void), clears the flag, and
  saves. The hand-off then logs the user in, in the Void, where
  creation runs.
- **A flagged record can't log in** (`users.LoginUser` refuses it with
  "That character is being deleted. Try again in a moment.").
- **Boot sweep** (`hooks.SweepDeletions`, called in `main.go` after
  `plugins.Load`): an offline flagged user gets the purge (the reset
  follows in `HandlePurge`); an online one (copyover kept it) gets the
  whole sequence again.
- **Masking** is per connection (`connections.SetInputMasked`). A prompt
  question can be marked `Masked`; after each input, `world.processInput`
  masks the connection exactly while the user's next question is a
  masked one. The echo handler sends `*` per printable byte instead of
  the input, the history handler skips it, and the switch sends
  `TEXTMASK:true/false` to the web client and `WILL/WONT ECHO` to
  Mudlet, as the login prompt does.
- **The lock** is a count of wrong passwords in the user's temp data,
  which a new login starts empty.

## Task 1: Masked prompt questions (`internal/prompt`, `internal/connections`, `internal/inputhandlers`, `world.go`, `internal/usercommands/password.go`)

- [x] Tests first: `Question.Masked` round trip; `SetInputMasked` sends
  `TEXTMASK` to a websocket and `WILL/WONT ECHO` to Mudlet, nothing
  twice; the echo handler stars masked input and passes it through
  otherwise; the history handler skips masked input;
  `SyncInputMask` masks while the next question is masked and unmasks
  when answered or cleared.
- [x] `prompt.Question.Masked`, `connections.SetInputMasked` /
  `InputMasked`, echo and history handlers, `users.SyncInputMask`
  called at the end of `world.processInput`.
- [x] `password` marks its questions masked.

## Task 2: The deletion record (`internal/users`)

- [x] Tests first (`internal/users/deletion_test.go`): the `deleting`
  flag round-trips; `ResetDeletedCharacter` keeps username, password,
  role, settings, macros, aliases, and tips, gives a new character in
  room -1, frees the old name in the character index, clears the flag,
  refuses an online user, and is idempotent;
  `DeletingUserIds` lists flagged records (the sweep splits online
  from offline);
  `LoginUser` refuses a flagged record.
- [x] `UserRecord.Deleting`, `ResetDeletedCharacter`,
  `DeletingUserIds`, the login refusal.

## Task 3: The sequence (`internal/events`, `internal/hooks`, `main.go`)

- [x] Tests first (`internal/hooks/deletion_test.go`): a flagged user's
  hand-off leave queues the purge and a hand-off to themselves; an
  unflagged one doesn't; `HandlePurge` with `KeepAccount` resets rather
  than removes; the sweep queues a purge for an offline flagged user and
  the whole sequence for an online one.
- [x] `UserPurged.KeepAccount`; `HandleLeave`; `HandlePurge`;
  `SweepDeletions`; the `main.go` call.

## Task 4: `delete character` (`internal/usercommands`)

- [x] Tests first (`internal/usercommands/delete_test.go`): refused in a
  fight, while down, in a replay, in the Void, and for anything but
  `delete character`; a wrong password cancels and counts; the third
  locks until the next login; a right password and wrong name cancels;
  both right flag the record, save it, and queue the hand-off leave; the
  password question is masked.
- [x] `Delete`, registered as `delete`.

## Task 5: Player help and tutorial

- [x] Tests first: `help delete` (and its aliases) renders the command
  and what's kept; `help password` links it;
  `TestTutorialHelpPointersExist` passes with the Departure lesson's new
  pointer.
- [x] `help/delete.template`, `keywords.yaml` (topic and aliases), a line
  in `help password`, the Departure lesson's hint.

Help aliases: `deletion`, `delete-character`, `reroll` (an alias can't
hold a space, so not `delete character`).

## Task 6: Wiring (`modules/tutorial/wiring_delete_test.go`)

- [x] Through `plugins.Load`, a real piped connection, the real hooks,
  and the real `delete` command: a character with a company out, a camp,
  cargo, gear and gold deletes themselves (the mount is covered by
  `modules/mount`'s purge test; 32f replaces the mount model); they land in the
  Void on the same connection with `start` running, and no module holds
  their old state; the file keeps the login; the old name is free; a
  wrong password or name deletes nothing; a restart between the flag and
  the purge is finished by the sweep; the clock doesn't move.

## Task 7: Verify, review, record

- [ ] `go test -race ./...`, `make generate`, `make validate`.
- [ ] Independent review over the phase diff; verify each finding.
- [ ] `docs/PROJECT_STATUS.md` entry with **Review:**.
