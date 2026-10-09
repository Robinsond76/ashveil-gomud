# A3: Landmark overlays

Read [00-standards.md](00-standards.md) first. Attach `landmark-sample.png`
and `scene-map.png` from A0, plus a few approved A2 tiles.

A landmark is drawn **over** its room's terrain tile and marks a notable
place (an inn, a gate, a temple). Each overlay is one **512×512 canvas
with transparency**. The object fills about 80%, sits on the lower part
of the canvas, and is centered.

**View:** the same slight 3/4 top-down as the figures, so buildings show
a roof and one front face. It must read as that place at 32×32. Use bold
shapes, one clear identifying feature and **no lettering on signs**. A
soft contact shadow (≤ 30% opacity) is allowed. There's no ground patch
around it; the tile shows through.

## Deliverables (`art/source/A3/map/landmarks/<id>.png`)

| ID | Marks | Design |
|---|---|---|
| `inn` | Inns | A timber-framed inn with a hanging sign (a tankard shape, no letters) and a warm lit window. |
| `bank` | Banks | A squat stone strongbox house with an iron-banded door and a coin emblem. |
| `shop` | General shops | A market stall with a striped, faded awning and goods on the counter. |
| `smithy` | Armorers, weaponsmiths | An anvil before a forge with an ember glow and a chimney. |
| `herbalist` | Herbalists, brewers | A small hut with herbs drying under the eaves and bottles on a sill. |
| `trainer` | Trainers | Crossed practice swords over a straw target. |
| `temple` | Temples, chapels | A small stone chapel with a bell in an open belfry. |
| `shaman` | Shamans | A hide tent with a bone-and-feather totem pole. |
| `hermit` | Hermits | A crooked moss-roofed hut with a lantern by the door. |
| `gate` | City gates | A stone gatehouse arch with a raised portcullis. |
| `wall` | Walls | A crenellated stone wall segment. |
| `bridge` | Bridges | A wooden plank bridge with rope rails. |
| `keep` | Keeps | A square stone tower with a faded banner. |
| `throne` | Throne rooms | A heavy throne under a cloth canopy. |
| `townsquare` | Town squares | A stone fountain on a paved square. |
| `village` | Villages | A cluster of three thatched cottages. |
| `caravan` | Caravans | A covered wagon with a canvas top. |
| `lake-house` | Lake houses | A small stilt house over water (the water is part of the object). |
| `cave-mouth` | Cave entrances | A dark cave opening in a rock outcrop. |
| `dungeon-stair` | Dungeon entrances | Stone stairs descending into darkness, edged by worn blocks. |
| `obelisk` | Obelisks | A tall stone carved with abstract runes (no letters), with faint lichen. |
| `pond` | Ponds | A small reed-fringed pond. |
| `rocks` | Rock piles | A pile of weathered boulders. |
| `desert-ruin` | Desert ruins | A half-buried broken column in sand. |
| `alts` | The alt-character room | An open book on a lectern. |
| `landmark` | Any other notable spot | A generic standing stone with a small cairn. |
| `boss-lair` | Boss lairs | A cracked skull on a stake with rags. Grim, not gory. |

## Check before delivering

Place each overlay on a `land`, `city` and `forest` tile at 32×32 and at
128×128. It must read at both sizes and never hide its whole tile.
