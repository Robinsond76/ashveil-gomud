# Sprite specification

Drafted 2026-10-05 for the [visual client milestone](2026-10-05-visual-client-milestone-design.md).
It lists every sprite the milestone needs, grouped into **art sets S0–S7**.
The sets are ordered so that the earliest are usable soonest. A generating
agent can work from this document alone.

Counts and rosters reflect the world and designs as of 2026-10-05. When new
classes, enemies, spells or statuses ship, add their rows here. The
[maintenance rules](#maintenance-rules) say how.

## Art sets at a glance

| Set | Used by phase | Contents | Approx. files |
|---|---|---|---|
| **S0** | Gate before all others | Master palette, style frames, proportion sheets | 6 |
| **S1** | 40a, 40b, 40i | Resource icons, map markers, camp, map sprites for the 6 base classes plus a fallback, app icons | ~40 |
| **S2** | 40c, 40d | Terrain tiles for every biome, fog, landmark overlays | ~55 |
| **S3** | 40f | Battle backgrounds, idle battle sprites for every class and enemy, battle UI icons | ~120 |
| **S4** | 40g | All remaining battle animations, hit effects, projectiles, spell effects, status overlays, damage digits | ~450 |
| **S5** | 40h | The 18 advanced (level-10) classes, map and battle | ~235 |
| **S6** | Later | Elite classes, expanded catalogue, creature recruits, race variants | ~1,100 |
| **S7** | Later, optional | Portraits, item icons, other-player markers | open |

**Priority order: S0 → S1 → S2 → S3 → S4 → S5 → S6.** S0 must be approved
by the owner before mass production. That keeps hundreds of files in one
consistent style.

---

## Global standards

These apply to every sprite unless a row says otherwise.

### Art direction (owner, 2026-10-05)

Ashveil is a **mature high fantasy**: a dangerous, unforgiving world. The
sprites must reflect that. **This section overrides any row description
that seems to say otherwise.**

- **Grounded, Tolkien-style high fantasy, in the Lord of the Rings
  tradition.** Think of weathered travelers in worn cloaks, practical mail
  and leather, and old stone. Magic is rare, smoky and costly. Monsters are
  genuinely threatening. Draw on that tradition's mood and realism, but
  **never copy designs from the films, games or any artist's work**. Every
  design is original.
- **Ogre Battle 64 is the sprite reference the owner likes.** Follow its
  grounded proportions, readable side-on battle poses, sober palette and
  the way a 3x3 formation reads at small size. Do not copy, trace or
  closely imitate its sprites.
- **Not anime in any way.** Specifically, no:
  - oversized heads or large, shiny eyes;
  - chibi or "super-deformed" proportions;
  - spiky, gravity-defying or brightly dyed hair;
  - oversized weapons or armor;
  - cute mascot monsters;
  - sparkles, hearts, sweat-drops or other cartoon emotes;
  - candy-bright outfits;
  - youthful, glamorous faces.
- **Realistic proportions:**
  - battle sprites about **7 heads tall** (an adult figure about 44–48 px in
    a 64 px frame);
  - map sprites as close to that as 32 px allows, about **5 heads tall**,
    still adult and never chibi.

  Faces are small and plain, often shadowed by hoods or helms.
- **Wear and weight.** Gear is used: dented helms, mud-stained hems, frayed
  cloaks, patched leather, nicked blades. Armor follows historical shapes:
  mail, gambesons, brigandines, simple plate. Weapons are real-world sized.
- **Danger.** Enemies are ugly, feral, gaunt or uncanny, never comic. Even
  small creatures look like they can hurt you.
- **Violence is physical, not gratuitous.** Hits may show a small spray of
  blood and the fallen lie still. There is no dismemberment and no gore
  close-ups.
- **Restrained magic.** Pale light, ember glow, smoke, frost, grave-mist.
  Effects are brief and dim beside the figures, never neon or glittery.
- **Mood and color:**
  - units are muted and naturalistic: earth, iron, wool, leather, bone;
  - each class has one **subdued** accent so it reads at 1x;
  - the **map is rich but natural**, like a hand-painted campaign map: deep
    greens, ochres, slate and river blues, not saturated candy colors.
- **Victory and emotion are sober.** Survivors lower their weapons and
  catch their breath. No cheering poses.

### Craft rules

- **Original pixel art**, hand-placed, in the spirit of late-90s tactical
  RPGs.
- **Light comes from the top-left.** Two to three shade steps per material.
  Avoid pillow shading.
- **Outline:** a 1 px selective outline on units, icons and markers, in the
  darkest palette color (never pure `#000`). Terrain tiles have no outline.
- **No text, letters or numbers** inside any sprite. The client draws
  numbers, using the S4 digit font.
- **No anti-aliasing against transparency.** Edges are hard. Internal
  anti-aliasing uses palette colors only.
- **Silhouette first.** Every class and enemy must be identifiable by
  silhouette alone at 1x.

### Palette

- One **master palette of at most 67 colors** (64 in S0, plus the `hair` ramp from Phase 72a, which the web client repaints with `skin` for a player's looks) is delivered in S0. Every
  sprite uses only master palette colors.
- **Class accent colors** (proposed; final in S0):

  | Lineage | Accent | Notes |
  |---|---|---|
  | Warrior | Iron grey with a faded oxblood surcoat | Broad, mailed, helm with nasal guard |
  | Rogue | Charcoal with dull plum | Hood and mask cloth, short blades, lean |
  | Ranger | Weathered moss green with tan leather | Hooded cloak, longbow, quiver |
  | Cleric | Undyed wool with tarnished brass | Mace, worn tabard, iron holy symbol |
  | Wizard | Slate blue-grey with pewter | Deep travel hood, long beard or gaunt face, tall staff |
  | Witch | Dusky heather with ash-moss green | Tattered layered shawls, bone and herb charms, crooked wand |

- **Alignment cue for promoted classes:** positive routes add tarnished
  gold or pale trim, unrestricted routes keep the lineage accent, and negative routes add
  ash-black and ember red. For example, Knight is gold-trimmed and Reaver is
  ember-trimmed.

### Technical

- **Format:** PNG-32 with an alpha channel, sRGB, transparent background
  (except terrain and backgrounds). Strip metadata.
- **Authored at 1x.** The client scales by whole numbers (2x, 3x, 4x) with
  nearest-neighbor filtering. Never deliver pre-scaled art.
- **Sheet layout:** **one file per sprite per animation**. Frames run left to
  right with no padding or gaps, and every frame is exactly the listed frame
  size. A file with directions has one **row per direction**, in the order
  listed. File width = frame width × frames; height = frame height × rows.
- **Anchor:**
  - Units: **bottom-center**, with the feet's baseline at `frameHeight − 2`.
  - Icons and markers: centered.
  - Tiles: full frame.
- **Facing:**
  - Map units face **down, up and side** (side faces right). The client
    mirrors the side row for left.
  - Battle units face **right**. The client mirrors enemies to face left.
  - Never bake mirroring into the art.
- **Location:** `_datafiles/html/public/static/sprites/` with the subfolders
  given in each table. File names are lowercase, hyphen-separated and use
  the IDs below.
- **Timing (client defaults, for reference):**
  - map idle: 500 ms per frame;
  - map walk: 120 ms;
  - battle idle: 180 ms;
  - battle actions: 90 ms;
  - effects: 70 ms;
  - camp fire: 150 ms.

### Size classes

| Class | Frame | Use |
|---|---|---|
| Map tile | 32×32 | Terrain, landmarks, map units |
| Icon | 16×16 | Resources, statuses, roles, morale, conditions, markers |
| Small icon | 8×8 | Path dots, the acting arrow |
| Battle S | 48×48 | Small creatures: rats, bats, hatchlings, floofs, faeries |
| Battle M | 64×64 | Humanoids and medium beasts. The figure is about 40 px tall, with room to swing a weapon |
| Battle L | 72×72 | Large: ogre, ent, crocodile, lich, golems (drawn at 96 and shrunk 3/4, so they cover two formation cells) |
| Battle XL | 96×96 | Bosses: spider queen; future drakes (drawn at 128 and shrunk 3/4) |
| Background | 320×180 | Battle backgrounds (scaled 4x to 1280×720) |
| Effect | 32×32 or 64×64 | Listed per effect |

---

## S0 — Style foundation (approval gate)

Deliver these first. **Stop for owner approval** before starting S1.

| ID | File | Size | Description |
|---|---|---|---|
| palette | `sprites/style/palette.png` + `palette.gpl` | 64 swatches, 8×8 | Master palette, named swatches in the `.gpl` |
| style-map | `sprites/style/style-map.png` | 320×180 | Must demonstrate the art direction. A mock map scene: forest, road, river, village and camp tiles, the Warrior and Witch map sprites, two resource icons |
| style-battle | `sprites/style/style-battle.png` | 320×180 | A mock battle: forest background, three company units (Warrior, Cleric, Wizard) on the left against a forest ogre and two goblins on the right |
| proportions-map | `sprites/style/proportions-map.png` | 6 × 32×32 | All 6 base classes facing down, side by side |
| proportions-battle | `sprites/style/proportions-battle.png` | 6 × 64×64 | All 6 base classes in battle idle, side by side |
| icon-sample | `sprites/style/icon-sample.png` | 8 × 16×16 | Sample icons: water, forage, bleeding, stunned, fighter, healer, yield, dark |

---

## S1 — Map basics (phases 40a, 40b, 40i)

### Resource icons — `sprites/map/resources/` (16×16, 1 frame)

Drawn in a tile corner. Every resource is wanted (owner, 2026-10-05): 40a
shows water, forage and shelter, and 40a2 adds the gathering resources.

| ID | Description |
|---|---|
| water | Fresh water: a spring or stream droplet over ripples |
| forage | Forage: a sprig of berries and leaves |
| herbs | Herbs: a bundle of medicinal leaves tied with twine |
| firewood | Firewood: two crossed logs |
| shelter | Shelter: a rock overhang or lean-to silhouette |
| fishing | Fishing: a fish arcing over a wave |
| game | Game: a deer track or hoofprint pair |
| unknown | Unsurveyed: a faint, abstract compass rose (no letters) |
| depleted | Overlay drawn over any gathering icon (herbs, firewood, fishing, game) whose room is picked clean for now (40a2): a dim, empty-basket mark |

### Map markers — `sprites/map/markers/`

| ID | Size | Frames | Description |
|---|---|---|---|
| here-ring | 32×32 | 4 | A pulsing ring under the current room's unit |
| company-badge | 12×12 | 1 | A shield-shaped badge background. The client draws the member count on it |
| ally-banner | 16×16 | 2 | A small fluttering pennant marking an allied company |
| walk-target | 16×16 | 2 | A destination flag for click-to-walk |
| walk-dot | 8×8 | 1 | A path breadcrumb dot |
| exit-up | 8×8 | 1 | An up-stairs chevron for rooms with an up exit |
| exit-down | 8×8 | 1 | A down-stairs chevron for rooms with a down exit |

### Camp — `sprites/map/camp/`

The camp is drawn on its room's tile. Only your own and allied camps are
shown.

| ID | Size | Frames | Description |
|---|---|---|---|
| tent | 32×32 | 1 | A small canvas tent, pegged, with a bedroll in front |
| tent-ally | 32×32 | 1 | The same tent with a small ally pennant (`ally-banner` colors) |
| camp-rough | 32×32 | 1 | A camp without a tent (40a3): two bedrolls on the ground around a stone fire ring, a pack leaning on a log |
| camp-rough-ally | 32×32 | 1 | The same, with a small ally pennant |
| embers | 16×16 | 3 | A banked fire glowing low after a rest (40a3), looping |
| fire-unlit | 16×16 | 1 | A ring of stones with unlit wood |
| fire-lit | 16×16 | 4 | A crackling campfire loop |
| smoke | 16×16 | 4 | A thin smoke curl loop, drawn above a lit fire |
| resting | 16×16 | 3 | A drifting sleep sigil loop. **No letters**: use a crescent moon with fading motes |
| inn-rest | 16×16 | 1 | A bed with a lantern, for resting at an inn |

### Map unit sprites — `sprites/map/units/<id>/`

The frame is 32×32. Each unit has **two files**:

| File | Rows (in order) | Frames | Total size |
|---|---|---|---|
| `idle.png` | down, up, side | 2 | 64×96 |
| `walk.png` | down, up, side | 4 | 128×96 |

Proportions are adult and realistic, about 5 heads tall (see the art
direction). The figure fills about 14×28 px of the frame, with its feet
anchored per the global standards.

| ID | Description |
|---|---|
| warrior | Broad, mail hauberk and faded surcoat, nasal helm, sword and round shield |
| rogue | Lean, hooded, dark leathers, mask cloth, two short blades |
| ranger | Hooded travel-stained cloak, longbow on back, quiver, long knife |
| cleric | Worn wool tabard over mail, mace, iron holy symbol on a cord |
| wizard | Long weathered robe and deep hood, tall gnarled staff with a dim stone |
| witch | Layered tattered shawls and hood, bone and herb charms, crooked ash wand |
| adventurer | Fallback for any class without art: plain traveler's cloak, pack, walking staff, neutral brown |

### App icons — `sprites/app/` (phase 40i)

| ID | Size | Description |
|---|---|---|
| icon-512 | 512×512 | App icon: an ember-lit campfire before a dark ridge (original emblem; no text) |
| icon-192 | 192×192 | The same icon, adjusted for small size (not just downscaled) |
| icon-maskable-512 | 512×512 | Maskable version: the emblem within the central 80% safe zone, background full-bleed |
| favicon-32 | 32×32 | Pixel version of the emblem |

---

## S2 — Terrain and landmarks (phases 40c, 40d)

### Terrain tiles — `sprites/map/terrain/` (32×32)

One file per biome: **3 variants side by side** (96×32). The client picks a
variant from the room ID. Animated biomes have a second file with 4 frames.
Tiles must sit beside any other biome without seams, so keep a 2 px
low-detail margin on each edge.

| Biome ID | Variants file | Animated file | Description |
|---|---|---|---|
| cave | `cave.png` | — | Dark stone floor, scattered rubble, faint mineral glints |
| city | `city.png` | — | Cobbled street, worn stones, an occasional drain |
| cliffs | `cliffs.png` | — | Sheer rock ledge with cracks, a drop shadow on the lower edge |
| default | `default.png` | — | Neutral packed earth (fallback) |
| desert | `desert.png` | `desert-anim.png` (4, drifting sand) | Wind-rippled sand, a few pebbles |
| dungeon | `dungeon.png` | — | Fitted flagstones, moss in the joints |
| farmland | `farmland.png` | — | Tilled rows, crops in three growth looks |
| forest | `forest.png` | — | Tree canopy tops, dark understory gaps |
| fort | `fort.png` | — | Packed parade ground with a timber or stone edge |
| house | `house.png` | — | Wooden floorboards with a rug corner |
| land | `land.png` | — | Open grassland with tufts and small flowers |
| mountains | `mountains.png` | — | Rocky slope with a snowcap highlight |
| road | `road.png` | — | Dirt road center with grass verges |
| shore | `shore.png` | `shore-anim.png` (4, lapping edge) | Sand meeting shallow water diagonally |
| slums | `slums.png` | — | Muddy alley, broken boards, refuse |
| snow | `snow.png` | `snow-anim.png` (4, light snowfall) | Snowfield with drifts and blue shadows |
| spiderweb | `spiderweb.png` | — | Web-strung ground, cocoons, pale silk |
| swamp | `swamp.png` | `swamp-anim.png` (4, bubbles) | Murky pools, reeds, lily pads |
| water | `water.png` | `water-anim.png` (4, waves) | Deep water, darker blue with wave crests |

Any biome added later needs a row here, following the same pattern.

### Fog and state tiles — `sprites/map/terrain/`

| ID | Size | Frames | Description |
|---|---|---|---|
| fog | 32×32 | 1 | Unexplored edge: soft dark mist (drawn only on known-but-unvisited edges) |
| unknown | 32×32 | 1 | A tile whose biome is unknown: dim neutral stone |
| night-mask | 32×32 | 1 | A soft vignette alpha mask the client multiplies for dark rooms (greyscale alpha only) |

### Landmark overlays — `sprites/map/landmarks/` (32×32, 1 frame, transparent background)

These replace the current letter symbols and `maplegend` entries. 40c maps
each existing symbol or legend to one of these IDs.

| ID | Replaces (legend or symbol) | Description |
|---|---|---|
| inn | Inn | A timber inn with a hanging sign (no lettering) |
| bank | Bank, `$` | A stone strongbox house with a coin emblem |
| shop | General shops | A market stall with an awning |
| smithy | Armorer and weaponsmith rooms | An anvil and forge glow |
| herbalist | Herbalist and brewer rooms | A hut with drying herbs and bottles |
| trainer | Trainer | Crossed practice swords over a target |
| temple | Temple, Chapel | A small chapel with a bell |
| shaman | Shaman | A hide tent with a bone totem |
| hermit | Hermit | A crooked hut with a lantern |
| gate | East-Gate, West-Gate | A gatehouse arch with an open portcullis |
| wall | Wall, `♜` | A crenellated wall segment |
| bridge | Bridge, `B` | A wooden plank bridge |
| keep | Keep | A square stone tower with a banner |
| throne | Throneroom | A throne under a canopy |
| townsquare | Townsquare | A fountain on a paved square |
| village | Village | A cluster of three cottages |
| caravan | Caravan | A covered wagon |
| lake-house | Lake House | A stilt house over water |
| cave-mouth | Cave, Entrance into caves | A dark cave opening in rock |
| dungeon-stair | Entrance or Exit into dungeons and catacombs | Stone stairs descending into darkness |
| obelisk | Obelisk | A tall rune-carved stone |
| pond | Pond | A small reed-fringed pond |
| rocks | Rocks | A boulder pile |
| desert-ruin | Desert | A half-buried column |
| alts | Alts (alt-character room) | An open book on a lectern |
| landmark | Any unique `★` or unmapped symbol | A generic standing-stone marker |
| boss-lair | Rooms flagged as a boss lair (future) | A cracked skull on a stake (shown only once visited) |

---

## S3 — Static battle screen (phase 40f)

### Battle backgrounds — `sprites/battle/backgrounds/` (320×180, 1 frame, opaque)

The layout follows OB64's side view. The company stands on the left and
the enemy on the right.

- **Keep the ground band clear:** y 100–176, x 16–304. Nothing tall may
  stand there.
- Depth cues (horizon and scenery) sit above y 100.
- The client darkens backgrounds for darkness and night. Don't paint night
  versions.

| ID | Used for biomes | Description |
|---|---|---|
| forest | forest | Tall dark trunks, a mossy clearing, shafts of light |
| deep-web | spiderweb | Web-choked trees, cocoons hanging, a pale green haze |
| plains | land, farmland, default | Rolling grass, distant hills, scattered stones |
| road | road | A rutted road through the field, a milestone, distant trees |
| city | city, fort | A street between timber and stone buildings, barrels and crates |
| slums | slums | A narrow muddy alley, sagging shacks, hanging laundry |
| interior | house | A tavern or house interior: beams, hearth, tables pushed aside |
| catacombs | dungeon | Burial niches, bone piles, candles in skull sconces |
| cave | cave | A cavern with stalactites, glowing crystal veins, a dark pool |
| snowfield | snow | A white expanse, frost-heavy pines, a grey sky |
| ice-keep | Stormwatchers Keep (zone override) | Ice-sheathed stone halls, frozen pillars |
| shore | shore, water | A lakeshore with reeds, an upturned boat, mist over the water |
| swamp | swamp | Black water, twisted roots, hanging moss |
| desert | desert | Dunes, a sun-bleached ruin, heat shimmer |
| highlands | mountains, cliffs | A rocky pass, a cliff face behind, scree |
| training-yard | Tutorial zone | A fenced yard with straw targets and weapon racks |

### Battle unit sprites — idle only

Each unit has a folder `sprites/battle/units/<id>/`. S3 delivers
**`idle.png`** only: **4 frames**, facing right, size per its size class.
S4 adds the remaining files for each unit. Generating whole units in one
pass is fine and helps consistency. S3 only requires `idle.png` to exist.

**Classes (Battle M, 64×64).** Same designs as the map sprites, drawn
larger and side-on. These cover the player and companions; recruits use
their class sprite.

| ID | Notes |
|---|---|
| warrior | Shield forward, sword low, weight braced |
| rogue | Crouched, blades reversed |
| ranger | Bow held, arrow nocked low |
| cleric | Mace and shield, symbol glinting |
| wizard | Staff planted, a faint glow at the stone |
| witch | Wand low, hood shading the face, a thin curl of grave-mist |
| adventurer | Fallback humanoid |

**Fallback silhouettes.** For any enemy without art:

| ID | Size | Description |
|---|---|---|
| unknown-humanoid | M | A hooded generic foe, muted |
| unknown-beast | M | A generic four-legged beast silhouette |
| unknown-large | L | A hulking generic shape |

**Enemies.** Grouped into families. A **variant** uses the same frames,
recolored, in its own folder. The roster is every combatant in the default
world as of 2026-10-05.

| ID | Size | Family / variant of | In-game names | Description |
|---|---|---|---|---|
| rat | S | rodent | rat | Mangy grey rat |
| rat-big | S | variant of rat | big rat | Larger, darker, scarred |
| wolf-timber | M | canine | timber wolf | Grey-brown wolf, lean |
| wolf-snow | M | variant of wolf-timber | snow wolf | White and blue-grey coat |
| dog-junkyard | M | canine | junkyard dog | Mangy mastiff, spiked collar |
| spider-hatchling | S | spider | spider hatchling, baby spider | Pale small spider |
| spider-large | M | spider | large spider | Hairy brown spider, raised forelegs |
| spider-warrior | M | variant of spider-large | spider warrior | Darker, with chitin plates and red markings |
| spider-queen | XL | spider (boss) | spider queen | Bloated abdomen, crown-like spines, egg sacs |
| skeleton | M | undead | skeleton | Bare skeleton with a notched sword |
| bone-warden | M | undead | bone warden | Skeleton in rusted plate, tower shield |
| bonecrafter | M | undead | bonecrafter | Hunched skeleton with bone tools and a fetish |
| lich | L | undead (boss) | lich | Robed skeletal mage, crown, green soulfire |
| acolyte-dark | M | cultist | dark acolyte, dark acolyte trainer | Black-robed cultist, ritual dagger |
| grave-chanter | M | cultist | grave chanter | Grey-robed chanter with a censer of grave smoke |
| brigand | M | bandit | road brigand | Leather, hood, axe |
| ruffian | M | bandit | ruffian | Street thug, club |
| ruffian-dangerous | M | variant of ruffian | dangerous ruffian | Scarred, knife |
| ruffian-enforcer | M | bandit | ruffian enforcer | Big brute, padded armor, cudgel |
| poacher | M | bandit | lake poacher | Oilskin coat, short bow |
| poacher-shieldman | M | bandit | poacher shieldman | Spear and board shield |
| bonesetter | M | bandit | poacher bonesetter, back-alley bonesetter | Satchel of splints, heals allies |
| shadow-trainee | M | shadow guild | shadow trainee | Masked novice, dagger |
| shadow-master | M | variant of shadow-trainee | shadow master | Black-clad master, twin blades |
| goblin | M | goblin | (base for the variants below and random encounters) | Small, sinewy, grey-green, ragged hide, crude spear; feral, not comic |
| goblin-hexer | M | variant of goblin | goblin hexer | Fetish-hung goblin shaman, bone staff |
| goblin-loot | M | variant of goblin | loot goblin | Wiry, feral goblin hunched under a heavy stolen sack |
| faerie | S | fey | faerie folk | Thin, pale, long-limbed fey with moth-like wings; an uncanny cold glow |
| imp-forest | S | fey | forest imp | Bark-skinned impish creature, twig horns |
| fungus | M | plant | sentient fungus | Walking mushroom cluster, spore puffs |
| ent | L | plant | ent | Treant: bark body, branch arms, mossy face |
| ogre-forest | L | ogre | forest ogre | Hulking ogre, tree-trunk club (Crushing Blow) |
| crocodile | L | reptile | crocodile | Long-bodied crocodile, low stance (frame L, wide) |
| creeper-cave | M | cave | cave creeper | Pale eyeless crawler, many legs |
| creeper-abyssal | M | variant of creeper-cave | abyssal creeper | Black-violet with bioluminescent spots |
| stalker-cave | M | cave | cave stalker | Gaunt clawed hunter, glowing eyes |
| bats-echo | S | cave | echo bats | A swarm of 3–4 bats in one sprite |
| ice-warrior | M | ice | ice warrior | Frost-armored humanoid with an ice blade |
| ice-guardian | L | ice | ice guardian | An ice-construct sentinel, crystalline |
| snow-floof | S | snow | snow floof | Squat, thick-furred tundra predator with hidden teeth; deceptively soft-looking, not cute |
| dummy-training | M | training | training dummy | A straw dummy on a post |
| straw-footman | M | training | straw footman | A straw figure with a wooden sword |
| straw-archer | M | variant of straw-footman | straw archer | A straw figure with a toy bow |
| guard | M | town guard | guard | Frostfang guard: mail, spear, city livery |
| guard-royal | M | variant of guard | king's guard | Polished plate, halberd |
| guard-captain | M | variant of guard | captain of the guard | Plumed helm, sword and shield |

Named NPCs that join as companions use their class sprite. Non-combat
townsfolk need no battle art.

### Formation and battle markers — `sprites/battle/ui/`

| ID | Size | Frames | Description |
|---|---|---|---|
| cell | 32×16 | 1 | A ground ellipse marking a formation cell (subtle) |
| cell-acting | 32×16 | 4 | A glowing pulse under the unit currently acting |
| cell-targeted | 32×16 | 2 | A red pulse under the current target |
| acting-arrow | 8×8 | 2 | A bobbing arrow above the acting unit |
| fallen | 16×16 | 1 | A planted weapon with a cloth, left at a fallen member's cell |
| surrendered | 16×16 | 1 | A white rag tied to a stick, beside a yielded foe |
| hp-frame | 32×6 | 1 | A health-bar frame. The client fills it |

### Status icons — `sprites/ui/status/` (16×16, 1 frame)

These are shown above units, and in the web client's Effects and Status
panels. The client draws the stack count.

| ID | Status (help name) | Description |
|---|---|---|
| bleeding | Bleeding | A blood drop with a slash |
| staggered | Staggered | Wobbly swirl lines |
| knocked-down | Knocked down | A falling figure glyph |
| stunned | Stunned | Circling stars |
| armor-broken | Armor broken | A cracked breastplate |
| exposed | Exposed | A target reticle over an open guard |
| hobbled | Hobbled | A shackled ankle |
| burning | Burning, On Fire | A flame |
| overloaded | Overloaded | A crackling arc over a hand |
| poisoned | Poisoned | A green drop with bubbles |
| asleep | Asleep (Witch, 38a), Sleeping | A closed eye with a crescent |
| paralyzed | Paralyzed (Witch, 38a) | Rigid figure bound by violet bands |
| blighted | Blighted (Witch, 38a) | A withered leaf over a cracked heart |
| weakened | Weakness, Curse of Frailty effects | A drooping arm |
| hamstrung | Hamstrung | A cut tendon line at the heel |
| winded | Winded | A puff of breath with sweat drops |
| tackled | Tackled | Two figures colliding |
| cold | Freezing, cold exposure | A snowflake over a shivering outline |
| regenerating | Regeneration, heal over time | A green rising spiral |
| lit | Illumination, Floating Light | A small glowing orb |
| hidden | Hidden, Very Hidden | A half-faded eye |
| chanting | Casting a spell (chant in progress) | An open hand with rising runes (abstract glyphs, not letters) |
| winding-up | Winding up a heavy blow | A drawn-back fist with motion lines |
| warded | Guarded by a guardian | A shield with a small figure behind |
| wounded-light | Light wound | A thin bandage strip |
| wounded-lasting | Lasting wound | A bloodied bandage |
| dread | Dread Whisper or morale shaken | A dark wisp over a trembling outline |
| tracked | Actively Tracking | A pawprint trail |

### Role icons — `sprites/ui/roles/` (16×16)

| ID | Description |
|---|---|
| fighter | Crossed swords |
| healer | A hand with a green glow |
| caster | A staff tip with an arcane spark |
| guardian | A tower shield |
| controller | A violet binding knot (the Witch role, 38a) |

### Morale icons — `sprites/ui/morale/` (16×16)

| ID | Description |
|---|---|
| hold | A planted banner |
| yield | A lowered white flag |
| flee | A running foot with dust |
| nerve | A steady flame (company nerve) |
| shaken | A guttering flame |

### Battlefield condition banners — `sprites/ui/conditions/` (16×16)

| ID | Condition (30f) | Description |
|---|---|---|
| dark | Darkness | A crescent moon over black |
| ambush | Ambush or surprise | An eye in the bushes |
| narrow | Narrow ground | Two walls pinching a path |
| cold | Cold | A frosted thermometer-like icicle (no numbers) |
| fatigue | Fatigue | A drooping figure |
| flanked | Flanking or leaping | An arrow curving around a block |
| cluster | Cluster attacks | Three dots with an impact burst |

---

## S4 — Battle animation and effects (phase 40g)

### Unit animation files

Each battle unit folder gets these files in addition to `idle.png`. Frames
face right, at the unit's size class.

**Standard set (every class and enemy):**

| File | Frames | Loop | Description |
|---|---|---|---|
| `walk.png` | 4 | yes | Stepping forward to strike and back again; also used for fleeing (mirrored) |
| `attack.png` | 6 | no | The class or creature's main blow. Frames 1–2 wind, 3–4 strike, 5–6 recover |
| `hurt.png` | 2 | no | A flinch back |
| `down.png` | 4 | no | Collapse. The last frame is held while fallen |
| `dodge.png` | 3 | no | Sidestep or lean away |

**Additional files by capability:**

| File | Frames | Who needs it | Description |
|---|---|---|---|
| `cast.png` | 6 | Cleric, Wizard, Witch, and any caster enemy (lich, dark acolyte, grave chanter, goblin hexer, bonesetter, bonecrafter) | Frames 1–4 chant (loopable), 5–6 release |
| `block.png` | 3 | Shield users: Warrior, Cleric, bone warden, poacher shieldman, guards | Raise the shield to absorb the blow |
| `parry.png` | 3 | Blade users: Warrior, Rogue, Ranger, shadow guild, ice warrior, guard captain | Deflect with the weapon |
| `windup.png` | 3 | Units with telegraphed heavy blows: forest ogre, ruffian enforcer, ent, ice guardian, Warrior | Rear back; the last frame is held while winding up |
| `shoot.png` | 6 | Ranged: Ranger, lake poacher, straw archer | Draw, release, follow-through. Replaces `attack.png` for ranged attacks |
| `prone.png` | 3 | Everyone except S-size swarms | Frame 1 knocked flat (held), 2–3 getting up |
| `yield.png` | 2 | Humanoid enemies that can surrender (bandits, cultists, shadow guild, goblins, guards) | Kneel and lower the weapon. The last frame is held |
| `victory.png` | 4 | Classes only | A sober end-of-battle pose: weapon lowered, catching breath (no cheering) |
| `guard-step.png` | 3 | Warrior and Cleric (guardian role) | A lunge sideways to intercept a blow for an ally |

**Unit file totals:** each class gets about 10 more files and each enemy
5–8. With 7 class sprites and 49 enemy and fallback sprites, S4 is about
390 unit files plus about 60 effect files.

### Hit effects — `sprites/battle/effects/hits/` (32×32, 4 frames, not looping)

| ID | Description |
|---|---|
| slash | A curved white-red arc |
| stab | A narrow thrust flash with a spark point |
| blunt | A round impact burst with dust |
| cleave | A heavy diagonal arc with chips |
| claw | Three parallel rake streaks |
| bite | Jaw-snap streaks with a small spray |
| arrow-hit | A small splinter burst |
| magic-hit | An arcane ring burst |
| crit | A larger starburst overlay layered on top of any hit |
| shield-bash | Shield-rim impact with stun stars starting (the interrupt) |
| chant-broken | Runes shattering outward (an interrupted chant) |
| guard-intercept | A shield flash between guardian and ward |

### Defense feedback icons — `sprites/battle/effects/feedback/` (16×16, 3 frames, pop and fade)

| ID | Description |
|---|---|
| miss | A whoosh streak |
| blocked | A shield with an impact tick |
| parried | Crossed blades with a spark |
| dodged | A blurred afterimage |
| resisted | A small ward sigil |

### Projectiles — `sprites/battle/effects/projectiles/` (16×8 unless noted, 2 frames, looping)

| ID | Description |
|---|---|
| arrow | An arrow with fletching |
| bolt | A crossbow bolt (future crossbows) |
| stone | A sling stone with a motion trail |
| magic-missile | A blue-white arcane dart, 16×16 |
| spark | A crackling ember, 8×8 |
| hex-bolt | A violet curse mote, 16×16 |
| spore | A drifting spore puff, 8×8 (sentient fungus) |
| web-shot | A strand of silk, 16×8 (spiders) |

### Spell effects — `sprites/battle/effects/spells/`

These are all spells in `_datafiles/world/default/spells/` plus the Witch
hexes from 38a. Single-target effects are drawn on the target. Area effects
span the target's formation (draw 96×64 to cover 3 columns).

| ID | Spell | Size | Frames | Description |
|---|---|---|---|---|
| magic-missile-impact | Magic Missile | 32×32 | 4 | Arcane burst; pairs with the `magic-missile` projectile |
| sparks | Shower of Sparks | 96×64 | 6 | A rain of embers over the enemy group |
| heal | Minor Heal | 32×32 | 6 | Rising golden-green motes and a soft glow |
| heal-all | Minor Heal All | 96×64 | 6 | A wide gentle light washing the company |
| tend | Tend | 32×32 | 4 | A bandage wrap glint |
| cure-poison | Cure Poison | 32×32 | 6 | Green bubbles drawn out and dispersing |
| aid | Aid | 32×32 | 4 | A steadying hand-glow |
| illumination | Illumination | 64×64 | 6 | A light bloom that brightens the area (used in darkness) |
| floating-light | Floating Light | 16×16 | 4 | A hovering orb, looping |
| hex | Hex (existing) | 32×32 | 6 | Violet runes coiling around the target |
| polymorph | Polymorph | 32×32 | 6 | A swirling transformation puff |
| slumber | Slumber (Witch) | 32×32 | 6 | Lilac mist settling and closing the eyes |
| earthbind | Earthbind (Witch) | 32×32 | 6 | Grave-soil hands dragging down |
| leaden-curse | Leaden Curse (Witch) | 32×32 | 6 | Grey weights clamping the limbs |
| miasma | Miasma (Witch) | 96×32 | 6 | A green-black poison cloud over one enemy row |
| binding-hex | Binding Hex (Witch) | 32×32 | 6 | Violet bands snapping tight |
| frailty | Curse of Frailty (Witch) | 32×32 | 6 | Cracks spreading over the target outline |
| dread-whisper | Dread Whisper (Witch) | 96×64 | 6 | Dark whispering wisps over the enemy group |
| blight | Blight (Witch) | 32×32 | 6 | Rot spreading from the heart, a withering glow |
| cast-arcane | Generic chant glow, arcane school | 32×32 | 4 (loop) | Fallback chant aura |
| cast-holy | Generic chant glow, restoration | 32×32 | 4 (loop) | Fallback chant aura |
| cast-hex | Generic chant glow, hexcraft | 32×32 | 4 (loop) | Fallback chant aura |
| cast-dark | Generic chant glow, necromantic or unholy (enemy casters) | 32×32 | 4 (loop) | Fallback chant aura |

### Status overlays (animated on units) — `sprites/battle/effects/status/` (32×32, looping)

| ID | Frames | Description |
|---|---|---|
| bleeding | 4 | Dripping drops |
| burning | 4 | Licking flames over the body |
| stunned | 4 | Circling stars at head height |
| asleep | 4 | A drifting crescent and motes |
| poisoned | 4 | Rising green bubbles |
| paralyzed | 2 | Violet bands flickering |
| overloaded | 4 | Crackling arcs |
| blighted | 4 | Dark motes sinking |
| staggered | 2 | Wobble lines |
| exposed | 2 | Cracks flickering |
| hobbled | 2 | A chain glint at the feet |
| warded | 4 | A faint shield shimmer |
| armor-shatter | 4 (no loop) | Plate shards bursting, once, when armor breaks |

### Damage digits — `sprites/ui/font/`

| ID | Size | Description |
|---|---|---|
| digits | 10 glyphs, each 6×9 (60×9 file) | Digits 0–9, white with a dark outline; the client tints them (damage, critical, heal) |
| digits-large | 10 glyphs, each 8×12 (80×12 file) | Larger digits for critical hits |

---

## S5 — Advanced classes (phase 40h, after 38b)

These are the 18 level-10 classes from the [branching design](2026-10-01-branching-class-progression-design.md)
and the [level impact design](2026-10-05-level-impact-class-power-design.md).
**Each class needs the full S1 map set** (`idle.png`, `walk.png`) **plus the
full S3/S4 battle set.** Each battle set has the standard files plus the
capability files its lineage uses, as listed in S4. Designs evolve from the
lineage sprite and apply the alignment cue. Until a class's art exists, the
client shows its lineage sprite.

| ID | Lineage | Gate | Visual direction |
|---|---|---|---|
| knight | Warrior | Positive | Plate, kite shield with a sun device, gold trim |
| mercenary | Warrior | Unrestricted | Mixed armor, a heavy blade on the shoulder, coin pouch |
| reaver | Warrior | Negative | Spiked dark plate, ember-red cloth, a jagged axe |
| scout | Rogue | Positive | Light cloak, spyglass, short bow and dagger |
| duelist | Rogue | Unrestricted | Fitted doublet, rapier and parrying dagger |
| assassin | Rogue | Negative | Black wraps, half-mask, vial bandolier |
| warden | Ranger | Positive | Green-gold cloak, longbow, small buckler |
| hunter | Ranger | Unrestricted | Furs, recurve bow, trophies |
| stalker | Ranger | Negative | Dark hooded leathers, barbed arrows, ember eyes |
| priest | Cleric | Positive | Pale undyed robes, a smoking censer, a sun-disc pendant |
| chaplain | Cleric | Unrestricted | Armored cassock, warhammer, prayer beads |
| hexer | Cleric | Negative | Ash-grey vestments, a cracked holy symbol, smoke |
| theurgist | Wizard | Positive | Silver-blue robes, warding circles, orb staff |
| arcanist | Wizard | Unrestricted | Layered blue robes, a floating tome |
| warlock | Wizard | Negative | Black-violet robes, a draining green flame |
| hedge-witch | Witch | Positive | Herb-laden shawl, warding charms, a pale lantern |
| coven-sage | Witch | Unrestricted | A deep layered hood, a staff hung with charms and small bones |
| hag | Witch | Negative | A hunched form, ragged black robes, ember-lit eyes |

**Approximate total:** 18 × (2 map files + about 11 battle files) ≈ 235
files.

---

## S6 — Later classes, recruits and variants (after 38c+)

S6 follows the same per-class file sets as S5. Produce it only once the
owning design is approved and scheduled.

**Elite classes (18).** paladin, warlord, dread-knight, pathfinder,
swordmaster, nightblade, sentinel, marksman, ravager, high-priest,
war-priest, hierophant-of-ash, archon, archmage, necromancer, wise-one,
coven-mother, crone-of-ash. Each elite develops its advanced class's
design: grander, with a stronger alignment cue.

**Expanded catalogue (40).** From the [expanded companion catalogue](2026-10-01-expanded-companion-classes-design.md):

- **Advanced (20):**
  - Warrior: hoplite, armiger, berserker, spellblade;
  - Rogue: saboteur, bard, shadowdancer, cutthroat;
  - Ranger: beastkeeper, skirmisher, trapper, storm-archer;
  - Cleric: oracle, shaman, exorcist, druid;
  - Wizard: sorcerer, hexweaver (renamed from the old Witch path),
    elementalist, illusionist.
- **Elite (20):**
  - Warrior: legionary, ironclad, juggernaut, rune-knight;
  - Rogue: demolitionist, troubadour, phantom, executioner;
  - Ranger: beastmaster, outrider, snaremaster, tempest-archer;
  - Cleric: seer, spiritkeeper, inquisitor, elder-druid;
  - Wizard: high-sorcerer, malison, elemental-savant, mirage-weaver.

**Creature recruits (20).** Each needs map and battle sets with the beast
standard set, without `yield` or `victory`:

| Base | Evolved |
|---|---|
| hound | warhound |
| hellhound | cerberus |
| stone-golem | runic-golem |
| iron-golem | ward-golem |
| gryphon | storm-gryphon |
| drake | elder-drake |
| treant | ancient-treant |
| wisp | lantern-wisp |
| skeleton-recruit | bone-sentinel |
| revenant | grave-warden |

Large forms use L or XL frames.

**Race variants (optional, pending owner decision 4).** Elf variants of the
6 base classes, as map and battle sets (`<class>-elf`).

## S7 — Optional later art

These are not required by any scheduled phase. List them here when they are
designed.

- **Class portraits** (64×64) for the company dock, OB64-style. One per
  class, plus named companions.
- **Item icons** (16×16) by weapon family and armor path from the
  [equipment design](2026-10-01-equipment-tiers-design.md), with tier and
  rarity frames from the [loot design](2026-10-05-loot-system-design.md).
  These need their own spec once the 36a/36b catalog ships.
- **Other-player markers** on the map (pending owner decision 1).
- **Concealed-camp art** for the deferred PvP camp-visibility feature: a
  hidden-camp shimmer, and a found-camp marker.
- **Enemy group markers** on the map, if scouting ever shows enemies on
  tiles.

---

## Delivery checklist for the generating agent

1. Deliver S0 and wait for owner approval.
2. Work set by set in priority order. Within a set, follow table order.
3. For every file:
   - the exact path and name;
   - the frame size;
   - the frame count;
   - the row order;
   - the anchor;
   - master-palette colors only;
   - transparent background (opaque only for terrain and backgrounds);
   - no text.
4. Deliver a contact sheet per set (`sprites/contact/<set>.png`) showing
   every sprite at 2x on mid-grey, for review.
5. Report any row you could not produce, or any deviation from its size or
   frame count. Never silently substitute.
6. All art must be original. Do not copy, trace or closely imitate existing
   games' sprites, including Ogre Battle 64.

## Maintenance rules

- A **new class** adds S1 map files and S3/S4 battle files, under S5 or S6
  by tier.
- A **new enemy** adds a row to the S3 enemy table: a family variant if it
  is a recolor, or a new family otherwise. Random encounters (37) and world
  building will add many. Until then the fallback silhouettes cover them.
- A **new spell** adds an S4 spell-effect row. Until then its school's
  `cast-*` glow covers it.
- A **new status** adds an S3 status icon and, if it is visible, an S4
  overlay.
- A **new biome** adds an S2 terrain row and an S3 background mapping.
- A **new landmark** legend maps to an existing landmark ID or adds a row.
