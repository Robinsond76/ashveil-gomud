# Phase 35a2: skill over hit points

Status: **design draft, awaiting owner approval** (handoff rule 20). It
follows [35a level impact](../plans/2026-10-05-phase-35a-level-impact.md)
(merged in PR #15) and comes **before** [35b caster power](../plans/2026-10-05-phase-35b-caster-power.md),
whose spell and heal numbers it changes. It amends the
[level impact design](2026-10-05-level-impact-class-power-design.md) §1c
and §2b, and the [30g6 amendment](2026-10-04-phase-30g6-amendment.md) E.

## Goal

Owner direction (2026-10-05):

> I didn't want HP to rise so much that a weapon took a long time to kill a
> character… I want weapons to do significant damage if they land… A high
> level character shouldn't have much more HP than a low level character,
> but it should be significantly harder to damage, because that character
> is skilled enough to avoid damage, and more skilled to land it as well.

So a level makes a character **better at fighting**, not **thicker**:

- a blow that lands always matters, at every level;
- a veteran lands more blows and avoids more blows;
- HP grows a little, so equal fights stay short and dangerous;
- every level-up shows a real gain.

## Why levels barely matter in a fight today

Every opposed chance (hit, crit, dodge, parry, block, the Strength bonus)
reads a **stat difference** over `StatEdgeSpan: 40` (30g6). Stats grow by
about 0.1–0.17 points per level (35a measurements), so a level-30 fighter
barely out-hits or out-dodges a level-10 one. The only number that grows
quickly is HP: a warrior has 70 at level 10 and 172 at level 60. Levels
show up as **longer fights**, the opposite of the goal.

## Decisions (proposed)

### 1. Two skill ratings: Prowess and Guard

Every character (player, companion, enemy) has two **derived** ratings:

- **Prowess**, skill at landing blows: `floor(level × ProwessRate)`;
- **Guard**, skill at avoiding them: `floor(level × GuardRate)`.

They are computed from level and archetype every time and **never saved**,
so nothing needs migration and death, level loss, copyover and replays
can't drift them. A lost level lowers both, which is the existing death
penalty working as intended.

Starting rates per archetype (config in the archetype overlay; enemies and
characters without an archetype use `Combat.DefaultProwessRate` and
`DefaultGuardRate`, both 1.0):

| Archetype | ProwessRate | GuardRate | Level 30 Prowess / Guard |
|---|---|---|---|
| Warrior | 1.0 | 1.0 | 30 / 30 |
| Ranger | 1.0 | 0.9 | 30 / 27 |
| Rogue | 0.9 | 1.1 | 27 / 33 |
| Cleric | 0.8 | 0.9 | 24 / 27 |
| Wizard, Witch | 0.7 | 0.8 | 21 / 24 |
| Enemies (default) | 1.0 | 1.0 | 30 / 30 |

Casters are worse with weapons and slower to get out of the way; their
power is spells and mana (decision 5). An enemy template may add a small
fixed offset (`prowess: 3`, `guard: -2`) for a trained captain or a
lumbering brute, within ±5.

### 2. One skill edge, added to the stat edge

A new `Combat.SkillEdgeSpan` (20) turns a rating gap into an edge:

```text
skill edge (attack)  = (attacker Prowess − defender Guard) / SkillEdgeSpan
skill edge (defense) = (defender Guard − attacker Prowess) / SkillEdgeSpan
```

Each opposed chance uses **one combined edge**: `clamp(skill edge + the
chance's existing stat edge, −1, 1)`, so stats still matter on top of
skill (a quick fighter is still harder to hit). With the span at 20, a
20-point rating lead is a full edge, about 20 levels between two
warriors.

| Chance | Today | Proposed: even / worst / best |
|---|---|---|
| Hit | 25–100, even 60, Speed edge | **60 / 10 / 95**, skill + Speed edge |
| Dodge | 5–30, only grows with a Perception lead | **12 / 3 / 40**, two-sided, skill + Perception edge |
| Parry | 5–30 + weapon modifier, Speed lead | **12 / 3 / 40** + weapon modifier, skill + Speed edge |
| Block | 15 + shield armor, ±Strength edge, 15–45 | **15 + shield armor**, skill + Strength edge, **8–50** |
| Crit chance | 5–30, even 15, Smarts edge | unchanged, plus **half** the skill edge |

- Dodge and parry become **two-sided** (an even value that a lead raises
  and a deficit lowers), so a novice facing a master defends less than
  today's floor, not the same.
- The 10% hit floor keeps a lucky blow possible: a pack of weaker foes
  still threatens a veteran, and a novice can still land the blow that
  matters.
- Burden still cuts dodge (30g3). Darkness, dual wielding and chemistry
  still move the hit chance. Tackle and bash keep their stat edges and add
  the skill edge.

### 3. HP grows a little

Halve every archetype's `HPPerLevel` and keep 35a's shape (full rate
through level 20), but give **25%** of the rate after level 20 instead of
60%:

- `HPPerLevel`: warrior 1.5, cleric and ranger 1.25, rogue 1.0, wizard
  0.75, Witch 0.9; `DefaultHPPerLevel` 1.25 (enemies too, unless a race or
  template overrides it);
- `HPFullLevels` 20; `HPAfterFull` 0.3125 (25% of the middle rate);
- `HPBase` 40 and `HPPerVitality` 0.5 unchanged.

HP before Vitality, today and proposed:

| Archetype | 1 | 10 | 20 | 30 | 60 | 60 ÷ 10 |
|---|---|---|---|---|---|---|
| Warrior today | 43 | 70 | 100 | 118 | 172 | 2.46× |
| **Warrior proposed** | 41 | 55 | 70 | 73 | 85 | **1.55×** |
| Cleric, Ranger proposed | 41 | 52 | 65 | 68 | 77 | 1.48× |
| Rogue proposed | 41 | 50 | 60 | 62 | 70 | 1.40× |
| Wizard proposed | 40 | 47 | 55 | 56 | 62 | 1.32× |

A level-60 warrior keeps 1.37× a wizard's HP (30g6 E keeps class
differences). Saved health clamps to the new maxima and is never refilled
(30g4 rule).

### 4. Weapons stay as they are

Weapon dice, the Strength bonus (`DamageBonusMin` 8 to `DamageBonusMax`
20), armor, sharpening, crits and wounds are unchanged. A landed blow
therefore stays near a third of a same-level target's HP at every level.

Rough model: two warriors, an average 1d10 weapon, the Strength bonus
growing to its cap, **before armor**. "Swings" counts swings until a kill.

| Fight | Swings that land | Landed hit, % of target HP | Swings to kill |
|---|---|---|---|
| Level 1 vs level 1 | 53% | 38% | ~5 |
| Level 10 vs level 10 | 53% | 34% | ~6 |
| Level 30 vs level 30 | 53% | 29% | ~6 |
| Level 60 vs level 60 | 53% | 30% | ~6 |
| Level 20 attacks level 10 | 72% | 36% | ~4 |
| Level 10 attacks level 20 | 26% | 26% | ~15 |
| Level 30 attacks level 10 | 92% | 39% | ~3 |
| Level 10 attacks level 30 | 6% | 25% | ~66 |

Today a level-60 warrior takes about 7 landed 1d10 hits (with the capped
Strength bonus) to fall; under this design, about 3–4.

### 5. Spells and healing fit the smaller pools (revises 35b §2b)

With HP this low, 35b's proposed spell numbers (Magic Missile ~56 at level
30) would kill in one cast. Spells keep the same rule as weapons: **sized
to a weapon hit, with skill deciding how well they land**.

- **Damage spells** (auto-hit, as 35b decides) take the skill edge as a
  damage factor: `× (1 + 0.5 × edge)`, so 50%–150% of the listed damage.
  A caster's edge uses Prowess against Guard; a master's missile is hard
  to shrug off and a novice's barely singes a veteran.
- **Magic Missile:** `8 + 1d6 + level/4 + Mysticism/8`, about 12 / 16 / 24
  at levels 1 / 10 / 30 (Mysticism 8 / 20 / 40), close to a weapon hit of the same level. Shower
  of Sparks and Withering Hex keep 35b's ratios to it.
- **Minor Heal:** `10 + 2d4 + level/3`, about 15 / 18 / 25: roughly one
  landed hit of the same level. Minor Heal All keeps its 55% share.
- **Hexes** (35b Witch): the resist roll adds the defense skill edge, held
  to 35b's 25–90%.
- **Mana** keeps 35b's large pools (§2d). Casters gain power by casting
  more often, not by casting bigger, which matches small HP.
- Class abilities keep 35b §2c, with Opening Strike's bonus reduced to
  `3 + level/4` so it stays near half a hit.

35b's plan is updated to these numbers before implementation.

### 6. Every level-up shows the gain

- The level-up report (35a) adds `Prowess 12 -> 13   Guard 11 -> 12`.
- `status` shows Prowess and Guard. `consider` describes the gap in words
  ("far more skilled", "evenly matched", "a novice next to you"), not
  numbers.
- Each level now gives Prowess and/or Guard, HP, mana, a training point,
  and on even levels a stat point (35a). Talents and promotions every
  fifth level are unchanged (38b).
- Stat points stay **every 2 levels**: with 59 points by level 60, a
  specialist could reach a full stat edge over an equal-level opponent, and
  level should decide fights more than a single maxed stat.

## Persistence and invariants

- Nothing new is saved. Ratings are derived from level, archetype and
  template offset; HP stays derived (30g4).
- No change to game time, rest or travel.
- Companions use their archetype's rates, the same as players.
- The combat simulator (`internal/combat/simulate.go`), `ExpectedDamage`,
  `CombatOdds` and enemy strategy previews use the same helpers, so
  predictions match fights.

## Integration points

- `internal/combat/calculations.go`: `hitChance`, `dodgeChance`,
  `parryChance`, `blockChance`, `critChance`, `BashChance` and Tackle
  (`strategy.TackleChance`) take the combined edge; a new `SkillEdge`
  helper beside `StatEdge`.
- `internal/characters`: `Prowess()` and `Guard()` from level and archetype
  rates; companions through the same path.
- `modules/archetype/files/data-overlays/config.yaml`: `ProwessRate`,
  `GuardRate`, halved `HPPerLevel`.
- `internal/mobs`: optional `prowess` and `guard` template offsets,
  validated to ±5.
- `_datafiles/config.yaml` and `internal/configs`: `SkillEdgeSpan`,
  default rates, new hit and defense bounds, HP values; the admin
  progression editor charts HP and ratings by level.
- Spell scripts and the 35b plan (decision 5).
- Level-up event and template (35a), `status`, `consider`.

## Help and tutorial

- **New page** `help prowess` (aliases `guard`, `skill`, `skill edge`):
  what the two ratings are, the per-class rates, the even/worst/best
  chances and an example of a 10-level gap.
- **Updated pages:** `defense`, `speed`, `perception`, `strength`
  (combined edge), `health` (new HP rates), `progression` (what a level
  gives), `combat` (hub link to `prowess`), `abilities` (Tackle and bash),
  `experience` (report line).
- `keywords.yaml` lists `prowess` under combat with its aliases.
- **Tutorial:** the combat lesson hint points to `help prowess`
  (`modules/tutorial/stages.go`); stale hints about HP growth corrected.

## Acceptance tests

With `ASHVEIL_BALANCE=1` and 100 fights a cell, these replace 30g6
acceptance's HP row and add to its mismatch rows:

1. **Small HP:** for each archetype, level 60 HP is 1.3–1.7× level 10 and
   at most 2.2× level 1; warrior at least 1.25× wizard at level 60.
2. **Blows matter:** at levels 1, 10, 30 and 60, an average landed weapon
   hit between equal warriors removes at least 20% of the target's HP
   after armor.
3. **Skill wins:** a level-20 company against a level-10 group of 3 wins
   ≥ 99% and loses ≤ 15% of its HP; a level-10 company against a level-20
   group of 2 loses at least 70% of fights.
4. **Equal fights stay short:** the spread mirror's median falls to 5–9
   rounds at every asserted level (it was 10–15; it remains a stress test).
5. **30g6 rows kept:** level 30 against level 10 wins ≥ 95% with no member
   lost in most; level 15 against level 10 wins more than half and loses at
   least once.
6. **Zone bands:** the level impact design's §4 band table is the tuning
   target, measured with these rules.
7. **Unit and wiring:** rating derivation (archetype, default, template
   offset, lost level); combined-edge bounds for every chance; a real
   `AttackPlayerVsMob` and `AttackMobVsPlayer` pass showing the skill edge
   changing hit and defense outcomes; the simulator matching real fights;
   the level-up report, `status` and `consider` lines; HP clamping on load
   without refill.
8. **Help:** every page above renders through `help`, and
   `TestTutorialHelpPointersExist` passes.

Every number above is a starting value. The balance tests decide the final
ones, and the owner approves any change to the shape of the system.

## Slices

- **35a2:** ratings, combined edge, HP rates, report and display, help,
  harness rows and tuning, in one phase (the parts can't be tuned apart).
- **35b:** proceeds with decision 5's numbers.

## Open questions for the owner

1. **Names.** "Prowess" and "Guard" read well, but "guard" was a retired
   targeting word and Guardian is a Warrior ability. Alternatives:
   Offense/Defense (clashes with armor's `help defense`), Attack/Evasion.
2. **Caster rates.** Wizards and Witches at 0.7/0.8 are much easier to hit
   than enemies of their level. Is the back row (formation) protection
   enough, or should their Guard be 0.9?
3. **Crits and skill.** Should a skill lead also raise crit chance (half
   weight proposed), or should crits stay purely Smarts?
4. **Armor growth.** Better armor at higher tiers (36a loot) would make a
   veteran tankier again. Should armor tiers be capped to keep the "a hit
   matters" goal, or is gear meant to add toughness?
