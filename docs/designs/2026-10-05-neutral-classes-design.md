# Neutral classes: eight new lineages with no good or evil path

Status: **owner-approved design (2026-10-05)**. The owner chose six
classes from a brainstorm in the project thread, then the Alchemist and
Arbalist from a second one, and answered every open question (see
**Owner answers** at the end; handoff rule 20). Every number is a starting
value for the balance harness. Builds on
the [branching class progression](2026-10-01-branching-class-progression-design.md),
[level impact and the Witch](2026-10-05-level-impact-class-power-design.md),
[35a2 skill over hit points](2026-10-05-phase-35a2-skill-over-hit-points-design.md),
[30f battlefield conditions](2026-10-02-phase-30f-battlefield-design.md) and
the approved [faith routes](2026-10-05-faith-routes-design.md).

## Owner direction (2026-10-05)

- A **glaive** class.
- More classes that are **neutral from the beginning**, with no good or evil
  path, inspired by **Ogre Battle** and **Unicorn Overlord**.
- The owner liked all six proposed classes and asked for this design, then
  added the **Alchemist** and **Arbalist** from a second batch.

## Summary

Each is a **new base lineage chosen at creation**, like the Witch. None has
an alignment gate at creation, at level 10 or at level 30; their routes
split by fighting style instead.

| Class | Inspiration | Row | Role | What only it does |
|---|---|---|---|---|
| **Halberdier** | Unicorn Overlord polearm fighters | Front or middle | Melee area damage | Sweeps a row with a glaive; braces to strike the first foe at its column |
| **Doll Master** | Ogre Battle's Doll Master | Back (doll in front) | Tank and controller | Drives a durable doll that stands in its own cell and acts on the Master's turn |
| **Beast Tamer** | Ogre Battle's Beast and Dragon Tamers | Middle | Striker with a beast | Raises a bonded beast that acts on its own turn and grows with the Tamer |
| **Gryphon Rider** | Unicorn Overlord's Gryphon Knight | Front | Back-line striker | Dives over a standing front line to hit a rear foe |
| **Samurai** | Ogre Battle's Samurai, Unicorn Overlord's Swordfighter | Front | Duelist | Acts first and strikes hardest with its opening blow |
| **Shaman** | Unicorn Overlord's Shaman | Back | Battlefield caster | Calls battle weather (fog, cold wind, rain) that changes both sides' odds |
| **Alchemist** | Original (camp consumables design) | Middle | Healer and buffer | Heals and buffs from brewed flasks, with no faith and no mana |
| **Arbalist** | Original (crossbows in the equipment catalog) | Back | Anti-armor ranged | Armor-piercing bolts that need a reload turn |

The rest of the second batch is listed in section 10 for later.

## 1. Rules shared by the neutral lineages

- **No alignment gates.** Promotion at 10 and 30 checks level only. Alignment
  still moves and still matters for recruiting and companion loyalty (32a2,
  33h); it never blocks these routes.
- **Same progression as every lineage.** Talents at 5, 15, 25 …; promotion at
  10 (advanced) and 30 (elite); **a rank every 5 levels** to 60 (faith routes,
  "Ranks"). Routes are final at launch, as for the faith routes.
- **Automatic play.** Every ability below is used by the character's
  strategy, never typed mid-battle (Ogre Battle rule). Each class names its
  strategy role.
- **Formation is locked in battle.** No ability moves a member to another
  cell during a fight (`internal/formationcombat` relies on this).
- **Companions too.** Each class joins the recruit rosters (32a2) with
  companion spell/ability levels (`CompanionLevels`), like the five base
  classes.
- **Polearm reach is not class-locked.** Any character holding a glaive or
  war spear already gets extended reach (`ResolveReach`, `ReachExtended`:
  the target column's frontmost foe or the one behind it). The Halberdier's
  identity comes from its skills, not from reach.

Archetype numbers (35a2 fields, starting values):

| Class | HPStart | HPPerLevel | Attack | Evasion | Armor | Shield | Weapons | Growth (weights) |
|---|---|---|---|---|---|---|---|---|
| Halberdier | 7 | 0.9 | 1.0 | 0.9 | medium | none | glaives, war spears | Strength 4, Vitality 3, Speed 2, Perception 1 |
| Doll Master | 1 | 0.55 | 0.7 | 0.8 | light | none | daggers, rods | Smarts 4, Perception 3, Speed 2, Mysticism 1 |
| Beast Tamer | 5 | 0.8 | 0.9 | 0.9 | medium | none | whips, spears, daggers | Perception 3, Vitality 3, Strength 2, Smarts 2 |
| Gryphon Rider | 5 | 0.8 | 1.0 | 1.0 | medium | buckler | spears, lances, swords | Speed 4, Strength 3, Perception 2, Vitality 1 |
| Samurai | 6 | 0.85 | 1.1 | 1.0 | medium | none | swords (one or two hands) | Speed 4, Strength 3, Perception 2, Smarts 1 |
| Shaman | 2 | 0.6 | 0.7 | 0.8 | light | none | staffs, rods | Mysticism 4, Vitality 2, Perception 2, Smarts 2 |
| Alchemist | 3 | 0.65 | 0.75 | 0.85 | light | none | daggers, staffs, slings | Smarts 4, Perception 2, Vitality 2, Speed 2 |
| Arbalist | 4 | 0.75 | 1.0 | 0.8 | medium | none | crossbows | Perception 4, Strength 3, Vitality 2, Speed 1 |

## 2. Halberdier (the glaive class)

A polearm fighter for the front or middle row. Two hands on the glaive, so
no shield; medium armor keeps it quicker than a warrior. Strategy role:
`striker` (prefers the most crowded enemy row).

| Level | Gains |
|---|---|
| 1 | **Sweep** (signature): one swing strikes the target and one foe beside it in the same row, each at 80% damage, each defending separately. Cooldown 3 rounds. Reuses the 30f sweep pipeline (each member struck at most once) |
| 3 | **Brace:** when the Halberdier's turn comes and no foe in its reach has struck yet this round, it may hold the turn; the first foe that strikes into its column takes the held blow at +25% damage. It spends the Halberdier's turn, so no free action |
| 5 | Talent |
| 6 | **Hook:** a glaive hit has a 20% chance to knock down a foe that is leaping, flying or mounted (40% at 20) |
| 8 | Sweep reaches every foe in the target's row |
| 10 | Promotion |

| Route | Elite | Signature (rank 10) | Elite signature (30) | Capstone (60) |
|---|---|---|---|---|
| **Sweeper** | **Reaper** | Sweep at 100% damage | Sweep also strikes the row behind the target at 50% | Sweep cooldown 2 rounds |
| **Vanguard** | **Linebreaker** | Brace protects the whole column; the held blow knocks down on a hit | Allies in its column take 10% less damage while it stands | Brace can trigger twice a round |
| **Valkyrie** | **Tempest Lancer** | **Charged Sweep:** spends mana (8) to add lightning damage (1d6 at 10, scaling by rank) that ignores armor | Lightning arcs to one foe in the next row | Charged Sweep paralyzes for 1 round (bosses resist; Witch no-loop rules apply) |

Kit: militia glaive (equipment tiers catalog), padded jerkin, rations,
waterskin, satchel. The glaive progression ends at the **Ashen Reaper**
relic, a natural Reaper prize.

## 3. Doll Master

Fights through a wooden doll. The Master stands in the back row; the doll
stands in a cell the player chooses, usually the front. The Master spends
**no mana**: its limits are the doll's health and the materials to repair
it. Strategy role: `controller`.

### The doll

- **A formation member that takes no company slot.** The company stays
  leader + 4; the doll takes one of the nine cells, like a summon, but it is
  **durable and saved** (unlike the faith routes' summons). Proposed: the
  Doll Master's formation editor places it; if the formation is full it
  stands in front of the Master.
- **No turn of its own.** It acts only when the Master acts. The company's
  number of actions doesn't grow.
- **Health and skill from the Master.** HP: 70% of a warrior's HP at the
  Master's level. Attack: the Master's level × 0.9. It never dodges
  (Evasion 0) but can parry with a weapon and is protected by armor.
- **Gear.** It wears armor and wields a melee weapon from the company's
  cargo. Being wooden, it ignores armor bulk's tempo penalty (it has no
  tempo), so heavy armor makes a sturdy doll without slowing the Master.
- **Breaking.** At 0 HP the doll breaks: out for the rest of the battle,
  never destroyed. It returns repaired after a camp rest at a cost (Mend
  below). If the Master falls, the doll goes limp until the battle ends or
  the Master is raised.
- **No alignment, no experience, no food.** It isn't a person, so it
  doesn't count for recruiting, morale or drift, and doesn't eat.
- **Persistence.** Saved on the Master (player or companion record): doll
  name, current HP, broken flag, cell, worn gear. Survives copyover and
  restart; gear on a doll is never lost when it breaks.

### Abilities

| Level | Gains |
|---|---|
| 1 | **Puppet Strike** (signature): the doll attacks a legal target from its cell with its weapon's reach. This is the Master's normal action |
| 1 | **Mend** (camp utility): repairs the doll during a camp rest for **doll parts** (a new good, sold in towns and salvaged from constructs); more HP per part as the Master levels |
| 5 | Talent. **Guard String:** the doll guards the most hurt ally next to it (the 30c2 guardian rule), twice a battle |
| 8 | Guard String three times a battle |
| 10 | Promotion |
| 12 | **Tangle:** strings snag one foe in reach of the doll; its action meter is pushed back by 50 (half a turn; 25 for bosses). Cooldown 3 rounds. After a Tangle, that foe can't be tangled again for 2 rounds |
| 18 | **Emergency Splice:** once a battle, when the doll would break, the Master spends its next turn and the doll stands back up at 25% HP |

The design doesn't move members in battle, so the brainstorm's "Switch"
(swapping the doll with a hurt ally) is replaced by Emergency Splice.

| Route | Elite | Signature (rank 10) | Elite signature (30) | Capstone (60) |
|---|---|---|---|---|
| **Puppeteer** | **Grand Puppeteer** | **Two dolls**, each with 50% of the doll HP; Puppet Strike drives one, Guard String works for both | Each doll has 65% HP | A Puppet Strike hits with both dolls |
| **Golemancer** | **Golem Lord** | **Golem doll:** 130% HP, armor +15, but it can't wear armor (its own body is its armor); Guard String 4 times a battle | The golem's blows knock down 20% of the time | The golem stands back up once a battle at 50% HP, on top of Emergency Splice |
| **Marionettist** | **String Sovereign** | Tangle hits 2 foes and its cooldown is 2 rounds | **Cut Strings:** a tangled foe's next attack has −15 Attack | Tangle hits 3 foes and pushes back 75 |

The Golemancer's doll is the Master's own construct; it doesn't depend on
the future creature recruits (Stone and Iron Golems in the expanded
catalogue).

## 4. Beast Tamer

Raises one bonded beast from level 1. The difference from the doll: the
beast is **alive**. It acts on its own turn (with its own action meter), can
panic in a morale break, needs food, and can't wear gear. Strategy role:
`striker`.

### The bonded beast

- **A formation member that takes no company slot**, durable and saved, like
  the doll.
- **Own turn,** at 80% of a normal tempo, with natural weapons (bite 1d8 at
  first). Its HP is 60% of a warrior's at the Tamer's level, and its Attack
  and Evasion use the Tamer's level at rate 0.9.
- **Food.** It eats one ration a day from company cargo (32f order: cargo,
  then the Tamer's pack).
- **Falling.** A fallen beast is out for the battle and wounded (30b). It
  never dies permanently; it recovers with rest like a member's wounds.
- **Persistence.** Saved on the Tamer: kind, name, HP, wounds, cell.

| Level | Gains |
|---|---|
| 1 | **Sic** (signature): the Tamer's action sends the beast at the Tamer's target with +10 Attack on its next strike. The Tamer's own whip has extended reach (whips count as polearm reach) |
| 3 | **Rally:** the Tamer heals its beast for a Minor Heal's worth, no mana, twice a battle |
| 5 | Talent |
| 8 | **Pack sense:** while the beast stands, the Tamer has +5 Evasion |
| 10 | Promotion. When creature recruits ship (expanded catalogue), every beast member in the company gains +10% damage while a Tamer stands |

| Route | Elite | Beast | Signature (rank 10) | Elite signature (30) | Capstone (60) |
|---|---|---|---|---|---|
| **Houndmaster** | **Packlord** | Warhound | The hound's bites hobble a fleeing or wounded foe | A second hound joins | Hounds strike first in each battle |
| **Bearward** | **Beastlord** | War bear | The bear guards the Tamer and one ally (30c2), with 120% HP | The bear's swipes hit two adjacent foes | The bear stands back up once a battle at 50% HP |
| **Dragon Tamer** | **Dragon Lord** | Drake hatchling | **Breath:** every 3 rounds, fire on the target and its neighbours (the 30f `sparks` shape) | The drake grows: 120% HP and armor +10 | Breath every 2 rounds |

This class replaces the expanded catalogue's ranger **Beastkeeper →
Beastmaster** route, whose job it takes over.

## 5. Gryphon Rider

Rides a gryphon into battle and dives on the enemy's back line. Strategy
role: `skirmisher` (prefers casters and healers in the rear).

- **The gryphon is the rider's mount,** in the riding-horse slot from 32f
  (one riding mount per member). It doesn't take a formation cell or act on
  its own; its strength shows through the rider's abilities. It eats twice
  a horse's feed from cargo. It's saved like a horse.
- **Flying.** Ranged attacks against the rider get +10 Attack. Indoors, in
  caves and on narrow ground (30f `narrow` tag, indoor rooms), the gryphon
  can't fly: no Dive, and the rider fights as an ordinary spear fighter.

| Level | Gains |
|---|---|
| 1 | **Dive** (signature): an attack that may pass a **standing** front-row foe to strike the middle or back row in lateral range (the 30f leap, without leap's "front cell open" condition). Cooldown 3 rounds. After a Dive, the rider has −10 Evasion until its next turn. Guardians can still intercept |
| 3 | **Talons:** a Dive that hits also makes the target bleed (existing status) |
| 5 | Talent |
| 8 | Dive deals +25% damage |
| 10 | Promotion |

| Route | Elite | Signature (rank 10) | Elite signature (30) | Capstone (60) |
|---|---|---|---|---|
| **Gryphon Knight** | **Gryphon Lord** | A Dive with a lance knocks down | Dive cooldown 2 rounds | A Dive knocks down every foe beside the target |
| **Skyscout** | **Falcon Marshal** | Spots ambushes from the air: the company's ambush evasion rises (as the ranger's tracker utility) and the rider throws javelins from any row | Allies get +5 Attack against the foe it dove on that round | The company always acts first in an ambush it would have suffered |
| **Wyvern Rider** | **Wyvern Lord** | Swaps the gryphon for a wyvern; Dive poisons (buff 13) | Poisoned foes it dives on take +25% damage | The wyvern's tail strikes a second foe on each Dive |

## 6. Samurai

A duelist who wins the fight in its first moment. Strategy role:
`duelist` (prefers the strongest foe in reach).

| Level | Gains |
|---|---|
| 1 | **Iaijutsu** (signature): its opening action meter starts 50 higher (it acts earlier) and its **first strike each battle** deals +50% damage with +10% critical chance |
| 3 | **Focus:** each round in which no blow lands on it, +5% critical chance, up to +15%; a hit resets it |
| 5 | Talent |
| 8 | **Zanshin:** when it fells a foe, its action meter gains 50 (half a turn), once a round |
| 10 | Promotion |

The cost is in its numbers: medium armor and no shield, so once its opening
is spent it's an ordinary front-line fighter with a good critical chance.

| Route | Elite | Signature (rank 10) | Elite signature (30) | Capstone (60) |
|---|---|---|---|---|
| **Kensai** | **Sword Saint** | Iaijutsu also ignores half the target's armor | Focus caps at +25% | Iaijutsu strikes twice |
| **Hatamoto** | **Shogun** | Bodyguard: guards the company leader (30c2) twice a battle | Every ally's opening meter starts 15 higher | Allies in its row gain +5 Attack |
| **Ronin** | **Kenshi** | For each fallen ally, +10% damage | When it's the last one standing, it can't be knocked down | Zanshin works every time it fells a foe |

## 7. Shaman

Calls weather into a battle. Strategy role: `support` (calls a weather at the
start, then casts damage). School `stormcraft` (new). Uses mana like other
casters (35a mana pools).

**Weather is battle-local.** A call changes the conditions of the current
battle only, through the existing 30f modifiers (light penalty, cold delays,
fatigue). It never touches the world's weather (`internal/climate`), which
is shared by every player. One weather at a time; a new call replaces it.

| Level | Gains |
|---|---|
| 1 | **Call Fog** (signature): for 3 rounds, enemies' ranged attacks and spells have the dim light penalty (−10 to hit). Chant 1, cost 6 |
| 1 | **Gust:** a small wind damage spell (a Magic Missile's worth at 80%) |
| 3 | **Chill Wind:** for 3 rounds, enemy chants and sling shots are cold-delayed (30f cold rules) |
| 5 | Talent |
| 6 | **Rain:** for 3 rounds, fire damage on both sides is halved and lightning damage is +25% |
| 8 | **Lightning:** strikes one foe; +50% in Rain. Chant 2, cost 12 |
| 10 | Promotion |

| Route | Elite | Signature (rank 10) | Elite signature (30) | Capstone (60) |
|---|---|---|---|---|
| **Stormcaller** | **Tempest Lord** | Lightning chains to a second foe at 50% | Rain lasts the whole battle | Lightning chains to every foe in the row |
| **Mistweaver** | **Veil Mother** | Fog also gives allies +5 Evasion against melee | Weather calls last 5 rounds | Fog hides the back row: enemy melee can't reach it with extended reach |
| **Earthspeaker** | **Mountain Speaker** | **Stoneskin:** one ally +10 armor for the battle | **Tremor:** each enemy in one row has a 25% chance to be knocked down | Stoneskin covers a row |

The expanded catalogue's cleric route named **Shaman** is renamed
**Spiritcaller → Spiritkeeper** so the name belongs to this lineage.

## 8. Alchemist

The one healer with no faith: it heals and strengthens the company from
**flasks** brewed at camp, not from prayer or mana. Strategy role: `healer`
(throws at the most hurt ally; buffs before the first blow lands).

### Flasks

- **A flask satchel** holds the flasks the Alchemist carries into battle:
  6 at level 1, +1 every 3 levels (16 at 30). A thrown flask is used up.
- **Brewing** (camp utility `brew`): during a camp rest, refills the satchel
  from **reagents**. Reagents are bought at apothecaries and, once 40a2
  gathering ships, made from gathered herbs. Brewing never advances world
  time; it is part of the rest, like cooking.
- **Flasks are items** with the camp consumables design's names and data
  (draughts, salves). Other members can't throw them in battle; an
  Alchemist's training is what makes them work at full strength. Outside
  battle anyone may drink one at half effect.
- **When the satchel is empty,** the Alchemist fights with a sling or dagger
  for the rest of the battle. That is its limit, as mana is a priest's.
- **Not holy.** Flask healing ignores alignment and costs no mana. Blight
  (Witch) halves it like any other healing.

| Level | Gains |
|---|---|
| 1 | **Healing Draught** (signature): thrown to any ally in range (the whole company), heals a Minor Heal's worth of the Alchemist's level. Uses one flask, no chant, so it can't be interrupted |
| 1 | **Brew** (camp utility) |
| 3 | **Antidote:** cures poison and bleeding |
| 5 | Talent |
| 6 | **Fire Flask:** fire damage to one foe and its neighbors (the 30f `sparks` shape) at 70% of Sparks |
| 8 | **Bracing Tonic:** one ally +5 Attack and +5 Evasion for 3 rounds; one per ally |
| 10 | Promotion |

| Route | Elite | Signature (rank 10) | Elite signature (30) | Capstone (60) |
|---|---|---|---|---|
| **Apothecary** | **Panacean** | Healing Draught heals 130% and a splash heals the ally beside the target for 30% | **Elixir:** once a battle, raises a fallen ally at 25% HP | Each Healing Draught also removes one harmful status |
| **Bombardier** | **Grenadier** | Fire Flask at 100%, and a **Smoke Flask** gives enemies the dim light penalty for 2 rounds | Flasks reach a whole enemy row | Fire Flasks leave foes burning (2 damage a round for 3 rounds) |
| **Mutagenist** | **Transmuter** | **Mutagen:** one ally +15% damage and +10 armor for the battle, at 10% of its HP | Mutagen lasts and costs no HP | Mutagen covers the ally's whole row |

The Alchemist heals less than a Priest per battle (it's capped by its
satchel), but it never fizzles, never chants and needs no mana, so it pairs
with a priest rather than replacing one.

## 9. Arbalist

A heavy crossbow user built to bring down armored tanks. Strategy role:
`marksman` (prefers the foe with the most armor). Crossbows are in the
equipment tiers catalog (hunting crossbow, arbalest; 36b). There is no
ammunition system, so bolts are not counted.

| Level | Gains |
|---|---|
| 1 | **Piercing Bolt** (signature): a crossbow shot that ignores half the target's armor. After shooting, the Arbalist **reloads**: its next turn is spent winding the crossbow (its meter still fills normally), so it shoots every other turn |
| 3 | **Steady Aim:** if no foe struck the Arbalist since its last shot, +10 Attack on the next one |
| 5 | Talent |
| 6 | **Crippling Bolt:** a hit hobbles the target for 2 rounds (existing status) |
| 8 | Piercing Bolt ignores all of the target's armor |
| 10 | Promotion |

The reload is the trade: a bolt hits about twice as hard as an arrow at the
same level, but the Arbalist shoots half as often, so it is weaker against
many light foes and strongest against one armored one.

| Route | Elite | Signature (rank 10) | Elite signature (30) | Capstone (60) |
|---|---|---|---|---|
| **Siegebreaker** | **Siege Master** | Bolts reduce the target's armor by 10 for the battle (stacks to 30) | **Ballista Bolt:** once a battle, a shot passes through the target to the foe behind it in the column | Reload takes half a turn (meter −50 instead of a whole turn) |
| **Sharpshooter** | **Deadeye** | +10% critical chance; criticals ignore block | Steady Aim gives +20 | First shot of each battle needs no reload |
| **Warden of the Wall** | **Bastion** | **Pavise:** sets up a shield at the start of battle; the Arbalist and the ally beside it have +10 armor against ranged attacks | A foe that strikes an ally in its column is shot first by the held reload turn, like the Halberdier's Brace | Pavise covers the Arbalist's whole row |

## 10. Further candidates (not yet chosen)

From the same brainstorm, not designed: **Ninja** (Ogre Battle; back-row
throws and ninjutsu), **Gladiator** (Unicorn Overlord; stronger as it's
hurt), **Thief** (Unicorn Overlord; steals in battle), **Chronomancer**
(bends the action meter).

## 11. State, persistence and integration

- **Archetypes.** Eight new entries in `modules/archetype/files/data-overlays/config.yaml`
  (fields above, kits, growth, companion levels), with creation text.
- **Promotion graph.** Three new parent → child edges per lineage at 10 and three at 30, all
  unrestricted, in the 38b promotion system.
- **Doll and beast.** A new durable "construct" or "bonded beast" record on
  the owning character (player save) and companion record (company save):
  kind, name, HP, broken/wounded flag, cell and (doll only) gear. It takes a
  formation cell, never a company slot; the formation, company panel, Battle
  view and `company` text show it beneath its owner. Copyover and restart
  keep it; a battle in progress during copyover resolves as other battles do.
- **Gryphon and wyvern.** A mount kind in the 32f mount slot, with its feed
  rate and a `flying` flag read by the narrow-ground and indoor checks.
- **Combat hooks.** Sweep reuses the 30f sweep path; Dive reuses the leap
  path with a "standing blocker allowed" flag; Iaijutsu, Zanshin and Tangle
  adjust the 30g5 action meter (runtime-only, never saved); Shaman weather
  is a runtime battle condition beside the 30f modifiers, never saved.
- **No global time.** Repairs, beast feeding and rests use the existing camp
  and food systems; nothing advances world time.
- **Flask satchel.** A per-Alchemist count of carried flasks by kind,
  saved with the character (player or companion); refilled by `brew` during
  a camp rest, which consumes reagents transactionally.
- **New items.** Militia glaive (catalog), long whip (exists, item 10020),
  doll parts (good), starting doll, lance, hunting crossbow (catalog),
  reagents and flasks (camp consumables names); tagged with bulk and weapon class.

## 12. Delivery

After 38b (promotion framework) ships. Proposed phases, one lineage each so
each passes its own review gate:

| Phase | Class | Depends on |
|---|---|---|
| 39a | Halberdier (needs the glaive from 36b) | 38b, 36b |
| 39b | Samurai | 38b |
| 39c | Shaman | 38b |
| 39d | Doll Master (construct record) | 38b |
| 39e | Beast Tamer (reuses the construct record as a living beast) | 39d |
| 39f | Gryphon Rider (flying mount) | 38b |
| 39g | Alchemist (flask satchel, brewing) | 38b |
| 39h | Arbalist (needs crossbows from 36b) | 38b, 36b |

Elite ranks (30–60) ship with 38c+ elite promotions, like the other
lineages.

## 13. Acceptance (each class phase)

With `ASHVEIL_BALANCE=1`:

1. **Balance.** A company with the new class in its usual row stays within
   the level impact §4 band targets, and wins within 5 points of the same
   company with the nearest existing class (Halberdier vs warrior, Samurai
   vs rogue, Shaman vs wizard, Doll Master vs warrior plus cleric support,
   Beast Tamer and Gryphon Rider vs ranger, Alchemist vs cleric, Arbalist vs
   ranger).
2. **Signatures through real paths.** Sweep through a real battle (each foe
   struck once); Brace spends the Halberdier's turn; Dive passes a standing
   front-liner but not indoors or on narrow ground; Iaijutsu only on the
   first strike; Tangle's meter push and its 2-round immunity; Shaman weather
   ends with the battle and never changes `internal/climate`; a flask throw
   can't be interrupted and an empty satchel ends flask use; the Arbalist's
   reload costs its next turn.
3. **Doll and beast.** They take a cell and no company slot; the doll never
   acts except on the Master's turn; breaking keeps gear; Mend repairs at
   camp for parts; a beast's fall wounds it but never kills it; both are
   saved and survive copyover and restart; neither adds to recruiting limits.
4. **No gates.** Promotion at 10 and 30 succeeds at alignment −100, 0 and
   +100.
5. **Companions.** Each class recruits, levels and uses its abilities by
   strategy role.
6. **Help.** A page per class and per route with rank tables (`help
   halberdier`, `help doll master`, `help beast tamer`, `help gryphon
   rider`, `help samurai`, `help shaman`, `help alchemist`, `help
   arbalist`, and each route), `help doll`, `help bonded beast`, `help
   flasks` and `help brew`, updates to `help classes`, `help promotion`,
   `help formation` (cells taken by dolls and beasts) and `help combat`;
   indexed in `keywords.yaml`; a tutorial hint in the class-choice lesson;
   rendered through `help` in tests and `TestTutorialHelpPointersExist`.

## Owner answers (2026-10-05)

1. **Doll and beast slots:** they take a formation cell but no company
   slot, so a Doll Master company has six bodies but five turns.
2. **Beast death:** a fallen beast is wounded and recovers with rest; it
   never dies.
3. **Names:** keep Samurai and Kensai (and Ninja, if it is designed later).
4. **Alchemist flasks:** a satchel refilled by brewing at camp from bought
   or gathered reagents, with no mana.
5. **Delivery order:** 39a–39h as listed, starting with the Halberdier,
   after 38b.

No questions are open.
