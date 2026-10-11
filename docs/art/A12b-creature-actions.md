# A12b and A12c: Battle action poses for enemies and creatures

Read [00-standards.md](00-standards.md), [A7-enemies-and-creatures.md](A7-enemies-and-creatures.md)
and [A12-battle-actions-pilot.md](A12-battle-actions-pilot.md) first. Attach
each creature's approved **A7 battle idle** as its reference, and two or
three approved A12 sheets so the motion matches the classes'.

The A12 pilot gave the 15 base classes real attack, hurt, shoot and cast
poses. Every enemy, summon and company creature still only breathes: when
it acts, the game slides its idle drawing. This order gives them the same
poses. The battle screen and the importer already play and import a pose
sheet for any figure, so this needs no engineering; each sheet shows in the
game as soon as it is imported.

It comes in **two batches**, by how often players meet each figure in the
world. Deliver A12b first; it is reviewed and imported before A12c starts.

## The poses

Same names, frame counts and beats as A12:

| Pose | Frames | What it shows |
|---|---|---|
| `attack` | 6 | The creature's own blow (the "Attack" column below): 1 ready, 2 wind-up, 3 strike under way, 4 **the blow at full reach** (the hit lands here), 5 follow-through, 6 recover to the idle stance. |
| `hurt` | 3 | Struck: 1 recoil (head and body snap back, weight on the back legs or foot), 2 the worst of it, 3 steadying. |
| `shoot` | 6 | A ranged attack: 1 draw, 2–3 aim, 4 the release, 5 follow-through, 6 recover. |
| `cast` | 4 | Chanting, **looping**: hands, staff or focus raised, a glow building and fading. Seamless loop. |

Beasts don't swing weapons: their blow is a bite, a claw, a lunge or a
slam, as listed. Keep them animal-real: a wolf's bite is a snap of the head
and shoulders with the hind legs planted, not a cartoon leap. A swarm
(`bats-echo`) surges forward as a group and scatters back. The rules of
A7 still hold: ugly, feral, gaunt or uncanny, never comic.

## Layout

Exactly as the creature's own A7 idle:

- **1 row × N frames** (N from the pose table), each frame the **same cell
  size as its idle**, with the **same transparent gutter width as its idle
  sheet** between frames.
- The **same art grain** as its idle, aligned to each cell's top-left
  corner: S units 768 px cells on a 6 px grain; M units 640 px on 5 px, or
  1024 px on 8 px, whichever the idle uses; L units 1728 px on 12 px; XL
  units 1536 px on 8 px. The importer refuses art off its grain or on a
  cell its size class doesn't use.
- The **same scale, ground line and facing (right)** as the idle. Colours,
  markings, gear and size match the idle exactly. A lunging foot or paw may
  leave the ground line; the planted ones stay on it.
- Keep the creature inside its cell. A strike may reach toward the cell's
  right edge but must not cross it; the game moves the whole figure toward
  its target itself.
- **No look masks:** creatures have no player-chosen skin or hair.

Deliver to `art/source/A12b/battle/units/<id>/<pose>.png` (and
`art/source/A12c/...` for the second batch), using the A7 ids.

## A12b: the creatures met most (25 figures, 53 sheets)

Every figure gets `attack` and `hurt`; the extra column lists `cast` or
`shoot` where the creature uses it in the game.

| ID | Size | Attack | Extra |
|---|---|---|---|
| `spider-hatchling` | S | A darting bite, forelegs raised. | |
| `fungus` | M | A lurching slam, spores bursting from the caps on the blow. | |
| `skeleton` | M | A hacking cut with its notched sword. | |
| `crocodile` | L | A sideways snap of the jaws, the tail swinging for balance. | |
| `snow-floof` | S | The fur parts and the hidden jaws strike. | |
| `imp-forest` | S | A raking swipe of its long fingers. | |
| `ruffian` | M | An overhead club blow. | |
| `rat` | S | A quick lunging bite. | |
| `rat-big` | S | A heavier lunging bite, shoulders hunched. | |
| `wolf-snow` | M | A snapping bite, head and shoulders driving forward. | |
| `wolf-timber` | M | The same bite as the snow wolf; draw it first and the snow wolf from it. | |
| `goblin-hexer` | M | A jab with the bone staff. | `cast`: the staff raised, the violet glow pulsing. |
| `bats-echo` | S | The swarm surges forward together, then scatters back. | |
| `spider-warrior` | M | A stab with both forelegs, body rearing. | |
| `acolyte-dark` | M | A ritual dagger thrust. | |
| `ruffian-dangerous` | M | A low knife slash. | |
| `poacher` | M | A short jab with the bow's end. | `shoot`: the short bow drawn and loosed. |
| `dog-junkyard` | M | A growling lunge and bite. | |
| `bonesetter` | M | A swing of the bone saw. | `cast`: a hand raised over the satchel, a dull red-brown glow (its healing). |
| `ice-warrior` | M | A slash with the ice blade. | |
| `stalker-cave` | M | A two-clawed rake, the eyes flaring. | |
| `spider-large` | M | A rearing stab of the forelegs. | |
| `unknown-humanoid` | M | A plain overhead strike, in shadow. | |
| `unknown-beast` | M | A plain lunging bite, in shadow. | |
| `unknown-large` | L | A heavy downward slam, in shadow. | |

25 `attack` + 25 `hurt` + 2 `cast` + 1 `shoot` = **53 sheets**. The three
`unknown-*` fallbacks are drawn for every enemy without art of its own, so
they come in this first batch.

## A12c: the rest (33 figures, 69 sheets)

| ID | Size | Attack | Extra |
|---|---|---|---|
| `spider-queen` | XL | A crushing stab of the forelegs, the abdomen rocking. | |
| `bone-warden` | M | A shield bash, then the blade. | |
| `bonecrafter` | M | A swing of its bone tools. | |
| `lich` | L | A clawed hand sweep, soulfire trailing. | `cast`: soulfire gathering in the raised hand and eyes. |
| `grave-chanter` | M | A swing of the smoking censer. | `cast`: the censer raised, grave smoke pulsing. |
| `brigand` | M | A hand-axe chop. | |
| `ruffian-enforcer` | M | The cudgel brought down off the shoulder. | |
| `poacher-shieldman` | M | A spear thrust over the shield. | |
| `shadow-trainee` | M | A dagger stab. | |
| `shadow-master` | M | A crossing cut of the twin blades. | |
| `goblin` | M | A spear thrust. | |
| `goblin-shaman` | M | A jab with the totem. | `cast`: the totem raised, its smoke thickening. |
| `goblin-loot` | M | A swing of the heavy sack. | |
| `faerie` | S | A darting rake of thin fingers, wings flaring. | |
| `ent` | L | A sweeping branch-arm blow. | |
| `ogre-forest` | L | The tree-trunk club brought down. | |
| `creeper-cave` | M | A lunge of the front legs. | |
| `creeper-abyssal` | M | The same lunge; draw it from the cave creeper. | |
| `ice-guardian` | L | A heavy crystalline slam. | |
| `dummy-training` | M | **No attack** (it never strikes): `hurt` only, rocking on its post. | |
| `straw-footman` | M | A stiff wooden-sword swing. | |
| `straw-archer` | M | A stiff jab with the practice bow. | `shoot`: the practice bow drawn and loosed. |
| `guard` | M | A spear thrust. | |
| `guard-royal` | M | A halberd chop. | |
| `guard-captain` | M | A sword cut behind the shield. | |
| `hound` | M | A lunging bite. | |
| `warhound` | M | A heavier lunging bite. | |
| `stone-golem` | M | A slow, heavy two-fisted slam. | |
| `doll` | S | Its strings jerk it into a stab. | |
| `angel` | L | A solemn sword stroke, wings opening. | |
| `demon` | L | A clawed swing, the wrist chain whipping. | |
| `war-bear` | L | A rearing paw swipe. | |
| `drake-hatchling` | M | A snapping bite, smoke at the nostrils. | |

32 `attack` + 33 `hurt` + 3 `cast` + 1 `shoot` = **69 sheets**.

## Check before delivering

As in A12: for each figure, a strip of its idle frame 1 and every pose
frame side by side at full size in the review sheet. Check that the scale,
colours and markings match, the ground line holds, `attack` frame 4 reads
as the moment of the blow, and `cast` loops cleanly. Show each sheet shrunk
to game size on mid-grey and on dark backgrounds, and check every frame for
tears (standards section 4).
