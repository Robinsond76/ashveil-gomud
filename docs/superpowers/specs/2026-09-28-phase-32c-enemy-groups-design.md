# Phase 32c: Enemy Groups — Design

From the owner's play-test notes of 2026-09-28
([roadmap](2026-09-28-playtest-feedback-roadmap.md)). Enemy groups get a
name and a description, a fight is started by naming a group, and `scout`
shows a group's formation before the fight.

## The owner's notes and rules

1. "I see *a pack of 4 creatures*. How do I attack it?"
2. There's no way to see an enemy's formation.
3. **Decided (owner, 2026-09-28):** only a **group** can be attacked, by
   the group's name. Naming a member is not a way to start a fight.

The rest are this design's recommendations, marked **(recommendation
applied)**, for the owner to confirm on review.

## Prior-art check

- **The room line** (`internal/rooms/mobparty_display.go`): hostile mobs
  are grouped by `mobparty.Assemble`. A party of one keeps its own line; a
  larger party is "a pack of N <name>s", or "a pack of N creatures" when
  the members differ. Nothing in the line says how to fight it.
- **Parties** (`internal/mobparty`, 11a; 29b2): grouped by spawn group
  when a mob has one (`spawn:<room>:<n>`, `encounter:<room>:<first>`),
  else by its first `groups` tag, else alone. Party ids change as
  members fall; the battle registry (`internal/battle`) matches a group
  by any member it shares. Each party's formation is assigned fresh on
  every call: toughest first, front row first.
- **Engine view** (`internal/enemyparty`): `Parties(room)` and
  `PartyOf(room, id)` over the room's non-charmed mobs. Combat and
  `formation reach` read enemy formations through it.
- **`attack`** (`internal/usercommands/attack.go`): `attack <name>` is
  `room.FindByName`, which matches a single mob or player by name
  (`util.FindMatchIn`, with GoMud's `name#2` form for the second match).
  `#<id>` and `@<id>` pick an exact mob or player; companions and GoMud
  party members join a fight with `attack #<id>`. A bare `attack` in a
  battle takes a foe from it (29b2), and `attack` at a waiting group is
  refused.
- **`cast`, `backstab`, `shoot`** name a single target the same way and
  refuse a waiting group up front (29b2).
- **`look <name>`** shows one mob; **`consider`** rates one mob;
  **`peep`** (skill) shows one mob's health and more.
- **The 3x3 grid** is drawn for the player's own company by `formation`
  (`modules/company`). `formation reach` names the enemies a member can
  strike, in a fight. There is no view of an enemy group's grid, and no
  `scout` command or skill.
- **Races** have `size` and `tameable`, but nothing that says what a
  group of them is called.
- **The tutorial's practice squad** (27c) is three straw footmen and a
  straw archer sharing a `groups` tag, spawned by `modules/tutorial`
  without a spawn group. The Combat lesson says `attack footman`.
- **The travel ambush** (29b2) spawns a pair of its foe as
  `encounter:<room>:<first>`.
- **The web client** gets the room's mobs one by one in GMCP
  `Room.Info.Contents.Npcs` (name, adjectives, aggro), with no group.
- **The battle summary** (29b, `internal/combatstream/summary.go`) names
  the outcome but not whom the fight was with.

## Decisions

### A. A group's name

1. **Every group has a name**, given when it forms and kept while it
   stands **(recommendation applied)**:
   - **generated:** a collective noun and the plural of its most common
     member ("a band of ruffians", "a pack of wolves", "a swarm of
     rats"); a tie goes to the toughest (the one in front). A mixed group
     is named for its most common kind; its description lists the rest;
   - **authored:** a room's spawn list may name the group it forms
     (`groupname: the Rat King's court`), and a module may name the
     groups it spawns (the tutorial's "straw squad", below).
2. **The collective noun** comes from the mob's race (`groupnoun` on the
   race file: humans, elves, goblins, and trolls a "band", canines a
   "pack", rodents and insects a "swarm", the undead a "host", the rest a
   "band"), and a mob file may override it (`groupnoun: patrol`). The
   exact table is fixed in the plan and reported for the owner's check.
3. **Stable while it stands.** The name is stored on each member at
   runtime (`GroupName`, beside `SpawnGroup`), so a band of ruffians stays
   a band of ruffians when its ruffians fall and only its rat is left. A
   survivor that regroups (29b2) takes the name of the group it joins.
   Not persisted: after a restart or copyover, rooms respawn and regroup,
   and the names are given again.
4. **A mob that isn't in a group** (a `solitary` troll, a shopkeeper, a
   lone straggler) is a group of one, named by its own name, so
   `attack troll` works as it does today.
5. **Two groups with the same name** in a room are listed as "a band of
   ruffians" and "a second band of ruffians", and told apart with
   GoMud's existing form: `attack ruffians#2`.
6. **Words that name a group:** its noun, its kind's plural, and the
   words of an authored name. So `attack band`, `attack ruffians`,
   `attack band of ruffians`, or `attack court` all work. A member's own
   name does not (decision C).

### B. Groups in the room

1. **One line per group** in `look`, the name first, then what's in it:
   ```
   A band of ruffians (4): two ruffians, a cutpurse, and a rat.
   A swarm of rats (2).
   ```
   A group of one kind needs no list. A group of one mob is shown as the
   mob is today.
2. **What it's doing**, appended when it's fighting: "(fighting you)",
   "(fighting Brom)", or "(waiting)" for a group set on you while you
   fight another **(recommendation applied)**.
3. **Hidden members** aren't counted or listed unless the viewer can see
   hidden things, the same as today's per-mob line. A group whose
   members are all hidden isn't shown.
4. **The web client:** each NPC in GMCP `Room.Info.Contents.Npcs` gains
   `group` (the group's name, empty for a mob alone), so the web client
   can group them. Drawing groups in the web client is 32g's Combat tab.

### C. Starting a fight: `attack <group>`

1. **`attack <group>`** starts a battle with that group. Your first aim
   is the member you'd turn on anyway: the weakest foe you can reach
   (29a's rule). Your companions join as they do today.
   ```
   You prepare to fight a band of ruffians!
   Dain prepares to fight a band of ruffians.
   ```
2. **Naming a member is refused** when you aren't already fighting its
   group (the owner's rule), and the refusal says what to type:
   ```
   The ruffian fights with a band of ruffians. Type attack ruffians.
   ```
3. **Inside your battle, naming a member aims at it** **(recommendation
   applied)**: `attack cutpurse` while you fight its band turns you on the
   cutpurse, with the usual reach check. The owner's rule is about
   starting a fight; picking a target within one is still useful, and 32d
   keeps it for players who want to override their automatic choice.
4. **A mob alone** is attacked by its own name, as today (A.4).
5. **Unchanged:** a bare `attack` (29b2), `attack` at another group while
   you're in a battle (refused, 29b2), `attack <player>` (PvP rules), the
   exact `#<id>` and `@<id>` forms companions and parties use, and the
   `*`, `*mob`, and `*user` random forms.
6. **`cast`, `backstab`, and `shoot`** follow the same rule
   **(recommendation applied)**: a group's name starts the fight, aimed
   at the member `attack` would choose (a shot, having no reach limit,
   takes the weakest member in the group); a member's name is refused
   unless you're already fighting its group. Otherwise a harmful spell or
   an arrow would be the way around the owner's rule. A helpful spell at
   anyone, and `look`, `consider`, and `peep` at a member, are unchanged.

### D. `look <group>`

`look <group>` describes the group:

```
A band of ruffians, four strong, idling by the door.
  a ruffian (unhurt), a ruffian (wounded), a cutpurse (unhurt), a rat (unhurt)
  Type scout ruffians to see how they stand, or attack ruffians to fight them.
```

- the name and count, and what it's doing (idle, fighting you, fighting
  another, waiting its turn);
- each visible member with a health word (the words `peep`'s first level
  gives);
- an authored group may carry its own description (`groupdesc` beside
  `groupname` in the spawn list), shown first.

`look <member>` still shows that member, as today.

### E. `scout`

1. **`scout <group>`** draws the group's formation, front row nearest
   you, in the same grid style as your own `formation`:
   ```
   A band of ruffians, as they stand (front row first):
            1              2              3
     1  ruffian        ruffian        -
     2  cutpurse       -              -
     3  -              rat            -
   The front of each column takes your first blows: plain melee reaches
   only them, a reach weapon one rank deeper, a bow anyone.
   ```
   Each cell shows the member and its health word. When you're in the
   grid yourself, the members you can reach from your place are marked
   (what `formation reach me` would name).
2. **`scout`** alone lists the groups in the room with their counts, the
   same as the room lines (B.1), and says to `scout <name>` one.
3. **Free and instant** **(recommendation applied)**: no skill, no roll,
   no round spent, in or out of a fight, including a group waiting its
   turn. Ogre Battle shows a unit's make-up to anyone who looks; a skill
   gate would hide the one thing the owner asked to see. A skill that
   reveals more (a group's archetype roles, 32d; its morale, 30e) can be
   added later.
4. **It sees what `look` sees:** hidden members are left out, and where
   the viewer can't see the room's mobs (darkness, 14), scout says it's
   too dark to make them out.
5. **The formation is the one combat uses now**, from
   `enemyparty.PartyOf`. It changes as members fall (the one behind steps
   up), and scout says so in `help scout`.

### F. The fight's name

The battle summary (29b) names the group: "The fight with a band of
ruffians is over." The stream's fight carries the group's name so later
presentation phases (29c, 32g) can use it.

### G. The tutorial

- The practice squad is named **"the straw squad"**, set by
  `modules/tutorial` when it raises the squad.
- The Combat lesson's hints change: `scout squad` to see how they stand,
  then `attack squad` to start the fight; `attack <name>` inside the
  fight picks a target; the reach hint names `formation reach` and
  `scout`.
- The travel ambush pair (29b2) is named like any group: two bandits
  are "a band of bandits".

## Module

- **`internal/races`, `internal/mobs`:** `groupnoun` (race, and a mob
  override); `GroupName` (runtime, on the mob).
- **`internal/mobparty`:** naming, pure: the noun, the most common kind,
  pluralisation (replacing `describeParty`'s "+s"), the count words, and
  "second"/"third" for repeats.
- **`internal/rooms`:** the name given when a spawn group forms or a
  survivor regroups; `groupname`/`groupdesc` on spawn entries; the room
  lines; a group lookup by name (`FindGroupByName`) beside `FindByName`.
- **`internal/enemyparty`:** a group's name, members, and formation for a
  room, for the commands below.
- **`internal/usercommands`:** `attack`, `cast`, `backstab`, `shoot`
  (C); `look` (D); `scout` (E, new).
- **`internal/combatstream`:** the fight's group name (F).
- **`modules/tutorial`:** the squad's name, the hints (G).
- **`modules/expedition`:** the ambush pair's name.
- **`modules/gmcp`:** `group` on each NPC (B.4).
- **Content:** `groupnoun` on the shipped races.

## Invariants

- **The clock:** nothing here advances time; `scout` spends no round.
- **Restart and copyover:** names are runtime only, like spawn groups;
  rooms regroup and name again after a restart. Nothing new is stored.
- **Locks:** everything runs on the game loop and reads mob instances as
  combat does. `internal/battle`'s mutex is not held across any of it.

## Acceptance criteria

- **Unit:** naming (noun from race and mob override, most common kind,
  ties, pluralisation, mixed groups, repeats as "second"); name
  stability as members fall; the words that match a group; the room line
  (counts, lists, doing, hidden members left out).
- **Wiring** (shipped config, real commands, `DoCombat`):
  - a room with a mixed group: `look` shows its name and list;
    `attack <noun>` and `attack <kind>` start a battle with it; the first
    aim is the weakest reachable member;
  - `attack <member>` with no battle is refused with the group's
    command; inside that battle it retargets;
  - two groups of the same name: `attack ruffians#2` takes the second;
  - a `solitary` mob and a shopkeeper are attacked by their own names;
  - `backstab`, `shoot`, and `cast` at a member are refused, and at the
    group start the fight;
  - `look <group>` and `scout <group>` in and out of a fight, with a
    waiting group, with a hidden member, and in the dark;
  - the name holds as members fall; a regrouped survivor takes its new
    group's name;
  - the battle summary names the group;
  - the tutorial: `scout squad` and `attack squad` pass the Combat
    lesson; `attack footman` before the fight is refused with the hint;
  - GMCP `Room.Info.Contents.Npcs` carries `group`;
  - 29a, 29b, and 29b2's fight tests pass (their `attack <member>` calls
    moved to the group's name where they start a fight).
- **Player help:** a new `help scout`; `help attack` rewritten for
  groups (replacing GoMud's page); `help targeting` (naming a group,
  retargeting inside a battle), `help combat` (the hub links `scout`),
  `help backstab`, `help shoot`, and `help cast` updated; `scout` and
  its aliases in `keywords.yaml`; the Combat lesson points to `scout`;
  `TestTutorialHelpPointersExist` passes.
- `go test -race ./...`, `make generate`, and `make validate` pass. The
  independent review is recorded.

## Deferred

- **Scouting the next room** (a group seen through an exit): later, with
  roaming groups.
- **`consider <group>`** (rating a whole group against the company).
- **What `scout` reveals with skill** (roles, morale): 32d, 30e.
- **Groups in the web client:** 32g's Combat tab.
- **Automatic targeting within a battle** for the player: 32d.
