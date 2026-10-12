# A15: Deaths on the battle screen

Read [00-standards.md](00-standards.md), [A12-battle-actions-pilot.md](A12-battle-actions-pilot.md)
and the masks section of [A11-look-masks.md](A11-look-masks.md#a11c-masks-for-the-action-poses-ordered-2026-10-11)
first. Attach each figure's approved battle idle (and, for player classes,
its idle masks) and two or three approved A12 or A12b sheets, so the motion
matches.

Today a figure that falls in battle is drawn as a flat placeholder shape.
This order gives every battle figure its own fall: one `death` sheet that
plays when it goes down, whose **last frame is the figure lying still**.
The battle screen holds that last frame for as long as the figure stays
down (a fallen companion can be raised again, and the screen then plays the
sheet backwards to stand it up).

**Tone:** grim but not gory. Figures crumple, fall and lie still; no
severed limbs, no pools of blood, no spatter. A hurt player character or
companion has *fallen*, not necessarily died: the last frame reads as a
body down and out of the fight.

## The pose

| Pose | Frames | What it shows |
|---|---|---|
| `death` | 6 | 1 the killing blow lands (as `hurt` frame 2), 2 staggering, 3 legs giving way, 4 falling, 5 striking the ground, 6 **lying still**: on the ground line, weapon dropped beside it. |

- The figure falls where it stands, backwards (to the left, away from the
  foe it faces) or crumpling forward; never out of its cell.
- Frame 6 lies along the idle's feet line, inside the cell.
- Some creatures don't fall like a body; draw what they would do:

| Who | Their death |
|---|---|
| `skeleton`, `bone-warden`, `bonecrafter` | Collapse into a heap of bones and rusted gear. |
| `lich` | The robe folds empty; the crown rolls; the soulfire gutters out. |
| `ice-warrior`, `ice-guardian` | Crack and shatter into a pile of ice shards. |
| `stone-golem` | Its blocks slump apart into a cairn of stone, the rune seams going dark. |
| `doll` | The strings go slack; it drops in a jointed heap. |
| `angel` | Kneels and fades into a pale, falling light, leaving only a few grey feathers. |
| `demon` | Burns down into embers and ash; the chain falls to the ground. |
| `fungus` | Deflates and slumps, a last puff of spores. |
| `bats-echo` | The swarm scatters and drops: a few small bodies on the ground. |
| `faerie` | Wings fold; it falls and lies like a dead moth. |
| `ent` | Topples like a felled tree, roots in the air. |
| `dummy-training`, `straw-footman`, `straw-archer` | The straw figure falls off its post or collapses, straw spilling. |
| the three `unknown-*` | The shadowed shape sinks into a dark smear on the ground. |

## Layout

Exactly as each figure's own battle idle and A12 poses: 1 row × 6 frames,
the idle's cell size, gutter width and art grain (640 px cells on a 5 px
grain for most figures; the creature's own size class otherwise), the
idle's scale, facing (right), colours and gear.

Deliver to `art/source/A15a/battle/units/<id>/death.png` (A15b:
`art/source/A15b/...`). For a player class marked with masks, also deliver
`art/source/A11/battle/units/<id>/death.skin.png` and `death.hair.png`,
exactly as A11c (white on real skin or hair only, consistent with the
idle masks). A helm or hood that comes off in the fall is fine: mask what
shows.

## A15a: the base classes and every creature (73 sheets, 28 masks)

- The 15 base classes: adventurer, alchemist, arbalist, beasttamer,
  cleric, dollmaster, gryphon-rider, halberdier, ranger, rogue, samurai,
  shaman, warrior, witch, wizard. Masks for all but the warrior (14 × 2).
  The gryphon rider falls from the saddle and the gryphon goes down beside
  it.
- All 58 A7 figures: every enemy, the fallbacks, the training figures and
  the company creatures and summons, with the deaths in the table above
  where they differ.

## A15b: the advanced and elite classes (86 sheets, 116 masks)

All 86 promoted classes (the A12d/A12e table lists them), each falling in
its own gear, with masks for the 58 whose idle has them. Order it after
A15a is reviewed.

## Check before delivering

For each figure, a strip of its idle frame 1 and the six death frames at
full size, and the sheet at game size on mid-grey and dark backgrounds;
the mask overlay strip for masked classes. Check that frame 6 sits on the
idle's feet line inside the cell, the figure's gear and colours match, and
nothing reads as gore. Keep review sheets under 5 MB.
