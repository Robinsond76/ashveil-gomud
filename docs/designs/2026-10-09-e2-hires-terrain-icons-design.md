# E2: High-resolution terrain, landmarks and icons (design, 2026-10-09)

The owner asked for E2 after E1 shipped (`docs/art/README.md`, engineering
row E2). E2 puts the approved A2–A5 art into the game: terrain tiles, road
and coast pieces, landmarks, camp pieces, map markers, resource icons, app
icons, interface icons and battle markers. E1 brought the class figures;
the battle screen is E3.

## Scope

- **In:**
  - an importer for the A2–A5 masters;
  - the generated sprite set and manifest;
  - road and coast auto-tiling on the map;
  - the replace-style terrain animation;
  - tests;
  - the `worldmap` help update.
- **Out:**
  - the battle screen drawing the new battle markers and interface icons.
    No client code draws `ui/*` or `battle/ui/*` today; E2 ships the files,
    and E3 draws them.
  - WebP, until the client checks support (decision 6).

## Prior art

- A manifest entry may declare `"density": N`. The map's `drawFrame`,
  `drawIcon` and `drawTileArt` draw such art at the same on-map size as 1x
  and smooth it (E1).
- `generate.py` copies `scripts/sprites/imported/` over its drawn output.
- The night mask is already drawn in code (`terrain.night_mask`). The
  client shades night in code and never loads it, so E2 leaves it alone.

## Decisions

1. **One density.** Every A2–A5 master is drawn 4 times larger than the
   standards' runtime size (section 3: a 512 px terrain master gives a
   128 px tile, 256 px icons give 64 px, a 192 px marker gives 48 px). The
   importer shrinks every master by 4 into a density-4 sheet. The frame
   sizes are the old 1x frames times 4, so every caller keeps its layout.
   App icons are the exception: they are written at their exact pixel sizes
   (512, 192, 512 maskable, 32), because browsers read them directly.
2. **Downscaling.** A 4×4 box average with premultiplied alpha. On art
   drawn on a 4 px grain, it maps each art pixel to one output pixel.
   Alpha stays soft (the fog's mist, landmarks' contact shadows); the map
   smooths density art anyway.
3. **Terrain sheets.**
   - A biome's three variant files become one 3-frame sheet
     `map/terrain/<biome>.png`, as before; a room's variant is still
     `room id mod 3`.
   - The four animated biomes (water, swamp, snow, desert) get a 4-frame
     `<biome>-anim.png` marked `"replace": true`. A2's frames are whole
     tiles (variant 1 plus the motion), not overlays.
     - Only rooms showing variant 1 (room id mod 3 is 0) play it, each at
       its own phase (offset by room id), so a lake doesn't pulse in step.
     - The other rooms keep their still variants 2 and 3, so a region is
       not one tile repeated (standards section 5, no wallpaper).
     - Every water, swamp, snow and desert tile and frame has identical
       edge pixels (measured), so mixing frames and variants leaves no
       seam.
     - With reduced motion, every room shows its variant.
   - `fog` keeps its transparency; `unknown` is opaque.
   - Shore no longer animates (A2: the coast pieces don't). The generated
     1x `shore.png` and `shore-anim.png` stay, used only if the pieces were
     missing.
4. **Road and coast pieces.** These are 16 files each,
   `map/terrain/road-<sides>.png` and `shore-<sides>.png`; `<sides>` lists
   the joined edges in n, e, s, w order, or is `none`. `MapTiles.piece`
   (`static/js/map-tiles.js`, pure and tested in Node) picks the piece:
   - **A road** joins a grid neighbour that is also road and has an exit
     between the two rooms.
     - A secret passage doesn't join, so the road never gives one away.
     - An exit that leaves the drawn map also joins, so a road runs on
       rather than dead-ending at the edge of what you have seen. That is
       an exit into the fog of an unvisited room, or into a visited room
       of another zone (a road that crosses a zone border).
     - Two road rooms with a wall between them don't join; the dark wall
       edge stays.
   - **A coast** faces each grid neighbour whose ground is `water`, exits
     or not (owner decision, 2026-10-07). That includes a lake cell
     (decision 8).
   - A piece that is still loading draws the classic colour square that
     frame, as other terrain does. A missing piece falls back to the
     biome's own sheet.
5. **Overlays and icons.**
   - Landmarks and the tent and camp pieces are 128 px frames.
   - Fire, smoke, embers, resting, the inn bed and resource icons are
     64 px.
   - Markers are 128, 64, 48 or 32 px, by their 1x sizes.
   - Animated rows keep their frame counts and timings.
   - The importer cuts each animated row into equal cells of the master's
     canvas size. One shared box per row means a frame never shifts
     against the others.
6. **Compression.** Sheets are saved as optimised PNGs. The standards'
   budgets apply per frame: 60 KB per terrain frame, 12 KB per icon frame.
   The test enforces them on every imported file. A file over budget is
   quantised to a 256-colour palette before saving; one that is still over
   fails the import. WebP waits for a client support check (out of scope).
7. **Import checks.** Each master must:
   - have the expected size (frames × canvas);
   - be opaque where the brief says opaque (terrain, the unknown tile, app
     icons);
   - be a known file name.

   The terrain **edge check** compares a 4 px band round each biome
   variant with the whole tile. If the band is much darker, the tile has a
   frame or vignette (standards section 5), and the import fails.

8. **Lakes from shape** (owner decision, 2026-10-09).
   - The shipped world has no `water` rooms. A lake is the empty middle of
     a ring of shore rooms: Frost Lake's 98, Alderbrook's pond, Marrowmere
     Fen. Without a rule, every coast would face nothing and draw as sand.
   - `MapTiles.lakes` finds the empty cells the drawn rooms fully enclose
     (4-connected, not reachable from outside the map's bounding box). It
     keeps a region only if every room bordering it is shore and no exit
     leads into it; a cell an exit leads into holds a room not yet seen.
   - The map draws those cells as `water` tiles, with variants, animation
     and night shading, and coasts face them.
   - It runs when a zone is replayed, so a lake appears once its ring has
     been walked.
   - Known gap: the 16 pieces have no inside corner. Where a lake's corner
     meets a shore room diagonally, that room is sand to its corner, and
     the water shows a square notch. Four inside-corner overlays are
     ordered as art phase A2b (`docs/art/A2b-coast-inner-corners.md`).

## State, persistence, multiplayer

There's no server or game state. The change is client art plus a client
rule for picking road and coast pieces from what the player's map already
holds. World time, persistence and the multiplayer invariant are untouched,
and the map still shows only what the game tells the player.

## Integration points

- `scripts/sprites/import_art.py`: the A2–A5 importer.
- `scripts/sprites/imported/` and `imported.json`: the committed runtime
  files. `generate.py` copies them in.
- `_datafiles/html/public/static/sprites/`: the generated set and
  `manifest.json`.
- `static/js/map-tiles.js`: loaded by `webclient-pure.html`.
- `window-map.js`:
  - `onwardExits`, `tileLook` and `drawTerrain` pick and draw pieces and
    replace-style animation;
  - `findLakes` (run by `replayZone`) and the lake pass draw lakes and
    shade them at night;
  - `state().drawn` records `pieces`, `replaced` and `lakes` for the
    browser check.

## Player help

The `worldmap` help page's Terrain section changes:

- roads join the roads they lead to and run on into the fog or another
  region;
- coasts follow the water, and a walked ring of shore fills with its
  lake;
- water, swamp, snow and desert move in places, and the shore no longer
  does.

There's no new command or topic. The tutorial's map hint ("the map window
draws terrain tiles") stays true.

## Acceptance

- Every expected A2–A5 file is imported:
  - 17 biome sheets, 4 animation sheets, 16 road and 16 coast pieces, fog
    and unknown;
  - 27 landmarks, 10 camp pieces, 7 markers and 9 resource icons;
  - 4 app icons;
  - 45 interface icons and 7 battle markers.
- Each is density 4 (app icons at their exact sizes), with 1x frames × 4
  and within budget. A Go test checks them all.
- `TestImportArt` builds synthetic masters and checks:
  - an animated row cuts into the right frames;
  - a wrong size, a transparent terrain tile, a framed tile, a missing or
    stray master and an `--only` that matches nothing are refused;
  - an over-budget file is quantised, and one still over is refused.
- `map-tiles` Node tests: road joins (exit, wall, secret, onward), coast
  sides, other ground, and lakes (ring, diagonal ring, open ring,
  non-shore border, exit into the middle).
- `map-check.mjs` checks that:
  - the harness's road and shore rooms draw their pieces;
  - an animated biome plays its replace sheet only on variant-1 rooms;
  - a road runs on into another zone;
  - a missing piece falls back to the biome;
  - a shore ring's middle draws as a lake its coasts face, shaded at
    night, and a ring with an exit into its middle has no lake.
- `help worldmap` renders the new terrain text.
- An independent reviewer checks the full diff before merge.
