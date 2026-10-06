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
| **Masters:** the agent's large originals (1–3 MB each, 200–400 MB in all) | The shared art folder, a cloud drive or bucket the owner chooses. Its layout mirrors `art/source/`: `A0/`, `A1/map/units/warrior.png` and so on. | **No.** Git would keep every version forever, and every clone would download them all. |
| **Local staging:** `art/source/` in a checkout | A copy of the phase being imported. It's listed in `.gitignore`. | No |
| **Runtime files:** the imported, compressed sheets in `_datafiles/html/public/static/sprites/`, and their copies in `scripts/sprites/imported/` | The repository | **Yes.** They're small (about 20–40 MB for the whole program) and are what players download. |

Back the shared folder up and keep old versions there if you want them. To
re-import at a different size later, import again from the masters.

Phases run in order. **A0 is an approval gate:** nothing else starts until
the owner signs off on the style anchors.

## Phases

| Phase | File | Contents | Files to deliver | Needs engineering |
|---|---|---|---|---|
| **A0** | [A0-style-anchors.md](A0-style-anchors.md) | Style anchors: lineup, sample tile, sample icons, mock scenes | 8 | none |
| **A1** | [A1-base-class-map-units.md](A1-base-class-map-units.md) | Map sprites for the 15 base figures (6 classic lineages, 8 neutral lineages, the fallback) | 15 | E1 |
| **A2** | [A2-terrain-tiles.md](A2-terrain-tiles.md) | 19 biomes × 3 variants, 5 animated biomes, fog and unknown tiles | 79 | E2 |
| **A3** | [A3-landmarks.md](A3-landmarks.md) | 27 landmark overlays | 27 | E2 |
| **A4** | [A4-map-icons-markers-camp.md](A4-map-icons-markers-camp.md) | Resource icons, map markers, camp pieces, app icons | 30 | E2 |
| **A5** | [A5-ui-icons.md](A5-ui-icons.md) | Status, role, morale and condition icons; battle markers | 52 | E2 |
| **A6** | [A6-battle-base.md](A6-battle-base.md) | 16 battle backgrounds and battle idles for the 15 base figures | 31 | E3 |
| **A7** | [A7-enemies-and-creatures.md](A7-enemies-and-creatures.md) | Battle idles for every enemy, fallback, summon and creature | 58 | E3 |
| **A8a** | [A8a-advanced-classic.md](A8a-advanced-classic.md) | 19 advanced classes of the 6 classic lineages, map and battle | 38 | E1, E3 |
| **A8b** | [A8b-advanced-neutral.md](A8b-advanced-neutral.md) | 24 advanced classes of the 8 neutral lineages, map and battle | 48 | E1, E3 |
| **A9** | [A9-elite-built.md](A9-elite-built.md) | The 19 elite classes in the game today, map and battle | 38 | E1, E3 |
| **A10** | [A10-elite-planned.md](A10-elite-planned.md) | The 24 planned elites, each made when its class is built | 48 | E1, E3 |

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
| **E2** | Importer kinds for terrain (3 variant files, 4 animation files, opaque, edge check), overlays (landmarks, camp) and icons (one shared box per animated row, so frames don't jitter). Per-kind density. Compression (quantized PNG; WebP once the client checks support), with the size budgets in the standards. Generate the `night-mask` in code. | A2–A5 |
| **E3** | High-resolution battle screen. Today it draws a 320×180 canvas scaled by CSS. It must draw at device pixels, with density on units and backgrounds. | A6, A7, battle half of A8–A10 |
| **E4** | Rewrite the sprite specification's craft and technical sections to point here. Contact sheets come from the imported art. | all, as phases land |

## Owner decisions to confirm

1. **Proportions.** The sample sheets are sturdier than the old "5 heads
   on the map, 7 in battle" rule. The standards now say about **4.5–5
   heads on the map** and **6–7 in battle**: adult and never chibi. Confirm
   this at A0.
2. **Source storage (decided 2026-10-06).** Masters stay outside git in
   the shared art folder, and only the compressed runtime files are
   committed (see "Where the files live"). The owner still needs to name
   the folder.
3. **Battle animations.** The battle screen animates with code effects
   today, so only idle sheets are drawn (S4 in the old spec). Full attack
   and hurt sheets can be a later phase once A7 is approved.
