# A2b: Coast inside corners

Read [00-standards.md](00-standards.md) and [A2-terrain-tiles.md](A2-terrain-tiles.md)
(coast pieces) first. Attach the approved `shore-*` pieces and `water-1.png`.

The 16 coast pieces handle water on a tile's own sides. Lakes on the map
are the empty middles of rings of shore rooms, so a coast often bends
**inward**: a shore tile's two neighbours along a lake corner are shore,
and the water sits only diagonally, touching one corner. Today that tile
is sand right to its corner, and the lake shows a square notch.

## What to deliver (`art/source/A2/map/terrain/shore-corner-<corner>.png`)

Four **512×512 overlays with transparency**, one per corner: `ne`, `es`,
`sw`, `nw` (the corner that touches the diagonal water).

- Each overlay is transparent except a **quarter-disc of water** in its
  named corner. That water continues the approved coast exactly: open
  water at the corner, then the stepped shallows bands and the broken foam
  line, then sand.
- The waterline meets the tile's two edges at **35% from the corner**,
  where the neighbouring pieces' waterlines arrive (A2 edge rule). Between
  those two points it curves round the corner, about 25–30% radius.
- The water matches `water-1.png` and the coast pieces where they touch.
  The overlay is drawn over any shore piece, so outside its quarter it
  must be fully transparent.
- Light from the top-left; draw each corner in its own orientation (don't
  rotate one image).

## Check before delivering

Build a 4×4 ring of `shore` pieces (`shore-s`, `shore-w`, `shore-n`,
`shore-e` on the sides, `shore-none` in the corners, each corner with its
overlay) around a 2×2 block of `water-1` tiles. Put it in the review sheet.
The waterline must run round the lake with no steps or notches.
