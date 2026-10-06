# A6: Battle backgrounds and base class battle figures

Read [00-standards.md](00-standards.md) first. Attach `lineup-battle.png`
and `scene-battle.png` from A0, and each class's approved A1 map sheet so
the designs match.

The battle screen is a side view. The company stands in a 3×3 formation
on the left, the enemy on the right. Units stand on a band of open ground
across the lower part of the picture.

## Battle backgrounds (`art/source/A6/battle/backgrounds/<id>.png`)

Each is **2560×1440, opaque**, side view, with no figures, UI or text.

- **Keep the ground band clear:** from about 55% of the height down to
  near the bottom, across the whole width, is open, fairly even ground.
  Nothing tall stands there. Units stand on it and must read clearly
  against it, so keep it quiet and mid-toned.
- Depth (horizon, trees, buildings) sits above the band.
- Paint daylight only. The game darkens the picture for night and dark
  places.

| ID | Used for | Design |
|---|---|---|
| `forest` | forest | Tall dark trunks around a mossy clearing, shafts of light. |
| `deep-web` | spiderweb | Web-choked trees, hanging cocoons, a pale green haze. |
| `plains` | land, farmland, default | Rolling grass, distant hills, scattered stones. |
| `road` | road | A rutted road through fields, a milestone, distant trees. |
| `city` | city, fort | A street between timber and stone buildings, barrels and crates. |
| `slums` | slums | A narrow muddy alley, sagging shacks, hanging laundry. |
| `interior` | house | A tavern or house interior: beams, a hearth, tables pushed aside. |
| `catacombs` | dungeon | Burial niches, bone piles, candles in sconces. |
| `cave` | cave | A cavern with stalactites, faint crystal veins and a dark pool. |
| `snowfield` | snow | A white expanse, frost-heavy pines, a grey sky. |
| `ice-keep` | Stormwatchers Keep | Ice-sheathed stone halls and frozen pillars. |
| `shore` | shore, water | A lakeshore with reeds, an upturned boat, mist over the water. |
| `swamp` | swamp | Black water, twisted roots, hanging moss. |
| `desert` | desert | Dunes, a sun-bleached ruin, heat haze. |
| `highlands` | mountains, cliffs | A rocky pass, a cliff face behind, scree. |
| `training-yard` | the tutorial zone | A fenced yard with straw targets and weapon racks. |

## Base class battle idles (`art/source/A6/battle/units/<id>/idle.png`)

Each is a **battle idle sheet**: 1 row × 4 frames, facing right, size
**M** (figure ≥ 320 px tall in the source). It's the same design as the
class's A1 map figure, drawn larger and side-on.

| ID | Stance |
|---|---|
| `warrior` | Shield forward, sword low, weight braced. |
| `rogue` | Crouched, blades reversed, ready to spring. |
| `ranger` | Bow held low with an arrow nocked. |
| `cleric` | Mace and shield ready; the holy symbol glints on one frame. |
| `wizard` | Staff planted; a faint glow at the stone swells and fades. |
| `witch` | Staff or wand low, the hood shading the face, a thin curl of grave-mist at the hem. |
| `halberdier` | Halberd lowered and angled forward, feet set wide. |
| `samurai` | A low ready stance, hand on the katana hilt (not yet drawn). |
| `shaman` | Totem staff raised slightly, the mask down, feathers stirring. |
| `dollmaster` | Control bar raised; the doll hangs on its strings and sways. |
| `gryphon-rider` | Spear leveled forward, the winged helm catching the light. |
| `beasttamer` | Whip uncoiled and trailing, the free hand open as if calling a beast. |
| `alchemist` | A flask raised ready to throw, the other hand at the bandolier. |
| `arbalist` | Heavy crossbow shouldered and aimed forward. |
| `adventurer` | Walking staff held across the body defensively. |
