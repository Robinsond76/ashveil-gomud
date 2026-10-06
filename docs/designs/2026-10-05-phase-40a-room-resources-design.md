# Phase 40a: room resources

Status: **approved 2026-10-06 under the owner's delegation**; open questions are decided in the [remaining roadmap](../plans/2026-10-06-remaining-roadmap.md#decisions-on-open-questions) (handoff rule 20). Part
of the [visual client milestone](2026-10-05-visual-client-milestone-design.md).
Roadmap item 1: data, icons and `look` text, with water wired into
survival. Art: set S1 resource icons in the
[sprite specification](2026-10-05-sprite-specification.md).

## Goal

Rooms say what they provide. A player can see it in `look`, in the map
tooltip, and as an icon on the map tile. Every marker is backed by a real
rule: if the map shows water, you can drink and fill a waterskin there.

## Prior art (code as of 2026-10-05)

- **Rooms** (`internal/rooms/rooms.go`) already carry `tags` (free strings
  used by modules, for example `camping` and `inn`), `biome`, `mapsymbol`,
  `maplegend` and coordinates.
- **GMCP `Room.Info`** (`modules/gmcp/gmcp.Room.go`) sends `details`, which
  includes every room tag. `World.Map` sends the same for visited rooms.
- **Water today** comes only from items. The waterskin (item 30015) is a
  `drinkable` with 5 uses and 40 hydration. Nothing refills it. `drink` and
  `company drink` provision through `survival.Provision`.
- **Forage** (`modules/camping/camp_specialists.go`) rolls a zone table
  (`Forage[zone]` in the camping config) when a camp rest finishes.
- **Camp rests** take their recovery from the zone weather's
  `RestRecoveryPct`, locked when the rest starts (Phase 16).
- `look` (`internal/usercommands/look.go`) appends module lines after the
  room description: sky lines, then camp lines, then recruiter lines.

## Owner decisions

- The idea, its purpose and its MUME inspiration (2026-10-05).
- The map shows icons for room resources; water is wired into survival.
- **Every resource is wanted** (2026-10-05): water, forage, shelter, herbs,
  firewood, fishing and game, even where new mechanics are needed. 40a
  ships the first three. [40a2 gathering](2026-10-05-phase-40a2-gathering-design.md)
  adds the other four with their mechanics.

## Proposed for approval

### Resource list

Every resource gets a real rule. 40a ships the three whose rules are
small. The four gathering resources are accepted in data now, and appear
when 40a2 gives them their mechanics, so an icon never promises nothing.

| Resource | Shown in 40a | Rule |
|---|---|---|
| `water` | Yes | `drink water`, `fill`, and `company drink` work here without spending supplies |
| `forage` | Yes | A camp rest finished here gets **+1 forage find** |
| `shelter` | Yes | A camp rest here halves the weather's rest penalty: recovery moves halfway from the weather's `RestRecoveryPct` toward 100 |
| `herbs`, `firewood`, `fishing`, `game` | From 40a2 | `gather herbs`, `gather firewood`, `fish` and `hunt`, with room pools and firewood for the camp fire. See 40a2 |

### Data model

- A new room field, `resources: [water, forage]`, in room YAML. It is
  validated against the list above when rooms load. An unknown entry logs
  a warning and is dropped.
- Resources are **world data**, not player state. Nothing new is
  persisted. Room instances copy the field like `tags`.
- Why not plain `tags`? Tags are free-form and already sent to the client
  as `details`. A separate field gives one validated vocabulary that `look`,
  survival and the map all read.

### Water rules

- **`drink water`** (or `drink source`) in a water room provisions the drinker (or a named
  member, as `drink` already allows) with the same hydration as one
  waterskin glug (40). It is refused in battle, like `drink`. A carried
  item named "water" takes precedence for `drink water`; `drink source`
  always means the room.
- **`fill [container]`** in a water room refills a refillable container to
  its full uses. The waterskin gains a new item-spec flag, `refillable:
  water`. With no argument, `fill` refills every refillable container the
  leader carries. It is refused in battle.
- **`company fill`** also refills refillable containers in company members'
  packs and the cargo.
- **`company drink`** in a water room waters every member from the source
  first. Items are spent only where there is no water source.
- Water takes no game time and never advances the world clock.

### Forage and shelter rules

- **Forage:** the camp-rest forage roll adds one find when the camp room has
  `forage`. The zone table still decides what is found. A zone without a
  table finds nothing extra.
- **Shelter:** when a camp rest starts in a `shelter` room, the locked
  recovery percent becomes `pct + (100 − pct) / 2`. Inn rests are
  unaffected.

### Presentation

- **`look`** adds one line after the sky lines, for example: `Here: fresh
  water, forage, shelter.` Order follows the table above. Rooms without
  shown resources print nothing.
- **GMCP:** `Room.Info` and each `World.Map` entry gain `resources`, an
  array of shown resource IDs. It is never empty, never null, and omitted
  when there are none. Reserved resources are not sent.
- **Map window** (`window-map.js`):
  - draws S1 icons in a tile corner, up to 3 at a time and then a "+" pip;
  - the tooltip lists the resources by name;
  - a legend toggle in map settings shows icons on or off;
  - before the art exists, each resource falls back to a small colored dot.

### World data

- Tag the existing default world where it fits: lakeshores, rivers, wells
  and springs get `water`; forest and farmland camps get `forage`; caves
  and overhangs get `shelter`.
- The tutorial's Survival room gets `water`, so the lesson can teach
  `fill`.
- An audit list goes in the plan; this design does not fix room IDs.

## Integration points

| Area | Change |
|---|---|
| `internal/rooms` | `Resources` field, load validation, accessor |
| `internal/items` | `refillable` spec field; refill to the spec's `uses` |
| `internal/usercommands` | `drink water`, `fill` |
| `modules/company` | `company fill`; `company drink` checks for a source first |
| `modules/camping` | forage bonus; shelter recovery |
| `internal/usercommands/look.go` | resource line |
| `modules/gmcp` | `resources` in `Room.Info` and `World.Map` |
| Web client | icons, tooltip, legend toggle |
| World data | resource tags; waterskin `refillable` |

## Player help and tutorial

- **New page:** `help resources`. It covers what each icon means, the water
  commands, the forage and shelter rules, and that some resources are not
  shown yet. Index it under `help:` with the aliases `water source`,
  `fill`, `spring` and `shelter`.
- **Updated pages:**
  - `help drink`: drinking from a source;
  - `help survival`: where water comes from;
  - `help forage`: the room bonus;
  - `help camp`: shelter;
  - `help company meal`/`company drink`: drinking at a source;
  - `help webclient`: map icons.
- **Tutorial:** the Survival lesson adds a hint to `fill` the waterskin in
  that room.

## Acceptance tests

1. Room YAML with `resources` loads, and an unknown entry is dropped with a
   warning. A test runs over the world data so every shipped resource is
   valid.
2. `look` prints the resource line in a water room and prints nothing in a
   room with none or only reserved resources.
3. `drink water`:
   - provisions the drinker in a water room;
   - is refused elsewhere and in battle;
   - waters a named companion.
4. `fill`:
   - restores a used waterskin to 5 uses;
   - ignores non-refillable items;
   - is refused away from water and in battle.

   `company fill` refills packs and cargo.
5. `company drink` at a source waters everyone and spends no items. Away
   from a source it behaves exactly as today.
6. A camp rest finished in a `forage` room yields one more find than the
   same rest elsewhere. The 15-minute reward cooldown is unchanged.
7. A camp rest started in a `shelter` room locks the halved weather
   penalty.
8. GMCP `Room.Info` and `World.Map` carry `resources`, and never carry
   reserved resources.
9. The world clock does not advance during any of the above.
10. `help resources` renders, and `TestTutorialHelpPointersExist` passes.
11. The map shows icons for the current and visited rooms (browser check).

## Decisions made while building (2026-10-06)

- Water freezing waits for the weather work; the +1 forage find and the
  halved shelter penalty are the shipped defaults.
- `drink water` yields to an item exactly named "water"; away from a
  source it keeps its old meaning (a carried waterskin), while `drink
  source` always means the room.
- A source drink gives the Hydrated buff like a waterskin glug.
- `company fill` refills cargo by withdrawing and re-depositing part-used
  stacks through the existing cargo API.

## Open questions (resolved)

1. Should water sources freeze in snow-biome blizzards? The proposal is
   later, with the weather work.
2. Confirm the forage (+1 find) and shelter (half the weather penalty)
   numbers as balance defaults.
