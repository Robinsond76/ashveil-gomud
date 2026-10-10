# E3 plan: high-resolution battle screen (2026-10-10)

Design: [2026-10-10-e3-hires-battle-design.md](../designs/2026-10-10-e3-hires-battle-design.md).
Branch: `e3-hires-battle` (worktree `.worktrees/e3-hires-battle`).

1. **Fetch masters.** Copy A6 (15 units, 16 backdrops), A7 (58), A8a (19), A8b (24), A9 (19) and A10 (24) battle masters into `art/source/`. Measure cell sizes and grains.
2. **Client.**
   - Backing store at device pixels (`pxScale`, `fit`, `draw`).
   - `artSmooth`.
   - Density and the feet anchor in `drawBody` and `drawBackground`.
   - `state()` fields for the checks.
3. **Importer.** `import_battle.py` checks the grain (with offset shift), cuts the 4 frames, shrinks exactly and keeps the roster metadata. Add the `warhound` roster unit and point `BEAST_SPRITES` at it.
4. **Looks.** The generator writes `look.png` for the base classes; the creation panel uses it.
5. **Generate.** Run `make sprites`.
6. **Tests.**
   - Go:
     - `TestCommissionedBattleArtIsImported`;
     - `TestImportBattle`;
     - density-aware anchor and backdrop rules, with float densities.
   - Browser:
     - `battle-art-check.mjs`;
     - the `battle-check` pixel probe;
     - the existing art, creation and pane checks.
7. **Help and tutorial.** Update `help appearance` and its test. The tutorial is unchanged.
8. **Docs.** Update the art README (E3 row, sizes), A11's later battle masks, and the project status, including the owner's living-map ideas.
9. **Gate.**
   - Run `make generate`, `make validate`, `go test -race ./...`, `make js-lint`, `make js-test`, the battle and map browser checks, and the mobile check.
   - An independent reviewer checks the full diff.
   - Merge and push.
