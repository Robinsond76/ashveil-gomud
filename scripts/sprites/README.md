# Sprite generator

Code-drawn pixel art for the visual client (roadmap: "Visual direction").
`make sprites` runs `generate.py`, which writes the PNG sheets to
`_datafiles/html/public/static/sprites/` plus `manifest.json`, and a review
contact sheet to `docs/verification/40s1-contact-sheet.png`. Requires Pillow.

- `palette.py`: the 64-color master palette (outline, 19 materials x 3 shades, ember).
- `pixels.py`: canvas, shaded parts (top-left light), 1 px outline, sheet joiner.
- `figures.py`: humanoid rig; 6 base classes + `adventurer` fallback; map (32px, 5 heads) and battle (64px, 7 heads).
- `icons.py`: resource icons, markers, camp pieces from ASCII pixel maps.
- `scenes.py`: style frames, mock enemies (throwaway until S3) and the app emblem.

Add a new sprite by adding a drawer and a `write_*` call, then run `make sprites`;
`go test ./scripts` checks sizes, the palette, hard edges, anchors and that the
committed PNGs equal the generator output. Any file can be replaced by commissioned art with no code change,
as long as its path, frame size and manifest entry stay the same.
