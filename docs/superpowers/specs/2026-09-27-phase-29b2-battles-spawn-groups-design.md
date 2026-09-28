# Phase 29b2: One Battle at a Time, and Spawn Groups — Design

Asked for by the owner on 2026-09-27, after 29b merged. It changes how
fights are formed (29a, 29b) and how enemies spawn (11a), before the
narration phases build on them.

## The owner's rules (2026-09-27)

1. **One battle at a time per player.** A room may hold several enemy
   groups. A player (with their company) fights one group at a time. When
   that battle is over, a new battle begins at once with the next hostile
   group that was already set on them.
2. **Another player takes the next group.** If a second player walks in,
   they can fight the next group while the first finishes theirs.
3. **No lone enemies.** A room never produces several groups of one mob
   each. Every hostile group has at least two members, unless the mob is
   marked as able to stand alone. The room's spawn list is the pool the
   group is built from, mixed kinds together ("think Ogre Battle").

Answers to the design questions (asked 2026-09-27):

- **Waiting groups hold back:** they don't attack a player who is fighting
  another group; when that battle ends, the next one starts, in the order
  the groups turned on the player. If another player is free, the next
  group fights them instead.
- **The spawn list builds the group:** spawning is changed so a room's
  hostile entries form a group, not lone mobs.
- **Standing alone** is a per-mob flag set by content authors.
- **Everyone:** the rule applies to solo players too, not only companies.

The rest are this design's recommendations, marked **(recommendation
applied)**, listed for the owner in `docs/PROJECT_STATUS.md`.

## Prior-art check

- **Spawning** (`internal/rooms/rooms.go`, `Room.Prepare`): each
  `spawninfo` entry with a `mobid` spawns one instance and tracks it
  (`InstanceId`, `DespawnedRound`, `RespawnRate`). Entries are
  independent; nothing groups them. Of the 84 hostile groups the shipped
  rooms produce today, **71 are a single, non-large mob**.
- **Parties** (`internal/mobparty`, 11a): mobs are grouped by their first
  `groups` tag (a zone-wide tag such as `slum-ruffians`), in chunks of up
  to 5; a mob with no tag is a party of one. Party ids are the tag plus a
  chunk index.
- **Picking fights:** a hostile mob's idle `lookfortrouble` sets its
  `Aggro` on a player (or a charmed companion) in the room. Any number of
  groups can attack one player at once.
- **29a's upkeep** keeps every party engaged with a company fighting, all
  at once. **29b's stream** opens one fight per company and room, holding
  every party it fights there, and only for a company with a companion
  present.
- **Races** have a size (small, medium, large), but the owner chose a
  per-mob flag instead.
- **Mob instances** are not kept across a restart or copyover; rooms
  respawn them (companions are restored by the company module).

## Decisions

### A. Battles (`internal/battle`, new)

1. **A battle** is one player (with their company, if any) against one
   enemy group. A small runtime registry, never persisted, keyed by the
   player, holds each player's battle: the room, the round it began, and
   the enemies it knows (instance ids, grown as the group is seen). A
   group is matched by any enemy it shares with the battle, so a group
   whose party id changes stays the same battle.
2. **Who is set on a player:** a group is *set on* a player when any of
   its living members in the room aims at the player or one of their
   companions, or the player or a companion aims at one of its members.
   The registry remembers the round each group was first set on each
   player, so groups are served in that order (ties by their order in the
   room).
3. **The battle each round,** decided at the top of `DoCombat`, before
   the upkeep and any blow:
   - a player whose battle's group is still standing in the room and still
     set on them keeps it;
   - otherwise the player's battle ends (decision C), and the earliest
     group set on them, if any, becomes their new battle at once.
4. **Waiting groups hold back:**
   - a member of a group set on a player who is in a battle with a
     different group doesn't strike that player or their companions. It
     keeps its aim, and waits, silently;
   - **another player free in the room is taken instead (rule 2).** A
     waiting group whose target is busy turns, as a whole, on a player in
     the room with no battle (the first such player, by user id), who is
     then fighting it: `The bandits turn on Brom.`;
   - a waiting group that becomes free to strike (its player's battle
     ended) starts that player's next battle the next round, with no
     message beyond the new battle's start.
5. **The player and their company stay on their battle:**
   - **`attack` on a waiting group is refused** while the player is in a
     battle: `You're fighting the bandits. Finish that fight first.`
     **(recommendation applied: the owner chose "hold back", not "can be
     pulled in")**;
   - 29a's upkeep keeps the company on the battle's group only: it no
     longer draws companions onto other groups set on the leader;
   - a companion's in-turn reassignment chooses only from the battle's
     group.
6. **Two players, one group:** a group may be in battle with several
   players at once (both chose to fight it, or a group joined). Each player
   still fights one group.
7. **Solo players too** (the owner's answer). A player with no companion
   has battles, waits, and refusals the same way. 29a's upkeep (turning
   and drafting) still needs a companion present, as before.

### B. Spawn groups (`internal/rooms`, `internal/mobparty`)

1. **A room's hostile spawn entries form a group** when they spawn:
   - after `Room.Prepare`'s spawn pass, the hostile mobs the room's list
     has spawned, alive in the room and not yet in a group, join a **spawn
     group**, a shared runtime id on each mob (`SpawnGroup`,
     `spawn:<room>:<n>`);
   - a newly respawned mob joins an existing spawn group in the room that
     isn't in a battle and has room (up to 5), else starts a new one.
2. **Parties follow spawn groups.** `mobparty.Assemble` groups a mob by
   its spawn group when it has one, else by its first `groups` tag as
   before. The zone-wide `groups` tags keep their other meaning
   (hostility shared across a zone).
3. **At least two (rule 3).** A spawn group with one member is topped up
   to two with a copy drawn from the room's hostile entries, in list
   order, so a room listing a rat and a ruffian gives a rat and a ruffian,
   and a room listing only a ruffian gives two ruffians **(recommendation
   applied)**. Top-up copies are not tracked by the spawn list: they don't
   respawn on their own, and are made again, if needed, when the group
   next forms.
4. **More than five:** a room whose hostile entries exceed five forms
   several groups of even size (six gives three and three, never five and
   one) **(recommendation applied)**. `mobparty`'s chunking of a large
   tagged crowd is changed the same way.
5. **Standing alone:** a mob with `solitary: true` in its file is never
   grouped or topped up (a troll, a dragon, a named boss).
6. **Only hostile mobs** (the mob's `hostile`, or the entry's
   `forcehostile`) are grouped. Shopkeepers, quest givers, townsfolk, and
   wildlife that only fight back are left as they are
   **(recommendation applied)**.
7. **Grouped mobs don't wander off alone** **(recommendation applied)**:
   a spawn-group member's wandering is turned off, so a group stays
   together. Roaming groups are deferred.

### C. The event stream and summary (29b, revised)

1. **A fight is a battle.** The stream opens a fight when a battle begins
   and ends it when the battle ends: one player (and company) against one
   group. 29b's "one fight per company and room" is replaced; it was a
   stopgap for the lone-mob rooms this phase removes.
2. **Solo players get fights and summaries** like a company
   **(recommendation applied)**. `set battlesummary` turns it off.
3. **Endings** are as in 29b: every company member dead (defeat); the
   group gone from the room (victory: slain, beaten, fled, or left); or
   neither side still fighting (broken off). A victory no longer waits for
   other foes in the room: they are the next battle.

## Module

- **`internal/battle`** (new, leaf): the registry (`Current`, `Begin`,
  `End`, first-set order), pure selection (`Next`), a mutex.
- **`internal/hooks`:** the battle pass at the top of `DoCombat`; the hold
  check before a mob strikes a player or companion; the free-player turn;
  the upkeep limited to the battle's group; the stream's fights opened and
  ended from battles.
- **`internal/usercommands/attack.go`:** the refusal.
- **`internal/mobs`:** `SpawnGroup` (runtime) and `Solitary` (from the
  file).
- **`internal/rooms`:** forming spawn groups and topping up after the
  spawn pass.
- **`internal/mobparty`:** grouping by spawn group, even chunks.
- **Content:** `solitary: true` on the shipped large or boss mobs that
  should stand alone (reported for the owner's check).

## Invariants

- **The clock:** nothing here advances time; battles read the round.
- **Restart and copyover:** battles and spawn groups are runtime only.
  After a restart, rooms respawn and regroup; a fight in progress resumes
  from `Aggro` as a new battle.
- **Locks:** `internal/battle` has one mutex, never held across a world
  call. The rest runs on the game loop.

## Acceptance criteria

- **Unit:** battle order and selection; grouping by spawn group; even
  chunks; top-up choice from the list; `solitary` exempt; non-hostile
  untouched.
- **Wiring through the real round** (shipped config, real commands,
  `DoCombat`, idle mobs):
  - a room with three hostile groups: the player fights one at a time,
    each a separate fight and summary; the waiting groups land no blow
    while they wait; the next battle starts the round after one ends;
  - a second player walks in: the next waiting group turns on them, and
    both battles run at once;
  - `attack` on a waiting group is refused;
  - a room spawning one hostile mob spawns a group of two; a room listing
    two kinds spawns them as one group; a `solitary` mob spawns alone;
  - 27c's practice fight and 29a/29b's tests pass (adjusted where 29b's
    one-fight-per-room behaviour is replaced).
- **Player help:** `help combat` and `help targeting` describe battles and
  waiting groups; `help battle-summary` covers solo players; the tutorial's
  Combat lesson says a group fights as one and others wait their turn.
- `go test -race ./...`, `make generate`, and `make validate` pass. The
  independent review is recorded.

## Deferred

- **Roaming groups** that wander together.
- **A group pulled into a battle by a blow** (the owner chose hold back).
- **Group composition rules** beyond the room's list (leaders, roles):
  30c.
