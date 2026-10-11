# E1b: chosen looks on the high-resolution figures (design, 2026-10-11)

Players choose a skin tone and a hair colour when they make a character
(Phase 72a). The 1x art showed them by swapping six exact palette colours.
The approved high-resolution art (E1 map figures, E3 battle figures) has
thousands of colours, so E1 and E3 show it untinted (`SpriteTint.applies`
is false for density > 1) and the creation panel kept a 1x battle preview
(`look.png`). Art phase A11 delivered a skin mask and a hair mask for every
one of the 101 class sheets, map and battle (owner decision 2026-10-09:
"masks, ship meanwhile"). E1b uses them.

## Behaviour

- A player's figure (and a party member's, wherever their look is known)
  shows their chosen skin and hair on the map and on the battle screen, at
  full resolution, keeping the art's own light and shade.
- The creation panel's preview shows the class's high-resolution battle idle
  in the chosen look once its masks exist; the 1x `look.png` remains the
  fallback.
- Figures without a look (companions, creatures, a player who chose nothing)
  are drawn as before. A class whose masks are both empty (a full helm) has
  no mask file and is drawn as before.
- The A12 action poses (E3c) have no masks yet, so during a blow, a flinch
  or a chant the figure shows its drawn skin. Masks for the 38 pose sheets
  are a small follow-up order (A11c); the client needs nothing more for them.

## Art pipeline

- Masters: `art/source/A11/map/units/<id>.{skin,hair}.png` (the map master's
  2272×832 layout, 4 px grain) and
  `art/source/A11/battle/units/<id>/idle.{skin,hair}.png` (the battle idle's
  layout, 5 px grain).
- `import_sheet.py` (map) and `import_battle.py` (battle) cut each mask
  exactly like its sheet (the same cells, the same shrink) and write one
  runtime mask per sheet beside it: `map/units/<id>/idle.look.png`,
  `walk.look.png` and `battle/units/<id>/idle.look.png`. A mask pixel is red
  for skin, green for hair, transparent elsewhere (a three-colour palette
  PNG, a few KB). The importer refuses a mask that doesn't match its master's
  size, marks a pixel the sheet leaves transparent, or overlaps skin and hair.
- The sheet's manifest entry gains `look_mask` (the mask's path); the mask
  has its own `look-mask` entry (frame, size, density as its sheet).

## Recolour (client)

`SpriteTint.recolour(data, maskData, look)` (pure, Node-tested): for each
material with a chosen colour, take the masked pixels' luminance; their
median is the reference. Each masked pixel becomes the chosen colour scaled
by its luminance over the reference: darker pixels darken toward black,
lighter ones lift toward white (at most 60% of the way), so shading, rim
light and outlines inside the mask survive. Pixels outside the masks are
untouched.

`SpriteTint.applies(info)` becomes true for a high-density sheet that has a
`look_mask`. `Sprites.tinted` and the battle screen's `tintedArt` load the
mask with the sheet and draw the sheet untinted until both are loaded;
`canvasFor` takes the mask image and caches per sheet and look, as before.

## Acceptance criteria

- Every class with a non-empty map or battle mask ships its runtime masks;
  each matches its sheet's size and marks only opaque pixels.
- On the map and the battle screen a player with a chosen look is drawn with
  it (masked pixels change, the rest don't); without a look, nothing
  changes.
- The creation panel shows the high-resolution idle in the chosen look.
- Help: `help appearance` says looks show on the new figures (and that
  action poses keep their drawn skin for now); the creation lesson's hint
  stays accurate.
- Tests: Node (`recolour`: masked pixels follow the chosen colour and keep
  their order of light and dark, unmasked pixels unchanged, a missing
  material untouched); Go (masks imported for every class with marks,
  sized like their sheets, only on opaque pixels; importer refusals);
  browser (`map-check`, `battle-art-check`, `creation-panel-check`: a
  chosen look changes the masked pixels and only them).

## State, persistence, multiplayer

Client art and rendering only. No server, world-time or persistence change.
