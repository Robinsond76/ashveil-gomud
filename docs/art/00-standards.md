# Art standards (read with every phase)

You are making the art for **Ashveil**, a browser-played multiplayer
fantasy game with a tile map and a side-view tactical battle screen. Each
phase file lists exactly what to make. This file says how. Where they
disagree, the phase file wins for that row.

## 1. The look

**Mood: mature, grounded high fantasy.** The tradition is Tolkien: weathered
travelers in worn cloaks, practical mail and leather, old stone. The world
is dangerous. Magic is rare, dim and costly. Every design must be
**original**: never copy, trace or closely imitate a film, game or artist,
including the games named below.

**Rendering: detailed pixel art at high resolution.** Think of a top-end
late-1990s tactical RPG sprite (the grounded tone of Ogre Battle 64 or
Final Fantasy Tactics), drawn with more detail and delivered large.

- Visible, deliberate pixel clusters and crisp shapes.
- **Art-pixel grain:** draw as if on a grid **twice the logical size** in
  section 3. A map figure reads like pixel art about 56 art pixels tall, a
  map tile like 64×64 art pixels, and an icon like 32×32. Keep that grain
  the same across every file, so a tile, an icon and a figure look like
  they belong together.
- 2–4 shade steps per material, with clean ramps. Light comes from the
  **top-left**, always. Avoid pillow shading (light in the middle of every
  shape).
- **Outline:** a dark, selective outline about 1 art pixel thick on
  figures, overlays and icons. Use the darkest local color (deep brown,
  deep slate, deep plum), never pure black. Where light hits, the outline
  may lighten or drop. Terrain tiles have no outline.
- Not painterly, not 3D-rendered, not smooth vector, not photographic.
  There's no bloom, lens flare, chromatic aberration or film grain.

**Proportions:**

- Adult, realistic and sturdy.
- Map figures are about **4.5–5 heads tall**; battle figures about
  **6–7**.
- Faces are small and plain, often shadowed by a hood or helm.

**Never anime.** None of these:

- big heads or big shiny eyes;
- chibi or super-deformed bodies;
- spiky or brightly dyed hair;
- oversized weapons or armor;
- cute mascot creatures;
- sparkles, hearts, sweat drops or emote marks;
- candy colors;
- youthful, glamorous faces.

**Wear and weight.** Gear is used: dented helms, mud on hems, frayed
cloaks, patched leather, nicked blades. Armor follows history (mail,
gambesons, brigandines, simple plate). Weapons are real-world sized.

**Color.**

- Units are muted and natural: earth, iron, wool, leather and bone.
- Each lineage has one subdued accent (table below) that reads even when
  small.
- The map is rich but natural, like a hand-painted campaign map: deep
  greens, ochre, slate, river blues. Never saturated candy.
- Magic is pale light, ember glow, smoke, frost or grave-mist. It is dim
  beside the figures, never neon.
- Use `_datafiles/html/public/static/sprites/style/palette.png` as a
  **color reference**, not a hard limit.

| Lineage | Accent | Silhouette cues |
|---|---|---|
| Warrior | Iron grey, faded oxblood surcoat | Broad, mailed, nasal helm, round shield, sword |
| Rogue | Charcoal, dull plum | Lean, hood and mask cloth, two short blades |
| Ranger | Moss green, tan leather | Hooded cloak, longbow, quiver |
| Cleric | Undyed wool, tarnished brass | Mace, worn tabard over mail, iron holy symbol |
| Wizard | Slate blue-grey, pewter | Deep hood, long robe, tall staff with a dim stone |
| Witch | Dusky heather, ash-moss green | Layered tattered shawls, charms, crooked wand or staff |
| Halberdier | Tawny coat, steel | Kettle helm, breastplate, upright halberd |
| Samurai | Lacquer red-brown, dark iron | Banded cuirass, flared shoulder guards, horned helm, katana |
| Shaman | Hide brown, bone | Hide mantle, fur collar, bone mask, feathered totem staff |
| Doll Master | Plum, brass | Long coat, control bar with strings, a wooden doll |
| Gryphon Rider | Tawny feather, steel | Riding jack, winged helm, spear held high |
| Beast Tamer | Hide, wool | Hide vest, fur collar, coiled whip |
| Alchemist | Wool, leather apron, glass | Apron, bandolier of flasks, a flask in hand |
| Arbalist | Slate padded jack, steel | Steel cap, heavy crossbow across the chest |

**Alignment cue for advanced and elite classes:**

- Positive routes add tarnished gold or pale trim.
- Unrestricted routes keep the lineage accent.
- Negative routes add ash-black and ember red.

An elite is its advanced class made grander: a longer cape, a taller crest,
richer trim and a stronger alignment cue. It keeps the same silhouette
family.

**Violence:** physical, not gratuitous. A small spray of blood is the limit.
There's no gore, and nothing sexualised.

## 2. Hard rules for every file

1. **PNG, RGBA, sRGB.** Never JPEG or WebP: lossy files leave fringes.
2. **A true transparent background** (alpha 0). This doesn't apply to
   terrain tiles and battle backgrounds, which are opaque.
   - No white or colored matte.
   - No fake checkerboard painted into the pixels.
   - No glow or halo around the edge.
3. **No text of any kind** inside a deliverable: no letters, numbers,
   labels, captions, signatures, watermarks or frame numbers. Put labels
   only on the separate review sheet.
4. **No ground, floor, platform, frame or border** behind figures,
   overlays or icons.
   - Units have **no shadow**; the client draws one.
   - Landmarks and camp pieces may have a soft contact shadow of at most
     30% opacity, inside the object's footprint.
5. **One subject per cell.** Leave **at least 32 px of clear transparent
   space** between cells, so the importer can find each frame.
6. **Consistency inside a file.** The same design is in every frame and
   row: the same colors, gear, proportions and scale, and the same hand
   for the weapon in the front and back views. Only the pose changes.
7. **Consistency across files.** Match the A0 anchors and earlier approved
   phases in grain, outline, light, palette and figure scale.
8. **Readable when small.** Every figure, icon and overlay must still read
   when shrunk to its logical size in section 3 (a map figure 28 px tall,
   an icon 16×16). Check it yourself before delivering.
9. **Facing:** side views face **right**. The game mirrors them for left,
   so avoid asymmetric heraldry or lettering that would read wrong when
   flipped.

## 3. Sizes

The **logical size** is the space the game lays out. The **source size** is
the minimum you deliver; larger is better if you keep the proportions. The
importer scales sources down to the runtime size: 4× logical (2× for a few
very large files).

| Kind | Logical | Source minimum | Runtime |
|---|---|---|---|
| Map figure (one frame) | 32×32, figure ~28 tall, feet at 30 | figure ≥ 224 px tall | 128×128 |
| Terrain tile | 32×32 | 512×512 | 128×128 |
| Landmark overlay | 32×32 | 512×512 canvas, object ~80% | 128×128 |
| Camp piece (tent, camp) | 32×32 | 512×512 | 128×128 |
| Small camp piece and icon | 16×16 | 256×256 | 64×64 |
| Marker (12 px) | 12×12 | 192×192 | 48×48 |
| Small marker (8 px) | 8×8 | 128×128 | 32×32 |
| Battle figure, size S | 48×48 frame | figure ≥ 240 px tall | 192×192 |
| Battle figure, size M | 64×64, figure ~40–46 tall | figure ≥ 320 px tall | 256×256 |
| Battle figure, size L | 72×72 (covers two cells) | figure ≥ 480 px tall | 288×288 |
| Battle figure, size XL | 96×96 (boss) | figure ≥ 640 px tall | 384×384 |
| Battle background | 320×180 | 2560×1440 | 1280×720 |
| App icon | 512×512 | 1024×1024 | 512×512 |

**Size budgets after import** (the lead's importer enforces these; you
don't need to):

- a map unit sheet ≤ 300 KB;
- a terrain variant ≤ 60 KB;
- an icon ≤ 12 KB;
- a background ≤ 400 KB.

## 4. Sheet layouts

**Map figure sheet** (one file per class): **3 rows × 8 columns.**

- Row 1 faces the viewer (down). Row 2 faces away (up). Row 3 is the side
  view, facing right.
- Columns 1–2 are **idle**: standing, with a barely visible breath (the
  chest or cloak shifts by one art pixel).
- Columns 3–8 are a **6-frame walk**: right contact, down, passing, left
  contact, down, passing. The loop must be seamless.
- Every figure in a row stands on the same ground line, with the feet's
  lowest point level. The walk's "down" frames dip by at most 1 art pixel.
- The view is a slight 3/4 from above, the same as the A0 lineup. Every
  frame uses the same scale.

**Battle idle sheet**: **1 row × 4 columns**, facing right.

- The side-on stance has a little 3/4 toward the viewer.
- A 4-frame breathing loop: frames 1–4 rise and settle. Cloaks and
  smoke may move a little.
- Feet stay on one ground line.

**Animated icon or overlay**: **1 row × N columns** (N from the phase
table).

- The object stays in exactly the same place and size in every frame.
- Only the moving part changes (a flame, a pennant, a pulse).
- The loop is seamless.

**Terrain:** **one file per variant or frame**, each a full opaque square.
Don't put several tiles in one image.

## 5. Terrain specifics

- **View:** straight top-down, like a campaign map, with soft top-left
  light. Trees are canopy tops, buildings are roofs, ground is seen flat.
- **Full-bleed edges, no frame.** Paint each tile edge to edge in **its
  own biome's colors**. Its edges look like its middle.
  - Never add a border, frame, vignette, darkened edge or shared "neutral"
    edge color. A shared edge color turns the map into a grid of boxes.
  - Textures continue across edges: canopy, grass and water join up when
    tiles touch. Large single objects (a boulder, a building) stay
    inside, clear of the edges.
- **Seamless with itself:** four copies placed 2×2 show no seam.
  Neighboring biomes may change color softly at the join, but never with
  a hard dark line.
- **Open biomes stay open.** `water` is water to all four edges; banks
  belong to `shore`. Paths (`road`) meet the middle of all four edges, or
  run as texture edge to edge, so they connect from any side.
- **Variants** share the base color and light. They differ only in where
  the details sit, and none of them stands out.
- **Animated frames** are the base tile plus the moving element. Frames
  1–4 loop and differ only where things move: waves, bubbles, snow,
  drifting sand.
- Detail is calm enough that a figure standing on the tile still stands
  out.

## 6. Icon specifics

- Centered, filling about 80% of the canvas, with a bold silhouette.
- Two or three main colors plus the outline. It must read at 16×16 on an
  ordinary (1x) screen.
- One family look: every icon in a set shares the outline weight, light,
  framing and level of detail. There's no background disc or plate
  unless the row says so.
- **One main object.** At 16×16 an icon has room for one bold shape, plus
  at most one small accent (a glow, a drop, a spark). Composite ideas
  blur at that size: a helm with stars, or a cross with a scroll and a
  hand. Draw the row's named object, not a substitute.
- Symbols are visual, never letters, numbers or real-world logos.

## 7. Prompt skeletons

Adapt these for each row. Always attach the A0 anchors as image
references.

**Map figure sheet:**

> Original high-detail pixel-art sprite sheet for a grounded, mature
> fantasy tactical RPG. Late-1990s tactical-RPG feel, crisp pixel clusters,
> 3–4 tone shading lit from the top-left, a dark brown-black selective
> outline, muted natural colors. Character: **[description from the phase
> table]**, an adult of realistic sturdy proportions (about 5 heads), no
> anime features. Layout: exactly 3 rows × 8 columns on a fully
> transparent background, with wide empty gaps between cells. Row 1 faces
> the viewer, row 2 faces away and row 3 is the side view facing right.
> In each row, columns 1–2 are a standing idle and columns 3–8 a 6-frame
> walk cycle. The same character, colors and scale are in every cell, with
> the feet level on one ground line. No text, labels, ground, shadow or
> border.

**Battle idle:**

> Same style. **[description]** in a side-on battle stance facing right
> (a slight 3/4 toward the viewer), **[stance note]**. Exactly 1 row × 4
> columns, a subtle breathing idle loop, transparent background, wide gaps
> between frames, feet level, no text or shadow.

**Terrain tile:**

> Same style, seen straight top-down like a hand-painted campaign map,
> soft top-left light. A seamless square map tile of **[biome
> description]**. Calm, low-detail edges in the base color, nothing
> crossing an edge, no outline, fully opaque, no text.

**Icon:**

> Same style. A single bold game icon of **[subject]**, centered, filling
> 80% of the square, dark selective outline, 2–3 main colors, lit from the
> top-left, transparent background, readable at 16×16, no text, no
> background plate.

## 8. Your check before delivering

Go through every file:

- [ ] Correct file name and folder, PNG, true transparency (or opaque
      where required).
- [ ] The right number of rows and columns, in the order the phase lists.
- [ ] No text, matte, checkerboard, ground, border or stray specks.
- [ ] Terrain: a 3×3 grid of each tile, and a mixed row of biomes, show no
      frame or seam. Put both in the review sheet.
- [ ] Shrunk to the logical size, it still reads. Shown next to the A0
      anchors, it looks like the same game.
- [ ] Every frame is the same design and scale. Feet are level, and the
      animation loops.
- [ ] Light comes from the top-left, the outline is dark but not black, and
      the colors are muted.

## 9. Delivering

- Deliver to the **shared art folder**: the Google Drive folder
  [Ashveil Art Masters](https://drive.google.com/drive/folders/1BgQaAZfopzUVc4mKYthn5ygdMqSapAza), in the phase's subfolder (`A0`, `A1` and
  so on). Ask the owner for access. **Never commit
  art to the git repository.** In the phase files, `art/source/` means the
  root of that folder. Use the exact relative names the phase file gives:
  for example, `A1/map/units/warrior.png` goes in the folder `A1/map/units/`.
- Add **one review sheet**, `art/source/<phase>/_review.png`. It shows
  every file small, on mid-grey, labeled. This is the only place text is
  allowed.
- List anything you couldn't make or had to change in
  `art/source/<phase>/_notes.md`. Never silently substitute.
