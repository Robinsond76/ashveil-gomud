# Phase 35a2: skill over hit points

Status: **design draft; all of the owner's answers are recorded under
Owner decisions (2026-10-05)** (handoff rule 20). Next: an execution plan. It
follows [35a level impact](../plans/2026-10-05-phase-35a-level-impact.md)
(merged in PR #15) and comes **before** [35b caster power](../plans/2026-10-05-phase-35b-caster-power.md),
whose spell and heal numbers it changes. It amends the
[level impact design](2026-10-05-level-impact-class-power-design.md) §1c
and §2b, and the [30g6 amendment](2026-10-04-phase-30g6-amendment.md) B
and E.

## Goal

Owner direction (2026-10-05):

> I didn't want HP to rise so much that a weapon took a long time to kill a
> character… I want weapons to do significant damage if they land… A high
> level character shouldn't have much more HP than a low level character,
> but it should be significantly harder to damage, because that character
> is skilled enough to avoid damage, and more skilled to land it as well.

So a level makes a character **better at fighting**, not **thicker**:

- an ordinary blow that lands takes **15–25%** of a same-level warrior's
  HP, at every level; a critical hit takes more;
- a veteran lands more blows and avoids more blows;
- HP grows a little; armor, not HP, makes a fighter tough, and only
  warriors are built to be truly tanky;
- every level-up shows a real gain.

## Why levels barely matter in a fight today

Every opposed chance (hit, crit, dodge, parry, block, the Strength bonus)
reads a **stat difference** over `StatEdgeSpan: 40` (30g6). Stats grow by
about 0.1–0.17 points per level (35a measurements), so a level-30 fighter
barely out-hits or out-dodges a level-10 one. The only number that grows
quickly is HP: a warrior has 70 at level 10 and 172 at level 60. Levels
show up as **longer fights**, the opposite of the goal.

## Owner decisions (2026-10-05)

- A landed ordinary hit takes **15% (weak roll) to 25% (strong roll)** of
  a warrior's HP; only a critical hit takes more.
- The ratings are named **Attack** and **Evasion**.
- Casters are **easier to hit than warriors and rogues**.
- **Crits stay Smarts:** the skill edge does not touch crit chance.
- **Gear adds toughness,** but only warriors should be highly tanky, and
  very tanky armor makes its wearer slow.
- **Block counts as evasion** for a shield-bearer.
- **Untrained armor is allowed, with a significant penalty** (not
  forbidden).
- **Shields:** warriors may use any shield; rangers only small shields
  (bucklers); no other class uses shields. **Clerics** fight with staffs,
  rods or maces only.
- **Clerics are priests,** a healer that can't fight well and needs
  protecting, as weak in a fight as the wizard. A fighting healer belongs to
  the warrior's promotion routes, good and evil, not the cleric's
  ([faith routes design](2026-10-05-faith-routes-design.md)).
- **Cleric weapons are maces, staffs and rods;** the cleric kit gets a
  **holy symbol**.

## Decisions

### 1. Two skill ratings: Attack and Evasion

Every character (player, companion, enemy) has two **derived** ratings:

- **Attack**, skill at landing blows: `floor(level × AttackRate)`;
- **Evasion**, skill at not being hurt by them (dodge, parry or block):
  `floor(level × EvasionRate)`.

They are computed from level and archetype every time and **never saved**,
so nothing needs migration, and death, level loss, copyover and replays
can't drift them. A lost level lowers both, which is the existing death
penalty working as intended.

Starting rates per archetype (archetype overlay config; enemies and
characters without an archetype use `Combat.DefaultAttackRate` and
`DefaultEvasionRate`, both 1.0):

| Archetype | AttackRate | EvasionRate | Level 30 Attack / Evasion |
|---|---|---|---|
| Warrior | 1.0 | 1.0 | 30 / 30 |
| Rogue | 0.9 | 1.1 | 27 / 33 |
| Ranger | 1.0 | 0.9 | 30 / 27 |
| Cleric, Wizard, Witch | 0.7 | 0.75 | 21 / 22 |
| Enemies (default) | 1.0 | 1.0 | 30 / 30 |

Casters, the cleric included, are easier to hit than warriors and rogues
(owner); their power is spells and mana (decision 6). An enemy template may add a small fixed
offset (`attackskill: 3`, `evasion: -2`) for a trained captain or a
lumbering brute, within ±5.

**Naming in the game:** `attack` is already a command with its own help
page, so the ratings are shown as **Attack** and **Evasion** in `status`
and the level-up report, and explained on a new `help evasion` page
(aliases `attack skill`, `combat skill`; `skill` already opens `help
skills`). `help attack` keeps
describing the command and links to it.

### 2. One skill edge, added to the stat edge

A new `Combat.SkillEdgeSpan` (20) turns a rating gap into an edge:

```text
attack edge  = (attacker Attack − defender Evasion) / SkillEdgeSpan
defense edge = (defender Evasion − attacker Attack) / SkillEdgeSpan
```

Each opposed chance uses **one combined edge**: `clamp(skill edge + the
chance's existing stat edge, −1, 1)`, so stats still matter on top of
skill (a quick fighter is still harder to hit). With the span at 20, a
20-point rating lead is a full edge, about 20 levels between two
warriors.

| Chance | Today | Proposed: even / worst / best |
|---|---|---|
| Hit | 25–100, even 60, Speed edge | **60 / 10 / 95**, skill + Speed edge |
| Dodge | 5–30, grows only with a Perception lead | **12 / 3 / 40**, two-sided, skill + Perception edge |
| Parry | 5–30 + weapon modifier, Speed lead | **12 / 3 / 40** + weapon modifier, skill + Speed edge |
| Block (shield) | 15 + shield armor, ±Strength edge, 15–45 | **15 + shield armor / 8 / 55**, skill + Strength edge |
| Crit chance | 5–30, even 15, Smarts edge | **unchanged** (owner: crits stay Smarts) |

- **Shields.** Only warriors and rangers carry them (decision 5b). A
  shield-bearer blocks instead of dodging or parrying (today's rule,
  kept): one block roll per blow, with the shield's own armor added to the
  chance. Block grows with Evasion like the other defenses, and its range
  is the widest (8–55), so a shield is the steadiest defense and a warrior
  with a good shield is the hardest target. A buckler's small armor keeps
  a ranger's block well below a warrior's. Bash (a counter after a block) keeps its Strength edge
  and adds the skill edge.
- Dodge and parry become **two-sided** (an even value that a lead raises
  and a deficit lowers), so a novice facing a master defends less than
  today's floor, not the same.
- The 10% hit floor keeps a lucky blow possible: a pack of weaker foes
  still threatens a veteran, and a novice can still land the blow that
  matters.
- Burden still cuts dodge (30g3); armor bulk cuts it further (decision
  5). Darkness, dual wielding and chemistry still move the hit chance.
  Tackle adds the skill edge to its stat edge.

### 3. HP: a higher floor, little growth, a warrior's head start

To make a landed hit 15–25% of a warrior's HP, starting HP rises a little
and growth stays small. Each archetype gets a **starting bonus** (new
`HPStart`), so warriors begin tougher instead of only pulling ahead with
levels:

- `HPBase` 48 (was 40); `HPPerVitality` 0.5 unchanged;
- `HPStart`: warrior 10, ranger 6, rogue 4, cleric 2, wizard and Witch 0;
- `HPPerLevel` through level 20: warrior 1.0, ranger 0.8, rogue 0.7,
  cleric 0.6, wizard and Witch 0.5; `DefaultHPPerLevel` 0.8 for enemies
  unless a race or template overrides it;
- `HPFullLevels` 20; after it, **25%** of the rate (`HPAfterFull` 0.2,
  the middle rate's quarter).

HP before Vitality:

| Archetype | 1 | 10 | 20 | 30 | 60 | 60 ÷ 1 |
|---|---|---|---|---|---|---|
| Warrior today | 43 | 70 | 100 | 118 | 172 | 4.0× |
| **Warrior** | 59 | 68 | 78 | 80 | 88 | **1.5×** |
| Ranger | 54 | 62 | 70 | 72 | 78 | 1.4× |
| Rogue | 52 | 59 | 66 | 67 | 73 | 1.4× |
| Cleric | 50 | 56 | 62 | 63 | 68 | 1.4× |
| Wizard, Witch | 48 | 53 | 58 | 59 | 63 | 1.3× |

A level-60 warrior keeps 1.4× a wizard's HP (30g6 E keeps class
differences). Saved health clamps to the new maxima and is never refilled
(30g4 rule).

### 4. Weapons: the dice decide, Strength adds a little

Today the Strength bonus is 8–20 on top of the weapon's dice, so it
outweighs a 1d10 and grows with level. It becomes **smaller and
flatter**, so the weapon matters most and a hit's size stays the same
share of HP at every level (amends 30g6 B):

- `DamageBonusMin` 6, `DamageBonusMax` 12 (were 8 and 20);
- `DamagePerStrength` 0.375 (was 1.25); `DamageEdgeMax` 2 (was 4).

A 1d10 weapon against an unarmored warrior of the same level (Strength
assumed 3 / 6 / 8 / 10 / 16 at levels 1 / 10 / 20 / 30 / 60):

| Level | Bonus | Warrior HP | Middle 80% of rolls | Average | Top roll | Average on a wizard |
|---|---|---|---|---|---|---|
| 1 | 7 | 59 | 15–27% | 21% | 29% | 26% |
| 10 | 8 | 68 | 15–25% | 20% | 26% | 25% |
| 20 | 9 | 78 | 14–23% | 19% | 24% | 25% |
| 30 | 9 | 80 | 14–22% | 18% | 24% | 25% |
| 60 | 12 | 88 | 16–24% | 20% | 25% | 28% |

- **Crits** (Smarts chance, Perception multiplier 1.5–3×) take about a
  third to a half of a warrior's HP and still leave 30b wounds.
- **Armor** cuts each hit by a random 0 to its total defense percent, so
  a warrior in heavy armor (total defense ~50) takes about 14–16% from an
  average hit.
- Weapon dice now separate weapons clearly: a dagger's 1d4 lands at about
  half a greatsword's blow.

Fights, two unarmored warriors, before armor ("swings" = swings until a
kill):

| Fight | Swings that land | Average landed hit | Swings to kill |
|---|---|---|---|
| Level 1 vs level 1 | 53% | 21% | ~9 |
| Level 10 vs level 10 | 53% | 20% | ~10 |
| Level 30 vs level 30 | 53% | 18% | ~10 |
| Level 60 vs level 60 | 53% | 20% | ~10 |
| Level 20 attacks level 10 | 72% | 21% | ~7 |
| Level 10 attacks level 20 | 26% | 17% | ~22 |
| Level 30 attacks level 10 | 92% | 21% | ~5 |
| Level 10 attacks level 30 | 6% | 17% | ~99 |

### 5. Armor bulk: toughness costs speed, and warriors carry it best

Owner: gear adds toughness, but only warriors should be highly tanky, and
very tanky armor makes its wearer slow. Today the only cost is weight
(burden), which a strong character can carry off. Add **bulk**:

- Every armor item gets `bulk: light | medium | heavy` (new item field).
  Items without one take it from their weight at load (6 kg or more is
  heavy, 2.5 kg or more medium, else light: a leather vest is medium, a
  breastplate heavy), and shipped items are tagged explicitly.
- A character's bulk is the heaviest piece worn:

  | Bulk | Tempo (turns per round) | Dodge | Parry, block |
  |---|---|---|---|
  | Light | — | — | — |
  | Medium | −8% | ×0.8 | — |
  | Heavy | −20% | ×0.5 | — |

  These stack with burden and are not offset by Strength: plate is slow
  however strong you are. Parry and block are unaffected, so a heavy
  warrior defends with sword and shield, not footwork.
- **Armor training** per archetype: warrior heavy; ranger medium; rogue,
  cleric, wizard and Witch light (a priest wears robes). Untrained armor
  can be worn, with a **significant penalty** (owner): its tempo and dodge penalties are
  doubled, the wearer loses **10 Attack and 10 Evasion** (half a full skill
  edge, about ten levels' worth), and a caster takes +1 round on every
  chant. `equip` warns before an untrained piece goes on.
- Result: a warrior in plate with a shield is the hardest target in the
  game and a little slow; a ranger in leather with a buckler is quick and
  fairly hard to pin down; a rogue in plate is a slow rogue who dodges
  nothing; a wizard in plate barely casts.

### 5b. Shields and cleric weapons

Owner: warriors and rangers may use shields if they want; rangers only
small shields (bucklers); clerics no shields, and only staffs, rods or
maces. These are **hard rules**, unlike armor's penalty.

- **Shield size:** a new item field `shieldsize: buckler | shield |
  tower` (a load-time default of `shield` for existing shields). A tower
  shield is also heavy bulk.

  | Class | Shields allowed |
  |---|---|
  | Warrior | Buckler, shield, tower |
  | Ranger | Buckler only |
  | Cleric, Rogue, Wizard, Witch | None |
  | Enemies | Any (unchanged) |

- **Cleric weapons:** **maces, staffs and rods only** (owner). A new
  weapon field `weaponclass` names them; clubs, cudgels and improvised
  weapons don't qualify. Shipped weapons are tagged: ash quarterstaff
  `staff`; ancient royal scepter `rod`; cudgel, crude cudgel, ogre's great
  club and tree trunk `club`; sharp stick, crowbar and boat oar
  `improvised`. An untagged weapon is never a cleric weapon.
- **Enforcement:** `equip` (and companion gear, `company equip`) refuses
  a disallowed shield or weapon with a reason ("Clerics fight with staffs,
  rods and maces."). Archetype data holds the rules (`ShieldSizes`,
  `WeaponClasses`), so later classes set their own.
- **Existing characters:** on load, a disallowed shield or weapon is moved
  from the hand to carried items and the player is told once; it counts
  toward burden like any carried item. Companions do the same.
- **Starting kits:** the cleric's crude cudgel (10015) and wooden shield
  (20004) are replaced by a new **acolyte's mace** (one-handed, 1d6,
  `mace`) and a new **holy symbol** (owner). The holy symbol is a light
  off-hand focus: it can't block, adds **+5% to the cleric's healing**,
  and is the focus the faith routes' rites use ([faith routes design](2026-10-05-faith-routes-design.md)).
  A cleric chooses between a two-handed staff (better parry) and mace and
  symbol (better healing). The ranger's kit is unchanged (its sling is two-handed); a new **leather buckler**
  (light, armor 3) is sold where shields are, for a ranger fighting with a
  one-handed weapon.
- The two shipped shields are `shield` size (wooden 5, iron 10). A
  buckler (3) and a tower shield (14, heavy bulk) are added to shops in
  the plan.

### 6. Spells and healing sized to a weapon hit (revises 35b §2b)

35b's proposed numbers (Magic Missile ~56 at level 30) would kill in one
cast. Spells follow the weapon rule: **about one average hit, with skill
deciding how well they land.**

- **Damage spells** (auto-hit, as 35b decides) take the attack edge as a
  damage factor: `× (1 + 0.5 × edge)`, 50%–150% of the listed damage. A
  caster's edge uses Attack against Evasion.
- **Magic Missile:** `7 + 1d6 + level/10 + Mysticism/15`, about 11 / 13 /
  16 at levels 1 / 10 / 30 (Mysticism 8 / 20 / 40). Shower of Sparks and
  Withering Hex keep 35b's ratios to it.
- **Minor Heal:** `8 + 2d4 + level/6`, about 13 / 15 / 18: one average
  hit. Minor Heal All keeps its 55% share.
- **Hexes** (35b Witch): the resist roll adds the defense edge, held to
  35b's 25–90%.
- **Mana** keeps 35b's large pools (§2d): casters grow by casting more
  often, not harder.
- **Opening Strike:** bonus `2 + level/6` (about half a hit). Other class
  abilities keep 35b §2c.

35b's plan is updated to these numbers before implementation.

### 7. Every level-up shows the gain

- The level-up report (35a) adds `Attack 12 -> 13   Evasion 11 -> 12`.
- `status` shows Attack and Evasion and the armor bulk. `consider`
  describes the gap in words ("far more skilled", "evenly matched", "a
  novice next to you"), not numbers.
- Each level gives Attack and/or Evasion, a little HP, mana, a training
  point, and on even levels a stat point (35a). Talents and promotions
  every fifth level are unchanged (38b).
- Stat points stay **every 2 levels**: with 59 points by level 60, a
  specialist could reach a full stat edge over an equal-level opponent,
  and level should decide fights more than a single maxed stat.

## Persistence and invariants

- Ratings are derived from level, archetype and template offset; HP stays
  derived (30g4). New item fields (`bulk`, `shieldsize`, `weaponclass`)
  are world data. Player and companion records change only when a
  disallowed shield or weapon is moved from the hand on load (decision 5b),
  which happens once and is saved with the record's normal atomic save.
- No change to game time, rest or travel.
- Companions use their archetype's rates and armor training, the same as
  players.
- The combat simulator (`internal/combat/simulate.go`), `ExpectedDamage`,
  `CombatOdds` and enemy strategy previews use the same helpers, so
  predictions match fights.

## Integration points

- `internal/combat/calculations.go`: `hitChance`, `dodgeChance`,
  `parryChance`, `blockChance`, `BashChance`, `damageBonus` and Tackle
  (`strategy.TackleChance`) take the combined edge; a new `SkillEdge`
  helper beside `StatEdge`; `critChance` unchanged.
- `internal/combat/tempo.go` and `burdenedDodge`: bulk penalties.
- `internal/characters`: `AttackSkill()`, `Evasion()` and `ArmorBulk()`.
- `internal/items`: the `bulk`, `shieldsize` and `weaponclass` fields,
  their load-time defaults and validation; new buckler and tower shield
  items.
- `internal/usercommands/equip.go` and company gear: shield and weapon
  rules, the untrained-armor warning; a load-time unequip for disallowed
  items.
- `modules/archetype/files/data-overlays/config.yaml`: `AttackRate`,
  `EvasionRate`, `HPStart`, new `HPPerLevel`, `ArmorTraining`,
  `ShieldSizes`, `WeaponClasses`, and the cleric kit.
- `internal/mobs`: optional `attackskill` and `evasion` template offsets,
  validated to ±5.
- `_datafiles/config.yaml` and `internal/configs`: `SkillEdgeSpan`,
  default rates, hit and defense bounds, HP and damage-bonus values, bulk
  penalties; the admin progression editor charts HP and ratings by level.
- The cast chant path (`internal/hooks/NewRound_DoCombat.go`): +1 chant
  round in untrained armor.
- Spell scripts and the 35b plan (decision 6).
- Level-up event and template (35a), `status`, `consider`.

## Help and tutorial

- **New page** `help evasion` (aliases `attack skill`, `combat skill`):
  what the two ratings are, the per-class rates, the even, worst and best
  chances, and an example of a 10-level gap.
- **Existing page** `help armor` gains bulk, its tempo and dodge costs, and
  each class's armor training (new aliases `bulk`, `heavy armor`).
- **New page** `help shields` (aliases `shield`, `buckler`, `block`):
  sizes, who may use them, and how block works; `help warrior`, `help
  ranger`, `help archetype` and the cleric's class text list their shields
  and the cleric's weapons; `help equip` mentions the refusals.
- **Updated pages:** `attack` (links to `evasion`), `defense` (block,
  armor and bulk), `speed`, `perception`, `strength` (smaller damage
  bonus), `health` (new HP), `progression` (what a level gives),
  `combatpace` (bulk and tempo), `combat` (hub links), `abilities`
  (Tackle, bash, Opening Strike), `experience` (report line).
- `keywords.yaml` lists `evasion` and `shields` under combat with their
  aliases and adds the new `armor` aliases.
- **Tutorial:** the combat lesson points to `help evasion`, and the gear
  lesson (or Departure) to `help armor` (`modules/tutorial/stages.go`);
  stale hints about HP growth corrected.

## Acceptance tests

With `ASHVEIL_BALANCE=1` and 100 fights a cell, these replace 30g6
acceptance's HP row and add to its mismatch rows:

1. **Hit size:** at levels 1, 10, 30 and 60, the middle 80% of 1d10
   weapon hits between equal unarmored warriors land within 13–27% of the
   target's max HP, and the average within 17–23%; crits are excluded.
2. **Small HP:** each archetype's level-60 HP is at most 1.6× its level-1
   HP; a warrior has at least 1.25× a wizard's HP at level 60.
3. **Skill wins:** a level-20 company against a level-10 group of 3 wins
   ≥ 99% and loses ≤ 15% of its HP; a level-10 company against a level-20
   group of 2 loses at least 70% of fights.
4. **Equal fights stay short:** the spread mirror's median is 8–12 rounds
   at every asserted level (30g6 targeted 10–15; it remains a stress test).
5. **Warriors are the tanks:** at level 30 in their best trained armor
   and shield, a warrior takes at least 30% less damage per enemy swing
   than a rogue and at most half what a wizard takes; a heavy-armored rogue
   gets at least 20% fewer turns than a light-armored one.
6. **30g6 rows kept:** level 30 against level 10 wins ≥ 95% with no member
   lost in most; level 15 against level 10 wins more than half and loses
   at least once.
7. **Zone bands:** the level impact design's §4 band table is the tuning
   target, measured with these rules.
8. **Unit and wiring:** rating derivation (archetype, default, template
   offset, lost level); combined-edge bounds for every chance, with crit
   chance unchanged; block used for a shield-bearer; bulk defaults and
   penalties, doubled with −10 Attack and Evasion when untrained; shield
   and cleric weapon refusals through the real `equip` and company gear
   paths; disallowed items unequipped once on load; the new cleric kit; a real `AttackPlayerVsMob` and
   `AttackMobVsPlayer` pass showing the skill edge changing hit and
   defense outcomes; the simulator matching real fights; the level-up
   report, `status` and `consider` lines; HP clamping on load without
   refill.
9. **Help:** every page above renders through `help`, and
   `TestTutorialHelpPointersExist` passes.

Every number above is a starting value. The balance tests decide the final
ones, and the owner approves any change to the shape of the system.

## Slices

- **35a2:** ratings, combined edge, HP, damage bonus, armor bulk and
  training, report and display, help, harness rows and tuning, in one
  phase (the parts can't be tuned apart).
- **35b:** proceeds with decision 6's numbers.

## Open questions for the owner

None open. Resolved 2026-10-05: untrained armor is penalized, not
forbidden; shields for warriors (any) and rangers (bucklers); cleric
weapons are maces, staffs and rods; the cleric kit gets a holy symbol; the
fighting healer belongs to warrior routes, good and evil, with the cleric
routes redesigned in the [faith routes design](2026-10-05-faith-routes-design.md).
