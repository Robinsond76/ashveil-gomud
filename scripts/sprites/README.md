# Sprite generator

Procedural and authored pixel art for the visual client (roadmap: "Visual direction").
`make sprites` runs `generate.py`, which writes the PNG sheets to
`_datafiles/html/public/static/sprites/` plus `manifest.json` and
`battle/mapping.json`, and two review contact sheets in `docs/verification/`
(`40s1-contact-sheet.png` for S0/S1, `40s-s2-s3-contact-sheet.png` for S2/S3, `40s5-contact-sheet.png` for S5). Requires Pillow.

- `palette.py`: the 64-color master palette (outline, 19 materials x 3 shades, ember).
- `pixels.py`: canvas, shaded parts (top-left light), 1 px outline, sheet joiner.
- `figures.py`: humanoid rig; 6 base classes + `adventurer` fallback; map (32px, 5 heads) and battle (64px, 7 heads).
- `icons.py`: resource icons, markers, camp pieces from ASCII pixel maps.
- `scenes.py`: style frames, the 3x3 `formation_scene` (the 40f layout) and the app emblem.
- `kit.py`: shared helpers (limbs, polygons, recolor, dither, noise).
- `humanoids.py`, `beasts.py`, `giants.py`: S3 enemy idles (data-driven humanoids on the class rig, canines, spiders, crawlers, large creatures).
- `roster.py`: every battle unit with size class, family, in-game names; feet are aligned to row frame-2 here.
- `promoted.py`: S5, the 23 advanced and built elite classes: the lineage's base figure, ramps swapped to the class's materials plus accessories (cape, horns, hat, halo); map (3 views, idle and walk) and battle idle.
- `summoned.py`: S5, the Hierarch's Angel and the Demonologist's Demon (large units).
- `uiicons.py`: S3 status, role, morale and condition icons and the formation markers.
- `backgrounds.py`: the 16 battle backgrounds (320x180, quiet ground band y 100-176).
- `terrain.py`, `landmarks.py`: S2 tiles (3 variants, flat 2 px margin), animated overlays, fog/state tiles, landmark overlays.

Add a new sprite by adding a drawer and a `write_*` call, then run `make sprites`;
`go test ./scripts` checks sizes, the palette, hard edges, anchors and that the
committed PNGs equal the generator output.

## Authored artwork

Put native-size replacement PNG sheets under `scripts/sprites/authored/`, using
the same relative path as the public sprite (for example
`authored/map/units/warrior/idle.png`). `Writer.png` loads these through
`authored.py`; regeneration preserves the authored artwork and keeps existing
manifest metadata. Missing replacements retain their procedural drawer.
Source PNGs must match the generated sheet dimensions, use the master palette
with binary alpha, and respect map-unit foot baselines. Invalid exports stop
generation. Run `python3 -m unittest discover -s scripts/sprites -p '*_test.py'`
for the import contract checks, and `go test ./scripts` for complete sprite checks.

The first authored pass covers Warrior, Ranger, Witch (idle/walk, down/up/right),
tent, allied tent, and lit fire. `authored/sources/` preserves the approved
concept, generated source atlases, and crop coordinates for provenance. These
large source atlases are references, not runtime assets. Native PNG sheets in
`authored/map/` are the editable source of truth; edit those for pixel cleanup.
They were exported with nearest-neighbor sampling, no palette dithering,
binary alpha, and bottom-center alignment. No image-generation service or
ImageMagick dependency is needed to rebuild the shipped sprites.

After `make sprites`, run `python3 scripts/sprites/preview_authored.py` to render
the actual native sheets at 1x/4x, on shipped terrain, and as a walking GIF in
`docs/verification/sprite-art-pass-01*`. These previews show production pixels,
not the concept board. Existing S0 concept scenes and battle/promoted sprites
remain procedural and are outside this first pass.
