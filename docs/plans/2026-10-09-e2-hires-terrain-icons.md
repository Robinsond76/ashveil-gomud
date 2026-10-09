# E2 plan: high-resolution terrain, landmarks and icons (2026-10-09)

Design: [2026-10-09-e2-hires-terrain-icons-design.md](../designs/2026-10-09-e2-hires-terrain-icons-design.md).
Branch: `e2-hires-terrain` (worktree `.worktrees/e2-hires-terrain`).

1. **Fetch masters.** Copy the approved A2 (101), A3 (27), A4 (30) and A5
   (52) files from Drive into the git-ignored `art/source/<phase>/`, and
   check their sizes and transparency.
2. **Importer.** Add `scripts/sprites/import_art.py`, with a table of every
   runtime file and its masters. It cuts animated rows (with or without
   32 px gutters), shrinks by 4 with premultiplied alpha, and writes app
   icons at exact sizes. It checks sizes, opacity and terrain edges, and
   enforces the budgets, quantising when a file is over. It writes into
   `scripts/sprites/imported/`.
3. **Generate.** Run `make sprites` and confirm the 164 entries.
4. **Client.**
   - Add `static/js/map-tiles.js` (`MapTiles.sides`, `MapTiles.piece`).
     Load it in `webclient-pure.html` and the map harness.
   - In `window-map.js`, `tileLook` and `drawTerrain` draw road and coast
     pieces and replace-style animation, with a fallback to the biome sheet.
5. **Tests.**
   - Node: `scripts/js/map-tiles.test.mjs`.
   - Go:
     - `TestCommissionedTerrainAndIconsAreImported` (all 160 density-4
       files and 4 app icons);
     - `TestImportArt` (synthetic masters);
     - the density-aware terrain and palette rules.
   - Browser: `map-check.mjs` E2 section (pieces on the mixed harness, a
     road network with a wall, a secret passage and fog, a missing piece's
     fallback, water's replace animation).
6. **Help and tutorial.** Update the `worldmap` Terrain section and
   `TestWorldMapHelp`. The tutorial's map hint stays true.
7. **Docs.** Update the sprites README importer section, the art README's
   E2 row and size note, and `docs/PROJECT_STATUS.md`.
8. **Gate.**
   - Run `make generate`, `make validate`, `go test -race ./...`,
     `make js-lint` and `make js-test`, plus `map-check.mjs` and
     `mobile-check.mjs`.
   - An independent reviewer checks the full diff.
   - Merge to master and push.
