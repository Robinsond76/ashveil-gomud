# A5: Interface icons and battle markers

Read [00-standards.md](00-standards.md) first, especially section 6
(icons). Attach `icon-sample.png` from A0 and the approved A4 icons.

These appear in the web client's panels and on the battle screen at 16×16.
Every icon is **256×256, transparent, 1 frame**, unless a row says
otherwise. Each set below is a family: the same outline, framing and
detail level. The sets differ only in subject.

## Status icons (`art/source/A5/ui/status/<id>.png`)

The game writes the stack count beside the icon. Keep the bottom-right
corner light.

| ID | Status | Design |
|---|---|---|
| `bleeding` | Bleeding | A blood drop crossed by a short slash. |
| `staggered` | Staggered | Wobbly swirl lines. |
| `knocked-down` | Knocked down | A small figure falling backwards. |
| `stunned` | Stunned | Small dull stars circling. |
| `armor-broken` | Armor broken | A cracked breastplate. |
| `exposed` | Exposed | A target reticle over an open guard. |
| `hobbled` | Hobbled | A shackled ankle. |
| `burning` | Burning, on fire | A flame. |
| `overloaded` | Overloaded | A crackling arc over an open hand. |
| `poisoned` | Poisoned | A green drop with bubbles. |
| `asleep` | Asleep, sleeping | A closed eye under a crescent. |
| `paralyzed` | Paralyzed | A rigid figure bound by violet bands. |
| `blighted` | Blighted | A withered leaf over a cracked heart. |
| `weakened` | Weakness, Curse of Frailty | A drooping arm. |
| `hamstrung` | Hamstrung | A cut line across a heel. |
| `winded` | Winded | A puff of breath. |
| `tackled` | Tackled | Two figures colliding. |
| `cold` | Freezing, cold exposure | A snowflake over a shivering outline. |
| `regenerating` | Regeneration, healing over time | A green rising spiral. |
| `lit` | Illumination, Floating Light | A small glowing orb. |
| `hidden` | Hidden | A half-faded eye. |
| `chanting` | Casting a spell | An open hand with abstract rising runes (glyph shapes, not letters). |
| `winding-up` | Winding up a heavy blow | A drawn-back fist with motion lines. |
| `warded` | Guarded by a guardian | A shield with a small figure behind it. |
| `wounded-light` | Light wound | A thin bandage strip. |
| `wounded-lasting` | Lasting wound | A bloodied bandage. |
| `dread` | Dread, shaken morale | A dark wisp over a trembling outline. |
| `tracked` | Being tracked | A trail of pawprints. |

## Role icons (`art/source/A5/ui/roles/<id>.png`)

| ID | Design |
|---|---|
| `fighter` | Crossed swords. |
| `healer` | An open hand with a soft green glow. |
| `caster` | A staff tip with a pale arcane spark. |
| `guardian` | A tower shield. |
| `controller` | A violet binding knot. |

## Morale icons (`art/source/A5/ui/morale/<id>.png`)

| ID | Design |
|---|---|
| `hold` | A planted banner, upright. |
| `yield` | A lowered white rag on a stick. |
| `flee` | A running foot kicking up dust. |
| `nerve` | A steady flame. |
| `shaken` | A guttering, bent flame. |

## Battlefield conditions (`art/source/A5/ui/conditions/<id>.png`)

| ID | Condition | Design |
|---|---|---|
| `dark` | Darkness | A crescent moon over black. |
| `ambush` | Ambush | An eye peering from bushes. |
| `narrow` | Narrow ground | Two walls pinching a path. |
| `cold` | Cold | A hanging icicle with frost. |
| `fatigue` | Fatigue | A drooping, tired figure. |
| `flanked` | Flanking | An arrow curving around a block. |
| `cluster` | Cluster attacks | Three dots with an impact burst. |

## Battle markers (`art/source/A5/battle/ui/`)

These are drawn on the battle screen's ground and over units, seen from the
battle's side-on angle.

| File | Canvas | Frames | Design |
|---|---|---|---|
| `cell.png` | 512×256 | 1 | A subtle flat ground ellipse marking a formation cell, pale and low-contrast. |
| `cell-acting.png` | 512×256 per frame | 4 | The same ellipse glowing pale gold, pulsing under the acting unit. |
| `cell-targeted.png` | 512×256 per frame | 2 | The same ellipse pulsing dull red under the target. |
| `acting-arrow.png` | 128×128 per frame | 2 | A small downward arrow that bobs above the acting unit, bone colored. |
| `fallen.png` | 256×256 | 1 | A weapon planted in the ground with a cloth tied to it, left where a member fell. |
| `surrendered.png` | 256×256 | 1 | A white rag tied to a stick, planted beside a yielded foe. |
| `hp-frame.png` | 512×96 | 1 | A slim health-bar frame: a dark iron rim with an empty, transparent inside. The game fills it. |
