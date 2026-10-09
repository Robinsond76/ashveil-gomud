# E1 plan: high-resolution map figures (2026-10-09)

Design: [2026-10-09-e1-hires-map-figures-design.md](../designs/2026-10-09-e1-hires-map-figures-design.md).
Branch: `spike/hires-map-art` (worktree `.worktrees/spike-hires-map-art`).

1. **Fetch masters.** Copy the 101 approved map sheets from Drive into the
   git-ignored `art/source/<phase>/map/units/`:
   - A1: 15 sheets.
   - A8a: 19.
   - A8b: 24.
   - A9: 19.
   - A10: 24.

   Check sizes (2272×832) and the class-ID list against the generated
   `map/units` folders.
2. **Importer grid mode.** Make `import_sheet.py` grid-only, with a folder
   mode. It cuts the exact 256 px cells (32 px gutters), checks feet at y 239,
   halves each cell to 128 px (box filter, alpha kept binary), writes
   `idle.png` and `walk.png` as optimised PNGs, and records manifest entries
   (density 4, feet 120, walk 6 frames at 120 ms). Then retire the spike's
   fit-to-figure witch import.
3. **Generate.** Run `generate.py`, and confirm the 101 entries in
   `manifest.json` and the file sizes.
4. **Tint.** `Sprites.tinted` returns the untinted art when density > 1. Add
   a Node test.
5. **Pixel ratio.** Place figures and icons on whole device pixels in
   `drawFrame` and `drawIcon` (sizes stay unrounded; see design decision 5).
6. **Tests.** Extend `sprites_test.go`:
   - every imported map unit is density 4, has 128 px frames, feet baseline
     120, a 6-frame walk and is under 300 KB;
   - all 101 IDs are present.

   Extend `scripts/browser/map-check.mjs`:
   - a high-resolution figure standing and walking on the map harness;
   - frame changes;
   - no tint applied;
   - screenshots at ratios 1, 1.5 and 2.
7. **Help.** Add the looks line to `appearance.template`, and check it
   renders through `help`.
8. **Art order.** Write `docs/art/A11-look-masks.md` and add it to the art
   README, together with E1b.
9. **Gate.** Run `make generate`, `make validate`, `go test -race ./...`,
   `make js-lint` and `make js-test`, plus the map browser checks. An
   independent reviewer checks the full diff. Then record the outcome in
   `docs/PROJECT_STATUS.md`, merge to master and push.
