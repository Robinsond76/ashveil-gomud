# Phase 39f: the Gryphon Rider

Source: [neutral classes design](../designs/2026-10-05-neutral-classes-design.md) §5.
A skirmisher with no alignment gate: it dives past a standing front line onto
the healers and casters behind it.

## What shipped

- **Lineage and data**: the `gryphon-rider` archetype (HP 5 and 0.8 a level,
  Attack and Evasion 1.0, medium armor, a buckler or no shield, spears and
  swords, speed 4 / strength 3 / perception 2 / vitality 1, the Skirmish skill,
  Keen Eye as its camp and road utility, an iron short spear and buckler kit),
  a Gryphon Rider recruit (mob 175) in the company templates and recruiter mix,
  creation text, a battle-screen hue.
- **Dive** (`internal/strategy`, `internal/hooks/combat_gryphon.go`): an
  automatic ability, cooldown 3 rounds, the rider's whole turn. One blow at the
  aimed foe that skips the front-row interception: the foe may be in the middle
  or back row, in the rider's column or the next. It needs a melee weapon and
  open sky (`!room.IsIndoor()`, not `narrow`); a foe's guardian can still step
  in. The rider has -10 Evasion for the rest of the round (the round's aura,
  reset when the next round's auras are set, "until its next turn").
- **Base ranks** (`internal/classes/routes_gryphon.go`): Dive (1), Talons (3:
  a landed Dive leaves the foe bleeding), Power dive (8: +25% damage).
- **Routes** (all open at any alignment; ranks 10/15/20/25): **Gryphon Knight**
  (Lance charge: a Dive with a two-handed reach weapon knocks the foe down;
  +4 armor; Falling stone: Dive +50% in all; Steady wings: no Evasion cost),
  **Skyscout** (Eagle eye: +6 Perception, +12 from 25, when the company looks
  for an ambush in the open; Quick stoop: Dive a round sooner; Hawk's mark: a
  landed Dive leaves the foe exposed), **Wyvern Rider** (Venom stoop: a landed
  Dive poisons, buff 13 and never an immune foe; +8% health; +2 Attack; +3
  Evasion). Gryphon Lord, Falcon Marshal and Wyvern Lord are registered as
  planned elites (39i). Talents: Keen Edge, Footwork, Toughness, Sharp Eye,
  Wing Drill (Dive a round sooner, once).
- **Aim**: `strategy.DefaultRule("gryphon-rider")` is `healers` (healers, then
  casters, then the weakest), the skirmisher's role without a new strategy
  role. A rider that can dive (`enemyparty.canDive`) sees every foe in its
  lateral range as reachable; on a round the Dive rests, a swing at a foe behind
  a standing front-liner is caught by the front as any swing is.
- **UI**: the ranks list in `class`, level-ups and companion level lines name
  each new rank ("New rank: Talons ..."), `strategy` lists Dive from level 1,
  `help gryphon-rider`, `help gryphon-rider-routes` (aliases for every rank and
  route name), updates to the archetype, classes, promotion, talents, strategy,
  abilities, combat, health, growth, armor, shields, evasion, progression,
  specialists and keeneye pages, and a creation-lesson hint in the tutorial.
  Dive shows on the battle screen through the ability event like Sweep.

## Decisions

- **No mount record yet.** The design makes the gryphon a mount in the 32f
  riding-horse slot with a flying flag and a double feed. Nothing in a
  battle needs the animal itself (dives read the rider's class and the ground),
  so the stabled-gryphon bookkeeping (herd kind, saddle, feed) is a follow-up;
  until it ships the gryphon is the rider's own and the help says so.
- **Lance charge uses a war spear.** No lance family exists in the 36b catalog
  (adding one touches shops, loot, salvage and the equipment tiers page). A
  two-handed reach weapon, the war spear, stands in; the rider's weapon classes
  are spear and sword.
- **Role**: a fighter with the `healers` default rule, not a new `skirmisher`
  role (as 39b decided for the duelist).
- **Dive at any foe in lateral range**, not only a shielded one: a dive at a
  front-row foe still carries its bleed and bonus damage for the same cost.
- **Skyscout** replaces the design's ranger-style javelins with the Perception
  that actually moves ambush detection; the "company acts first in an ambush"
  capstone is left for the elite (39i).
- **Wyvern Rider** keeps the design's poison (buff 13, as the Witch's hex),
  skipping `immune` creatures to match 43b's poison susceptibility.
- IDs: mob 175, skill `skirmish`, buff 13 (existing).

## Tests

`internal/classes` (lineage, base ranks, routes, talents, level-up text),
`internal/strategy` (Dive's decision, defaults), `modules/company`
`wiring_gryphon_test.go` (the real round: dive past the front row, cooldown and
Evasion cost, talons at level 3, indoor and narrow ground, guardian
interception, column range, abilities off, unarmed, a player rider, route
effects, healer aim, the Skyscout's ambush detection), `balance_gryphon_test.go`
(opt-in), the help render test, and the archetype, company and tutorial suites
updated for the tenth archetype.

## Balance

100 fights a cell, five-foe group of the company's level, one member swapped
for the class (`balance_gryphon_test.go`, `ASHVEIL_BALANCE=1`). Win rate at
levels 5 / 10 / 20: Warrior 79 / 94 / 72, Ranger 77 / 89 / 69, Gryphon Rider
83 / 89 / 75, within about 6 points of the Ranger everywhere and inside the
noise at this sample size. Routes (base / Gryphon Knight / Skyscout / Wyvern
Rider): level 15 76 / 82 / 88 / 83, level 25 81 / 84 / 84 / 89. No tuning was
needed; every route beats the base class. Elite ranks are balanced in 39i.
