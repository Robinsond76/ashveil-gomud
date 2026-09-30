# Phase 30g: Combat Tempo, Personal Load, and Active Defense — Design

A roadmap design for the next combat-balance work, in six slices
(30g1–30g6). Written 2026-09-30 from a design conversation with the owner;
each slice gets its own plan (and, where it settles an open decision
below, a short amendment) before code.

## Owner decisions (2026-09-30)

Agreed in conversation; not to be re-asked:

1. **Attack frequency** comes from a character's Speed and their own
   carried and worn weight. Extra actions are spread across combat rounds
   by an **action meter**, not stacked into one round, and the rate is
   **capped** so a 5v5 doesn't flood the round with attacks.
2. **Weight** for combat agility is each character's **personal load**.
   Company cargo capacity, packs, and mounts never make a heavily
   equipped fighter faster.
3. **Dodge** is driven directly by Perception and personal load.
4. **Shields:** a shield is required to block. A shield-bearer attempts a
   **block instead of dodging**; a failed block goes to armor, with no
   dodge fallback.
5. **Without a shield:** an equipped weapon can **parry a melee strike**;
   dodge remains available when parry can't apply. **One active defense
   attempt per strike.**
6. **Armor:** once shields can block, remove the shield's **×1.5 to all
   armor**. The shield's own armor value still counts, and armor still
   applies after a failed active defense.
7. **Positioning:** attack frequency does not depend on the formation
   grid position.
8. **Balance target:** an evenly matched 5v5 lasts **10–15 combat rounds**
   (80–120 seconds at 29f's 8-second combat round). HP, damage, action
   frequency, defenses, healing, and focus fire are reviewed **together**
   against it. The owner's concern: HP grows too fast against weapon
   damage. D&D and Pathfinder may guide the shape of progression.
   (The HP shape and level cap were open here; settled since by
   decisions 11, 12, and 13.)
9. **Sequencing** (the lead's recommendation, accepted; reordered by
   decision 13): measure first (30g1), then defenses (30g2), personal
   load (30g3), progression (30g4: stat steps, archetype HP, the XP
   knee), the action meter (30g5), and the tuning pass (30g6), each
   measured against the 30g1 baseline.

Settled in a second round (2026-09-30):

10. **Shield bash (was open decision C).** The owner didn't want a bash
    on every blocked strike, and offered two options: a bash only when
    an attacker **fumbles**, or **a small stat-based chance, automatic,
    when a block happens**. The lead chose the second (the owner left it
    open; easy to flip before 30g2's plan): on a **blocked melee strike**,
    a bash chance of **5–20% from the bearer's Strength against the
    attacker's**, once a round per bearer; 30d1's 1d4 and stun 25% are
    kept. A plain miss no longer triggers a counter. Why not a fumble: it
    depends only on the attacker's roll, not on the shield or its bearer,
    and it would be a new idea better given to everyone (below).
11. **Level cap (was open decision F).** **No hard cap.** Leveling slows
    steeply past a knee: the XP curve stays as it is (quadratic) to about
    **level 60**, which already takes a long time, then each level costs a
    fixed percentage more than the last, so **about 100** is the
    practical ceiling. The balance band (decision 8) must therefore hold
    from level 1 to 60, with a sanity check at 100, and no HP or damage
    formula may run away with level.

Settled in a third round (2026-09-30):

12. **HP: small numbers, by archetype (was open decision G).** HP grows
    slowly: a veteran is tougher, but not a damage sponge, and level
    shows mostly in gear, accuracy, defenses, and abilities. **Archetypes
    differ in HP** (a fighter gains more per level than a wizard), D&D
    style. Aim (tuned in 30g6): a level-60 has about 2–3× a level-10's
    HP, not today's ~5×.
13. **Stats grow in steps (replaces open decision E).** The owner found
    a per-level Speed reference messy. Instead, **automatic stat growth
    comes every few levels** (the owner offered 5 or 10; open decision I)
    rather than every level, so stats, and with them Speed and tempo,
    stay in a small range. Tempo then reads **raw Speed** against one
    fixed reference, with no level term. A level advantage stays modest,
    but **a level 30 beats a level 10 with relative ease**, through
    better gear plus a small stat edge (a few steps) and more HP.
14. **Balance baseline: the spread-out fight (was open decision H).**
    The 10–15 round target applies to a fight with **no focus on either
    side**. Tactics help but don't always work cleanly: **enemy tactics
    (30c1's targeting personalities) can trouble the company too**, so
    the player has to adapt. A focused company against a passive enemy
    should win faster; an enemy focusing on the company should hurt.

Settled in a fourth round (2026-09-30): the owner accepted the lead's
recommendations on the remaining open decisions ("Ok go"):

15. **Parry or dodge (A):** when both could apply, roll whichever
    chance is higher, once.
16. **Parry by weapon (B):** swords and staves +5, axes and maces 0,
    daggers −5, two-handed polearms +5; unarmed and claws can't parry.
17. **Armor suffix (D):** `(5 damage, 2 absorbed)`.
18. **Stat step size (I):** every **5** levels (`StatStepLevels: 5`).

## Prior-art check (against `master` at `66fc6dc`)

- **Hit, dodge, armor** (`internal/combat/combat.go`, the strike loop): a
  hit roll from the Speed delta (`hitChance`, `ToHitMin/Max` 25–100), then
  `Dodges(defPerc, atkPerc)` (Perception delta, `DodgeChanceMin/Max`
  5–30) unless the target has `status.FlagNoDodge` (stunned), then
  `applyDefenseReduction(damage, GetDefense())`: a random 0..armor percent
  off each strike. Dodge is a line of text; it is not a combatstream
  event.
- **Shields** (`internal/characters/character.go`): `HasShield()` (a
  non-weapon offhand with `DamageReduction > 0`, false under `no-block`);
  `GetDefense()` sums every slot's defense, **×1.5 when `HasShield()`**,
  halves under `armor-broken`, caps at 100. There is no block roll: the
  shield only raises armor. `internal/combat/armor_rank.go` ranks armor
  with the same ×1.5, and `help armor` says so.
- **Shield counter** (30d1, `internal/hooks/combat_interrupt.go`
  `afterBlow`/`counterBlow`): on a **missed** melee blow against a
  shield-bearer, 50% to bash for 1d4, stun 25%, once a round.
- **Attack count** (`combatAttackCount`): armed fighters get one turn a
  round (a weapon's own `attacks` dice and dual wield add strikes);
  weaponless and claws get `1 + extraAttackCount(atkSpd, defSpd)` (0–3
  from the Speed delta **against the current target**) plus the
  `attacks` stat mod. No parry exists anywhere.
- **Weight** (28, 32f): every item weighs grams (`Item.Weight()`);
  `carriedGrams` in `internal/usercommands/skill.peep.panels.go` already
  weighs one character's carried and worn items. 32f's company capacity
  (`encumbrance.MemberCapacity`: `MemberBaseKg` 20 + `StrengthKg` 0.5 per
  Strength + best pack, plus mounts) is a company-level limit and is not
  reused for agility (decision 2).
- **HP** (`RecalculateStats`): `HPBase 5 + level × HPPerLevel 1 +
  Vitality × HPPerVitality 4 + mods`, and Vitality itself grows with
  level (`StatInfo.Recalculate`, soft-capped). Weapon dice don't grow
  with level; only the Strength-delta damage bonus (0–10) does.
- **Cadence** (29f): `CombatEveryRounds: 2` of 4-second rounds; the 29f
  known issue (regeneration, survival, buff durations tick twice per
  combat round) is due its Phase 30 retune, which 30g6 takes.
- **Prior design** (`2026-09-22-combat-design-reference-external.md`,
  prototypes 1–2): Effective Speed = AGI × 100 / (100 + Weight × 0.75)
  filling an action meter to 100 made weight matter at once, but was too
  steep; its agreed follow-ups were a gentler weight curve, block and
  parry as derived chances, and slowness in wind-ups rather than in
  initiative for heavy enemies (30d2).
- **Harnesses:** `internal/combat/simulate.go` (`SimulateCombat`, one mob
  against one mob, not the real round); `modules/company`'s `newBrawl`
  drives the real `DoCombat` with a company and a bandit group (fixed
  health for determinism, dice injectable).

## Slices

### 30g1 — Balance harness and baseline (measure first)

As built (amended from this design's first draft by the 30g1 plan, the
first run, and review; the plan holds the detail):

- An opt-in test, `TestBalance5v5` in `modules/company`, skipped unless
  `ASHVEIL_BALANCE=1`, drives an evenly matched 5v5 through the real
  `DoCombat` with real dice and 30a's statuses: the company (Aria and
  four companions: a shield fighter, a cleric who heals, a sellsword,
  and a ranger, by 32d strategy) against a mirror group with the same
  kits, both at levels 1, 5, and 10. Company modes: spread, default (the
  shipped strategies, already focus fire), focus (30c1). Enemy modes:
  spread (random aims), default (the shipped weakest). The baseline is
  spread × spread (decision 14).
- Fights are not seeded (`util.Rand` is the global `math/rand`); each
  cell runs `ASHVEIL_BALANCE_FIGHTS` fights (default 50) and reports
  distributions.
- It reports, per cell and per side: company wins, rounds to the end
  (p10, median, p90), stalls (still going at 200 rounds), fallen, damage
  and healing, turns per standing fighter per round, hit and crit
  rates, shield bashes, and status-tick damage. Output is a table in the
  test log; nothing is asserted until 30g6. Per-defense outcomes arrive
  with 30g2's `Defense` field.
- The baseline table goes in the 30g1 work-log entry. Each later slice
  records its new table, so every change is measured, not felt.
- Fixture mobs are in the test (like `banditMob`); no shipped content
  changes.

### 30g2 — Active defense and armor

One **active defense** per strike, after a strike hits and before armor:

| Defender | Strike | Defense rolled |
|---|---|---|
| Holds a shield (`HasShield`) | any weapon strike, melee or ranged | **block** only |
| Weapon in hand, no shield | melee | **parry** or dodge (open decision A) |
| Weapon in hand, no shield | ranged (`items.Shooting`) | dodge |
| Unarmed or claws, no shield | any | dodge |
| Stunned (`no-block`, `FlagNoDodge`) | any | none |

- **Block** (new `blockChance`): from the shield's own armor value and
  the defender's Strength against the attacker's, clamped to new
  `BlockChanceMin/Max` (proposed 15–45; it must beat a typical dodge,
  since a shield-bearer gives dodge up). A blocked strike does no damage
  and leaves no status or wound. Personal load does not lower it (heavy
  shield fighters are the point).
- **Parry** (new `parryChance`): from the Speed delta, clamped to new
  `ParryChanceMin/Max` (proposed 5–30, dodge's scale), with a small
  per-subtype modifier (open decision B). Melee only. A parried strike
  does no damage.
- **Dodge**: unchanged in shape here (Perception delta, 5–30); 30g3 adds
  personal load.
- **A failed defense** goes to `applyDefenseReduction` exactly as today.
- **Armor:** `GetDefense()` loses the ×1.5; the shield's own
  `DamageReduction` still sums through the offhand slot. `armor_rank.go`
  and `help armor` drop it too.
- **Shield bash** (30d1's counter, decision 10): moves from "a missed
  melee blow" (50%) to "a **blocked** melee strike", with a chance of
  5–20% from the Strength delta (new `BashChanceMin/Max`); once a round
  per bearer, 1d4, and stun 25% are unchanged, and it breaks no chant or
  wind-up (owner, 2026-09-30, with 30d2: a counter strike only). A plain
  miss no longer triggers it.
- **Lines** in the 29c/29d voice, pronoun-aware: block ("Tamsin catches
  the blow on her shield."), parry ("Garrick turns the blow aside with
  his longsword."), dodge as today. The armor suffix's word (open
  decision D).
- **Events:** the attack event gains a `Defense` outcome (`dodged`,
  `parried`, `blocked`, empty); the battle summary counts each side's
  blocks, parries, and dodges on one line, shown only when non-zero.
- **Unchanged:** spells are not dodged, parried, or blocked (as today);
  the hit roll; crits (a defended strike can't crit, as a dodged one
  can't today).

### 30g3 — Personal load and agility

- **Personal load**: a character's carried and worn grams, the same for
  players, companions (their live mob), and enemies. `carriedGrams`
  moves to `Character.PersonalGrams()` (the peep panel keeps using it).
  Company cargo, packs, and mounts are not in it and add nothing to the
  capacity below (decision 2).
- **Agility capacity**: `AgilityBaseKg + AgilityStrengthKg × Strength`,
  new `Combat` config keys (proposed 15 kg and 0.5 kg), separate from
  32f's cargo keys so tuning one never moves the other.
- **Burden** `b` in 0–1: `clamp((load / capacity − AgilityFreeLoad) /
  (1 − AgilityFreeLoad), 0, 1)`, with `AgilityFreeLoad` proposed 0.35: a
  light kit costs nothing; only real armor and packed gear count. Using
  a fraction of capacity (not raw kilograms) keeps a strong fighter in
  plate from being punished as hard as a weak one.
- **Dodge** becomes `dodgeChance(defPerc, atkPerc) × (1 − 0.6 × b)`
  (decision 3). Parry and block are not reduced by burden.
- **Burden words** (unburdened, lightly burdened, burdened, heavily
  burdened) in `status`, the web Character overview, and `scout`/`look`
  of a member; never raw ratios.
- Burden is computed when needed from the live items, so nothing new is
  saved.

### 30g4 — Progression: stat steps, archetype HP, and the XP knee

Structure first, with provisional numbers; 30g6 tunes them.

- **Stat steps (decision 13).** `StatInfo.GainsForLevel`
  (`internal/stats/stats.go`) counts **steps** instead of levels: a
  character at level L has `1 + floor((L − 1) / StatStepLevels)` steps
  (new progression key; 5 or 10, open decision I), and today's racial
  formula runs on the step count instead of the level. Earned stat points
  follow the same rhythm through the existing `StatPointsEveryNLevels`.
  Players and mobs share `Recalculate`, so both change together.
- **Each step is worth less than the levels it replaces:** the aim is a
  small stat edge per step (the owner's "small stat advantage"), so a
  level 30 is a few steps ahead of a level 10, not twenty levels of
  growth. The step's size is 30g6's tuning.
- **Levels between steps still count:** each gives HP (below), training
  points (skills), and progress toward the next step, so no level is
  empty. `status` and the level-up message say when the next stat step
  comes.
- **Archetype HP (decision 12).** `HealthMax = HPBase + the archetype's
  per-level HP × levels up to HPFullLevels + a small fixed HP per level
  after + Vitality × HPPerVitality + mods`. Per-archetype values sit
  with the archetypes (the 22a seam, `internal/archetypes`; companions
  through `company.CompanionArchetype`); enemies take one from their race
  or template, defaulting to the middle archetype's. Proposed start
  (tuned in 30g6): fighter 6, cleric and ranger 5, rogue 4, wizard 3 per
  level to `HPFullLevels` 20, then 1 a level; `HPPerVitality` reduced so
  Vitality matters less than archetype. The old `HPPerLevel` is retired.
- **XP knee (decision 11).** `XPTL` (`internal/characters/character.go`)
  keeps today's formula (`(XPBase + L^XPLevelPower × XPLevelFactor ×
  XPBase) × TNLScale`) up to a new `XPKneeLevel` (proposed 60); past it,
  each level's cost is the previous level's times a new `XPKneeGrowth`
  (proposed 1.10, making level 100 about 45× a level-60 level). Clamped
  to `MaxInt` as today. The admin progression editor
  (`/admin/progression`) charts the steps, HP by archetype, and the knee;
  `MaxLevel` stays a display value.
- **Existing characters:** level, XP, trained stats, and spent points are
  kept; the racial part of their stats and their max HP are recomputed
  on load, as they are today on every level change, so nothing new is
  saved. A character's racial stats may drop (fewer steps than levels);
  30g4's plan checks the shipped companions and starter characters and
  records the before/after in the work log.

### 30g5 — The action meter

- **Tempo** per combatant, recomputed each combat round: `speedTerm ×
  (1 − 0.35 × b)`, clamped to `TempoMin`–`TempoMax` (proposed 0.6–1.5).
  `speedTerm = 1 + (Speed − TempoSpeedRef) / TempoSpeedSpan`, from **raw
  Speed** against one fixed reference (decision 13: with stats in steps,
  Speed stays in a small range, so no level term is needed). Never
  against the target, so a fighter's tempo is the same whoever they
  strike. Formation position plays no part (decision 7).
- **Meter:** each combat round the meter gains `100 × tempo`; each full
  100 is one **turn** (the whole of today's round for that character:
  their weapons, dual wield, and a weapon's own `attacks`). At most
  `MaxTurnsPerRound` (proposed 2) turns in one round; after acting the
  meter keeps at most 99, so nobody banks turns for a burst. The meter
  starts at `100 − 100 × tempo`, so everyone acts in the opening round
  and slower fighters then fall behind.
- **Rate cap in a 5v5:** at `TempoMax` 1.5 and two turns a round, ten
  fighters average at most 15 turns a round and never exceed 20.
- **Replaces** `extraAttackCount` (the target-relative Speed bonus for
  weaponless and claws); the `attacks` stat mod becomes a tempo bonus
  (+0.1 each, open to 30g6). A weapon's own `attacks` dice are unchanged.
- **Chants** step once per combat round regardless of tempo (casters'
  slowness is in their chants, per the prior design); a chanting
  character's meter still fills but takes no weapon turn.
- **Statuses** (30a) that cost a round's actions cost all of that
  round's turns; the meter still fills. Guards (30c2) and counters
  (30d1) are reactions, not turns, and use no meter.
- **State:** the meter lives with the battle in memory, is cleared at
  fight end, and is dropped by restart/copyover exactly as aggro and
  chants are (nothing saved; a restarted fight starts fresh meters).
  It fills only on combat rounds; it never reads or advances the world
  clock.
- **Output:** a round with more turns produces more lines; 29f's pacing
  already spaces them, and 30g1's harness reports lines per round so the
  8-second round stays readable.

### 30g6 — Tuning HP, damage, and healing against the target

- **The anchor (decision 14: the spread-out fight).** Let `e` be one
  fighter's expected damage per combat round after hit, active defense,
  and armor, and `H` a fighter's health. With no focus on either side,
  blows land evenly and everyone falls at about `H / e`, so 10–15 rounds
  means **H ≈ 10–15 × e** before healing (each fighter survives about a
  dozen average landed hits). When one side focuses, its first kill comes
  after about `H / (5e)` and the other side loses damage early, which is
  why focus fire wins faster.
- Today `H` grows with level through both `HPPerLevel` and Vitality's
  own growth, while `e` grows only through the capped Strength bonus, so
  fights lengthen with level. 30g4's steps and archetype HP change the
  shape; 30g6 sets the numbers so `H / e` stays in band at every level
  from 1 to 60, with small numbers (decision 12).
- **Scope:** the stat step size, archetype HP values, `HPFullLevels`,
  how damage grows (weapon tiers, the Strength bonus, tempo), healing
  (heal spells, 32d healer thresholds, and 29f's doubled in-combat
  regeneration), and the 30g2–30g5 numbers, all tuned together through
  the harness.
- **Also retuned here:** 29f's known issue (regeneration and round-timed
  buffs ticking twice per combat round).
- **The harness grows** to cover:
  - **even fights** at levels 1, 5, 10, 30, and 60 (asserted) and 100
    (reported only);
  - **tactics both ways** (decision 14): company focus none or weakest,
    against enemies with no targeting or a 30c1 personality (weakest,
    casters);
  - **mismatches** (decision 13): a level-30 company against level-10
    enemies, and level 15 against level 10.
- **It asserts:**
  - no focus on either side: median rounds 10–15 at every asserted
    level;
  - a focused company against passive enemies wins more often, and
    sooner, than the no-focus fight;
  - enemies focusing on a passive company take more of its health than
    passive enemies do;
  - level 30 against level 10: the company wins at least 95% of fights
    and loses no member in most;
  - level 15 against level 10: the higher side is clearly favored but can
    lose.

## Open decisions (all settled; kept for the reasoning)

- **A. Parry or dodge, when both could apply** (settled: decision 15) (a melee strike on an
  armed fighter without a shield). Recommend: **roll whichever chance is
  higher, once**. It is still one attempt per strike (decision 5), and
  nobody is worse off for holding a weapon. The alternative (always
  parry) makes a light, perceptive fighter weaker for drawing a blade.
- **B. Parry by weapon** (settled: decision 16). Recommend a small subtype modifier: swords and
  staves +5, axes and maces 0, daggers −5, two-handed polearms +5;
  unarmed and claws can't parry (they dodge). The owner may prefer no
  modifier at first.
- **C. What triggers the shield bash.** Settled: decision 10.
- **D. The armor suffix** (settled: decision 17). Today's `(5 damage, 2 blocked)` means armor,
  which will read wrongly once shields block. Recommend
  `(5 damage, 2 absorbed)`; `help narration` and 29c's tests follow.
- **E. Tempo's Speed reference.** Replaced by decision 13 (stats in
  steps; tempo from raw Speed against one fixed reference).
- **F. Level cap.** Settled: decision 11 (no cap; an XP knee at ~60,
  ~100 practical).
- **G. HP formula shape.** Settled: decision 12 (small numbers, by
  archetype).
- **H. Which fight the target describes.** Settled: decision 14 (the
  spread-out fight; enemy tactics cut both ways).
- **I. Stat step size** (settled: decision 18, every 5).
  Recommend **5**. Every 10 gives only six steps by level 60, so each
  step must be large to mean anything, and nine levels in a row pass with
  no stat change. Every 5 gives twelve smaller steps, keeps the gap
  between a level 30 and a level 10 at four steps, and makes a new step
  a regular event.

## Durable model and invariants

- **Nothing new is saved.** Defenses and burden are computed per strike
  from live stats and items; the meter lives with the battle and is
  dropped with it (restart/copyover drops fights, as it does aggro and
  chants today).
- **No world clock change.** The meter fills on combat rounds only;
  nothing reads or advances game time. Travel and rest are untouched.
- **Game loop only; no new locks.** Everything runs inside `DoCombat` and
  the attack resolution it already calls.
- **Config, not constants,** for every tuned number (new `Combat` keys:
  `BlockChanceMin/Max`, `ParryChanceMin/Max`, `BashChanceMin/Max`, `AgilityBaseKg`,
  `AgilityStrengthKg`, `AgilityFreeLoad`, `TempoMin/Max`,
  `TempoSpeedRef`, `TempoSpeedSpan`, `MaxTurnsPerRound`; new
  `Progression` keys: `StatStepLevels`, `HPFullLevels`, `XPKneeLevel`,
  `XPKneeGrowth`; archetype HP with the archetypes), so 30g6 tunes
  without code.
- **Restart/copyover:** stat steps, max HP, and XP cost are all derived
  from saved level, archetype, and training on load, so they survive
  restart without a save change.

## Constraints and deferrals

- Spells stay undefended by block, parry, and dodge (a magic defense is
  a later idea from the prior design, not this phase).
- 30d2's wind-ups are separate; the meter doesn't slow heavy attacks.
- **Adaptive enemy tactics** (a later idea): enemies that change focus
  mid-fight (turning on a healer who starts working, guarding their own
  casters) would build on 30c1's personalities; this phase only measures
  the personalities that exist.
- **Fumbles** (a later idea, not this phase): the worst few percent of
  anyone's hit rolls could leave the attacker `exposed` (30a's status),
  for every fighter, not only against shields.
- Guard reactions as a limited resource (the prior design's "Guard
  Reaction") are not built; 30c2's guardian stays as it is.
- Mounted combat (30f) doesn't read burden yet.
- No player-facing numbers for tempo or burden (words only), matching
  `scout`'s health words.

## Acceptance criteria (per slice; each slice's plan repeats its own)

- **30g1:** `ASHVEIL_BALANCE=1 go test ./modules/company -run
  TestBalance5v5` prints the table for levels 1/5/10 and every pair of
  modes; the baseline is recorded; the test is skipped by default; the
  always-on harness tests keep it working (a fight ends, the sides stay
  even, statuses land).
- **30g2, unit (`internal/combat`):** `blockChance`, `parryChance` bounds
  and deltas; the defense-choice table above (shield → block only, ranged
  → no parry, unarmed → dodge, stunned → none); `GetDefense` without the
  ×1.5 (a shield's own armor still counts, `armor-broken` still halves).
  **Wiring through `DoCombat`:** a blocked strike, a parried strike, a
  failed block that reaches armor, a shield-bearer never dodging, a
  bash on a blocked melee strike only (roll forced; the chance from the
  Strength delta, 5–20%) and never on a plain miss, the summary's
  defense line.
- **30g3, unit:** `PersonalGrams` (worn + carried, not cargo), burden's
  free band and clamp, dodge falling with burden; **wiring:** a heavily
  burdened fighter's dodge chance through the real round; a mount or
  cargo change leaving burden unchanged; the burden word in `status`.
- **30g4, unit:** `GainsForLevel` flat between steps and rising at each
  (level 1–4 equal, 5 above them, with `StatStepLevels` 5); stat points
  earned on the step rhythm; `HealthMax` by archetype (a fighter above a
  wizard of the same level and Vitality; full gains to `HPFullLevels`,
  then the small fixed gain); an enemy's HP from its race or template
  default; `XPTL` unchanged up to the knee, growing by `XPKneeGrowth` a
  level past it, clamped at `MaxInt`. **Wiring:** a character loaded
  from a save before 30g4 keeps its level, XP, and training, with
  recomputed stats and HP; a level-up to a step raises stats, and one
  between steps raises only HP; a companion's HP follows its archetype.
- **30g5, unit:** tempo from raw Speed and its clamps, meter fill, the
  two-turn cap, the 99 carry cap, the opening-round start; **wiring:** a
  fast fighter taking two turns in one round and never three, a slow one
  skipping a round, a stunned fighter losing both turns, a chant stepping
  once a round, a fresh meter after the fight ends.
- **30g6:** the harness assertions listed under 30g6 pass; the tuned
  values and the final harness table are recorded in its work-log entry.
- **Player help (every slice that ships behavior):**
  - new `help defense` (block, parry, dodge, the one-attempt rule,
    armor after), aliases `block`, `parry`, `dodge`, `shield`,
    `shields`; linked from `help combat`;
  - new `help tempo` (the meter, turns, burden, the cap), aliases
    `speed`, `actions`, `turns`, `burden`, `agility`; linked from
    `help combat` and `help encumbrance`;
  - updated: `armor` (no 50% bonus), `interrupts` (counter on a block),
    `narration` (the armor suffix), `statuses` (stunned: no defense),
    the leveling page (stat steps and when the next comes, HP by
    archetype, slower past 60; 30g4), and the archetype choice text at
    creation (each archetype's toughness),
    `cargo`/`encumbrance` (combat agility is personal, mounts don't help);
  - the Practice Yard hint (`modules/tutorial/stages.go`) points to
    `help defense` and `help tempo`;
  - each page renders through `help` (the `help_combat_test.go`
    pattern) and `TestTutorialHelpPointersExist` passes.
- Each slice: `go test -race ./...`, `make generate`, `make validate`,
  the independent review, and its **Review:** line in
  `docs/PROJECT_STATUS.md`.
