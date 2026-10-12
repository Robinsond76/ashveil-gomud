# A12f: Block and parry poses

Read [00-standards.md](00-standards.md), [A12-battle-actions-pilot.md](A12-battle-actions-pilot.md)
and the masks section of [A11-look-masks.md](A11-look-masks.md#a11c-masks-for-the-action-poses-ordered-2026-10-11)
first. Attach each figure's approved battle idle (and idle masks for player
classes) and its approved `attack` and `hurt` sheets.

When a blow is stopped, the battle screen shows how: a figure with a shield
**blocks** it, a figure with a weapon that can turn a blow **parries** it,
and anyone else twists aside. Today a block or parry plays the `hurt`
sheet, so a well-made defense looks like taking the hit. This order gives
the figures that block or parry their own pose. The battle screen already
plays these sheets when they exist (and falls back to `hurt` when they
don't), so each one shows as soon as it is imported.

## The poses

Each figure gets **one** of the two, by what its idle carries:

| Pose | Who | Frames | What it shows |
|---|---|---|---|
| `block` | A figure carrying a **shield** (in the game a shield-bearer only blocks) | 3 | 1 the shield brought up square to the blow, weight on the back foot, 2 **the blow lands on the shield**: the arm braced, the figure driven back a little, 3 recovering behind the shield. |
| `parry` | A figure holding a **melee weapon** that can turn a blow (sword, axe, mace, spear, glaive, staff, dagger, club) and no shield | 3 | 1 the weapon swept across to meet the blow, 2 **the blades meet**: the attacker's line turned aside, the figure's weight shifting, 3 recovering to guard. |

A figure with neither (beasts, the unarmed, bowmen without a melee weapon,
the swarm) gets no sheet: it dodges, and keeps its `hurt` sheet for that.
Draw the contact on frame 2 without a painted spark or flash; the game adds
its own hit effect there (A16a `block` and `parry`).

## Layout

Exactly as the figure's own idle and A12 poses: 1 row × 3 frames, the
idle's cell size, gutter width and art grain, scale, feet line, facing
(right), colours and gear. Deliver to
`art/source/A12f/battle/units/<id>/<pose>.png`. For a player class with
idle masks, also deliver `art/source/A11/battle/units/<id>/<pose>.skin.png`
and `<pose>.hair.png`, exactly as A11c.

## Who

- **The 15 base classes:** choose `block` or `parry` for each from its idle
  (the warrior's shield blocks; the halberdier's glaive, the samurai's
  blade and the rogue's daggers parry; casters parry with their staff or
  mace if they carry one). Masks for every class but the warrior.
- **The armed A7 figures** (A12b and A12c): every humanoid enemy and
  company creature that holds a weapon or shield in its idle: the
  skeletons and undead, cultists, bandits, poachers, the shadow guild,
  goblins, the ice warrior, the guards, the straw footman, the angel and
  the `unknown-humanoid` fallback. Beasts, spiders, creepers, the fungus,
  the bats and other unarmed creatures get none.
- In the delivery notes, list which pose each figure got, so the import
  can be checked against it.

The advanced and elite classes get theirs in A12d and A12e.

## Check before delivering

For each figure, a strip of its idle frame 1, its `hurt` frame 2 and the
three defense frames at full size, and the sheet at game size on mid-grey
and dark backgrounds; the mask overlay strip for masked classes. Check the
shield or weapon meets the blow on frame 2, the feet line and gear match
the idle, and the figure stays inside its cell. Keep review sheets under
5 MB.
