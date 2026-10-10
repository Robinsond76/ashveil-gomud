# A12: Battle action poses (pilot)

Read [00-standards.md](00-standards.md) and [A6-battle-base.md](A6-battle-base.md) first. Attach each class's approved A6 battle idle as its reference.

Today a battle figure only breathes. When it acts, the game slides the idle drawing: a step in, a nudge back when struck. This pilot draws real action poses for the **15 base classes**, so we can judge them in the game before ordering the other classes and the creatures. The battle screen already plays a pose sheet when one exists. The owner decided on 2026-10-10 to order the pilot after E3.

## What to deliver (`art/source/A12/battle/units/<id>/<pose>.png`)

| Pose | Who | Frames | What it shows |
|---|---|---|---|
| `attack` | every class | 6 | A melee blow with the class's weapon. 1 ready, 2 wind-up, 3 swing, 4 the blow at full reach (the hit lands here), 5 follow-through, 6 recover to the idle stance. |
| `hurt` | every class | 3 | Struck: 1 recoil (head and shoulders snap back, weight on the back foot), 2 the worst of it, 3 steadying. |
| `shoot` | ranger, arbalist, alchemist | 6 | A ranged attack. 1 draw or raise, 2–3 aim, 4 the release (bow, crossbow bolt, or the alchemist's throw), 5 follow-through, 6 recover. |
| `cast` | cleric, wizard, witch, shaman, dollmaster | 4 | Chanting, **looping**: hands and focus raised, a glow building and fading. The loop must be seamless; the game plays it while the spell is prepared and at its release. |

That's 15 `attack` + 15 `hurt` + 3 `shoot` + 5 `cast` = **38 sheets**.

The class ids are adventurer, alchemist, arbalist, beasttamer, cleric, dollmaster, gryphon-rider, halberdier, ranger, rogue, samurai, shaman, warrior, witch and wizard. The halberdier wears the approved glaive design.

## Layout

Exactly as the battle idles (standards section 4):

- **1 row × N frames** (the frame count from the table), each a **640×640 cell**, with **40 px transparent gutters** between frames. So a 6-frame sheet is 6×640 + 5×40 = 4040 px wide, and a 3-frame sheet is 2000 px.
- The same **5 px art grain** as the idle, aligned to each cell's top-left corner. Every art pixel is a 5×5 block of one colour; the game shrinks the sheet by exactly 5.
- The **same scale and feet line** as the class's idle: the body is 400 px from head to boots, and the feet stand at y≈579. A lunging foot may leave the ground line, but the standing foot stays on it.
- Facing **right**, with the same slight 3/4 view as the idle. Every piece of gear matches the idle.
- Keep the figure inside its cell. A swing may reach toward the cell's right edge but must not cross it; the game moves the whole figure toward its target itself, so don't draw travel across the cell.

## Check before delivering

For each class, put a strip of its idle frame 1 and every pose frame side by side at full size in the review sheet. Check that:
- the scale and gear match;
- the feet line holds;
- `attack` frame 4 reads as the moment of the blow;
- `cast` loops cleanly.

Also show each sheet shrunk to 128 px tall (the game's size) on a mid-grey and on a dark background. Check every frame for tears (standards section 4).
