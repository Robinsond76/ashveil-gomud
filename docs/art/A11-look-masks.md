# A11: Skin and hair masks

Read [00-standards.md](00-standards.md) first.

Players choose a skin tone and a hair colour when they make a character.
The old 1x art was repainted by swapping a few exact palette colours. The new
art can't be, so the game needs to be told which pixels are skin and which
are hair. This phase draws that information: two masks per class sheet. You
don't redraw any art.

## What to deliver

For every **approved map-unit master** (A1, A8a, A8b, A9, A10: 101 sheets)
deliver two mask sheets beside it, with the same name and a suffix:

- `art/source/A11/map/units/<id>.skin.png`
- `art/source/A11/map/units/<id>.hair.png`

Each mask is **exactly the size and layout of its master** (2272×832, the
same cells and gutters). Each mask pixel is either:

- **opaque white** where the master shows that material (visible skin:
  face, neck, hands, bare arms; or visible hair: head hair, beard,
  eyebrows), or
- **fully transparent** everywhere else.

There's no grey and no partial alpha. Mark only pixels that are opaque in
the master, and follow the master's 4 px art grain.

## Rules

- Mark only what is really skin or hair. Leather, wood, fur trim, rope and
  bone stay unmarked even when they're brown. Masks, helmets and hoods that
  hide the face mean nothing is marked there.
- A figure with no visible skin or hair (a full helm, a masked assassin)
  gets an empty mask. Deliver it anyway.
- Painted shading inside skin or hair stays in the master; the game keeps
  it when recolouring. Don't try to mark highlights or shadows separately.
- Every frame of every row needs its mask; the game recolours each frame
  on its own.

## Check before delivering

For each class, overlay the masks on the master (skin in magenta, hair in
cyan) and put a strip of those overlays in the review sheet. No marked pixel
may fall outside the figure, and no gear may be marked.

## Later

The high-resolution battle screen (E3) has landed, so the battle-idle sheets
of the 15 base classes and the 86 advanced and elite classes (A6, A8a–A10)
need the same masks, `art/source/A11/battle/units/<id>/idle.skin.png` and
`idle.hair.png`, each the size and layout of its master (1 × 4 frames). Only
player and companion classes need them; creatures (A7) don't. Deliver them
after the map masks, as a second batch of this order.
