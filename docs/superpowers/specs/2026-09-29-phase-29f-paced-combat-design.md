# Phase 29f: Paced Combat Output — Design

Builds the [owner-approved direction](2026-09-26-combat-pacing-design.md),
including its settled cadence decision (combat resolves every second game
round, giving an 8-second combat round). Part of the
[combat presentation roadmap](2026-09-26-combat-presentation-roadmap.md);
29b–29e have finished the text this phase paces.

Status: **decisions confirmed by the owner (2026-09-29): all six
recommendations, the combat-balance shifts accepted for now, and the
prompt's health held with the lines (D7).**

## Prior-art check (2026-09-29, against `135b72e`)

- **Cadence.** `world.go`'s `MainWorker` queues `events.NewTurn` every
  `TurnMs` (50 ms) and `events.NewRound` every `TurnsPerRound()` turns
  (`RoundSeconds: 4`). `internal/hooks/hooks.go` registers `DoCombat` as a
  `NewRound` listener. Company, weather, camping, walking, market,
  exposure, follow, and the GMCP feeds all listen to `NewRound` too and
  must keep the 4-second beat.
- **Everything combat resolves is inside `DoCombat`.** Battle choice
  (29b2), engagement upkeep (29a), strategies (32d), player and mob
  attacks, spell casting and `Aggro.RoundsWaiting` countdowns, flee
  attempts, idle-battle closing, and settling all run there. Those
  counts become combat rounds automatically when `DoCombat` fires less
  often. Buffs, bleeding out, survival drain, alignment drift, and
  hostility timers run elsewhere and stay on game rounds, as the cadence
  decision requires.
- **Tests call `hooks.DoCombat` directly** (for example
  `modules/company/wiring_combat_test.go`), with explicit round numbers.
  Gating the cadence where the listener is *registered*, not inside
  `DoCombat`, leaves those tests and their round numbers valid.
- **Delivery.** Combat text goes out through about 60 `user.SendText` /
  `room.SendText` calls in `internal/hooks` (`NewRound_DoCombat.go`,
  `combat_battle.go`, `combat_formation.go`, `combat_engagement.go`,
  `combat_narration.go`). They queue `events.Message`, which
  `hooks.Message_SendMessage` writes to each connection. Room messages
  reach every player in the room, not only the fighters. Follow-on text
  arrives through later events: a companion's death notice through
  `MobDeath` → `modules/company`, the battle summary at fight end, and
  experience and level lines.
- **Single-threaded loop.** `DoCombat` and every listener run inside
  `events.ProcessEvents` under `util.LockMud`, and one drain processes
  everything queued during it. So all of a round's lines are known before
  the drain returns. Few goroutines other than the loop queue events: the
  input worker queues `Input`, and a few system/web paths queue
  `Broadcast` or `WebClientCommand`, never a `Message`.
- **Player settings.** `UserRecord.ConfigOptions` is saved with the user,
  and `set` already lists and toggles per-player options (29b added
  `battlesummary`). `UserRecord.ScreenReader` is saved too.
- **Combat stream (29b).** `internal/combatstream` carries structured
  events, not text, so pacing works on text messages. The stream and the
  battle summary are unaffected.
- **Live battle view (32g2).** `modules/gmcp/gmcp.Company.go`'s feed
  refreshes every round and after every command, and sends only when the
  JSON changes. Its known issue, "the view can lead the round's narration
  by a few lines", was deferred to this phase.
- **Help that goes stale.** `help combat` says "Combat resolves once a
  round (a round is 4 seconds)". `help narration`, `help set`, and
  `help battle-summary` describe combat text delivery. `help market` is
  about game rounds and stays as it is.

## Scope

1. **Combat cadence.** A new `Timing.CombatEveryRounds` setting (default
   `2`, minimum `1`) in `_datafiles/config.yaml`. `DoCombat` runs only on
   game rounds whose number is a multiple of it. `1` restores today's
   cadence. The world clock, round count, and every other `NewRound`
   listener are untouched.
2. **A per-player paced queue** (`internal/combatpace`). Text caused by a
   combat round is held per recipient and released line by line on
   `NewTurn`, over the pace window the player chose.
3. **`set combatpace fast|normal|slow|off`.** It is saved in
   `ConfigOptions`. The default is `normal`, and `off` for a player with
   `ScreenReader` on. `set` with no arguments lists it.
4. **Spacing:**
   - a longer gap before the pain line or death line that follows a
     critical hit;
   - short gaps for indented follow-up lines (area spells, multi-heals);
   - causal order kept, so a companion's death notice follows its death
     line and the battle summary follows the closing line.
5. **Catching up.**
   - When a round has too many lines, the gaps shrink to fit the window.
   - When the next combat round starts, anything still held is flushed at
     once, before that round's first line.
   - Output never runs past the next combat round.
6. **Non-combat text is never delayed:** says, tells, command output, and
   prompts pass straight through, and input is never blocked.
7. **The live battle view (32g2)** holds its update while that player has
   held lines, then sends it when they drain (decision D4).
8. **Player help and tutorial** (below).

Out of scope:
- the Phase 31 battle panel. Its spec already assumes "updated as each
  line is released", and this phase supplies the release hook it will
  use;
- per-line animation in the web client;
- pacing non-combat bursts such as travel;
- changing buff and spell durations, which stay in game rounds.

## Durable model

- **Saved:** `ConfigOptions["combatpace"]`, one of `fast`, `normal`,
  `slow`, or `off`, absent until the player sets it. The default is
  derived when it is read, so a later `ScreenReader` toggle still gives
  the right default.
- **Runtime only:** the held-line queues. They are about 8 seconds of text
  at most, so they are flushed, not saved, at a player's quit or at
  copyover (D6). A crash loses at most one round's pending lines, which is
  acceptable. Nothing about pacing touches the round count or world clock.
  The pacer reads wall-clock time through an injectable clock, so tests
  use a fake clock.

## Module and integration

- **`internal/combatpace`** (new engine package, pure). It holds the
  per-user queues, the pace table, the scheduler, and the beat marks. Its
  API is roughly:
  - `Hold(userId, round, text, beat)`;
  - `Due(now) []Release`;
  - `Flush(userId)` and `FlushAll()`;
  - `Busy(userId)`;
  - `OnDrained(func(userId))`.

  It has its own mutex, never calls out while holding it, and returns what
  to send instead of sending it. Its only caller is the game loop, which
  already holds `LockMud`.
- **`internal/events`: causal combat tagging (D1).** Events gain an
  unexported "cause round". While the `DoCombat` listener runs, and while
  any event queued during it is dispatched, events queued inherit that
  round. `events.CombatRound()` exposes the round of the event being
  dispatched. This adds no field to `events.Message` and changes no
  existing event contract.
- **`internal/hooks`:**
  - `hooks.go` registers a cadence wrapper instead of `DoCombat` directly;
  - `DoCombat` opens the cause scope, and at its start flushes every
    queue (the previous round's leftovers);
  - `Message_SendMessage` checks the round and routes each tagged
    recipient's text into `combatpace` according to that recipient's pace
    (`off` sends it at once);
  - a `NewTurn` listener sends due lines through the existing
    connection-write path and queues `RedrawPrompt`;
  - `RoomChange`, `PlayerDespawn`, and copyover flush that player's queue
    (D6).
- **Beat marks (D3).** Where narration builds a pain or death line for a
  critical hit (`internal/combat` for 29e reactions and
  `hooks/combat_narration.go` for deaths), it marks the rendered strings
  as dramatic for the current round. The pacer gives a marked line the
  longer gap. A line whose text starts with whitespace gets the short gap.
- **`internal/usercommands/set.go`** adds `set combatpace`.
- **`modules/gmcp`:** the battle-view part of the company feed skips its
  send while `combatpace.Busy(userId)`, and `OnDrained` triggers a refresh
  (D4). `modules/gmcp` still imports only engine packages.

**Lock ordering:** `LockMud` → the `combatpace` mutex. `combatpace` takes
no other lock, and `events` does not call `combatpace`.

## Pace table (D2, D5)

| Pace | Gap per line | Crit follow-up gap | Window cap | Default for |
|---|---|---|---|---|
| fast | 0.4 s | 0.8 s | 3.0 s | |
| normal | 0.8 s | 1.4 s | 6.0 s | everyone else |
| slow | 1.0 s | 1.8 s | 7.5 s | |
| off | 0 | 0 | — | screen readers |

- The round's first line goes out at once.
- If the gaps would exceed the window, all of them scale down to fit it.
- The window is also clamped to 90% of the real combat round
  (`CombatEveryRounds × RoundSeconds`), so a server with shorter rounds
  still never spills over.
- The owner's reference text (8 lines, about 6 seconds, with 0.8–0.9 s
  gaps) is the normal row.

## Decisions (confirmed by the owner, 2026-09-29, as recommended)

- **D1: How combat text is recognised.**
  - *Recommended:* causal tagging in `events`, which catches every line a
    combat round causes, including module and script follow-ons, with no
    change at the call sites.
  - *Alternative:* explicit helpers at each call site. This is more
    visible but misses follow-on events such as the companion death notice
    and scripts.
- **D2: Spacing model.**
  - *Recommended:* a fixed gap per pace, compressed to fit the window.
    Short rounds stay brisk.
  - *Alternative:* always spread a round's lines across the whole window.
    Two lines would then sit 6 seconds apart.
- **D3: Marking crit follow-ups.**
  - *Recommended:* per-round beat marks registered by the narration code.
  - *Alternatives:* a new `events.Message` field, which touches ~60 call
    sites and the event contract; or matching the `(critical hit …)` text,
    which is brittle.
- **D4: Hold the 32g2 live battle view until a player's lines drain.**
  - *Recommended:* yes. This closes 32g2's known issue.
  - *Alternative:* defer to Phase 31.
- **D5: Pace values and default.**
  - *Recommended:* the table above, with `normal` for existing players.
- **D6: Held lines when the player leaves.**
  - *Recommended:* flush at once on moving rooms, quitting, and copyover,
    so text is never lost, only un-paced.
  - *Alternative:* drop them.

- **D7 (added, owner: yes): hold the prompt's health with the lines.**
  At each combat round's start, the prompt of every player whose pace
  isn't `off` is snapshotted. While that player has held lines, a prompt
  redraw shows the snapshot. When the lines drain, the live prompt is
  drawn. By the same rule, the web client's `Char.Vitals` and `Company`
  payloads wait for the drain, and a `CombatPaceDrained` event then
  resends them.
- **Balance shifts (owner: accept for now, retune in Phase 30).** Game-round
  systems now tick twice per combat round:
  - passive regeneration heals more between blows;
  - a fight costs about twice the hunger, thirst, and fatigue;
  - round-timed buffs and spell effects last half as many combat rounds;
  - bleeding out gives allies fewer combat rounds to save someone.

  `CombatEveryRounds: 1` reverts the cadence.

## Implementation notes (found while building)

- **Room goings-on wait behind held lines.** The real-round wiring test
  found a mob's post-round engagement line ("The bandit captain goes for
  Ysolde.", from `IdleMobs`, outside the combat round) arriving ahead of
  the round's held narration, before the death that caused it. Now, while
  a player has held lines, an untagged room message joins the end of their
  queue (`combatpace.Follow`). Say/emote (`IsCommunication`) and anything
  addressed to them directly still go out at once.
- **An open round.** The per-round web refresh can run before the round's
  lines are held, so `StartRound` opens the round for every pacing player.
  They are `Busy` from the round's start until their lines drain, or until
  the first turn if they had none. Every pacing player gets
  `CombatPaceDrained` once per combat round. The prompt is redrawn only if
  its round-start snapshot was actually shown.
- **Death lines are always dramatic.** The pacer can't tell whether a
  killing blow was critical, so every death line gets the longer gap, and
  so does every pain line (which only a critical hit causes).
- **Cadence phase.** Combat runs on game rounds whose number is a multiple
  of `CombatEveryRounds`, a single phase shared by the whole world.

## Constraints and deferrals

- Never advance or fast-forward the world clock or round count. Pacing is
  wall-clock only, and the cadence only skips `DoCombat`'s own firing.
- Survive restart and copyover:
  - the pace setting is saved;
  - held lines are flushed, not saved;
  - a mid-fight copyover resumes on the next due combat round, as today.
- Fights take about twice as long in wall-clock time. A command such as
  `flee` or `cast` waits up to 8 seconds for its combat round. `help
  combat` says so.
- Deferred:
  - the battle panel's per-line updates (Phase 31);
  - pacing text from mobs' own scripts that runs outside a combat round.

## Player help and tutorial

- **`help combat`:** combat rounds are 8 seconds (every second game
  round); why actions wait; the pace setting; a link to `help combatpace`.
- **New `help combatpace`** (`.template`):
  - what pacing does;
  - the four paces, with their gaps and windows;
  - the screen-reader default;
  - catch-up and flush rules;
  - that chat and commands are never delayed.

  It goes in `keywords.yaml` under the combat category, with aliases
  `pace`, `pacing`, and `combat-pace`.
- **`help set`:** lists `combatpace`.
- **`help narration`:** the dramatic pause before a pain or death line.
- **Tutorial:** a hint in the Combat lesson (`modules/tutorial/stages.go`)
  pointing to `set combatpace` and `help combatpace`.

## Acceptance criteria

**Timing and ordering** (a wiring test with a fake clock, through the real
`DoCombat` → `Message_SendMessage` → `NewTurn` path):
- A round's lines come out in order over the configured window.
- They never spill past the next combat round; leftovers flush before its
  first line.
- Non-combat messages sent to the same player during a round are
  delivered at once.
- `off`, and the screen-reader default, deliver at once.
- A crit's pain or death line gets the longer gap. Indented lines get the
  short one.
- A companion's death notice (through `MobDeath` → `modules/company`)
  follows its death line, and the battle summary follows the closing
  line.

**Cadence:**
- With `CombatEveryRounds: 2`, `DoCombat` fires on alternate `NewRound`
  events only.
- A test shows survival drain, alignment drift, and the world clock and
  round count are unchanged by the cadence.
- `CombatEveryRounds: 1` restores today's behaviour.

**Settings and lifecycle:**
- `set combatpace` saves the choice, survives a save and load, rejects
  bad values, and shows in `set`.
- Moving rooms, quitting, and copyover flush held lines; none are lost.

**Web client:**
- The 32g2 view does not update ahead of a player's held lines, and does
  update when they drain (D4).

**Help:**
- `help combatpace`, the updated `help combat`, `help set`, and `help
  narration` render through `help`.
- `TestTutorialHelpPointersExist` passes.

**Review and verification:**
- Focused tests pass, then `make generate`, `make validate`, and
  `go test -race ./...`.
- An independent full-diff review passes before merge, and its outcome is
  recorded in `docs/PROJECT_STATUS.md`.
