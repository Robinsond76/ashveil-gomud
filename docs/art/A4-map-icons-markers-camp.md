# A4: Map icons, markers, camp and app icons

Read [00-standards.md](00-standards.md) first, especially section 6
(icons). Attach `icon-sample.png`, `camp-sample.png` and `scene-map.png`
from A0.

These sit on map tiles. Resource icons sit in a tile's corner, three to a
corner, so they must read at 16×16 next to each other. Animated rows are
**1 row × N frames** on transparency, with the object fixed in place
(standards section 4).

## Resource icons (`art/source/A4/map/resources/<id>.png`, 256×256, 1 frame)

| ID | Design |
|---|---|
| `water` | A fresh water droplet over small ripples, clear blue. |
| `forage` | A sprig of red berries with two leaves. |
| `herbs` | A bundle of medicinal green leaves tied with twine. |
| `firewood` | Two crossed split logs. |
| `shelter` | A rock overhang with a dark hollow, or a lean-to of branches. |
| `fishing` | A fish arcing over a curl of wave. |
| `game` | A pair of cloven hoofprints. |
| `unknown` | Unsurveyed: a faint, abstract compass rose, pale and low-contrast (no letters). |
| `depleted` | Drawn **over** any gathering icon when it's picked clean: a dim, empty woven basket with a slash of shadow. Partly transparent, so the icon beneath stays faintly visible. |

## Map markers (`art/source/A4/map/markers/`)

| File | Canvas | Frames | Design |
|---|---|---|---|
| `here-ring.png` | 512×512 per frame | 4 | A flat ellipse ring on the ground under the current figure (seen from above at the figure's 3/4 angle). It pulses softly brighter and dimmer in pale bone-gold. |
| `company-badge.png` | 192×192 | 1 | A small shield-shaped plate, dark iron with a bone rim. The game writes the member count on it, so leave the middle plain. |
| `ally-banner.png` | 256×256 per frame | 2 | A small pennant on a short pole, slate blue and bone, fluttering between the 2 frames. |
| `walk-target.png` | 256×256 per frame | 2 | A small destination flag planted in the ground, waving between frames, ochre cloth. |
| `walk-dot.png` | 128×128 | 1 | A small path breadcrumb: a pale, slightly worn stone dot. |
| `exit-up.png` | 128×128 | 1 | An upward chevron like stone stairs going up, pale. |
| `exit-down.png` | 128×128 | 1 | A downward chevron like stairs going down, darker. |

## Camp (`art/source/A4/map/camp/`)

Your own and allied camps are drawn on the camp's tile. The tent and camp
are drawn at the tile's size; the fire pieces are drawn at icon size beside
them.

| File | Canvas | Frames | Design |
|---|---|---|---|
| `tent.png` | 512×512 | 1 | A small, pegged canvas tent with a bedroll in front and a pack. Weathered canvas. |
| `tent-ally.png` | 512×512 | 1 | The same tent with a small ally pennant (as `ally-banner`) on its pole. |
| `camp-rough.png` | 512×512 | 1 | A camp without a tent: two bedrolls on the ground around a stone fire ring, and a pack leaning on a log. |
| `camp-rough-ally.png` | 512×512 | 1 | The same, with the ally pennant on a stick. |
| `fire-unlit.png` | 256×256 | 1 | A ring of stones with unlit split wood. |
| `fire-lit.png` | 256×256 per frame | 4 | The same ring and wood with a crackling fire: flames shift and sparks rise. Looping. Warm but not neon. |
| `embers.png` | 256×256 per frame | 3 | The ring with banked embers glowing low after a rest, pulsing gently. Looping. |
| `smoke.png` | 256×256 per frame | 4 | A thin grey smoke curl drifting up, drawn above a lit fire. Looping, transparent. |
| `resting.png` | 256×256 per frame | 3 | A sleep sign: a pale crescent moon with fading motes drifting up. **No letters, no "Z".** Looping. |
| `inn-rest.png` | 256×256 | 1 | A bed with a turned-down blanket and a small lantern. |

The fire pieces share one stone ring and wood pile, so they swap without a
jump. Draw `fire-lit`, `embers` and `fire-unlit` from the same base.

## App icons (`art/source/A4/app/`)

| File | Size | Design |
|---|---|---|
| `icon-512.png` | 1024×1024, opaque | The Ashveil emblem: an ember-lit campfire before a dark mountain ridge under a night sky. No text. |
| `icon-192.png` | 1024×1024, opaque | The same emblem simplified for small sizes (fewer, bolder shapes), not just a reduced copy. |
| `icon-maskable-512.png` | 1024×1024, opaque | The same, with the emblem inside the central 80% and the background running to every edge. |
| `favicon-32.png` | 256×256, transparent | A tiny version: just the fire and a ridge line, legible at 32×32. |
