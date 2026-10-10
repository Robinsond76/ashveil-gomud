# E3: High-resolution battle screen (design, 2026-10-10)

The owner asked for E3 after E2 (`docs/art/README.md`, engineering row E3).
E3 puts the approved battle art (A6–A10) on the battle screen:
- 15 base classes and 16 backdrops (A6);
- 58 enemies and creatures (A7);
- 86 advanced and elite classes (A8a–A10).

It also makes the screen ready for action-pose art. The owner chose (2026-10-10) to order a pilot of attack, hurt and cast or shoot sheets after E3 (decision 6).

## Scope

- **In:** drawing the battle canvas at device pixels, a battle importer, the generated sprite set, the creation panel's look preview, the warhound's own sprite, tests, and help.
- **Out:**
  - action-pose art itself (a later art order);
  - skin and hair on the new battle figures, which waits for masks (A11's "Later" order, E1b);
  - drawing the A5 battle markers and interface icons, which no code draws yet.

## Decisions

1. **Device pixels.** The battle picture keeps its 320×180 coordinate space, so every drawing call is unchanged.
   - The canvas's backing store is the shown size times the pixel ratio (`pxScale`). `draw()` sets that transform once.
   - Code-drawn shapes, bars and text become sharp, and art is sampled at the screen's resolution.
   - `fit()` keeps its whole-number display scale (1×–4×) and resizes the backing store when the scale or ratio changes.
   - Pointer hit-testing already maps to picture coordinates through the shown size.
2. **Exact pixel art.** The battle masters are pixel art on a coarse grain. Each art pixel is a square block of master pixels, measured on the delivered files:

   | Kind | Master cell | Grain | Runtime frame | Density |
   |---|---|---|---|---|
   | Person (M) | 640 | 5 | 128 | 2 |
   | Beast (M) | 1024 | 8 | 128 | 2 |
   | Small creature (S) | 768 | 6 | 128 | 8/3 |
   | Large (L) | 1728 | 12 | 144 | 2 |
   | Boss (XL) | 1536 | 8 | 192 | 2 |
   | Backdrop | 2560×1440 | 4 | 640×360 | 2 |

   - The importer shrinks each frame by its grain, so every art pixel becomes exactly one runtime pixel. Nothing is lost, and the files are small (159 units and 16 backdrops come to about 7.6 MB).
   - All 175 delivered masters sit exactly on their grain. A frame whose grid is offset from its cell's corner is shifted onto it; art that isn't on a grain is refused.
   - This replaces the standards table's nominal 4× runtime size (256 px for M) with the art's real resolution.
3. **Crisp where it can be.** Where an art pixel lands on a whole number of device pixels, art is drawn with nearest-neighbour sampling, so it stays pixel-crisp. That is `pxScale × pose scale / density` being whole: for example density 2 at 2× or 4×. Otherwise (odd scales, 8/3 creatures, a squashed or shrunk figure) it's smoothed, so pixels don't land unevenly. 1x art is always crisp.
4. **Feet.** An imported sheet records its own feet row (`feet_baseline`, the lowest opaque row). The bottom of that row lands 1 picture pixel above the unit's ground point, exactly where the 1x sheets' feet (row frame − 2) land, so old and new figures stand on one line. A floating unit (echo bats) keeps its centre anchor.
5. **Skin and hair** (as the owner decided for the map: "masks, ship meanwhile").
   - The new battle figures can't be repainted by the palette swap, so they show the class's own colours. `help appearance` says so.
   - The creation panel previews looks on `battle/units/<class>/look.png`. That is a 1x copy of the drawn base-class figure, which the generator keeps for the 15 base classes, so the preview still changes colour as the player chooses.
   - Battle masks join the A11 follow-up.
6. **Action poses.**
   - The screen already plays `battle/units/<key>/<pose>.png` when listed: attack, hurt, cast, shoot, block, parry, dodge, windup, walk, down, victory. Otherwise it nudges the idle figure in code.
   - E3 makes those sheets density-aware like the idle. So a pilot order (the 15 base classes' attack, hurt and cast or shoot) drops in with only an importer table entry.
7. **The warhound.** A7 drew the Beast Tamer's warhound, which the screen had shown as the junkyard dog. It joins the roster (its 1x placeholder is still the dog), and `BEAST_SPRITES` points at it.

## State, persistence, multiplayer

There's no server or game state. The change is client art and client rendering. World time, persistence and the multiplayer invariant are untouched.

## Integration points

- `scripts/sprites/import_battle.py`: the A6–A10 battle importer. It shares `shrink` and `encode` with `import_art.py`.
- `scripts/sprites/roster.py`: the `warhound` unit. `generate.py` writes `look.png` for base classes.
- `window-battle.js`:
  - `pxScale` and `fit()`;
  - `artSmooth`;
  - `drawBackground` and `drawBody` (density, feet);
  - `BEAST_SPRITES`;
  - `state()` exposes `pxScale`, `backing`, `backdrop` and each unit's `art`.
- `window-creation.js` `figure`: the look sheet, and density.

## Player help

The `appearance` page now says chosen skin and hair show only in the creation panel, not yet on the map or battle figures. There's no new command or topic, and the tutorial has no hint about looks.

## Acceptance

- All 159 battle units and 16 backdrops are imported as exact pixel art, with the densities above and within budget. Each base class has a 1x `look.png`. `TestCommissionedBattleArtIsImported` checks this.
- The density-aware battle anchor and backdrop rules pass.
- `TestImportBattle` checks:
  - exact 5 px grain shrink, feet row and roster metadata;
  - an offset grain is shifted;
  - refusals: a gutter pixel, art off any grain, a wrong cell, an unknown unit.
- `scripts/browser/battle-art-check.mjs` at ratios 1, 1.5 and 2:
  - the backing store at device pixels;
  - the backdrop, base and promoted members, and M, S and L creatures drawing their imported art;
  - the warhound's own sheet.
- `battle-check.mjs` (its pixel probe scaled by `pxScale`), `class-art-check`, `doll-art-check`, `elite-art-check`, `creation-panel-check` and `battle-pane-check` pass.
- `help appearance` renders the new text.
- An independent reviewer checks the full diff before merge.
