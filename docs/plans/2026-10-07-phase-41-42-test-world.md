# Phases 41 and 42: the test-only world (levels 2-33)

Spec: [the roadmap](2026-10-06-remaining-roadmap.md) ("41 and 42 World
building") and the [tile-ready conventions](../designs/tile-ready-conventions.md).
Built under Robinson's full-autonomy rule. His 2026-10-06 ruling (11:03): the
current world is stock GoMud and **will all be replaced**, so this builds only
what testing the shipped features needs. Every decision has its reason.

## What shipped

- **Nine new tile-ready zones**, chained east from Alderbrook through its
  Heather Slope (room 2125). Each is one room per map tile with hand-placed
  coordinates, 22-25 rooms, a hearth inn, a campable clearing and 4-6
  encounter rooms (never more than a third of the zone). Ids 3100-3999.

  | Zone | Band | Biome | Ids | Lair |
  | --- | --- | --- | --- | --- |
  | Brindle Downs | 4-6 | land | 3101-3123 | none |
  | Marrowmere Fen | 6-8 | shore | 3201-3225 | none |
  | Greywatch Pass | 9-11 | mountains | 3301-3324 | ice guardian (52, no relic) |
  | Cinder Hollow | 12-14 | land | 3401-3424 | none |
  | Thornreach Wood | 15-17 | forest | 3501-3525 | great timber wolf (56, no relic) |
  | Hollowweb Deep | 18-20 | forest, dungeon | 3601-3625 | forest ogre (85) |
  | Ashen Barrows | 21-24 | dungeon, cave | 3701-3724 | bone warden (92, no relic) |
  | Glassvault Depths | 25-28 | cave | 3801-3822 | lich (14) |
  | Stormcrown Heights | 29-33 | snow, cliffs | 3901-3925 | ent, spider queen or abyssal creeper |

- **Alderbrook** gains a band (2-4), a `fringe` table (rats, dogs, brigands, a
  poacher trio with a bonesetter) and eight encounter rooms, so the first
  level has something to fight between the tutorial and Brindle Downs.
- **Encounter tables** reuse the shipped mob templates only (no new mobs):
  each zone has one ordinary table (five or six compositions of 2-4 foes, a
  healer group of at most a fifth of the weight, one four-foe group) and,
  where a lair exists, a `lair` table of boss compositions (boss plus 2-3
  escorts, no healers). A lair room's chance is 40, against 15 for the zone.
- **Relic bosses placed.** All five (85, 34, 37, 14, 25) lair in this chain,
  so every relic from 36d can be earned in play, in level order.
- **Recipe pages placed.** Page 30063 (thyme-roasted game) lies in Brindle
  Downs' Drover's Leanto and 30064 (hunter's stew) in Marrowmere Fen's
  Trapper's Cache, one each, on the floor.
- **Elite and tier 4-6 reach.** The chain tops out at band 29-33, so a company
  gets to the level-30 elite promotion (38c, 39i, 39i2) and to item levels 30-35,
  which draw Masterwork (tier 4) gear and now and then Runeforged (36d, the
  tier is the item level divided by 10, plus 1, one tier either side 30% of
  the time), without a new zone.
- **Help and tutorial.** New `help regions` (road category, aliases `zones`,
  `level guide`, `where to go`, `bands`) lists the chain and its bands;
  `help encounters` and `help travel` link it; the Departure lesson's
  encounter hint points to it.

## Decisions and reasons

| Decision | Reason |
| --- | --- |
| Nine zones, one hearth inn and one camp each, not a full settlement per zone | The world is temporary; rest, cooking and camping are all that testing needs |
| No new mob templates; zones reuse skeletons, wolves, ruffians and so on at the band's levels | Encounters scale levels from the band, so a new template adds only a name; the replacement world will author its own |
| Each zone is a hand-drawn 8x4 grid with generated titles and descriptions | Tile-ready rules (one room per tile, adjacent exits, short filler text) are mechanical; prose is placeholder |
| Chain is linear, west entry and east exit | Walking order matches level order, and `walkto` plans over it without special cases |
| Brindle Downs, Marrowmere Fen, Cinder Hollow have no lair | A lair is a relic's home; the five relic bosses fill the five lairs at 9-11, 15-17, 18-20, 21-24 and 25-28, and the summit holds two |
| Summit lair table is two boss compositions (ogre, ent), 50/50 | Exercises a lair with several bosses and keeps two relic sets earnable at the top |
| Entry chance stays 15 and a lair's is 40 | The 37 pacing numbers (8.7 entries between battles) stand, and a lair must spring often enough to be found |
| The inn/camp gap rule: one inn and one camp per zone of ~24 rooms | 37's rule is a rest point at least every ~20 eligible entries |
| Recipe pages and relic bosses placed by hand in zone rooms, not a shop | Phase 56 left them to the world, and the economy rule says gathered or bought goods must not resell for profit |
| Dungeon, cave and web rooms keep the dark biomes | They exercise the light module and its to-hit penalties |
| Pages lie on the floor and stay until picked up | The world is temporary; a page respawn clock is a world-replacement concern |

## Review changes (2026-10-07)

- **Lairs follow relic wear levels.** 36d sets relic ilvl to boss level + 5
  and a relic is worn at ilvl - 5, so each relic boss belongs where its own
  level (band low + 3) is its relic's wear level: ogre 22, lich 30, ent 35,
  spider queen 40, abyssal creeper 45. The build lairs gave a level-12
  company ogre relics it could not wear until 22 (and the ent and spider
  queen before the lich). Now the ogre lairs in Hollowweb Deep (boss 21),
  the lich in Glassvault Depths (boss 28), and the summit holds the ent,
  spider queen and creeper as trophies to grow into (their bands, 34-60,
  wait for the replacement world). Greywatch Pass, Thornreach Wood and Ashen
  Barrows keep lairs with relic-less bosses so lair play is testable at
  every band. `TestTestWorldLairsMatchTheirRelicLevels` holds this.
- **Recipe pages respawn.** A floor item is gone for every other company
  once one picks it up, so the pages are `spawninfo` entries that return an
  hour after being taken.
- **Aliases.** `help regions` no longer claims `zone` (the admin zone
  command's page), `levels` or `world`.

## Deferred

- Named settlements, merchants beyond the existing markets, quest givers, a
  boss respawn clock and zone `loot:` profiles (the item-level default of
  band to top plus 2 applies).
- Elite-level bands beyond 33 (levels 34-60); the replacement world places
  them, and with them the ent, spider queen and creeper lairs at their relic
  levels. The pilot lairs (Dark Forest ogre at 5-7, Catacombs lich at 10-12)
  predate this and still sit under their relics' wear levels.
- Landmark art for lairs beyond the existing legends (Cave, Throneroom, Pond,
  Keep, Hermit), and a map screenshot pass.

## Tests

`modules/encounters/world_test.go` reads the shipped data and checks: the
band chain (valid tables, no gaps, reaches 30); encounter pacing (at most a
third of rooms, one inn and a camp per zone, tile-ready, start room in the
zone); every encounter room names a table, lairs are boss tables with
chance 20 or more, and every `loot.RelicBosses()` boss lairs somewhere;
both recipe pages are placed; every composition builds at its planned
levels; and a user walks the whole chain through the real `Go` command from
Alderbrook to the summit lair, passing every zone in band order.
`internal/rooms` `TestShippedTileReadyZonesFollowTheConventions` covers the
new zones by itself. `internal/usercommands` renders `help regions`, checks
its aliases, band lines against the shipped zone configs, and the links from
`help encounters` and `help travel`; `TestTutorialHelpPointersExist` covers
the new hint.

## Gates

`make generate`, `make validate`, `make js-lint`,
`go test -race -timeout 30m ./...`; `make smoke-world`.
