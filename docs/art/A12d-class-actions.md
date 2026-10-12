# A12d and A12e: Battle action poses for the advanced and elite classes

Read [00-standards.md](00-standards.md), [A12-battle-actions-pilot.md](A12-battle-actions-pilot.md)
and the masks section of [A11-look-masks.md](A11-look-masks.md#a11c-masks-for-the-action-poses-ordered-2026-10-11) first.
Attach each class's approved battle idle (A8a, A8b, A9 or A10) and its idle
masks, and two or three approved A12 sheets of the same line, so the motion
and the masks match.

The A12 pilot gave the 15 base classes attack, hurt, shoot and cast poses,
and A11c gave them look masks, so a player's chosen skin and hair hold
through every blow. The 86 advanced and elite classes still only breathe:
when one acts, the game slides its idle drawing. This order gives them the
same poses and masks. The importer and the battle screen already handle
them, so each sheet shows in the game as soon as it is imported.

It comes in **two batches**: A12d, the 43 advanced classes (A8a and A8b);
then A12e, the 43 elite classes (A9 and A10). Deliver A12d first; it is
reviewed and imported before A12e starts.

## The poses

Exactly the A12 poses, frame counts and beats:

| Pose | Frames | What it shows |
|---|---|---|
| `attack` | 6 | A blow with the class's own weapon, as its idle carries it: 1 ready, 2 wind-up, 3 strike under way, 4 **the blow at full reach** (the hit lands here), 5 follow-through, 6 recover to the idle stance. |
| `hurt` | 3 | Struck: 1 recoil, 2 the worst of it, 3 steadying. |
| `shoot` | 6 | The ranger, arbalist and alchemist lines: 1 draw or raise, 2–3 aim, 4 the release (bow, crossbow bolt, or a thrown flask), 5 follow-through, 6 recover. |
| `block` or `parry` | 3 | One of the two for every class, by what its idle carries: `block` for a shield-bearer, `parry` for a melee weapon and no shield, none for neither. As described in [A12f](A12f-defense-poses.md). Not listed in the tables below; list which each class got in the delivery notes. |
| `cast` | 4 | The cleric, wizard, witch, shaman and dollmaster lines, and the knight and paladin (who heal by laying on hands): chanting, **looping**, hands, staff or focus raised and a glow building and fading in the class's own colour (a censer's smoke, soulfire, storm light, the doll strings tightening). Seamless loop. |

Each class's blow is its own: a knight's sword over its shield, a
blackguard's jagged axe, a duelist's thrust, a hatamoto's guarding cut, a
sweeper's wide glaive arc. The knight's and paladin's `cast` is the laying on
of hands: the sword lowered, a gauntleted hand raised with a pale, warm glow. Mounted classes (the gryphon-rider line) strike
from the saddle, the mount lunging with the rider; keep the mount on the
idle's ground line. The beast tamer line strikes itself; its beasts have
their own A12c sheets.

## Layout

Exactly as each class's own battle idle: 1 row × N frames, each the
**idle's cell size** (640×640 for most) with the **idle's gutter width**
(40 px at 640), on the **idle's art grain** (5 px at 640) aligned to the
cell's top-left corner, with the idle's scale, feet line, facing (right) and
gear. Keep the figure inside its cell. The importer refuses a pose whose
frame differs from its idle's.

Deliver to `art/source/A12d/battle/units/<id>/<pose>.png` (A12e:
`art/source/A12e/...`).

## Masks

For every class marked *yes* below, deliver its masks with the poses, as
A11c did: `art/source/A11/battle/units/<id>/<pose>.skin.png` and
`<pose>.hair.png` for each pose sheet, exactly the sheet's size and layout,
on its grain, opaque white on real skin or hair only, everything else
fully transparent, no overlap, nothing outside the opaque art. Keep each
class's choices consistent with its idle masks. Classes marked *no* have no
idle masks (a full helm or hood hides both), so their poses need none.

## A12d: the advanced classes (43 classes, 112 sheets, 158 masks)

| ID | Line | Poses | Masks |
|---|---|---|---|
| `apothecary` | alchemist | attack, hurt, shoot | yes |
| `bombardier` | alchemist | attack, hurt, shoot | yes |
| `mutagenist` | alchemist | attack, hurt, shoot | yes |
| `sharpshooter` | arbalist | attack, hurt, shoot | yes |
| `siegebreaker` | arbalist | attack, hurt, shoot | yes |
| `warden-of-the-wall` | arbalist | attack, hurt, shoot | yes |
| `bearward` | beasttamer | attack, hurt | yes |
| `dragon-tamer` | beasttamer | attack, hurt | yes |
| `houndmaster` | beasttamer | attack, hurt | yes |
| `blood-priest` | cleric | attack, hurt, cast | yes |
| `druid` | cleric | attack, hurt, cast | yes |
| `priest` | cleric | attack, hurt, cast | yes |
| `golemancer` | dollmaster | attack, hurt, cast | yes |
| `marionettist` | dollmaster | attack, hurt, cast | yes |
| `puppeteer` | dollmaster | attack, hurt, cast | yes |
| `gryphon-knight` | gryphon-rider | attack, hurt | no (helm or hood hides skin and hair) |
| `skyscout` | gryphon-rider | attack, hurt | yes |
| `wyvern-rider` | gryphon-rider | attack, hurt | no (helm or hood hides skin and hair) |
| `sweeper` | halberdier | attack, hurt | yes |
| `valkyrie` | halberdier | attack, hurt | yes |
| `vanguard` | halberdier | attack, hurt | yes |
| `hunter` | ranger | attack, hurt, shoot | yes |
| `stalker` | ranger | attack, hurt, shoot | no (helm or hood hides skin and hair) |
| `warden` | ranger | attack, hurt, shoot | yes |
| `assassin` | rogue | attack, hurt | no (helm or hood hides skin and hair) |
| `duelist` | rogue | attack, hurt | yes |
| `scout` | rogue | attack, hurt | no (helm or hood hides skin and hair) |
| `hatamoto` | samurai | attack, hurt | no (helm or hood hides skin and hair) |
| `kensai` | samurai | attack, hurt | no (helm or hood hides skin and hair) |
| `ronin` | samurai | attack, hurt | no (helm or hood hides skin and hair) |
| `earthspeaker` | shaman | attack, hurt, cast | no (helm or hood hides skin and hair) |
| `mistweaver` | shaman | attack, hurt, cast | no (helm or hood hides skin and hair) |
| `stormcaller` | shaman | attack, hurt, cast | no (helm or hood hides skin and hair) |
| `blackguard` | warrior | attack, hurt | no (helm or hood hides skin and hair) |
| `knight` | warrior | attack, hurt, cast | no (helm or hood hides skin and hair) |
| `mercenary` | warrior | attack, hurt | no (helm or hood hides skin and hair) |
| `coven-sage` | witch | attack, hurt, cast | yes |
| `hag` | witch | attack, hurt, cast | yes |
| `hedge-witch` | witch | attack, hurt, cast | yes |
| `arcanist` | wizard | attack, hurt, cast | yes |
| `sorcerer` | wizard | attack, hurt, cast | yes |
| `theurgist` | wizard | attack, hurt, cast | yes |
| `warlock` | wizard | attack, hurt, cast | yes |

## A12e: the elite classes (43 classes, 112 sheets, 158 masks)

| ID | Line | Poses | Masks |
|---|---|---|---|
| `grenadier` | alchemist | attack, hurt, shoot | yes |
| `panacean` | alchemist | attack, hurt, shoot | yes |
| `transmuter` | alchemist | attack, hurt, shoot | yes |
| `bastion` | arbalist | attack, hurt, shoot | no (helm or hood hides skin and hair) |
| `deadeye` | arbalist | attack, hurt, shoot | yes |
| `siege-master` | arbalist | attack, hurt, shoot | yes |
| `beastlord` | beasttamer | attack, hurt | yes |
| `dragon-lord` | beasttamer | attack, hurt | yes |
| `packlord` | beasttamer | attack, hurt | yes |
| `demonologist` | cleric | attack, hurt, cast | yes |
| `elder-druid` | cleric | attack, hurt, cast | yes |
| `hierarch` | cleric | attack, hurt, cast | yes |
| `golem-lord` | dollmaster | attack, hurt, cast | yes |
| `grand-puppeteer` | dollmaster | attack, hurt, cast | yes |
| `string-sovereign` | dollmaster | attack, hurt, cast | yes |
| `falcon-marshal` | gryphon-rider | attack, hurt | yes |
| `gryphon-lord` | gryphon-rider | attack, hurt | no (helm or hood hides skin and hair) |
| `wyvern-lord` | gryphon-rider | attack, hurt | no (helm or hood hides skin and hair) |
| `linebreaker` | halberdier | attack, hurt | no (helm or hood hides skin and hair) |
| `reaper` | halberdier | attack, hurt | yes |
| `tempest-lancer` | halberdier | attack, hurt | no (helm or hood hides skin and hair) |
| `marksman` | ranger | attack, hurt, shoot | yes |
| `ravager` | ranger | attack, hurt, shoot | yes |
| `sentinel` | ranger | attack, hurt, shoot | yes |
| `nightblade` | rogue | attack, hurt | no (helm or hood hides skin and hair) |
| `pathfinder` | rogue | attack, hurt | yes |
| `swordmaster` | rogue | attack, hurt | yes |
| `kenshi` | samurai | attack, hurt | no (helm or hood hides skin and hair) |
| `shogun` | samurai | attack, hurt | no (helm or hood hides skin and hair) |
| `sword-saint` | samurai | attack, hurt | yes |
| `mountain-speaker` | shaman | attack, hurt, cast | no (helm or hood hides skin and hair) |
| `tempest-lord` | shaman | attack, hurt, cast | no (helm or hood hides skin and hair) |
| `veil-mother` | shaman | attack, hurt, cast | no (helm or hood hides skin and hair) |
| `dread-knight` | warrior | attack, hurt | no (helm or hood hides skin and hair) |
| `paladin` | warrior | attack, hurt, cast | no (helm or hood hides skin and hair) |
| `warlord` | warrior | attack, hurt | no (helm or hood hides skin and hair) |
| `coven-mother` | witch | attack, hurt, cast | yes |
| `crone-of-ash` | witch | attack, hurt, cast | yes |
| `wise-one` | witch | attack, hurt, cast | yes |
| `archmage` | wizard | attack, hurt, cast | yes |
| `archon` | wizard | attack, hurt, cast | yes |
| `high-sorcerer` | wizard | attack, hurt, cast | yes |
| `necromancer` | wizard | attack, hurt, cast | yes |

## Check before delivering

As in A12 and A11c: for each class, a strip of its idle frame 1 and every
pose frame side by side at full size; the same strip with the magenta/cyan
mask overlay for the masked classes; each sheet shrunk to game size on
mid-grey and dark backgrounds. Check that the scale, gear and feet line
match the idle, `attack` frame 4 reads as the blow, `cast` loops cleanly,
and every frame is free of tears (standards section 4). Keep review sheets
under 5 MB each.
