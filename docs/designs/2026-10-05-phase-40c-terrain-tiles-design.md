# Phase 40c: terrain and landmark tiles

Status: **design draft, awaiting owner approval** (handoff rule 20). Part
of the [visual client milestone](2026-10-05-visual-client-milestone-design.md).
Roadmap item 3. Art: set S2 in the
[sprite specification](2026-10-05-sprite-specification.md).

## Goal

Replace the map's colored squares with colorful terrain tiles for each
biome. Replace letter symbols with landmark pictures (inn, gate, temple…).
Keep a toggle back to the classic look. This is client-only work: no
gameplay changes.

## Prior art (code as of 2026-10-05)

- `window-map.js` draws each visited room as a square or circle. Its color
  comes from `colorForSymbol`, using the per-biome colors and symbol
  overrides in the biome table from `World.Map`.
- Rooms are spaced on a grid (`roomSpacing` > `roomSize`), with connection
  lines between rooms that share an exit. Exits leading to unvisited rooms
  are drawn as stubs (`zoneExitStubs`). Locked and secret exits get
  badges.
- **Biomes:** 19 ids in `_datafiles/world/default/biomes/`. Every room
  reports its biome as `environment`, with the zone default when the room
  sets none.
- **Landmarks:** rooms use `maplegend` (Inn, Bank, Trainer, Gate…) and
  `mapsymbol` letters.

## Owner decisions

- A colorful tile map that updates as you move, keeping the current
  browser look otherwise (2026-10-05).

## Proposed for approval

### Two map styles

Map settings gain `style`: `tiles` (the new default once S2 art ships) or
`classic` (today's rendering, unchanged).

### Tile rendering

- Each room draws its biome's tile at the room's size. The variant is
  `roomId mod 3`, so it is stable and the same for every viewer. Animated
  biomes (water, shore, swamp, snow, desert) cycle their 4 frames at 250 ms.
  The animation runs only while the window is visible, and not at all with
  reduced motion.
- **Contiguous mode:** in `tiles` style, spacing defaults to the room size,
  so neighbouring rooms touch and read as terrain.
- **Walls between tiles:** two touching tiles with **no exit** between them
  get a dark 2 px edge on the shared side. Otherwise the map would suggest
  you can walk where you can't.
- **Exits:** an exit between touching tiles needs no line. A longer exit
  (non-adjacent coordinates, as in layouts that aren't tile-ready) keeps
  today's connection line. Locked and secret badges are unchanged.
- **Up and down exits** get `exit-up` and `exit-down` chevrons (S1). Only
  the current z-level is drawn, as today.
- **Fog:** the end of each unvisited-exit stub gets the `fog` tile, so the
  map's edge looks unexplored. It is never drawn for secret exits the
  player hasn't found; the stub rules for secret exits are unchanged.
- **Unknown biome:** the `unknown` tile.
- Pixel art is drawn at whole-number scale with smoothing off. Zoom snaps
  to steps that keep tiles crisp in `tiles` style.

### Landmarks

- A client mapping file (`sprites/map/landmarks.json`) maps `maplegend`
  values (case-insensitive) and leftover `mapsymbol` letters to S2
  landmark IDs. The sprite specification's landmark table is its source.
- A matched room draws its landmark overlay over the terrain. An unmatched
  symbol keeps today's glyph, drawn over the tile with a contrasting
  outline. The tooltip still shows the legend text.
- Resource icons (40a) and camp and unit layers (40b) draw above
  landmarks. The order is: terrain, walls, landmark, resources, camp,
  units.

### Loading and fallback

- The 40b sprite loader fetches terrain and landmark sheets lazily, per
  biome and landmark seen. While an image loads, or if it fails, the room
  draws its classic color square. The map never blanks.
- Performance: each zone's static terrain layer is cached to an offscreen
  canvas and redrawn only when rooms are added or settings change. Animated
  tiles and units draw on top every frame.

## State and persistence

Client only: settings live in browser storage. The server is unchanged,
apart from serving the new static files.

## Integration points

| Area | Change |
|---|---|
| `window-map.js` | style setting, tile renderer, walls, fog, chevrons, landmark overlay, layer order, zone cache |
| Static files | `sprites/map/terrain/*`, `sprites/map/landmarks/*`, `landmarks.json` |
| `help` | `help worldmap` styles section |

## Player help and tutorial

- `help worldmap` (from 40b) gains a styles section. It covers terrain,
  walls, fog, landmarks, and switching to classic.
- `help webclient` notes the setting.
- **Tutorial:** none needed beyond 40b's pointer. This changes how the map
  looks, not what you can do.

## Acceptance tests

1. **Browser check** (scripted Chromium, screenshots kept with the PR) on a
   zone with mixed biomes:
   - tiles draw;
   - two touching rooms without an exit show a wall edge;
   - landmarks replace their letters;
   - fog sits on unvisited stubs;
   - `classic` restores today's look exactly.
2. Deleting one terrain image falls back to the color square without
   errors.
3. A test over the world data checks that every `maplegend` in shipped
   rooms maps to a landmark ID in `landmarks.json`, or is listed as an
   intentional glyph.
4. Reduced motion stops animated tiles.
5. `make js-lint` passes, and `help worldmap` renders.

## Open questions

1. Should `tiles` become the default immediately, or opt-in until S2 art
   is complete? The proposal: opt-in until every biome has a tile.
