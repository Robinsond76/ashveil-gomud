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
| `road` | A dirt road with ruts down the middle and grass verges. The road runs through the tile so it continues into neighbors. |
| `shore` | Pale sand meeting shallow water along a diagonal, with a little foam. |
| `slums` | A muddy alley with broken boards, puddles and refuse. |
| `snow` | A snowfield with drifts and cool blue shadows. |
| `spiderweb` | Web-strung ground with pale silk strands and a cocoon. Eerie, not cartoonish. |
| `swamp` | Murky pools, reeds, lily pads and sodden earth. |
| `water` | Deep water, darker blue, with a few soft wave crests. |

**The road needs extra care:** a road tile may touch a road above, below,
left or right. Run it across the middle in **both directions**, so it
reads as a crossroads, or keep the road texture edge to edge so it joins
from any side. Note which you chose in `_notes.md`.

## Animated tiles: 4 frames (`art/source/A2/map/terrain/<biome>-anim-<1..4>.png`)

These replace the base tile while the biome animates. Each frame is a full
512×512 tile matching variant 1, with only the moving element changed. The
4 frames loop.

| Biome | Motion |
|---|---|
| `water` | Wave crests slide and shimmer. |
| `shore` | The water's edge laps in and out; foam forms and fades. |
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
