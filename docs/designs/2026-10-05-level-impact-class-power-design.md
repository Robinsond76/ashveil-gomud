# Level Impact, Class Power, and the Witch

Status: owner-requested design, 2026-10-05. Documentation only; no gameplay
implemented. The owner's direction (listed under **Owner decisions**) is
settled; every number below is a proposed default to verify with the balance
harness before it ships.

Related: [30g6 amendment](2026-10-04-phase-30g6-amendment.md),
[branching class progression](2026-10-01-branching-class-progression-design.md),
[skill and spell progression](2026-10-01-skill-spell-progression-design.md),
[random room encounters](2026-10-01-random-room-encounters-design.md),
[loot system](2026-10-05-loot-system-design.md).

## Problem

A review of master `e8a1032` (30g6) found four connected problems:

1. **Most levels feel empty.** Automatic stats and stat points arrive only at
   levels 5, 10, 15, … (`StatStepLevels: 5`, `StatPointsEveryNLevels: 5`).
   Archetype HP slows after level 10 (`HPFullLevels: 10`, `HPAfterFull: 1.3`).
   The other four levels in five give one training point and +1 mana.
2. **Casters are weak and unreliable.**
   - Magic Missile does a flat `1d6+2` at every level, while a melee hit adds
     `DamageBonusMin` 8 or more from Strength.
   - Its difficulty of 75 leaves a new wizard roughly a 1-in-3 chance to cast
     (`GetBaseCastSuccessChance`: 100 − difficulty + proficiency + Mysticism/5).
   - Every spell fails on a roll of 100, because the check is
     `roll >= successChance`.
   - Mana is `4 + level + 3 × Mysticism`, enough for a handful of casts.
3. **Healing falls behind damage.** Minor Heal is `2d3 + level` (5 at level 1)
   against melee hits of about 14. The 30g6 harness shows healing at roughly
   2–30% of the damage a company takes.
4. **Fights cost too much.** The harness's even 5v5 is a coin flip in which the
   winner loses about three members. Wounds, online-only recovery and
   15-round fights make the loop one fight, then a long rest.

The harness measures mirrors. The game the owner intends is different, and
this design tunes for it.

## Owner decisions (2026-10-05)

- **The encounter contract.** Ordinary zone groups are **2–3 enemies below
  the company's level**, an occasional **4**, still below. A **boss** group
  can have **up to 5** (the boss and 4 escorts), with **no strategy**: no
  coordinated focus or healing (coordination tier 0, nearest-target aim).
  Mirror fights stay in the harness as a stress test, not the tuning target.
- **Every level should matter.** Stat changes and other levers are open.
- **Healing grows with level**; spells and player skills get buffs.
- **Spells should rarely fizzle.** Casters can cast many spells without
  fizzling or running dry; mana pools grow greatly.
- **A Witch class** with hexes that paralyze, knock down and put to sleep,
  affecting more targets as the Witch levels.
- **First class promotion at level 10.** (This matches the branching design's
  proposal; it is now an owner decision.)

## 1. Every level counts

Aim: every level-up changes at least three numbers the player can see, and
every fifth level adds something new to do.

### 1a. Smooth stat growth (replaces the 5-level steps)

Set `StatStepLevels: 1` and rescale the racial growth formula so a
character's automatic stats at levels 5, 10, 15, … stay near today's values.
Growth is spread across the levels in between instead of arriving all at once.
Each level then moves a stat by about one point. Under the stat edge
(`StatEdgeSpan: 40`), each point moves hit, crit, dodge and the Strength edge
by 1/40 of their range. That is a small but real gain every level.

- Archetype `Growth` weights choose which stats rise each level (a Warrior's
  Strength and Vitality first).
- Fractional growth accumulates and is never lost. Store whole points only,
  and derive the remainder from level and archetype so it can't be duplicated.
- Existing characters keep spent points. Their automatic stats are recomputed
  from level, as 30g4 already does.

### 1b. Stat points: one every 2 levels

Change `StatPointsEveryNLevels` from 5 to **2**. That gives 30 points by level
60 instead of 12. The player still chooses where they go. The harness checks
that a level-30 company does not flatten level-25 content.

### 1c. Restore HP growth

Set `HPFullLevels` to **20** and `HPAfterFull` to **60% of the archetype rate**
(instead of 1.3 for everyone). Each level then adds visible HP for every class.
Recheck the 30g6 limit on HP at level 60 versus level 10 (2.09–2.20×). The
proposed new band is 2.4–2.8×, because content runs below the company's level.

### 1d. Abilities and spells scale every level

Every class ability and spell gains a per-level term (section 2), so a
level-up visibly changes Magic Missile's damage, Minor Heal's healing and
Opening Strike's bonus.

### 1e. A milestone every 5 levels

| Level | Milestone (proposed) |
|---|---|
| 1 | Base class role and first ability or spell (as today) |
| 3 | Second base ability or spell (moved down from 5) |
| 5 | Talent choice: pick 1 of 3 small passives from the class list |
| 10 | **Class promotion** (advanced class and its signature) |
| 15 | Talent choice |
| 20 | Advanced signature improves; a new ability for the advanced class |
| 25 | Talent choice |
| 30 | **Elite promotion** |
| 35, 45, 55 | Talent choices |
| 40, 50 | Signature modifiers (from the skill and spell progression design) |

Talents are small, explicit passives with caps, such as +10% Minor Heal, one
extra hex target at a lower rank, or +1 parry. They are one-time choices saved
with the character's or companion's progression owner, and no talent grants a
free action. The full talent lists belong in the class routes delivery
(section 6).

### 1f. Level-up feedback

The level-up message and `experience` list the actual changes. For example:

```text
You reach level 7.
  Health 82 -> 88   Mana 96 -> 104   Strength 21 -> 22
  Magic Missile 14-19 -> 15-20 damage   Minor Heal 17-22 -> 18-23
  Next: a talent at level 10 (Class promotion).
```

Companions get a one-line version in the company's level-up report.

## 2. Spells and abilities

### 2a. No fizzling for known spells

- A spell a character owns, within their level band, **always casts in
  battle**. The fizzle roll is removed for these casts.
- The risk to a caster is the chant being broken (30d1/30d1b), not a dice roll.
- Out-of-battle utility casts and spells cast from scrolls or items keep the
  difficulty roll.
- Fix the `roll >= successChance` bug, so a 100% chance never fails.
- `difficulty` keeps a job: it sets how easily a blow breaks the chant (harder
  spells break more easily) and the proficiency a spell shows in the spellbook.

### 2b. Spell power scales with level and Mysticism

Use one shared form for spells: `base + dice + perLevel × level + Mysticism / 4`.

| Spell | Today | Proposed | Level 1 / 10 / 30 (Mysticism 8 / 20 / 40) |
|---|---|---|---|
| Magic Missile | 1d6+2 | 5 + 1d6 + 1.25 × level + Myst/4 | ~12 / ~26 / ~56 |
| Shower of Sparks (all foes) | script dice | 60% of Magic Missile per target | ~7 / ~16 / ~34 |
| Withering Hex | script dice | damage over 3 rounds totalling 120% of Missile | — |
| Minor Heal | 2d3 + level | 8 + 2d4 + 1.5 × level + Myst/3 | ~16 / ~33 / ~66 |
| Minor Heal All | half bonus | 55% of Minor Heal per patient | ~9 / ~18 / ~36 |

Target: a Minor Heal restores about one ordinary enemy hit of the same level,
roughly 30–35% of a frontliner's HP. Spells auto-hit, so Missile sits a little
below an average melee hit of the same level. Wound limits still cap healing.

### 2c. Class abilities scale

| Ability | Proposed scaling |
|---|---|
| Tackle | Knockdown chance + level/2 percent (within 30g6 bounds); the stagger lasts 1 more round at level 20 |
| Opening Strike | Bonus damage 4 + level/2 |
| Aimed Shot | Crit chance + level/3 percent; damage bonus 2 + level/3 |
| Guardian | Steps in up to 2 + level/10 times a battle (the cap of 2 stays at levels 1–9) |

The second base option moves to level 3 (milestone table): Shield Bash/Cleave
(Warrior), Feint (Rogue), Pinning Shot (Ranger), Minor Heal All (Cleric),
Shower of Sparks (Wizard) and Sleep (Witch). Each needs a full specification
(trigger, action cost, cooldown, AI priority) in its delivery plan.

### 2d. Mana: larger pools and sustain

Give each archetype its own mana growth, the way it already has its own HP.

| Archetype | Mana base | Per level | Level 1 / 10 / 30 (+3 × Mysticism) |
|---|---|---|---|
| Wizard, Witch | 20 | 8 | ~52 / ~148 / ~380 |
| Cleric | 18 | 6 | ~45 / ~120 / ~290 |
| Others | 4 | 1 | as today |

- **In-battle sustain:** each caster regains `1 + level/10` mana per combat round.
- **Out-of-battle sustain:** after a battle's grace, mana returns to full in
  about **60 seconds** of real time.
- **Spell costs stay flat.** Higher levels therefore sustain more.
- **Target:** a caster casts on every earned turn for **two typical encounters**
  (2–3 foes) before using the reserve. The 33e mana reserve stays.

## 3. The Witch

### 3a. Placement (proposed)

The Witch becomes a **sixth base class at creation**, so its control grows from
level 1, which is what the owner described. The expanded catalogue's Wizard
advanced path named "Witch" is renamed **Hexweaver**, with elite **Malison**,
keeping its role: debuffs such as Slow and Expose. Its old elite name, Coven
Sage, moves to the Witch lineage.

The alternative is to keep the Witch as a level-10 Wizard path. That gives the
Witch nothing until level 10, so this design does not recommend it.

| | |
|---|---|
| Role | Control caster: takes enemy turns away instead of dealing damage |
| HP per level | 1.75 |
| Growth | Mysticism 4, Smarts 3, Perception 2, Vitality 1 |
| School | `hexcraft` (new) |
| Kit | Ash wand or staff, a hooded robe, rations, waterskin, satchel |
| Utility (proposed) | Weather Sense; later, alchemy and poison recipes from the [poison](2026-10-01-weapon-poisons-design.md) and [camp consumables](2026-10-01-camp-consumables-design.md) designs |

### 3b. Hexes

Every hex uses the existing status engine (`internal/status`) and the
combat-round timing. Two new statuses are needed: **asleep** and
**paralyzed**.

| Hex | Level | Effect | Chant | Cost |
|---|---|---|---|---|
| Sleep | 1 | Asleep for 2 rounds: loses its turns; the first damage wakes it, and that blow gets +25% hit | 1 | 6 |
| Trip | 3 | Knocked down (existing status) | 1 | 6 |
| Hobble | 5 | Hobbled (existing): slower turns | 1 | 8 |
| Paralysis | 10 | Paralyzed for 1 round, 2 at level 20; not broken by damage | 2 | 12 |
| Dread (talent or advanced) | 15 | One morale check against the target group (30e) | 2 | 14 |

**Targets grow with level.** A hex affects 1 foe at level 1, **2 at level 8**,
**3 at level 16**, **4 at level 24**, and the whole enemy group at elite level
30+. Extra targets come from the same formation row, then the nearest others.

**Resist rolls.** Each target resists with the stat edge: the Witch's
Mysticism against the target's Mysticism, or against Vitality for Paralysis.
The chance is held between 25% and 90%. Bosses add +25% resist, and their
durations are halved, rounding down to at least 1 round.

**No lock loops.** After a hex lands, that target is immune to that hex's
status for 2 combat rounds. A target can't be asleep or paralyzed for more than
50% of the rounds in any fight. This follows the skill and spell design's rule
of no repeated stun loops.

**Automatic use.** Strategy role `controller` (new). The Witch hexes whichever
unhexed group has the most living foes, preferring:

1. a foe who is winding up (30d2);
2. an enemy caster who is chanting;
3. the nearest foe.

It stops hexing when no foe can be hexed and falls back to a weak damage curse
(Withering Hex).

### 3c. Witch routes (level 10 / level 30)

| Advanced | Gate | Elite | Signature |
|---|---|---|---|
| Hedge Witch | Positive | Wise One | Wards: a hex that also shields the most hurt ally; Sleep lasts longer |
| Coven Sage | Unrestricted | Coven Mother | Reach: +1 hex target and shorter chants |
| Hag | Negative | Crone of Ash | Curses: hexed foes take +15% damage; Dread spreads to an adjacent group |

## 4. Fight cost and downtime

Changes that shorten the loop of one fight, then a long rest:

1. **Tune to the encounter contract.** Add harness cells for a 5-member company
   against 2, 3 and 4 foes at 1–3 levels below, and against a boss of +2 levels
   with 4 escorts at tier 0. Proposed acceptance:

   | Cell | Win rate | Members fallen | HP lost | Rounds (median) |
   |---|---|---|---|---|
   | 2–3 foes, 1–3 levels below | ≥ 97% | 0 in ≥ 85% of fights | ≤ 30% | 4–8 |
   | 4 foes, 1–2 levels below | ≥ 90% | ≤ 1 | ≤ 45% | 6–10 |
   | Boss + 4 escorts | 70–85% | — | — | 10–15 |

   The 10–15-round mirror target from 30g6 remains a stress check, not a goal.
2. **After-battle recovery.**
   - Healers spend spare mana on a free field mend once a battle ends.
   - The whole company then recovers HP quickly while out of battle and not
     travelling: about **5% of max HP per combat round**, reaching full in
     about 2–3 minutes. Wounds still cap that recovery.
3. **Fewer wounds from easy fights.** A crit from a foe 3 or more levels below
   leaves a light wound only, and only on a crit that would knock the member
   below 50% HP. Lasting wounds stay for boss and even-level fights.
4. **No change to world time.** Recovery and regeneration are per-character
   online timers. Nothing advances global game time (handoff invariant).
5. **Keep the risk in bosses and travel ambushes.** Mercy, morale and retreat
   remain the tools for fights that go badly.

## State, persistence and integration points

- Progression config: `_datafiles/config.yaml` (`Progression`, the new
  per-archetype `ManaBase`/`ManaPerLevel`), `internal/configs`, the admin
  progression editor and its charts.
- Archetype data: `modules/archetype/files/data-overlays/config.yaml` (the
  Witch, mana growth, talents later).
- Spells: `_datafiles/world/default/spells/*.yaml|js`; cast resolution and the
  fizzle in `internal/hooks/NewRound_DoCombat.go`, and cast chance in
  `internal/characters/character.go`.
- Statuses: `internal/status` (asleep, paralyzed, hex immunity windows).
- Automatic combat: `internal/strategy` (the `controller` role and hex
  targeting).
- Balance: the `modules/company` harness gets the encounter-contract cells.
- Talents and Witch class identity persist with the existing archetype
  registry (player) and the company record (companion). Old saves keep their
  class and gain derived values. Recomputed maxima clamp and never refill.

## Delivery (proposed slices)

1. **Level impact:** smooth stats, stat points, HP shape, level-up report and
   encounter-contract harness cells. Small and mostly config; ships first.
2. **Caster power:**
   - remove the fizzle and fix the roll-100 bug;
   - scale spells and abilities with level;
   - per-archetype mana and sustain;
   - after-battle recovery and wound changes.
3. **Witch base class:** the hexes, the two new statuses, the controller role,
   recruit candidates and help.
4. **Talents and milestones** with class promotion (see the roadmap order in
   Project Status).

## Player help and acceptance

Ship indexed help in the same delivery as its mechanics:

- `help witch`, `help hexes`, `help talents`;
- the `asleep` and `paralyzed` entries in `help statuses`;
- updates to `help progression`, `help stat-edge`, `help spellbook`,
  `help mana`, `help heal`, `help abilities`, `help wounds`, `help rest`,
  `help strategy` and `help classes`.

Add a creation-step pointer for the Witch and a Practice Yard hint for hexes.
Help is published only with its mechanics.

Acceptance tests:

- real level-ups (player and companion) report the right changes;
- old saves recompute without duplicating points;
- casts in a real battle never fizzle for owned spells, and a 100% cast never
  fails;
- mana sustain over a two-encounter sequence;
- hex target counts at each level boundary, resists, immunity windows and the
  sleep wake on damage;
- boss halving and the lock-loop cap;
- after-battle recovery stops on travel, rest and battle;
- copyover keeps current mana, HP, statuses and talents;
- the balance cells above pass.

An independent full-diff review runs before each merge.
