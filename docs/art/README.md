# Ashveil high-resolution art program

Drafted 2026-10-06. The owner wants higher-quality character art, map tiles
and icons, made by an image-generating art agent. This folder holds that
agent's work orders. It **supersedes the craft and technical rules** of the
[sprite specification](../designs/2026-10-05-sprite-specification.md): 1x
hand-placed pixels, the 64-color palette and hard edges no longer apply.
The specification's **rosters, logical sizes and file paths still apply**,
because the client lays the map and battle screen out in those units.

The proof of concept is on branch `spike/hires-map-art`. The map draws a
`"density": 4` sheet in the same space as a 1x one. The map canvas
renders at the screen's pixel density. `scripts/sprites/import_sheet.py`
turns a delivered sheet into the client's sheet layout.

## How to use these orders

1. Give the art agent **[00-standards.md](00-standards.md)** and **one
   phase file**. Every phase file says what to render. The standards file
   says how. The agent needs both, every time.
2. Also attach the **style anchors** approved in A0 (and, once they exist,
   the approved sheets of earlier phases). Consistency across hundreds of
   files comes from those references, not from the words alone.
3. The agent delivers its source files to the **shared art folder**,
   outside git (see "Where the files live" below). It uses the folder
   names the phase lists, plus one review contact sheet.
4. The lead copies that phase into a local `art/source/` (ignored by git)
   and imports it. Then the lead checks it in the running client and
   records the outcome in `docs/PROJECT_STATUS.md`. The next phase starts
   after that.

## Where the files live

| Files | Where | In git? |
|---|---|---|
| **Masters:** the agent's large originals (1–3 MB each, 200–400 MB in all) | The shared art folder: the Google Drive folder [**Ashveil Art Masters**](https://drive.google.com/drive/folders/1BgQaAZfopzUVc4mKYthn5ygdMqSapAza). It has one subfolder per phase (`A0` to `A10`), and inside each the layout mirrors `art/source/`: `A1/map/units/warrior.png` and so on. The folder is private; the owner shares it with the art agent. | **No.** Git would keep every version forever, and every clone would download them all. |
| **Local staging:** `art/source/` in a checkout | A copy of the phase being imported. It's listed in `.gitignore`. | No |
| **Runtime files:** the imported, compressed sheets in `_datafiles/html/public/static/sprites/`, and their copies in `scripts/sprites/imported/` | The repository | **Yes.** They're what players download. E1's 101 map units take about 12 MB of stored blobs (git keeps one copy of the identical pair, so a checkout holds about 24 MB); E2's 164 terrain, landmark and icon files about 1.7 MB more; E3's 159 battle units and 16 backdrops, stored as exact pixel art at their 5–12 px grain, about 3.8 MB. |

Drive keeps earlier versions of a replaced file, so a regenerated sheet
can be uploaded over the old one. To re-import at a different size later,
import again from the masters.

**Getting a phase to the lead:** the owner's claude.ai account has the
Google Drive connector, so a lead session can find the phase's files and
download them into `art/source/<phase>/`. If a session doesn't have the
connector, the owner downloads the phase folder from Drive as a zip and
attaches it instead.

Phases run in order. **A0 is an approval gate:** nothing else starts until
the owner signs off on the style anchors.

## Phases

| Phase | File | Contents | Files to deliver | Needs engineering |
|---|---|---|---|---|
| **A0** | [A0-style-anchors.md](A0-style-anchors.md) | Style anchors: lineup, sample tile, sample icons, mock scenes | 8 | none |
| **A1** | [A1-base-class-map-units.md](A1-base-class-map-units.md) | Map sprites for the 15 base figures (6 classic lineages, 8 neutral lineages, the fallback) | 15 | E1 |
| **A2** | [A2-terrain-tiles.md](A2-terrain-tiles.md) | 17 biomes × 3 variants, 16 road pieces, 16 coast pieces, 4 animated biomes, fog and unknown tiles | 101 | E2 |
| **A2b** | [A2b-coast-inner-corners.md](A2b-coast-inner-corners.md) | 4 coast inside-corner overlays, so a lake's corners have no notch | 4 | E2b |
| **A3** | [A3-landmarks.md](A3-landmarks.md) | 27 landmark overlays | 27 | E2 |
| **A4** | [A4-map-icons-markers-camp.md](A4-map-icons-markers-camp.md) | Resource icons, map markers, camp pieces, app icons | 30 | E2 |
| **A5** | [A5-ui-icons.md](A5-ui-icons.md) | Status, role, morale and condition icons; battle markers | 52 | E2 |
| **A6** | [A6-battle-base.md](A6-battle-base.md) | 16 battle backgrounds and battle idles for the 15 base figures | 31 | E3 |
| **A7** | [A7-enemies-and-creatures.md](A7-enemies-and-creatures.md) | Battle idles for every enemy, fallback, summon and creature | 58 | E3 |
| **A8a** | [A8a-advanced-classic.md](A8a-advanced-classic.md) | 19 advanced classes of the 6 classic lineages, map and battle | 38 | E1, E3 |
| **A8b** | [A8b-advanced-neutral.md](A8b-advanced-neutral.md) | 24 advanced classes of the 8 neutral lineages, map and battle | 48 | E1, E3 |
| **A9** | [A9-elite-built.md](A9-elite-built.md) | The 19 elite classes in the game today, map and battle | 38 | E1, E3 |
| **A10** | [A10-elite-planned.md](A10-elite-planned.md) | The 24 planned elites, each made when its class is built | 48 | E1, E3 |
| **A11** | [A11-look-masks.md](A11-look-masks.md) | Skin and hair masks for the 101 map-unit masters, then the 101 class battle idles, so players' chosen looks recolour the new art | 404 | E1b |
| **A12** | [A12-battle-actions-pilot.md](A12-battle-actions-pilot.md) | Pilot battle action poses for the 15 base classes: attack, hurt, and shoot or cast | 38 | E3c |

Base figures come first, then the whole map (tiles, landmarks, icons),
then the battle screen, and the advanced and elite classes come last.
Until a class has new art, the client shows its lineage's art, so the game
never has gaps.

## Engineering prerequisites (lead, not the art agent)

The art agent can work ahead of these. Its files wait in the shared art
folder until the matching step is merged.

| Step | Work | Unblocks |
|---|---|---|
| **E1** | Land the spike as a phase. Density on map units, a high-DPI map canvas, `import_sheet.py` reading the A1 layout (2 idle and 6 walk columns per row). Make the tests' rules depend on density. Even out 1x pixel art on fractional pixel ratios. | A1, map half of A8–A10 |
| **E1b** | Recolour the new map and battle figures from the A11 masks: within each mask, map the art's own shading onto a ramp built from the player's chosen skin or hair colour (`SpriteTint`), keeping light and shadow. Until then high-density sheets are shown untinted (`SpriteTint.applies`). | A11 |
| **E2** | Importer kinds for terrain (3 variant files, 4 animation files, opaque, edge check), overlays (landmarks, camp) and icons (one shared box per animated row, so frames don't jitter). Per-kind density. Compression (quantized PNG; WebP once the client checks support), with the size budgets in the standards. Generate the `night-mask` in code. Road auto-tiling: the map picks `road-<sides>` from the neighbors that are road and joined by an exit. Coast auto-tiling: `shore-<sides>` from the grid neighbors whose biome is `water`, exits or not. | A2–A5 |
| **E2b** | Draw the A2b inside-corner overlay on a shore piece in each corner whose diagonal neighbour is water (a lake cell or a water room) while the two sides beside it are not. | A2b |
| **E3** | High-resolution battle screen: the 320×180 picture drawn at device pixels, with the battle units and backdrops imported as exact pixel art at their measured grain (density 2; 8/3 for small creatures). | A6, A7, battle half of A8–A10 |
| **E3c** | Import the A12 pose sheets: `import_battle.py` learns pose files (frame counts from the order, the idle's grain); the battle screen already plays `battle/units/<id>/<pose>.png` when listed, standing it on the idle's feet row. | A12 |
| **E4** | Rewrite the sprite specification's craft and technical sections to point here. Contact sheets come from the imported art. | all, as phases land |

## Owner decisions to confirm

1. **Proportions.** The sample sheets are sturdier than the old "5 heads
   on the map, 7 in battle" rule. The standards now say about **4.5–5
   heads on the map** and **6–7 in battle**: adult and never chibi. Confirm
   this at A0.
2. **Source storage (decided 2026-10-06).** Masters stay outside git in
   the Google Drive folder [Ashveil Art Masters](https://drive.google.com/drive/folders/1BgQaAZfopzUVc4mKYthn5ygdMqSapAza), and only the
   compressed runtime files are committed (see "Where the files live").
3. **Battle animations.** The battle screen animates with code effects
   today, so only idle sheets are drawn (S4 in the old spec). Full attack
   and hurt sheets can be a later phase once A7 is approved.
