# Phase 40d: tile-ready showcase region and click-to-walk

Status: **built 2026-10-06** (see [As built](#as-built)); approved under the owner's delegation; open questions are decided in the [remaining roadmap](../plans/2026-10-06-remaining-roadmap.md#decisions-on-open-questions) (handoff rule 20). Part
of the [visual client milestone](2026-10-05-visual-client-milestone-design.md).
Roadmap item 4: an area density pass on one region, plus click-to-walk.
It reuses art set S2.

## Goal

1. Build one showcase region where **one room is one tile**: a dense, walkable
   area laid out on the map, with resources (40a) and landmarks (40c).
2. Let players **travel** to a visited room by clicking its tile, or with a
   text command. Travel walks step by step through ordinary moves, so
   denser areas don't mean tedious typing.
3. Write down **tile-ready conventions** and a validator, so later zones
   follow them.

## Prior art (code as of 2026-10-05)

- **Starting region:** Dunmar (rooms 2001, 2003, 2004, 2006 and 2007),
  Old Kings Road (2002 and 2005) and Fernhollow (2008 and 2009). Every room
  already has hand-placed coordinates. Room 2002 (Fork at the Black Oak) is
  the camping proving room. 2003 is the Waymark Inn, and the tutorial's
  Departure leads there.
- **`internal/mapper`** lays out rooms and has an A* pathfinder
  (`mapper.path.go`). It is used today by admin and NPC tools.
- **`modules/walking`:** every step costs survival strain by terrain,
  eased by Rested, Well Rested and the Pathfinder specialist. Companions
  follow the leader through the company's own movement hooks.
- **Random room encounters** (Phase 37) will interrupt movement in the
  wilds.
- **Zone config** (`zone-config.yaml`) holds `name`, `roomid` and
  `defaultbiome`.
- The map client already has a click handler, used for an admin teleport
  menu.

## Owner decisions

- Areas become denser: multiple rooms represent an area, laid out as tiles
  (2026-10-05).
- Walking must never advance shared world time (standing invariant).
- **A showcase area may be built** to show off the new map features.
  Full world building comes later, once everything is in place
  (2026-10-05).
- **Travel as proposed is approved:** one step every 1.5 s, at most 60
  steps, leader only (2026-10-05).

## Proposed for approval

### Tile-ready conventions

A zone opts in with `tileready: true` in its `zone-config.yaml`. In a
tile-ready zone:

1. Every room has hand-placed coordinates, and no two rooms on one z-level
   share a coordinate.
2. Each compass exit leads to the adjacent coordinate in its direction
   (north is y − 1, and so on). Diagonals are allowed; `up` and `down`
   change z. Any exception declares `mapdirection`, as GoMud allows.
3. Every room has a `biome` (or inherits the zone default) and its
   `resources`.
4. **Filler rooms** (plain terrain between landmarks) have short
   descriptions of 1–2 sentences, so walking stays quick to read.
5. Landmark rooms set a `maplegend` that maps to an S2 landmark.

A **validator test** over the world data enforces 1, 2 and 5 for every
tile-ready zone. Zones that aren't tile-ready are untouched.

### Showcase region

**Recommendation:** build a **new, compact showcase area** rather than
rebuilding the starting rooms. Dunmar, Old Kings Road and Fernhollow keep
their rooms and tests untouched, and the showcase connects to them by an
exit.

- **Placement and size:** off the Old Kings Road (from room 2005,
  Trappers' Post), about 30–50 tile rooms. It is a single zone marked
  `tileready`.
- **It must show off every new feature:**
  - several biomes: road, meadow, forest, a stream and lakeshore, a rocky
    hillside with a cave mouth, and a hamlet;
  - every resource type (40a and 40a2);
  - landmarks;
  - an up and down exit;
  - a locked door;
  - a campable clearing;
  - one dense loop to test `travel`.
- **Content stays light:** short filler descriptions and existing mobs.
  World building later decides whether to keep, grow or replace it.
- New rooms take free IDs. The exact layout is drawn in the plan, for
  owner review, before any rooms are written.

The alternative, kept below for reference, is to densify the starting
region itself:

- **Region:** the starting region of Dunmar, Old Kings Road and
  Fernhollow, unless world building has replaced it by the time 40d runs.
  In that case, use the first levels 1–15 zone.
- **Target size:** about 40–60 rooms across the three zones. For example:
  - Dunmar becomes a small town block with streets;
  - the Old Kings Road becomes a road of 10–15 tiles through forest and
    fields, with side clearings;
  - Fernhollow becomes a forest hamlet with a stream (water).
- **Existing room IDs and their tags are kept:**
  - 2002 keeps `camping`;
  - 2003 keeps `inn`;
  - tutorial and recruiter references are unchanged.

  New rooms take free IDs in the zone's block. The exact layout is drawn in
  the plan, for owner review, before any rooms are written.
- Encounters, spawns and NPCs keep their rooms unless the layout plan says
  otherwise.

### Travel (click-to-walk)

- **Command:** `travel [room]` and `travel stop`.
  - The web map sends `travel <roomId>` when a visited tile is clicked,
    after a one-tap confirm.
  - Text players can use `travel <landmark>` for a visited room in the
    current zone whose legend matches (for example `travel inn`), and
    `travel` alone to see where they're headed.
- **Path:** the mapper's A*, restricted to:
  - **rooms the player has visited**;
  - exits that are not secret, plus secret exits the player has already
    used;
  - locked doors the player can open with a key they carry.

  At most 60 steps. No path means the command refuses and explains why.
- **Walking:**
  - one ordinary move per step, through the same movement path as typing
    the direction, so strain, encounters, followers, separation,
    hunger/thirst and weather all apply;
  - **one step every 1.5 seconds of real time** (proposed);
  - **never advances the world clock** (invariant);
  - only the company leader can travel, and the company follows as
    usual.
- **Stops:** travel stops on:
  - arrival;
  - any battle or aggression;
  - a blocked or changed exit;
  - entering a room with a hostile;
  - a survival warning threshold;
  - any command typed by the player, other than `look`, `map` or
    `travel`;
  - quit, death or copyover.

  The reason is told in one line.
- **State:** travel is a per-user in-memory plan. It is **not persisted**.
  After a restart or copyover, the player simply isn't traveling.
- **Map display:** the planned path shows `walk-dot` markers and
  `walk-target` on the destination (S1 art). Both clear on stop.

## Integration points

| Area | Change |
|---|---|
| World data | the tile-ready showcase zone, `tileready` zone flag |
| `internal/rooms` zone config | `tileready` field |
| Validator test | coordinates, exit-direction and landmark checks |
| New `travel` command (`internal/usercommands` or a `modules/travel` module) | plan, step timer, stop rules |
| `internal/mapper` | a visited-and-usable-exit filter for its path search |
| `modules/gmcp` | `Char.Travel` `{target, path: [roomIds], reason}` while traveling, `{}` otherwise |
| `window-map.js` | click to travel (non-admin), path markers. The admin teleport menu stays for admins |

## Player help and tutorial

- **New page:** `help travel`, with the aliases `click to walk`,
  `autowalk` and `walk to`. It covers how to travel, which rooms you can
  reach, its speed, what stops it, and that strain and encounters still
  apply.
- **Updated pages:** `help worldmap` (clicking tiles), `help webclient`.
- **Tutorial:** a Departure lesson hint: "Click a tile on the map, or
  type `travel inn`, to walk there."

## Acceptance tests

1. The validator passes on the showcase zone. It fails on a fixture with
   overlapping coordinates or a mismatched exit direction.
2. Kept room IDs keep their tags: the `camping`, `inn` and tutorial tests
   still pass.
3. `travel`:
   - refuses an unvisited room, and a room past a locked door without its
     key;
   - never uses a secret exit the player hasn't used.
4. Travel walks the planned path one move per tick. Each step costs strain
   like a typed move. Companions follow. **The game clock is unchanged.**
5. Travel stops on combat, on a typed command, and at arrival, each with
   its reason line.
6. GMCP `Char.Travel` shows the path while traveling and clears on stop.
7. Browser check: clicking a visited tile walks there and draws the path.
8. `help travel` renders, and `TestTutorialHelpPointersExist` passes.

## Open questions

1. Approve the showcase-area recommendation, with its placement and size.

Resolved: the travel numbers (approved). World building comes after the
new features are in place, so it can adopt the tile-ready conventions from
the start.

## As built

Built on 2026-10-06. Where the build departs from the proposal above:

- **The command is `walkto`, not `travel`.** `travel` already belongs to
  journeys (`travel status`, `travel resume`, `travel return`), so the
  click-to-walk command is `walkto [place]` and `walkto stop`, with the
  alias `autowalk`. The GMCP namespace is `Walkto` for the same reason.
  The help topic is `help walkto` (aliases `click-to-walk`, `walk to`).
- **Planning is its own breadth-first search** (`internal/walkto`), not the
  mapper's A*: the mapper walks every exit of the zone map, while a walk may
  use only visited rooms and usable exits, across zones. It is pure over a
  `Graph` and unit tested.
- **Landmark words search the zone you stand in.** A room number works
  anywhere you have been; `walkto inn` finds the nearest visited room in the
  current zone whose `maplegend` is `Inn`, else whose name starts with or
  contains the word.
- **Exits with an `exitmessage` are not walked** (they delay and requeue the
  move), like journey exits.
- **Showcase region: Alderbrook**, a new zone of 44 rooms (ids 2101 to 2144)
  east of the Trappers' Post, not a rebuild of the starting rooms.
- **Validator** is `rooms.ValidateTileReady`, covering conventions 1 to 5 of
  [tile-ready-conventions](tile-ready-conventions.md).
- **Stops** are the listed ones, checked before each step; a need that already
  warned when the walk began does not stop it again (only a new crossing, or
  fatigue at zero). The step timer only queues an event; steps run on the game
  loop.
- **Web map:** a click offers `Walk to [room]` through the existing menu
  (one pick confirms; admins keep the teleport items), the path is drawn
  with the S1 `walk-dot` and `walk-target` markers, and two 40c follow-ups
  are folded in: S1 resource icons replace the dots (dots remain on the
  classic style, on tiles under 24 px and for an icon still loading), and
  outdoor tiles are shaded by the game's time of day (`Gametime`; an hour of
  dusk and dawn; indoor and dark biomes, which `World.Map` biomes now flag,
  are not shaded; a Day/night switch in the map settings).
