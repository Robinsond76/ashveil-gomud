# A13: Light markers for the living map

Read [00-standards.md](00-standards.md) first, especially section 6
(icons) and the animated-icon rule. Attach the approved A4 `fire-lit.png`
and `here-ring.png`, and one approved A1 map figure (the warrior) for
scale.

The map is learning to show what a character can see (living map, LM2:
[design](../designs/2026-10-10-living-map-design.md)). At night or
underground the map darkens. A character carrying a light gets a small
marker beside their figure, and the game draws the soft pool of light
around it in code. The game has two carried lights:
- a **torch**, used up after about six game hours;
- an oil **lantern**, which can be put out and lit again.

A party light (a spell) floats over the one who cast it.

**Paint no glow, halo or light pool around these objects.** The game draws
the glow itself, warm for a flame and cool for the orb. A flame or a
lantern's glass may be bright, but the pixels around the object stay fully
transparent.

## What to deliver (`art/source/A13/map/markers/`)

| File | Canvas | Frames | Design |
|---|---|---|---|
| `torch.png` | 256×256 per frame | 2 | A hand torch held upright: a short wooden haft wrapped in pitch-dark cloth, with a lively flame. Only the flame moves: it shifts and licks between the 2 frames. Warm orange-gold, never neon. |
| `lantern.png` | 256×256 per frame | 2 | A small iron-and-horn hand lantern hanging from a ring, with a steady candle flame behind the panes. Between frames the flame breathes a little and a highlight on the glass shifts; the lantern itself doesn't move. |
| `lantern-out.png` | 256×256 | 1 | The same lantern, unlit: dark panes and no flame. It is shown when the lantern is carried but doused or out of oil. |
| `light-orb.png` | 256×256 per frame | 2 | A small floating mote of pale light, a soft core in a ring of faint sparks, cool white with a hint of blue. It pulses slightly larger and brighter between frames. |

The torch and lantern sit beside a figure's shoulder:
- Draw each object about **60% of the canvas tall**, centred left to right.
- Put its **hand point** (the haft's grip for the torch, the hanging ring
  for the lantern) on the canvas's centre line, at **y = 160**. The game
  lines that point up with the figure's hand.

Draw the orb centred in its canvas, about a third of the canvas wide, so it
can hover over a head.

### Optional

| File | Canvas | Frames | Design |
|---|---|---|---|
| `art/source/A13/map/camp/bonfire.png` | 512×512 per frame | 4 | A bigger campfire for the camp drawn as your marker (LM1): the same stone ring as `fire-lit`, scaled to a tile, with taller flames and rising sparks, looping. No glow on the ground; the game adds it. |

## Rules

- **Grain:** the 8 px art-pixel grain of every 256 px icon (32×32 art
  pixels per canvas; 16 px for the 512 px bonfire: 32×32 art pixels per
  tile, the same grain on screen).
- **Light and outline:** light from the top-left, and a dark selective
  outline about 1 art pixel thick, as on the A4 markers.
- **Animation:** 1 row of frames with 32 px transparent gutters. The object
  stays in exactly the same place and size in every frame; only the flame,
  glass glint or pulse changes, and the loop is seamless.
- **Shared design:** `lantern.png` and `lantern-out.png` are the same
  lantern drawn the same way, so the game can swap them without a jump.
- **Readable small:** each marker must still read at 16×16 beside a
  28-pixel figure: the torch as a torch, the lantern as a lantern, the orb
  as a light.

## Check before delivering

- Each marker at full size and at 64 px (game size), on mid-grey and on a
  near-black background, beside the A1 warrior shrunk to 128 px. Hold the
  torch and lantern at its hand.
- The torch and lantern frames played as loops (a strip is enough).
- Every pixel outside the objects is fully transparent.
