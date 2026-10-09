# E1: High-resolution map figures (design, 2026-10-09)

The owner asked to start E1 after art phases A0–A10 were approved
(`docs/art/README.md`). E1 puts the commissioned map figures on the world
map: 101 sheets covering every built class (15 base, 43 advanced, 19 built
elites, 24 planned elites). Terrain, icons and landmarks are E2; the battle
screen is E3.

## Scope

- **In:** importing the 101 approved map-unit masters as density-4 sheets,
  generating them into the client's sprite set and manifest, the map client
  changes they need, tests, and the player-help note below.
- **Out:** battle art (E3), terrain and icons (E2), and the skin and hair
  recolour of the new art (E1b, below).

## Prior art (shipped on the spike branch, landing here)

- A manifest entry may declare `"density": N`. The map draws such a sheet at
  the same on-map size as 1x art, smoothing it only when density > 1
  (`window-map.js` `drawFrame`, `artSmoothing`).
- The map canvas renders at `devicePixelRatio`.
- `scripts/sprites/generate.py` copies `scripts/sprites/imported/` over its
  drawn output; `sprites_test.go` reads density from the manifest.

## Decisions

1. **Runtime size.** Masters use 256 px cells with 32 px gutters, feet on
   y 239 and a 224 px body (standards section 4). The importer cuts each cell
   exactly from that grid and halves it to a 128 px frame: density 4, feet
   baseline 120, the same anchor as 1x (30 of 32). There's no per-figure
   rescaling. The masters are already at one scale, and the old fit-to-figure
   cutting was what shrank tall figures in A1.
2. **Frames.** Idle is 2 frames at 500 ms, as 1x. Walk is the masters'
   6 frames at 120 ms; 1x had 4. The client already reads the frame count
   from the manifest.
3. **Which art.** The 101 class IDs that have masters replace their 1x sheets.
   Creatures (`hound`, `stone-golem`) have no map master and keep 1x art.
   The fallback chain (class, lineage, adventurer) is unchanged.
4. **Player looks (owner decision, 2026-10-09: "masks, ship meanwhile").**
   `SpriteTint` repaints six exact palette colours. The new art has thousands
   of colours, so the swap would do nothing or, worse, recolour a stray
   matching pixel. E1 skips tinting for density > 1 sheets, so chosen skin
   and hair don't show on the new map figures for now. Battle sprites and the
   creation panel still use 1x art and keep showing looks. Art phase A11
   (`docs/art/A11-look-masks.md`) orders a skin mask and a hair mask per
   class sheet. A follow-up engineering step, E1b, will recolour the new art
   from those masks, keeping the art's own shading.
5. **Pixel ratio.** The canvas already renders at the screen's ratio. 1x art
   on a fractional ratio (1.25, 1.5) is drawn at a whole number of device
   pixels per art pixel, so its pixels stay even. That's the README's
   "even out 1x pixel art" item. High-density art is smoothed and needs
   nothing.
6. **Size.** Each imported sheet is saved as an optimised PNG, and the test
   holds every map unit sheet under the standards' 300 KB budget. Quantising
   and WebP are E2's job.

## State, persistence, multiplayer

There's no server or game state. The change is client art and a client
rendering rule. World time, persistence and the multiplayer invariant are
untouched.

## Integration points

- `scripts/sprites/import_sheet.py`: a grid import mode for approved masters,
  run per sheet or over a whole folder.
- `scripts/sprites/imported/`: the committed runtime sheets and manifest
  entries. `generate.py` copies them in.
- `_datafiles/html/public/static/sprites/`: the generated sheets and
  `manifest.json`.
- `sprites.js` `tinted`: returns the art untouched when density > 1.
- `window-map.js`: snaps 1x art to whole device pixels on a fractional ratio.

## Player help

The `appearance` help page gains one line: the new map figures show your
class, but not yet your chosen skin and hair (the battle screen and the
creation panel do). There's no new command, and the help index is unchanged.
The tutorial has no hint about looks, so nothing there goes stale.

## Acceptance

- All 101 class IDs have density-4 `idle.png` (2 frames) and `walk.png`
  (6 frames) with 128 px frames, feet baseline 120 and source "imported".
  Each sheet is under 300 KB, and generation is deterministic.
- `go test ./scripts -run Sprite` passes with density-aware rules.
- `sprite-tint` JS test: a density-4 sheet comes back untinted.
- `scripts/browser/map-check.mjs` passes, and a new check draws a
  high-resolution figure walking (6 frames) and standing on the real map
  harness. Screenshots are taken at ratios 1, 1.5 and 2.
- The `appearance` help page renders through `help`.
- An independent reviewer checks the full diff before merge.
