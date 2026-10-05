# Level Impact, Class Power, and the Witch

Status: owner-approved design, 2026-10-05. Execution plans for 35a–35c are in
the [phase 35 handoff](../plans/2026-10-05-phase-35-handoff.md). 35a level impact
is implemented on its feature branch and pending PR integration; 35b and 35c
remain pending implementation. The [35a measurements](../plans/2026-10-05-phase-35a-measurements.md)
record the sparse human automatic-stat limitation against the "most levels"
goal. The draft [35a2 skill over hit points design](2026-10-05-phase-35a2-skill-over-hit-points-design.md)
proposes replacing §1c's HP growth and §2b's spell numbers. The owner's
direction (listed under **Owner decisions**) is settled;
every number below is a proposed default to verify with the balance harness
before it ships.

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

- **The encounter contract.** Ordinary groups are **2–3 enemies**, with an
  occasional **4**. A **boss** group has **up to 5** (the boss and 4
  escorts). Enemy levels come from the **zone**, never from the player:
  - A zone recommended for levels 8–10 has groups a company of levels 8–10
    can defeat. A lower-level company there risks losing.
  - In practice its foes sit at or just below the zone's band.
- **Strategy grows with enemy level**, for bosses too. Low-level enemies have
  none. Higher-level enemies coordinate through the existing 33i2
  coordination tiers, whose bands change at levels 10, 25 and 45.
- Mirror fights stay in the harness as a stress test, not the tuning target.
- **Every level should matter.** Stat changes and other levers are open.
- **Healing grows with level**; spells and player skills get buffs.
- **Spells should rarely fizzle.** Casters can cast many spells without
  fizzling; mana pools grow greatly.
- **Mana comes back only from rest or mana potions.** Rest means camp or inn;
  mana potions are special and not cheap. There is no passive mana
  regeneration. Healers use their larger pools to patch the company up
  between fights. When mana runs out, the company must rest in camp.
- **A Witch is a sixth starting class,** with hexes that paralyze, knock
  down, put to sleep and poison, affecting more targets as the Witch levels.
  Hexes have spell names, not physical skill names.
- **A stat point every 2 levels,** provided growth stays balanced across levels.
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

### 1b. Stat points: one every 2 levels (owner decision)

Change `StatPointsEveryNLevels` from 5 to **2**. That gives 30 points by level
60 instead of 12. The player still chooses where they go. The owner's
condition is that things scale appropriately. Smooth automatic growth (1a)
and chosen points together must keep each zone band's encounter targets
(section 4) within bounds at the band's low, middle and high level. The
harness checks that a company at a band's top does not flatten the next
band's lowest zone.

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
(roadmap item 4).

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
Shower of Sparks (Wizard) and Earthbind (Witch). Each needs a full specification
(trigger, action cost, cooldown, AI priority) in its delivery plan.

### 2d. Mana: large pools, refilled only by rest or potions

Mana is the company's expedition clock (owner decision). It comes back only
from:
- **a camp rest or inn stay,** which refills it fully;
- **mana potions,** which are special and expensive.

Passive regeneration is removed: `ManaPerRound` drops to 0 for players and
companions, and 33h2's online recovery no longer covers mana. Enemies keep
their current mana behaviour. Since pools can't trickle back, they must be
large enough to carry a run of several fights. Each archetype gets its own
mana growth, the way it already has its own HP.

| Archetype | Mana base | Per level | Level 1 / 10 / 30 (+3 × Mysticism) |
|---|---|---|---|
| Wizard, Witch | 40 | 10 | ~74 / ~200 / ~460 |
| Cleric | 36 | 9 | ~69 / ~186 / ~426 |
| Others | 4 | 1 | as today |

- **Target run:** a full caster pool lasts about **3–4 typical encounters**
  of the zone's band. The caster casts on every earned turn, and a healer also
  patches the company up after each fight (section 4). Then the company must
  rest.
- **Spell costs:** base spells keep flat costs. Advanced and elite spells cost
  more, so pools and costs grow together.
- **The 33e mana reserve stays,** so a healer keeps something back for the
  next fight.
- **Mana potions:**
  - Three sizes: a minor, lesser and greater mana draught restoring about
    25%, 40% and 60% of a caster's pool at its tier.
  - Priced as a premium sink: a minor draught costs about one tier-1 Fine
    weapon.
  - Sold only by alchemists and apothecaries in settlements. They are a rare
    drop (loot design).
  - Drunk out of battle only. 33f's "no items in a fight" rule stands unless
    the owner changes it.
  - The existing small blue potion (30014) becomes the minor draught, with
    its price raised. The wizard starting kit keeps one.

## 3. The Witch

### 3a. Placement

The Witch is a **sixth base class at creation** (owner decision, 2026-10-05),
so its control grows from level 1. The expanded catalogue's Wizard
advanced path named "Witch" is renamed **Hexweaver**, with elite **Malison**,
keeping its role: debuffs such as Slow and Expose. Its old elite name, Coven
Sage, moves to the Witch lineage.

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
combat-round timing. Three new statuses are needed: **asleep**,
**paralyzed** and **blighted**. Poison reuses the existing `poisoned` buff.
Names are hex names, not physical skill names (owner).

| Hex | Level | Effect | Chant | Cost |
|---|---|---|---|---|
| Slumber | 1 | Asleep for 2 rounds: loses its turns; the first damage wakes it, and that blow gets +25% hit | 1 | 6 |
| Earthbind | 3 | Grave-soil drags the target down: knocked down (existing status) | 1 | 6 |
| Leaden Curse | 5 | The target's limbs grow heavy: hobbled (existing), with slower turns | 1 | 8 |
| Miasma | 7 | A poison cloud over one enemy row: poisoned (existing buff 13) for 3 rounds; Cure Poison removes it | 2 | 10 |
| Binding Hex | 10 | Paralyzed for 1 round, 2 at level 20; damage does not break it | 2 | 12 |
| Curse of Frailty | 12 | Exposed (existing): the target takes more damage | 1 | 10 |
| Dread Whisper | 15 | One morale check against the target group (30e) | 2 | 14 |
| Blight | 18 | Healing the target receives is halved for 3 rounds (new `blighted` status); answers enemy healers in coordinated groups | 1 | 12 |

**Targets grow with level.** A hex affects 1 foe at level 1, **2 at level 8**,
**3 at level 16**, **4 at level 24**, and the whole enemy group at elite level
30+. Extra targets come from the same formation row, then the nearest others.
Miasma covers one row up to the same count.

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
3. an enemy healer, for Blight;
4. the nearest foe.

It stops hexing when no foe can be hexed and falls back to a weak damage curse
(Withering Hex).

### 3c. Witch routes (level 10 / level 30)

| Advanced | Gate | Elite | Signature |
|---|---|---|---|
| Hedge Witch | Positive | Wise One | Wards: a hex that also shields the most hurt ally; Slumber lasts longer |
| Coven Sage | Unrestricted | Coven Mother | Reach: +1 hex target and shorter chants |
| Hag | Negative | Crone of Ash | Curses: hexed foes take +15% damage; Dread spreads to an adjacent group |

## 4. Fight cost and downtime

Changes that shorten the loop of one fight, then a long rest:

1. **Tune to zone bands.**
   - Each combat zone gets a recommended level band, for example 8–10, in its
     zone config.
   - Its encounter tables use enemy levels from the band's low end to one
     below its top, for example 7–9. The coordination tier those levels earn
     comes from 33i2.
   - Harness cells run a 5-member company at the band's low, middle and high
     level against the zone's groups. Proposed acceptance:

   | Cell | Win rate | Members fallen | HP lost | Rounds (median) |
   |---|---|---|---|---|
   | 2–3 foes, company at band middle | ≥ 97% | 0 in ≥ 85% of fights | ≤ 30% | 4–8 |
   | 2–3 foes, company at band low | ≥ 85% | ≤ 1 | ≤ 45% | 5–10 |
   | 4 foes, company at band middle | ≥ 90% | ≤ 1 | ≤ 45% | 6–10 |
   | Boss + 4 escorts, company at band top | 70–85% | — | — | 10–15 |
   | 2–3 foes, company 3 levels under band low | 30–60% | — | — | — |

   - The last row is the intended risk for an under-levelled company, not a
     failure.
   - The 10–15-round mirror target from 30g6 remains a stress check, not a
     goal.
2. **After-battle patching up, paid in mana.**
   - When a battle ends, the company's healers automatically cast their
     ordinary heals on hurt members, at normal mana cost, until each member is
     above the company's healing threshold (30c1 tactics) or the healer
     reaches its mana reserve. This takes no game time, only the usual chant
     pacing.
   - `company patch` repeats it on demand outside battle.
   - When healers run dry, the company rests in camp (owner). Healing still
     stops at the wound limit.
   - **Passive HP regeneration outside rest** (owner decision, 2026-10-05):
     - a slow trickle that stops at 50% of max HP, so a company with no
       healer can still limp to camp;
     - nothing above 50% without a healer, a potion, an inn or a camp rest;
     - nothing in battle;
     - applies to players and companions, using 33h2's online-only recovery,
       capped at 50%.
3. **Fewer wounds from easy fights.** A crit from a foe 3 or more levels below
   leaves a light wound only, and only on a crit that would knock the member
   below 50% HP. Lasting wounds stay for boss and even-level fights.
4. **No change to world time.** Patching up and any HP trickle are
   per-character actions and timers. Nothing advances global game time
   (handoff invariant).
5. **Keep the risk in bosses and travel ambushes.** Mercy, morale and retreat
   remain the tools for fights that go badly.

## 5. Companion training and optional skills

Owner decision (2026-10-05): companions can learn utility skills after
joining, so the player shapes each member. Companion skills split into two
kinds:

- **Class skills stay automatic.** These are the 33f specialist capabilities
  that define a class: Read the Trail, Pathfinder, Forage, Keen Eye, Haggle,
  Camp Watch, Field Smith, Vigil and Weather Sense. Their ranks keep coming
  from the class at levels 1/10/20/30. A ranger never has to be told to
  track.
- **Optional skills are trained.** These are Scribe, Cooking, and later
  Alchemy and other crafts. A companion learns them with training points the
  player directs.

**Training points for companions.**
- A companion earns training points per level at the player's rate
  (`TrainingPointsPerLevel`).
- The points are **derived, not banked**: points available = points earned
  by level, minus ranks bought. That matches 33h's derived training, so
  death, level regain, re-summon and copyover can't mint extra points.
- Trained ranks persist on the companion's company record. A death that drops
  a companion below a rank's cost keeps the rank, and the next points go to
  repaying the shortfall first.
- Stat points stay automatic, as 33h's archetype-weighted growth.

**Training rules.**
- **Command:** `company train [member] [skill]`, at a trainer offering that
  skill and rank, or in an established camp for ranks 1–2.
- **Cost:** the same 1+2+3+4 point cost as a player.
- **Eligibility:** the companion's class must be allowed the skill. Scribe is
  for casters, Cooking for anyone, and Alchemy per its design.
- **Preview:** shows the cost and the points left.
- **Recruits:** candidates may come with optional skills already trained, at
  a higher price, as a head start.
- **Browser view:** the member's Skills tab shows trained ranks and points
  available.

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
   - per-archetype mana pools, no passive mana regeneration, and mana
     draughts;
   - after-battle patching up and the wound changes.
3. **Witch base class:** the hexes, the two new statuses, the controller role,
   recruit candidates and help.
4. **Companion training** (section 5): derived companion training points,
   `company train`, and trained optional skills, starting with Cooking.
   Scribe arrives with the loot model.
5. **Talents and milestones** with class promotion (see the roadmap order in
   Project Status).

## Player help and acceptance

Ship indexed help in the same delivery as its mechanics:

- `help witch`, `help hexes`, `help talents`, `help mana`;
- `help company train` and the companion section of `help skills`;
- `help draughts` and `help patch`;
- the `asleep`, `paralyzed` and `blighted` entries in `help statuses`;
- updates to `help progression`, `help stat-edge`, `help spellbook`,
  `help heal`, `help abilities`, `help wounds`, `help rest`,
  `help strategy` and `help classes`.

Add a creation-step pointer for the Witch and a Practice Yard hint for hexes.
Help is published only with its mechanics.

Acceptance tests:

- real level-ups (player and companion) report the right changes;
- old saves recompute without duplicating points;
- casts in a real battle never fizzle for owned spells, and a 100% cast never
  fails;
- a full pool carries a 3–4-encounter run, mana never regenerates outside
  rest or draughts, and rest and draughts restore exactly their amount;
- after-battle patching up stops at the mana reserve;
- passive HP recovery stops at 50% of max HP, never runs in battle, and
  rest, inns and potions still heal above it;
- hex target counts at each level boundary, resists and immunity windows;
- Slumber's wake on damage, Miasma's row coverage, and Blight halving real
  heals;
- boss halving and the lock-loop cap;
- copyover keeps current mana, HP, statuses and talents;
- companion training through the real command at trainers and camps: class
  eligibility, derived points across death, regain and copyover with no
  minting, and recruits arriving with skills;
- the balance cells above pass.

An independent full-diff review runs before each merge.
