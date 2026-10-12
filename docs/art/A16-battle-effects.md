# A16: Projectiles, hits and spells on the battle screen

Read [00-standards.md](00-standards.md) first. Attach `scene-battle.png`
from A0, a few approved battle idles for scale, and the A13 light markers
for how light is handled (no painted glow halos; the game adds its own
light).

Today the battle screen draws its effects in code: a short line for an
arrow, a small square for a spell, rings and strokes for blows. The screen
already looks for effect art by name; this order supplies it. Every effect
plays at a fixed spot (a figure's centre, or travelling from one figure to
another) and fades on its own, so draw each one complete, on a transparent
background.

**Look:** the A0 rule holds: magic is rare, dim and costly. Effects are
small, crisp and coloured by their school, never neon, never bloom or lens
flare. They must read at game size against both day and night backdrops.

## Sizes and layout

The battle screen is 320×180 logical pixels, drawn at twice that. Effects
are drawn on the battle figures' **5 px grain**, at 10 master pixels per
logical pixel:

| Kind | Logical size | Master cell | Frames |
|---|---|---|---|
| Projectile | 16×16 | 160×160 | 2 (a flicker or spin, looping) |
| Hit or impact | 32×32 | 320×320 | 4 (burst, spread, fade, last wisps) |
| Spell aura (around the caster, or a lasting effect on a target) | 64×64 | 640×640 | 4, looping |
| Beam segment (tiles sideways) | 32×16 | 320×160 | 3, looping |

1 row of frames with **40 px** transparent gutters for every kind (a multiple
of the 5 px grain, as in A12). Projectiles point **right**; the game turns them to
their path. Fully transparent or fully opaque pixels only; fade with
dither. Deliver to `art/source/A16a/battle/effects/<group>/<id>.png` and
`art/source/A16b/battle/effects/<group>/<id>.png`.

## A16a: projectiles and weapon hits (26 files)

**Projectiles** (`projectiles/`):

| ID | What |
|---|---|
| `arrow` | A fletched arrow in flight, a faint motion streak behind it. |
| `crossbow-bolt` | A short, heavy quarrel. |
| `throwing-knife` | A spinning knife (2 frames: edge, flat). |
| `flask` | A stoppered clay flask tumbling (the alchemist's throw). |
| `fire-flask` | The same flask, its rag fuse burning. |
| `stone` | A sling stone or thrown rock. |

**Weapon hits** (`hits/`), the ids the screen already uses:

| ID | What |
|---|---|
| `slash` | A bright arc of a cutting blow. |
| `cleave` | A heavier, wider arc with a few chips flying. |
| `stab` | A short thrust line and a spark at its point. |
| `blunt` | A dull ring of impact and a little dust. |
| `claw` | Three raking strokes. |
| `bite` | Jaws snapping shut: two jagged arcs closing. |
| `arrow-hit` | An arrow striking: splinters and a puff. |
| `bolt-hit` | A quarrel striking: a harder spark. |
| `flask-burst` | A clay flask breaking: shards and a splash. |
| `crit` | A critical hit: a sharp starburst laid over the blow. |
| `shield-bash` | A shield's edge striking: a flat flash and stars of light. |
| `guard-intercept` | A guard stepping in: a bright parry arc. |
| `block` | A blow stopped on a shield: a spark and a ring. |
| `parry` | Blades crossing: a spark at the cross. |
| `miss` | A whoosh past the target: a pale streak. |
| `chant-broken` | A spell breaking apart: blue-white shards scattering. |
| `armor-shatter` | Armor plates breaking off and falling. |
| `poison-tick` | A sickly green bubble and drip (a poisoned figure suffering). |
| `bleed-tick` | A dark red throb (bleeding); no spatter. |
| `burn-tick` | A lick of flame (burning). |

## A16b: spells (40 files)

Every spell belongs to a school, and each school has its colour and its
three effects: the **aura** around the caster while chanting (looping), the
**bolt** that travels to a harmful single-target spell, and the **impact**
where it lands. Use these for every spell of the school, then the signature
effects below for the spells that need their own.

| School | Colour and feel | Spells |
|---|---|---|
| `conjuration` | Pale arcane blue-white, crisp sigils | Magic Missile, Arcane Lance, Arcane Ward, Shower of Sparks |
| `restoration` | Warm gold and soft green, gentle light | the heals, Bless, Ward, Barkskin, Grove, Rejuvenation, Cure Poison, Entangle, Siphon, Raise the Fallen, Call the Host, Bind the Fiend |
| `hexcraft` | Dim violet and smoky black, crawling | Withering Hex, Binding Hex, Blight, Curse of Frailty, Leaden Curse, Miasma, Slumber, Dread Whisper, Earthbind |
| `stormcraft` | Storm grey-blue with white-violet lightning | Lightning, Gust, Chill Wind, Call Fog, Rain, Stoneskin |
| `alchemy` | Amber, copper and fume green | Fire Flask, Healing Draught, Bracing Tonic, Antidote |
| `illusion` | Shimmering pale lilac, glassy | Floating Light, Illuminate, Polymorph |

Files (`spells/`): for each of the six schools, `<school>-aura.png`
(64×64, 4 looping), `<school>-bolt.png` (16×16, 2) and
`<school>-impact.png` (32×32, 4): **18 files**. Then the signature effects:

| ID | Size | What |
|---|---|---|
| `heal` | 32×32 | Motes of warm light rising over the healed. |
| `heal-all` | 64×64 | A ring of warm light washing over a figure. |
| `ward` | 64×64 | A faint shield of light settling around a figure (looping). |
| `barkskin` | 64×64 | Bark plates closing over the body. |
| `stoneskin` | 64×64 | Grey stone flakes closing over the body. |
| `entangle` | 32×32 | Roots and vines bursting up and gripping. |
| `siphon` | beam | A thin red-gold stream drawn from foe to caster. |
| `raise` | 64×64 | A column of pale light lifting a fallen figure. |
| `summon` | 64×64 | A circle of runes on the ground and a rising light (the Host, the Angel, the Fiend). |
| `arcane-lance` | beam | A focused arcane beam. |
| `lightning` | beam | A forked lightning arc that jumps from figure to figure. |
| `sparks` | 32×32 | A shower of falling sparks. |
| `gust` | 32×32 | A curling wind streak with dust. |
| `chill` | 32×32 | Frost crawling over the target. |
| `fog` | 64×64 | A bank of fog around a figure (looping). |
| `fire-burst` | 32×32 | A flask of fire bursting into flames. |
| `hex-mark` | 32×32 | A crawling violet sigil over the hexed (looping). |
| `slumber` | 32×32 | Drifting dim motes settling over the target (no Z marks). |
| `dread` | 64×64 | Shadowy wisps coiling round the target. |
| `polymorph` | 32×32 | A glassy shimmer and puff. |
| `light` | 32×32 | The floating light: a small steady mote. |
| `drain` | 32×32 | A dark pull at the target (Blight, Miasma). |

That is 18 + 22 = **40 files**.

## Check before delivering

Each effect at full size and at game size, over a day and a night battle
backdrop, next to a battle idle for scale; looping effects as strips to
check the loop; projectiles shown pointing right. Keep review sheets under
5 MB.
