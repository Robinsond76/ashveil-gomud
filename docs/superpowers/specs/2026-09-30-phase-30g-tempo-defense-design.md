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
   damage. D&D and Pathfinder may guide the shape of progression; **no HP
   formula or level cap is agreed yet** (30g5's decisions).
9. **Sequencing** (the lead's recommendation, accepted): measure first
   (30g1), then defenses (30g2), personal load (30g3), the action meter
   (30g4), and the HP/damage tuning pass (30g5), each measured against the
   30g1 baseline.

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
  combat round) is due its Phase 30 retune, which 30g5 takes.
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

- An opt-in test, `TestBalance5v5` in `modules/company`, skipped unless
  `ASHVEIL_BALANCE=1`, drives an evenly matched 5v5 through the real
  `DoCombat`: a company of five (fighter, fighter, healer, caster,
  archer by 32d strategy) against a five-member enemy group of the same
  levels, at levels 1, 5, and 10, both with no focus and with a 30c1
  focus, over N seeded fights (default 200) each.
- It reports, per level and focus: rounds to the end (median, p10, p90),
  who won, damage and healing per side, landed strikes, defenses by
  outcome, and each fighter's actions per round. Output is a table
  written to the test log; nothing is asserted until 30g5 (it only fails
  on a fight that never ends within 100 rounds).
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
  per bearer, 1d4, stun 25%, and its chant break are unchanged. A plain
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

### 30g4 — The action meter

- **Tempo** per combatant, recomputed each combat round: `speedTerm ×
  (1 − 0.35 × b)`, clamped to `TempoMin`–`TempoMax` (proposed 0.6–1.5).
  `speedTerm` compares the character's Speed with a **level reference**
  (open decision E), never with their target, so a fighter's tempo is the
  same whoever they strike. Formation position plays no part (decision 7).
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
  (+0.1 each, open to 30g5). A weapon's own `attacks` dice are unchanged.
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

### 30g5 — HP, damage, and healing against the target

Gated on the owner's decisions F–H below. The anchor for it:

- Let `e` be one fighter's expected damage per combat round after hit,
  active defense, and armor, and `H` a fighter's health. When both sides
  focus fire, wiping five takes about `(1 + ½ + ⅓ + ¼ + ⅕) × H / e ≈
  2.28 H / e` rounds, so 10–15 rounds means **H ≈ 4.4–6.6 × e** (each
  fighter survives about five average landed hits). When damage is spread
  evenly, everyone falls at about `H / e`, so the same target means
  **H ≈ 10–15 × e**. Real fights, with healing, sit between; 30g1's
  harness says where.
- Today `H` grows with level through both `HPPerLevel` and Vitality's
  own growth, while `e` grows only through the capped Strength bonus, so
  fights lengthen with level. 30g5 fixes the ratio, not a raw number:
  it keeps `H / e` inside the band at every level up to the cap.
- **Scope:** the HP formula (`HPBase`, `HPPerLevel`, `HPPerVitality`,
  possibly a per-archetype hit die), how damage grows (weapon tiers,
  Strength bonus, tempo), healing (heal spells, 32d healer thresholds,
  and 29f's doubled in-combat regeneration), and the 30g2–30g4 numbers,
  all tuned together through the harness.
- **Also retuned here:** 29f's known issue (regeneration and round-timed
  buffs ticking twice per combat round).
- **No cap (decision 11):** the band must hold from level 1 to 60, so
  30g5 extends the harness to levels 1, 5, 10, 30, and 60 (asserted) and
  100 (reported only, a sanity check). With stats growing every level
  (`StatInfo.Recalculate`, soft-capped) and no ceiling, HP and damage
  must grow at the **same rate**, so `H / e` stays flat rather than being
  held in place by a cap.
- The harness then **asserts** the band: median rounds 10–15 at levels
  1, 5, 10, 30, and 60, focus and no focus.

### 30g6 — The XP knee (progression, decision 11)

- `XPTL` (`internal/characters/character.go`) keeps today's formula
  (`(XPBase + L^XPLevelPower × XPLevelFactor × XPBase) × TNLScale`) up to
  a new `XPKneeLevel` (proposed 60); past it, each level's cost is the
  previous level's times a new `XPKneeGrowth` (proposed 1.10, making
  level 100 about 45× a level-60 level). Clamped to `MaxInt` as today.
- The admin progression editor (`/admin/progression`) charts the knee;
  `MaxLevel` stays a display value.
- Existing characters keep their XP and level; only the cost of future
  levels past the knee changes. Nothing new is saved.
- Independent of 30g1–30g5 and can ship any time; `help experience`
  (or the leveling page) says leveling slows sharply after 60.

## Open decisions (lead's recommendations)

- **A. Parry or dodge, when both could apply** (a melee strike on an
  armed fighter without a shield). Recommend: **roll whichever chance is
  higher, once**. It is still one attempt per strike (decision 5), and
  nobody is worse off for holding a weapon. The alternative (always
  parry) makes a light, perceptive fighter weaker for drawing a blade.
- **B. Parry by weapon.** Recommend a small subtype modifier: swords and
  staves +5, axes and maces 0, daggers −5, two-handed polearms +5;
  unarmed and claws can't parry (they dodge). The owner may prefer no
  modifier at first.
- **C. What triggers the shield bash.** Settled: decision 10.
- **D. The armor suffix.** Today's `(5 damage, 2 blocked)` means armor,
  which will read wrongly once shields block. Recommend
  `(5 damage, 2 absorbed)`; `help narration` and 29c's tests follow.
- **E. Tempo's Speed reference.** Recommend **the Speed an untrained
  human has at the character's level** (`GainsForLevel`), with
  `speedTerm = 1 + 0.5 × clamp((Speed − ref) / ref, −0.5, 1)` (0.75–1.5).
  It is stable whoever the fighter faces. The alternative (the battle's
  median Speed) keeps every battle averaging one turn a round but moves a
  fighter's tempo with the enemy, which decision 1 set out to avoid.
- **F. Level cap.** Settled: decision 11 (no cap; an XP knee at ~60,
  ~100 practical).
- **G. HP formula shape** (30g5). Options: keep today's formula and grow
  damage to match; a D&D-like hit die per archetype with a small Vitality
  term; or flatter HP growth. Recommend choosing after 30g1's baseline and
  the cap, by whichever keeps `H / e` in band with the fewest moving parts.
- **H. Whether focus fire should shorten fights** (30g5). With the band
  above, a focused company wins faster than a spread one; recommend
  keeping that (it rewards `tactics`) and tuning the no-focus fight to
  the top of the band.

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
  `MaxTurnsPerRound`), so 30g5 tunes without code.

## Constraints and deferrals

- Spells stay undefended by block, parry, and dodge (a magic defense is
  a later idea from the prior design, not this phase).
- 30d2's wind-ups are separate; the meter doesn't slow heavy attacks.
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
  TestBalance5v5` prints the table for levels 1/5/10, focus and no
  focus; the baseline is recorded; the test is skipped by default.
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
- **30g4, unit:** tempo clamps, the level reference, meter fill, the
  two-turn cap, the 99 carry cap, the opening-round start; **wiring:** a
  fast fighter taking two turns in one round and never three, a slow one
  skipping a round, a stunned fighter losing both turns, a chant stepping
  once a round, a fresh meter after the fight ends.
- **30g5:** the harness asserts median rounds 10–15 at levels 1, 5, 10,
  30, and 60, focus and no focus, and reports 100; decisions G–H
  recorded here.
- **30g6, unit:** `XPTL` unchanged up to the knee, growing by
  `XPKneeGrowth` a level past it, clamped at `MaxInt`; a character past
  the knee keeps its level and XP.
- **Player help (every slice that ships behavior):**
  - new `help defense` (block, parry, dodge, the one-attempt rule,
    armor after), aliases `block`, `parry`, `dodge`, `shield`,
    `shields`; linked from `help combat`;
  - new `help tempo` (the meter, turns, burden, the cap), aliases
    `speed`, `actions`, `turns`, `burden`, `agility`; linked from
    `help combat` and `help encumbrance`;
  - updated: `armor` (no 50% bonus), `interrupts` (counter on a block),
    `narration` (the armor suffix), `statuses` (stunned: no defense),
    the leveling page (slower past 60, 30g6),
    `cargo`/`encumbrance` (combat agility is personal, mounts don't help);
  - the Practice Yard hint (`modules/tutorial/stages.go`) points to
    `help defense` and `help tempo`;
  - each page renders through `help` (the `help_combat_test.go`
    pattern) and `TestTutorialHelpPointersExist` passes.
- Each slice: `go test -race ./...`, `make generate`, `make validate`,
  the independent review, and its **Review:** line in
  `docs/PROJECT_STATUS.md`.
