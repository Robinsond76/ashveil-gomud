# A7: Enemies, summons and creatures

Read [00-standards.md](00-standards.md) first. Attach `scene-battle.png`
from A0 and two or three approved A6 battle idles for scale.

Every file is a **battle idle sheet**: 1 row × 4 frames, **facing right**
(the game mirrors enemies to face left), at the listed size class. Source
figure heights are in standards section 3: S ≥ 240 px, M ≥ 320, L ≥ 480,
XL ≥ 640.

**Enemies are ugly, feral, gaunt or uncanny, never comic.** Even small
creatures look like they can hurt you. Humanoid foes are worn and grim.
Beasts are animal-real.

**Variants** are the same creature with a different coat, gear or color.
Make the base first, then make the variant from the approved base as an
image reference, so the two read as kin.

## Deliverables (`art/source/A7/battle/units/<id>/idle.png`)

### Beasts and vermin

| ID | Size | Design and idle |
|---|---|---|
| `rat` | S | A mangy grey rat, sniffing. |
| `rat-big` | S | Variant of rat: larger, darker and scarred, teeth bared. |
| `wolf-timber` | M | A lean grey-brown timber wolf, head low, hackles raised. |
| `wolf-snow` | M | Variant: a white and blue-grey snow wolf. |
| `dog-junkyard` | M | A mangy mastiff in a spiked collar, growling. |
| `bats-echo` | S | A swarm of 3–4 leathery cave bats in one sprite, wings beating. |
| `snow-floof` | S | A squat, thick-furred tundra predator with hidden teeth. It looks soft and is dangerous. Not cute. |
| `crocodile` | L | A long, low crocodile with jaws slightly open; wide in its frame. |

### Spiders

| ID | Size | Design and idle |
|---|---|---|
| `spider-hatchling` | S | A small pale spider, legs twitching. |
| `spider-large` | M | A hairy brown spider with raised forelegs. |
| `spider-warrior` | M | Variant of spider-large: darker, with chitin plates and dull red markings. |
| `spider-queen` | XL | Boss: a bloated abdomen with egg sacs and crown-like spines, legs splayed wide. |

### Undead and cultists

| ID | Size | Design and idle |
|---|---|---|
| `skeleton` | M | A bare yellowed skeleton with a notched sword, swaying. |
| `bone-warden` | M | A skeleton in rusted plate behind a tower shield. |
| `bonecrafter` | M | A hunched skeleton with bone tools and a fetish of knuckle bones. |
| `lich` | L | Boss: a robed skeletal mage with a tarnished crown and pale green soulfire in its hand and eyes. |
| `acolyte-dark` | M | A black-robed cultist with a ritual dagger, the face hidden. |
| `grave-chanter` | M | A grey-robed chanter swinging a censer of grave smoke. |

### Bandits and outlaws

| ID | Size | Design and idle |
|---|---|---|
| `brigand` | M | A road brigand in leather and a hood, with a hand axe. |
| `ruffian` | M | A street thug with a club. |
| `ruffian-dangerous` | M | Variant: scarred, with a knife held low. |
| `ruffian-enforcer` | M | A big brute in padded armor with a cudgel on the shoulder. |
| `poacher` | M | A lake poacher in an oilskin coat with a short bow. |
| `poacher-shieldman` | M | A spear and a battered board shield, in a fishing-village jerkin. |
| `bonesetter` | M | A grim bandit healer with a satchel of splints and a bone saw. |
| `shadow-trainee` | M | A masked novice of a thieves' guild, with a dagger. |
| `shadow-master` | M | Variant: a black-clad master with twin blades. |

### Goblins, fey and growing things

| ID | Size | Design and idle |
|---|---|---|
| `goblin` | M | Small, sinewy and grey-green, in ragged hide, with a crude spear. Feral, not comic. |
| `goblin-hexer` | M | Variant: hung with fetishes, with a bone staff and a dim violet glow. |
| `goblin-shaman` | M | Variant: a bone mask and a smoking totem, beads and feathers. |
| `goblin-loot` | M | Variant: wiry and hunched under a heavy stolen sack. |
| `faerie` | S | Thin, pale and long-limbed fey with moth-like wings and an uncanny cold glow. |
| `imp-forest` | S | A bark-skinned imp with twig horns and long fingers. |
| `fungus` | M | A walking cluster of mushrooms that puffs spores. |
| `ent` | L | A treant with a bark body, branch arms and a mossy, ancient face. |
| `ogre-forest` | L | A hulking ogre with a tree-trunk club on the shoulder. |

### Caves and ice

| ID | Size | Design and idle |
|---|---|---|
| `creeper-cave` | M | A pale, eyeless crawler with many legs. |
| `creeper-abyssal` | M | Variant: black-violet with dim bioluminescent spots. |
| `stalker-cave` | M | A gaunt, clawed hunter with faintly glowing eyes. |
| `ice-warrior` | M | A frost-armored humanoid with an ice blade. |
| `ice-guardian` | L | A crystalline ice construct, a sentinel. |

### Training yard and the town watch

| ID | Size | Design and idle |
|---|---|---|
| `dummy-training` | M | A straw dummy on a post (idle: it creaks slightly). |
| `straw-footman` | M | A straw figure with a wooden sword. |
| `straw-archer` | M | Variant: a straw figure with a practice bow. |
| `guard` | M | A Frostfang city guard in mail and city livery, with a spear. |
| `guard-royal` | M | Variant: a king's guard in polished plate, with a halberd. |
| `guard-captain` | M | Variant: a plumed helm, sword and shield. |

### Company creatures, summons and pets

| ID | Size | Design and idle |
|---|---|---|
| `hound` | M | A recruited war dog: lean and muscular in a leather collar and harness. Alert. |
| `stone-golem` | M | A recruited construct of fitted stone blocks with faint rune seams, heavy and slow. |
| `doll` | S | The Doll Master's fighting doll: a jointed wooden figure, its strings running up out of the frame. Uncanny. |
| `angel` | L | The Hierarch's summoned angel: tall and armored, with grey-white wings and a pale, restrained glow. Solemn, not glittery. |
| `demon` | L | The Demonologist's bound demon: a gaunt horned brute with ember cracks in ash-black hide and a chain at the wrist. |
| `warhound` | M | The Houndmaster's warhound: heavier than the hound, in a studded collar and a scarred hide. |
| `war-bear` | L | The Bearward's war bear: a brown bear in a leather barding, standing guard. |
| `drake-hatchling` | M | The Dragon Tamer's drake: a young wingless drake, scaled green and ember, smoke at the nostrils. |

### Fallbacks for anything without art

| ID | Size | Design and idle |
|---|---|---|
| `unknown-humanoid` | M | A hooded, generic foe in muted, shadowed colors. Featureless but menacing. |
| `unknown-beast` | M | A generic four-legged beast in shadow. |
| `unknown-large` | L | A hulking generic shape in shadow. |

## Note for the lead

Today the battle screen draws `warhound` as `dog-junkyard`, the bear as
`unknown-large` and the drake as `unknown-beast` (`BEAST_SPRITES` in
`window-battle.js`). Point those at the new IDs when importing.
