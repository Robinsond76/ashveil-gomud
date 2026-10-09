# A2: Terrain tiles

Read [00-standards.md](00-standards.md) first, especially section 5
(terrain). Attach `tile-sample.png` and `scene-map.png` from A0.

Every room on the map is one square tile of its biome. Neighboring rooms
touch, so the tiles make one continuous painted map. Figures, landmarks
and icons stand on top of the tiles, so keep detail calm.

## Base tiles: 3 variants per biome (`art/source/A2/map/terrain/<biome>-<1|2|3>.png`)

Each variant is a full, opaque **512×512** square, seen straight top-down.
The game picks a variant per room, so any variant can touch any other.

| Biome | Variants show |
|---|---|
| `cave` | Dark stone floor with scattered rubble and faint mineral glints. |
| `city` | A cobbled street of worn, uneven stones with an occasional drain grate or crack. |
| `cliffs` | A sheer rock ledge with cracks and lichen; the lower part is shadowed as if it drops away. |
| `default` | Neutral packed earth with a few pebbles (the fallback). |
| `desert` | Wind-rippled sand with a few pebbles and a dry tuft. |
| `dungeon` | Fitted grey flagstones with moss in the joints. |
| `farmland` | Tilled rows of dark soil; the variants show three crop stages (bare shoots, green, ripening gold). |
| `forest` | Dense tree canopy tops with dark gaps of understory. |
| `fort` | A packed-earth parade ground with timber or stone edging. |
| `house` | Worn wooden floorboards with a rug corner or a dropped item. |
| `land` | Open grassland with tufts, small pale flowers and a stone. |
| `mountains` | A rocky slope of grey stone and scree with a snow highlight on the peaks. |
| `road` | **Not drawn as variants:** see "Road pieces" below. |
| `shore` | **Not drawn as variants:** see "Coast pieces" below. |
| `slums` | A muddy alley with broken boards, puddles and refuse. |
| `snow` | A snowfield with drifts and cool blue shadows. |
| `spiderweb` | Web-strung ground with pale silk strands and a cocoon. Eerie, not cartoonish. |
| `swamp` | Murky pools, reeds, lily pads and sodden earth. |
| `water` | Deep water, darker blue, with a few soft wave crests. |

## Road pieces: 16 tiles (`art/source/A2/map/terrain/road-<sides>.png`)

The game picks a road tile by which neighbors it connects to: a neighbor
that's also road, with an exit between the two rooms. So the road is drawn
as **pieces**, not variants (owner decision, 2026-10-07). Each piece is a
full, opaque **512×512** tile:

- grass verges, as on the approved A0 `land` tile, everywhere the road
  isn't;
- a rutted dirt road, as on the approved A0 road sample, running from
  the center to the **middle of each named edge**;
- the road the **same width** (about 30% of the tile) and in the same
  position on every edge, so any two pieces line up;
- no road touching an edge the piece doesn't name.

`<sides>` lists the connected edges in the order **n, e, s, w** (north is
the top edge):

| File | Shape |
|---|---|
| `road-none.png` | No connections: a small worn dirt patch or a milestone in grass |
| `road-n.png`, `road-e.png`, `road-s.png`, `road-w.png` | Dead end: the road comes in from that edge and fades out at the center |
| `road-ns.png`, `road-ew.png` | Straight |
| `road-ne.png`, `road-es.png`, `road-sw.png`, `road-nw.png` | Corner: a smooth curve between the two edges |
| `road-nes.png`, `road-esw.png`, `road-nsw.png`, `road-new.png` | T-junction: three edges, the fourth side grass |
| `road-nesw.png` | Crossroads |

Light comes from the top-left on every piece. Draw each piece in its own
orientation; don't rotate one image, which would turn the light.

**Check before delivering:** lay the pieces out as a small road network
(a loop with a T-junction, a crossroads and a dead end) on `land` tiles.
Put it in the review sheet. The road must run continuously with no steps
or width changes.

## Coast pieces: 16 tiles (`art/source/A2/map/terrain/shore-<sides>.png`)

A shore tile is drawn by **which of its four neighbors are water**, as the
roads are (owner decision, 2026-10-07). `<sides>` lists the edges that
face water, in the order **n, e, s, w**. Each piece is a full, opaque
**512×512** tile of pale beach sand, shallow water and a foam line.

**Edge rule.** Every piece follows it, so any two pieces line up:

- An edge that faces water is **open water** along its whole length, and
  matches the approved `water` tile where they meet.
- The **waterline** runs parallel to each water-facing edge, **35% of
  the tile in from it**. A strip of shallows lightens toward the sand, and
  a thin foam line marks the waterline.
- An edge that doesn't face water is **sand**, except within 35% of a
  water-facing corner, where the water from that side continues. So a
  straight coast of `shore-s` pieces joins side by side.
- Where two water-facing edges meet at a corner, the waterline curves
  round it widely (a radius of about 25–30% of the tile), not tightly.
- Between the fixed 35% crossing points, the waterline **wanders
  naturally**, up to about 8% in or out, with small coves and points, so
  coasts never look ruled.
- Foam and shallows are pixel art: a broken line of light pixel clusters
  for the foam, and 2–3 stepped bands of lighter water for the shallows.
  Never a soft glow or a smooth gradient.

| File | Shape |
|---|---|
| `shore-none.png` | No water beside it: a small sandy beach patch. Sand to every edge, a few shells and pebbles. |
| `shore-n.png`, `shore-e.png`, `shore-s.png`, `shore-w.png` | Straight coast: water along the named side |
| `shore-ns.png`, `shore-ew.png` | A sand strip between water on two opposite sides |
| `shore-ne.png`, `shore-es.png`, `shore-sw.png`, `shore-nw.png` | Outer corner: water on two adjacent sides, sand in the opposite corner |
| `shore-nes.png`, `shore-esw.png`, `shore-nsw.png`, `shore-new.png` | Headland: water on three sides, sand reaching in from the fourth |
| `shore-nesw.png` | A sandbar islet: water on all sides, a small sandy island in the middle |

Light comes from the top-left; don't rotate one image. The coast pieces
have no animation. The water tiles beside them already move.

**Check before delivering:** build a small coastline on `land` and `water`
tiles. Include a straight stretch, an outer corner, a headland, a strip
between two waters and an islet. Put it in the review sheet. The
waterline must join with no steps.

## Animated tiles: 4 frames (`art/source/A2/map/terrain/<biome>-anim-<1..4>.png`)

These replace the base tile while the biome animates. Each frame is a full
512×512 tile matching variant 1, with only the moving element changed. The
4 frames loop.

| Biome | Motion |
|---|---|
| `water` | Wave crests slide and shimmer. |
| `swamp` | A few bubbles rise and pop; reeds sway slightly. |
| `snow` | Light snowfall drifts down across the tile. |
| `desert` | Sand streams drift across the ripples. |

## State tiles (`art/source/A2/map/terrain/`)

| File | Contents |
|---|---|
| `fog.png` | 512×512 with **transparency**: soft dark drifting mist, densest in the middle and fading at the edges. It covers known but unvisited edges of the map. |
| `unknown.png` | 512×512 opaque: dim neutral stone, for a room whose biome isn't known. Quieter than any biome. |

The night mask is generated in code; don't draw it.

## Check before delivering

Tile every biome 3×3 using its variants in a mixed order. Then place each
biome next to `land`, `forest` and `water`. There should be no visible
seams, no cut objects and no variant that stands out.
