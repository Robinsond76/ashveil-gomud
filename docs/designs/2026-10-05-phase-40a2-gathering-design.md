# Phase 40a2: gathering (herbs, firewood, fishing, game)

Status: **approved 2026-10-06 under the owner's delegation**; open questions are decided in the [remaining roadmap](../plans/2026-10-06-remaining-roadmap.md#decisions-on-open-questions) (handoff rule 20). Part
of the [visual client milestone](2026-10-05-visual-client-milestone-design.md).
It follows [40a room resources](2026-10-05-phase-40a-room-resources-design.md),
which supplies the room data, `look` line, GMCP field and map icons. Art:
S1 resource icons, including `depleted`, in the
[sprite specification](2026-10-05-sprite-specification.md).

## Goal

Make the four gathering resources real. A room marked with herbs,
firewood, fishing or game lets a company spend real time and effort to
gather from it. Gathering:
- exposes the company to danger;
- strips the room for a while;
- feeds cooking, camping, and later brewing and poison-making.

In a dangerous, unforgiving world, supplies are earned in the wild, not
only bought.

## Prior art (code as of 2026-10-05)

- **Camp Forage** (Phase 33f3, `modules/camping/camp_specialists.go`): a
  finished camp rest rolls the zone's forage table. On the Old Kings Road
  the finds are raw game meat, wild thyme and mushrooms. Rewards come at
  most once per `CampRewardCooldown` (15 real minutes). The best Forage
  specialist adds finds by level.
- **Cooking** exists. Raw game meat and wild thyme become seared or
  thyme-roasted game.
- **Camp fire:** `camp fire` lights a fire for free
  (`CampingModule.lightFire`). Resting needs a lit fire.
- **Survival exertion:** `ApplyCompanyExertion` charges fatigue and strain
  to the whole company. Walking strain depends on terrain
  (`modules/walking`).
- **Planned systems** that will consume gathered goods:
  - [camp consumables](2026-10-01-camp-consumables-design.md) (emberleaf,
    coolmint, bitter moss, watchsage, clearroot);
  - [weapon poisons](2026-10-01-weapon-poisons-design.md) (Bitterleaf,
    Leechbane, Leadroot, Mirethorn ingredients);
  - [loot goods](2026-10-05-loot-system-design.md) (pelts, hides,
    provisions that spoil).
- **Random room encounters** (Phase 37) define a zone's encounter chance.

## Owner decisions

- Herbs, firewood, fishing and game are wanted, even if they need new
  mechanics now or later (2026-10-05).
- Ashveil is a dangerous, unforgiving world (2026-10-05).
- Never advance shared world time for travel or rest (standing invariant).
  Gathering is treated the same way.
- **Camps need firewood** (2026-10-05). The fuel rule below is approved.
  [40a3 camp gear](2026-10-05-phase-40a3-camp-gear-design.md) refines it
  (one bundle per rest; embers) and adds the fire steel for damp wood.

## Proposed for approval

### Commands

| Command | Room needs | Time (real) | Tool |
|---|---|---|---|
| `gather herbs` | `herbs` | 20 s | None. A knife or sickle adds +1 yield |
| `gather firewood` | `firewood` | 20 s | None (deadfall). A hatchet or axe doubles the yield |
| `fish` | `fishing` (or `water` on shore or water biomes) | 30 s | A fishing line (new, cheap item). Without one, `fish` is refused |
| `hunt` | `game` | 30 s | A bow, crossbow or sling equipped by any company member present. Without one the company sets snares instead: one roll at half chance |

- `gather` alone lists what the room offers and what you can do.
- Only the **company leader** gives the order. The whole company takes
  part, so companions present add effort and their specialists apply.
- **Interruption:** the work is a timed action, like a camp rest.
  - Moving, being attacked, starting a battle, or typing any command
    other than `look` or `conditions` cancels it with nothing gained.
  - It cannot start in battle, while resting, or while traveling (40d).
- **It never advances the world clock.** The timer is real time, the way
  the inn's rest timer is.

### Cost and risk

- **Effort:** each attempt charges the company the strain of one step on
  the room's terrain, through `ApplyCompanyExertion`. Fishing charges
  half.
- **Danger:** once Phase 37 ships, each attempt rolls the zone's random
  encounter chance once, plus 10 points for `hunt` (the noise and blood
  draw attention). Before 37, there is no extra roll.
- **Night and weather:** gathering in darkness halves herb yield (you
  cannot see what you're picking). Heavy rain halves the firewood that's
  fit to burn (the bundles come back `damp`; see Firewood).

### Yields

Each success yields a roll on a **biome or zone table** in data, the same
shape as today's forage tables, so content can grow without code.

| Resource | Base yield | Specialist and skill bonus | Launch items |
|---|---|---|---|
| Herbs | 1–2 | Forage specialist: +1 per 2 levels (as camp Forage). A caster with **Scribe**: 10% chance of one rarer herb | wild thyme, spotted mushroom, mushroom; plus reserved ingredient items for consumables and poisons, which drop only once those systems ship |
| Firewood | 2 bundles | Tool doubles; a warrior Field Smith adds +1 | firewood bundle (new) |
| Fishing | 0–2 fish (a 40% chance per catch) | Ranger: +10% per catch | raw fish (new; spoils like meat) |
| Game | 1 carcass on success (a 50% base chance) | Ranger: +5% per level, up to 90%; Forage specialist: +1 meat | raw game meat ×2, plus a goods drop (pelt or hide) once the loot goods (36a/36b) exist |

- **Unskilled foraging risk:** with no Forage specialist present, a herb
  gather has a 15% chance to return one **bitter weed** instead: a junk
  item worth nothing. Telling good leaves from bad is a skill.
- **Cooking:** a new recipe turns raw fish into grilled fish, sized like
  seared game meat.

### Room pools (depletion)

- Each gathering resource in a room has a **pool**: herbs 3, firewood 4,
  fishing 4, game 2. Each successful attempt spends one charge. Fishing and
  hunting spend a charge even when the catch fails.
- A pool regrows **one charge every 20 real minutes** (game every 40
  minutes).
- Pools are **shared by every player**: the world is shared, and the first
  company to arrive takes the bounty.
- **Empty pools:**
  - the command says the room is picked clean for now;
  - `look` adds "(picked clean)" after that resource;
  - the map draws the `depleted` overlay on its icon;
  - GMCP `Room.Info` gains `depleted: ["herbs"]`.
- **Persistence:** pools are durable world state. A module store saves
  `roomId → resource → {charges, lastRegrowUTC}`, so a restart or copyover
  never refills the world early. Regrowth is computed from real time
  elapsed on load and at each read. Only rooms that have been gathered
  from are stored.

### Firewood and the camp fire

- **`camp fire` needs fuel:** the camp room is a `firewood` room (gather
  costs nothing extra; lighting draws on the deadfall), **or** the
  company spends one **firewood bundle** from the leader, the packs or the
  cargo.
  - With neither, the fire cannot be lit, and so the company cannot rest
    at camp. Inns are unaffected.
  - This is a behavior change that suits the harsh world. Approved by the
    owner (2026-10-05).
- **Damp bundles** light only on a second try and give no warmth bonus.
- **Prepared kit:** a bundle is light enough to carry two or three.
  Provisioners sell bundles, so you can prepare before an expedition.
- **Tutorial:** the Camp lesson's room becomes a `firewood` room, and the
  tutorial kit gains one bundle, so the lesson still works.

### Items (new)

| Item | Type | Notes |
|---|---|---|
| firewood bundle | Supply | Weight 2 kg. Sold by provisioners. Consumed by `camp fire` |
| damp firewood bundle | Supply | As above; needs two tries to light; no warmth bonus |
| fishing line | Tool | Cheap. Breaks on 5% of uses |
| raw fish | Food (raw) | Cookable; spoils |
| grilled fish | Food | Cooked result |
| bitter weed | Junk | Worthless; teaches the lesson |

Item IDs come from the next free block. Their icons belong to the S7 item
icon set, and the text works without them.

## State and persistence

- **Pools:** a durable module store (above).
- **Gathering in progress:** an in-memory timed action per company. It is
  **not** persisted; a restart or copyover simply cancels it, and nothing
  is gained or lost.
- **Gathered items:** they enter inventory or cargo through the existing
  item and cargo paths, which already persist.

## Integration points

| Area | Change |
|---|---|
| A new `modules/gathering` module (`init()` registration, `make generate`) | commands, timed action, pools store, yield tables in its config overlay |
| `modules/camping` | `camp fire` fuel rule; tutorial kit |
| `internal/survival` | effort through `ApplyCompanyExertion` |
| Cooking recipes | grilled fish |
| `modules/gmcp` | `depleted` in `Room.Info` and `World.Map` |
| `internal/usercommands/look.go` | "(picked clean)" |
| Phase 37 hook | the encounter roll, when it exists |
| World data | items; `herbs`, `firewood`, `fishing` and `game` tags on fitting rooms; provisioner stock |

## Player help and tutorial

- **New page:** `help gathering`, with the aliases `gather`, `herbs`,
  `firewood`, `fish`, `fishing`, `hunt`, `hunting` and `snares`. It covers:
  - each command, its time, tool, yields and the specialist bonuses;
  - the dangers;
  - picked-clean rooms and regrowth;
  - bitter weeds;
  - firewood for the camp fire.
- **Updated pages:**
  - `help resources` (40a);
  - `help camp` (fuel rule);
  - `help forage` (the difference between camp Forage and gathering);
  - `help cooking` (fish);
  - `help survival`.
- **Tutorial:** the Camp lesson explains firewood. The Survival lesson
  hints `gather`.

## Acceptance tests

1. Each command:
   - is refused in a room without its resource, in battle, while resting,
     and while traveling;
   - succeeds after its real-time delay;
   - is cancelled by moving, by an attack and by a typed command, with
     nothing gained.
2. The world clock does not advance during gathering.
3. Yields:
   - follow the tables;
   - tools, specialists and Scribe apply;
   - the bitter-weed chance applies without a Forage specialist and not
     with one.
4. Pools:
   - spend charges;
   - refuse when empty;
   - regrow by real time;
   - **persist across a save and load**, and a copyover fixture does not
     refill them;
   - are shared between two companies.
5. Empty pools show "(picked clean)" in `look` and `depleted` in GMCP.
6. `camp fire`:
   - burns a bundle from the leader, packs or cargo;
   - is free in a firewood room;
   - fails with neither;
   - inn rests are unaffected;
   - the tutorial Camp lesson still completes.
7. Effort charges strain. `hunt` adds the encounter bonus once Phase 37
   ships; that test is pending until then.
8. `help gathering` renders, and `TestTutorialHelpPointersExist` passes.

## Open questions

1. Confirm the times, pools, regrowth rates and success chances as balance
   defaults.
2. Should pelts and hides wait for the loot goods (36a/36b)? The proposal
   is yes: until then, `hunt` gives meat only.
