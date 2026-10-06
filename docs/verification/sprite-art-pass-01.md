# Authored sprite pass 01

The owner approved the Direction D style proof, then requested production
integration. This first pass replaces nine sheets (60 frames): Warrior, Ranger,
and Witch map idle/walk animations in down/up/right directions, the tent and
allied tent, and four lit-fire frames. Characters and tents use native 32x32
frames; fire uses 16x16. Other classes, battle sprites, resources, terrain,
and the app icon remain outside this pass.

- [Native pixels at 1x/4x and on shipped terrain](sprite-art-pass-01.png)
- [Walking and fire animation preview](sprite-art-pass-01-walk.gif)
- [Authored source workflow](../../scripts/sprites/README.md#authored-artwork)

The original concept and generated source atlases are retained in
`scripts/sprites/authored/sources/`, with crop-coordinate records. The native
PNG sheets in `scripts/sprites/authored/map/` are the source of truth for future
pixel edits. The generator validates and publishes them at the existing paths;
no client changes or new animation metadata are required. This preserves the
64-color palette, binary transparency, original timings, and foot anchors.

The review images are assembled from the actual packed sprites. The terrain
panel uses shipped land and road tiles; it is a compositing preview, not a
live-client screenshot. The GIF uses manifest timings for both walking and
fire. Existing S0 concept scenes still show procedural figures.

Independent review found no blocking issues in the importer, native sheets,
or scoped artwork. Its one minor finding (the preview fire running at walk
timing) was fixed by using separate manifest-driven animation clocks.

Verification: native sprite layout, palette/alpha, map anchors, authored import
contracts, and byte-for-byte regeneration tests pass. Full repository check
results are recorded in `docs/PROJECT_STATUS.md`.
