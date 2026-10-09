# A0: Style anchors (approval gate)

Read [00-standards.md](00-standards.md) first.

These files fix the look that every later phase copies. **Stop after A0
and wait for the owner's approval.** If the owner asks for changes, redo
A0 until it's approved. Later phases attach these files as references, so
time spent here pays off everywhere.

Reference the sample sheets the owner liked: the witch, warrior and ranger
walk sheets and the camp pieces. They're in the owner's chat of
2026-10-06. Ask the lead for copies if you don't have them. Match their
quality and grain, but make every design your own.

## Deliverables (`art/source/A0/`)

| File | Size | Contents |
|---|---|---|
| `lineup-map.png` | 6 figures, each ≥ 224 px tall, in a row on transparent | The 6 classic lineages facing the viewer, standing: warrior, rogue, ranger, cleric, wizard, witch. Same scale, feet level. This fixes map proportions. |
| `lineup-battle.png` | 6 figures, each ≥ 320 px tall, in a row on transparent | The same 6 in battle stance, facing right. This fixes battle proportions. |
| `tile-sample.png` | 4 tiles, each 512×512, opaque, side by side with 32 px gaps | `land`, `forest`, `road` and `water` tiles, one each. They must meet without a seam when placed edge to edge. |
| `icon-sample.png` | 8 icons, each 256×256, in a row on transparent | water, forage, firewood (resources); bleeding, stunned (statuses); fighter, healer (roles); hold (morale). |
| `landmark-sample.png` | 2 overlays, each 512×512 canvas | `inn` and `gate`, as described in A3. |
| `camp-sample.png` | `tent` at 512×512, `fire-lit` frame 1 at 256×256 | As described in A4. |
| `scene-map.png` | 1920×1080, opaque | A mock of the map at zoom 2. It's a grid of square tiles (forest, land, a road, a river of water and shore), with an inn landmark and a camp on a tile. The warrior and witch stand on two tiles, one at the side view, and there are two resource icons in a tile corner. Make the tiles 192 px. Seen top-down; figures stand upright on the tiles. |
| `scene-battle.png` | 2560×1440, opaque | A mock battle in a forest clearing, seen side-on. Three company figures (warrior, cleric, wizard) stand on the left in a loose formation and face right. A forest ogre and two goblins stand on the right and face left. Keep the lower band (y 800–1400) clear ground where units stand. No UI, no text. |

## Approval checklist (owner)

- [ ] Mature, grounded, original. Not anime and not cute.
- [ ] Proportions: map about 4.5–5 heads, battle 6–7. Adjust here if wanted.
- [ ] Each lineage is identifiable by silhouette in the map lineup when
      shrunk to 28 px tall.
- [ ] Tiles meet without seams, and figures stand out on them.
- [ ] Icons read at 16×16.
- [ ] The scenes look like one game.
