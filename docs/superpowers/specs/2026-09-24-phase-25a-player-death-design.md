# Phase 25a: Player Death and Church Return

Implements the first half of the
[death and resurrection spec](2026-09-23-death-resurrection-design.md),
fourth on the [onboarding roadmap](2026-09-23-company-life-onboarding-roadmap.md).

**Split.** The parent spec has two mechanics that share little code. They are
split the same way as 21a/21b and 23a/23b:

- **25a (this document):** a player's death costs one level, and the player
  wakes at the church of the last city they visited, with every living
  companion.
- **25b (next):** a companion's durable `Dead` state, the rescue allowance
  counted only in the leader's online time, `resurrect` at a city church or
  village shaman, expiry, and village shaman content.

Until 25b ships, a companion that dies behaves as it does today: its gear is
cleared, and it respawns with the leader at the leader's next login.

The open decisions below were settled by applying this design's
recommendations, as in Phases 19b–24, under the owner's "work on the next
phase" instruction (2026-09-24).

## Prior-art check

- **Engine death** (`internal/usercommands/suicide.go`): a player at
  `Health <= -10` is sent the `suicide` command by the combat loop
  (`NewRound_DoCombat`) or by `NewRound_AutoHeal` (every 3 rounds, out of
  combat). Players can also type `suicide`. The command gives a script the
  chance to take over (`TryUserDieEvent`), announces the death, updates
  kill/death stats, queues `events.PlayerDeath`, and handles permadeath. For
  players above `Death.ProtectionLevels` (5), it drops equipment and gold by
  `EquipmentDropChance` and applies `Death.XPPenalty` (`none` in the shipped
  config). It then leaves a corpse, cancels buffs, sets health to −10, and
  moves the player to `DeathRecoveryRoom` (75, the Shadow Realm; an
  ephemeral copy per player). There, buff 24 heals the player and a portal
  eventually leads back to the start room.
- **Levels** (`internal/characters`): reaching level `L+1` takes
  `XPTL(L)` experience; the floor of level `L` is `XPTL(L-1)` (0 at level
  1; `Validate` raises experience below 1 to 1). `LevelUp` grants `TrainingPoints`/`StatPoints` on every level gained,
  so a level lost to death and then earned again would grant the points a
  second time. `Validate` recalculates stats and clamps health.
- **Durable per-character markers:** Phase 22a keeps its kit marker in
  `Character.MiscData`, saved in the same user file as the items it covers.
- **Company** (`modules/company`): live companions are tracked in
  `m.instances`; `Runtime.IsAttached` tells whether one still serves its
  leader. A companion that dies is untracked (`onMobDeath`) and respawned on
  the leader's next `PlayerSpawn`. Optional provider interfaces on
  `company.FormationProvider` (`ArchetypeProvider`, `AlignmentProvider`,
  `ChemistryProvider`) let engine packages call the module without
  importing it.
- **Travel** (`modules/expedition`): the leader waits in the origin room
  while a session runs. On completion, the module moves the leader to the
  destination from *wherever they are*, so a leader who died mid-journey
  would be teleported from the church to the destination. `Traveling →
  Cancelled` is a legal transition; `Interrupted` leaves through
  `ReturnForProfile`.
- **Camping** (`modules/camping`): a camp belongs to its leader and a room.
  A resting camp can't be broken, and a rest completes on its timer
  wherever the leader is. Inn stays work the same way.
- **Room tags:** `room.HasTag` (`inn`, `market`, `camping`, `blackmarket`).
  Frostfang's Sanctuary of the Benevolent Heart is room 18, with a
  clergyman (mob 4). Dunmar has no church.
- **Locks:** commands and events run on the game loop under the world lock.
  The expedition and camping modules take their own mutex inside it (world
  lock → module lock); their timers take only the module lock.

## Decisions

1. **The Ashveil death path replaces the engine's penalty and destination,
   through a seam.** A new `internal/death` package has a provider seam
   (`SetProvider`/`Active`). `Suicide` is unchanged when no provider is
   registered, as in tests or another world. With one registered:
   - the script override, announcements, kill/death stats, and
     `PlayerDeath` event stay as they are. The event is never
     `Permanent`, and the engine's permadeath branch doesn't run;
   - item and gold drops and the corpse stay governed by the death config,
     including its protection levels ("existing corpse and item-drop rules
     remain governed by the death config");
   - the engine's `XPPenalty` is skipped;
   - instead of setting health to −10 and moving the player to the Shadow
     Realm, `Suicide` hands the player to the provider's `Respawn`.
2. **One level, always.** `Character.LoseLevel()` drops one level (none at
   level 1) and sets experience to the floor of the resulting level, so at
   level 1 the player keeps level 1 with no progress. It then calls
   `Validate`, which recalculates stats and clamps health (and keeps
   experience at 1 or more, so the level-1 floor is 1). Mana is clamped too.
   Training and stat points, whether spent or unspent, are untouched.
   Protection levels don't apply.
3. **No re-granted points.** A new durable `Character.PeakLevel` records the
   highest level ever reached; 0 on an older character means its current
   level. `LevelUp` grants training and stat points only for a level above
   the peak, so re-earning a lost level grants nothing a second time.
   `LoseLevel` raises the peak to the level being lost before dropping it.
4. **A durable pending death, charged once.** `suicide` hands the provider
   either a **new death** or a **retry**. A new death applies `LoseLevel`
   and sets `MiscData["death-pending"]` to an operation ID
   (`death-<userId>-<round>`, which reads the round counter and never writes
   it). The level and the marker live in the same user file, so they are
   saved together, by the engine's usual user saves. Once the player is at
   the church, the marker is cleared. `suicide` checks, before anything else:
   - a player still **down** (health below 1) with the marker set is a
     **retry**: only the return is attempted, with no announcement, corpse,
     drop, or level, and the stale killer fields and damage record are
     cleared;
   - a **living** player returned to a church this round or the last
     (`JustReturned`, in memory) is ignored. The combat loop and AutoHeal can
     both queue a `suicide` for the same death in one round; without this,
     the second would be a whole new death at the church (review finding);
   - anything else is a new death, including a pending player who was healed
     and then killed again: two deaths, two levels.

   A crash before the user file is saved loses the whole death, which is the
   engine's existing behaviour for deaths, drops, and corpses.
5. **Checkpoint.** A settlement registry in the death module's config lists
   each settlement's zone, kind (`city` or `village`), and service room.
   A **valid church** is a `city` entry whose service room loads and
   carries the `church` room tag. When a player enters a room (a
   `RoomChange` event, and on `PlayerSpawn`) in a city zone whose church is
   valid, that church's room ID is stored in `MiscData["death-checkpoint"]`.
   When the checkpoint changes, the player is told once: *"Should you fall,
   you will wake in <church>."* Villages never change it. Like the
   character's room, the checkpoint rides the normal user saves.
6. **Destination.** The destination is the checkpoint, if it is still a
   valid church. Otherwise it is the configured `FallbackRoomId` (18,
   Frostfang's Sanctuary), if that room loads. If neither will do, the
   death stays pending: the player stays where they fell at −10 health,
   out of any fight,
   told once that *"The way back is closed to you. The gods have been
   told."* An error is logged. The engine's own triggers retry `suicide`
   every few rounds (AutoHeal every 3 rounds out of combat; the combat
   loop in combat). Each retry goes straight to `Respawn` (decision 4), so
   the move is retried without losing another level, until an operator
   repairs the church or the config.
7. **Close journeys first.** Before moving the player, `Respawn` asks two
   new seams to abandon the leader's sessions through their own durable
   lifecycle:
   - `expedition.AbandonForDeath` removes the leader's session, whatever
     its state (Traveling, Interrupted, or a Completed or Cancelled record
     that is only waiting for cleanup), and stops its timer. It doesn't
     move anyone, and a travel encounter mob stays where it was. Without
     this, a completed record would move the leader from the church to
     the destination.
   - `camping.AbandonForDeath` removes the leader's camp, resting or not,
     and any inn stay. A rest still running grants no recovery and no tier,
     and an inn stay refunds nothing. A rest that has already finished keeps
     its recovery: it is synced first, as `camp status` would.

   Each removal is a single save, done at once. If a save fails, the
   session and its timer are kept, the death stays pending (decision 6),
   and it is retried. If the module's data couldn't be read at load, the
   abandon fails too: the unread file may hold a journey that would move
   the player once repaired (review finding).
   A travel ambush's hostile mob is left on the road, as it would be if the
   company had fled.
8. **Arrival.** `Respawn` then clears the player's aggro and moves them to
   the destination (`rooms.MoveToRoom`, the player's normal room-change
   path, so the church is recorded as visited). It sets health and mana to
   `RespawnVitalsPct` (50) percent of their new maximums, with health at
   least 1. Next it moves the company (decision 9), clears the pending
   marker, and queues a `look`. The player is told *"You lose a level (now
   level N)."*, or *"You lose what you had learned toward level 2."* at
   level 1. Then comes *"You wake before the altar of <church>."* and,
   when anyone came along, *"Your company is with you."* The church room
   sees *"<name> is carried in and laid before the altar."* The Shadow
   Realm and its recovery buff are no longer part of an Ashveil death.
9. **The company comes too.** A new optional provider on
   `company.FormationProvider`, `RelocationProvider.RelocateCompany(leader,
   room)`, moves every tracked companion whose mob is live, attached to the
   leader, and alive (`Health >= 1`) into the church. It clears their
   aggro and reports how many moved. It works on live mobs only: records,
   formation, gear, alignment, and chemistry service are unchanged. A crash
   needs no recovery, because companions respawn with the leader on login.
   A dead companion has no live mob and doesn't come (in 25a it respawns at
   the leader's next login, as today). A companion awaiting restoration is
   restored with the leader later, through the normal `PlayerSpawn` path.
10. **Shipped content.** Frostfang's Sanctuary (18) gets the `church` tag.
    A new **Chapel of the Wayfarer** (room 2007, Dunmar, `church` tag) sits
    east of Dunmar Market Square (2004). Its keeper is **Sister Maren**
    (mob 65): non-hostile, carrying nothing, with no gold and no drops.
    She is the chapel's resurrection NPC in 25b. The config lists
    Frostfang (city, 18) and Dunmar (city, 2007). No village is shipped in
    25a; the kind is supported and tested, and 25b ships the first shaman.
11. **No new command.** The checkpoint message (decision 5) tells players
    where they will wake. A read-only display belongs to the
    information-surfaces phase.
12. **No new locks.** The death module runs on the game loop. The two
    abandon seams take their module's own mutex inside the world lock, the
    same order as the `travel` and `camp` commands.

## Scope

**In scope:**

- `internal/characters`: `PeakLevel`, `LoseLevel`, and the `LevelUp` grant
  rule.
- `internal/death` (pure and seam): settlement kinds, the registry
  (`NewRegistry` with validation, `ChurchFor(zone)`, `IsChurch(room)`),
  `Destination`, the provider seam, and MiscData key constants.
- `internal/usercommands/suicide.go`: the provider branch.
- `internal/expedition` and `internal/camping`: the `AbandonForDeath`
  seams. `modules/expedition` and `modules/camping` implement them.
- `internal/company`: `RelocationProvider` and `RelocateCompany`;
  `modules/company` implements it.
- `modules/death`: config parsing, the checkpoint listener, and `Respawn`.
  `make generate` wires it.
- Content: room 18's tag, room 2007, the 2004 exit, mob 65, and the
  config.
- Module `AGENTS.md`, config comments, and `docs/PROJECT_STATUS.md`.

**Deferred to 25b:** companion `Dead` records, the online-time allowance,
`resurrect`, expiry, and village shaman content.

**Deferred beyond:** a checkpoint display (information surfaces) and death
tutorial text.

## Constraints

- Never advances or writes the world clock or round count; the operation
  ID only reads it.
- Survives restart and copyover: the checkpoint, peak level, level, and
  pending marker are all in the user file; abandoned sessions are saved in
  their own stores before the player moves.
- Exactly once: the pending marker keeps a death from costing a second
  level, whether it is retried or replayed.
- No new locks; the lock order is world lock → module lock.

## Acceptance criteria

- Pure: `LoseLevel` at levels 1, 2, and 10 (level, experience floor, peak,
  stats recalculated, health clamped); `LevelUp` grants points only above
  the peak; the registry rejects bad entries (unknown kind, a room ID ≤ 0,
  or a zone listed twice); a village is never a checkpoint; the destination
  falls back from an invalid checkpoint to the fallback, then to none.
- `Suicide` with no provider behaves as before (the Shadow Realm); with a
  fake provider it skips the XP penalty and permadeath, keeps drops, and
  calls `Respawn` once; a pending player skips straight to `Respawn`.
- Module: the checkpoint is set on entering a city with a valid church,
  never by a village or a city without one; the player is told once per
  change. A death costs one level (the protection levels don't help), wakes
  the player at the checkpoint church with the configured vitals, and
  clears the marker. An invalid checkpoint uses the fallback. No loadable
  church keeps the death pending without moving the player, and the retry
  costs no second level. A failed travel or camp abandon keeps the death
  pending. Living, attached companions arrive with the player; dead,
  detached, or unattached ones don't.
- Expedition and camping: `AbandonForDeath` removes Traveling, Interrupted,
  and terminal records, and resting and idle camps and inn stays. It never
  moves the leader, a failed save restores the session, and a later timer
  does nothing.
- Wiring: through `plugins.Load` with the shipped config and rooms, a
  player walks into Dunmar (the checkpoint is set), then dies through the
  real `suicide` command via `usercommands.TryCommand` while travelling
  with a live companion. The player wakes in the Chapel of the Wayfarer one
  level lower, with the companion. The travel session is gone, and the
  checkpoint and level survive a user save and reload. Separately, a death
  with no checkpoint wakes in Frostfang's Sanctuary.
- `go test -race ./...`, `make generate`, `make validate` pass.
