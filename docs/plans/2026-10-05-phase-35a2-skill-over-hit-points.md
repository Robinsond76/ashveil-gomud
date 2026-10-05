# Phase 35a2 — skill over hit points

Implements the [35a2 skill over hit points design](../designs/2026-10-05-phase-35a2-skill-over-hit-points-design.md).
Branch and worktree: `phase-35a2-skill-over-hp`.

Status: plan written 2026-10-05; the design is **owner-approved
(2026-10-05)**. Implementation has not started and begins only when the
owner asks for it. 35b waits on this phase.

## Goal

A level makes a character better at fighting, not thicker:
- derived **Attack** and **Evasion** ratings add a skill edge to every
  opposed chance (hit, dodge, parry, block, bash, Tackle), never to crits;
- HP grows a little from a higher floor with a class head start, and a
  smaller Strength damage bonus keeps an ordinary landed hit at 15–25% of a
  same-level warrior's HP;
- armor **bulk** makes heavy armor slow, with per-class armor training;
- shields are for warriors (any size) and rangers (bucklers); clerics use
  maces, staffs and rods, with a holy symbol;
- spells and heals are resized to a weapon hit (this phase changes the
  shipped spell scripts; 35b's plan is updated to match).

Out of scope: no-fizzle, mana pools and potions, after-battle patching
(35b); talents, promotions and route ranks (38b); the Witch (35b).

## Code context (master `48bc7f9`)

- **Opposed chances:** `internal/combat/calculations.go`: `StatEdge`,
  `statAdvantage`, `edgeChance`, `advantageChance`, `hitChance`,
  `critChance`, `dodgeChance`, `burdenedDodge`, `blockChance`,
  `parryChance`, `BashChance`, `damageBonus`. `activeDefense`
  (`combat.go:219`) picks block for a shield-bearer, else the better of
  parry and dodge. Tackle: `strategy.TackleChance` (`internal/strategy/abilities.go:171`).
- **Predictions:** `simulate.go`, `expectedDPS`, `ExpectedDamage`,
  `CombatOdds` (`calculations.go:414–560`) must use the same helpers.
- **HP:** `ProgressionConfig.HealthAtLevel` and `HealthAfterFull`
  (`internal/configs/config.progression.go:224`); `Character.HealthGainPerLevel`
  (`internal/characters/character.go:2495`) reads the archetype rate
  through `archetypes.HealthPerLevel` (`modules/archetype/archetype.go:874`).
  The admin editor calls `HealthAtLevel` (`internal/web/api_v1_progression.go:197–215`).
- **Tempo and burden:** `combat.Tempo` (`internal/combat/tempo.go:12`)
  and `Character.Burden` (`internal/characters/load.go:55`).
- **Chants:** `Character.SetCast(waitRounds, …)` is the single entry for
  player, companion and enemy casts; `ColdDelay` (`internal/characters/cold.go`)
  is the precedent for an added delay.
- **Items:** `ItemSpec` (`internal/items/itemspec.go:209`);
  `Character.HasShield` (`character.go:611`) treats any non-weapon
  off-hand with armor as a shield. Highest ids: weapons 10022, armor 20045.
- **Equip paths:** `internal/usercommands/equip.go`; company gear
  `modules/company/equipment.go` (`equipmentCommand`, `equipmentActor`).
- **Kits:** archetype overlay `Kit` lists; `modules/archetype/kit.go`
  grants once. The cleric kit is `[10015 crude cudgel, 20004 wooden
  shield, 20008, 30004, 30015, 30001, 30001, 38]`.
- **Spells:** `_datafiles/world/default/spells/mm.js`, `heal.js`,
  `healall.js`, `sparks.js`, `hex.js`; Opening Strike in
  `internal/hooks/combat_abilities.go:269`.
- **Display:** `status` (`internal/usercommands/status.go`,
  `status.panels.go`), `consider` (`consider.go`), level-up template
  `templates/character/levelup.template` and `events.LevelUp` (35a).
- **Harness:** `modules/company/balance_test.go` (opt-in, zone-band cells
  from 35a); `TestBalanceClassHPShape` (`wiring_balance_test.go:265`)
  asserts the cleric and ranger share the middle HP rate, which 35a2
  deliberately changes.

## Implementation decisions

1. **One edge helper.** Add `SkillEdge(attack, evasion int) float64` beside
   `StatEdge` (span `Combat.SkillEdgeSpan`, default 20) and
   `combinedEdge(skill, stat float64) float64` (clamped to ±1). Each
   chance function takes the attacker and defender characters (or their
   ratings) instead of bare stats, so callers can't forget the skill term.
   `critChance` is untouched.
2. **Two-sided defenses.** Dodge and parry move from `advantageChance`
   (one-sided) to `edgeChance` with new `DodgeChanceEven` and
   `ParryChanceEven` (12), bounds 3–40. Block: `BlockChanceMin` 8,
   `BlockChanceMax` 55, even = 15 + shield armor. Hit: `ToHitMin` 10,
   `ToHitMax` 95, `ToHitEven` 60.
3. **Ratings are derived.** `Character.AttackSkill()` and `Evasion()` =
   `floor(level × rate) + template offset`. Rates come from the archetype
   (`AttackRate`, `EvasionRate`), else `Combat.DefaultAttackRate` and
   `DefaultEvasionRate` (1.0). Mob templates gain `attackskill` and
   `evasion` offsets, validated to ±5 at load. Nothing is saved.
4. **HP.** Add archetype `HPStart` and `Progression.HPBase` 48;
   `HealthAtLevel` gains a `start` argument (callers pass the archetype's,
   enemies 0). New `HPPerLevel`: warrior 1.0, ranger 0.8, rogue 0.7,
   cleric 0.6, wizard 0.5; `DefaultHPPerLevel` 0.8; `HPAfterFull` 0.2.
   `TestBalanceClassHPShape` drops the "cleric equals ranger" assertion
   (design decision: clerics are casters) and checks the new shape
   (acceptance 2). Saved health clamps on load, never refills (30g4).
5. **Damage bonus.** `DamageBonusMin` 6, `DamageBonusMax` 12,
   `DamagePerStrength` 0.375, `DamageEdgeMax` 2.
6. **Bulk.** `ItemSpec.Bulk` (`light|medium|heavy`), defaulted at load from
   weight for armor without one (≥ 6 kg heavy, ≥ 2.5 kg medium), and tagged
   explicitly on shipped armor. `Character.ArmorBulk()` is the heaviest
   piece worn. Penalties are config (`Combat.BulkTempo`, `BulkDodge` per
   bulk), applied in `Tempo` and after `burdenedDodge`. Untrained bulk
   (archetype `ArmorTraining`) doubles both, subtracts 10 from Attack and
   Evasion, and adds `ArmorChantDelay()` (1) inside `SetCast`, beside
   `ColdDelay`.
7. **Shields and weapon classes.** `ItemSpec.ShieldSize`
   (`buckler|shield|tower`, default `shield` for existing shields) and
   `ItemSpec.WeaponClass` (`mace|staff|rod|club|improvised|…`). Archetype
   `ShieldSizes` and `WeaponClasses` (empty = any). One check,
   `archetypes.CanWield(id, spec) (ok bool, reason string)`, used by
   `equip`, company gear, kit granting and the load-time unequip.
8. **Load-time unequip.** On user load (beside 35a's stat-point catch-up,
   saved with the same atomic save) and on companion load, a disallowed
   shield or weapon moves from the hand to carried items with a one-time
   message. Idempotent; tested across reload and copyover.
9. **New items:** `10023` acolyte's mace (1d6, `mace`, one hand);
   `20046` holy symbol (off-hand, no armor so never a shield, +5% healing
   via a stat mod read by heal spells); `20047` leather buckler (armor 3,
   `buckler`, light); `20048` tower shield (armor 14, `tower`, heavy).
   The cleric kit becomes `[10023, 20046, 20008, 30004, 30015, 30001,
   30001, 38]`. Existing characters keep their granted kit (exactly-once
   rule); the load-time unequip handles their shield and cudgel.
10. **Spells.** Damage spells multiply by `1 + 0.5 × edge` (caster Attack
    vs target Evasion, shared helper exposed to scripts). Magic Missile
    `7 + 1d6 + level/10 + Mysticism/15`; Minor Heal `8 + 2d4 + level/6`;
    Minor Heal All 55% of it; Sparks and Withering Hex keep their ratio to
    Magic Missile; Opening Strike bonus `2 + level/6`. The holy symbol's
    +5% applies to heals.
11. **Display.** `status` shows Attack, Evasion and bulk; the level-up
    report adds `Attack a -> b   Evasion c -> d`; `consider` describes the
    gap in words by thresholds (≥ 10 ahead "far more skilled", 4–9
    "more skilled", ±3 "evenly matched", and the mirror phrases behind).
    Companion level lines gain the same pair.
12. **Admin editor** charts HP per archetype (with `HPStart`) and the two
    ratings by level; new config keys are editable.

## Tasks

- [x] **Owner approval** of the design recorded in Project Status (2026-10-05).
- [ ] **Tests first**, red before the code:
  - `SkillEdge`/`combinedEdge` bounds; every chance at even, ±10 and ±20
    rating gaps with equal stats; crit chance unchanged by skill;
  - ratings per archetype, default, template offset (±5 validation), and a
    lost level lowering both;
  - HP per archetype at 1/10/20/30/60 matches the design table;
  - damage bonus at Strength 3/6/8/10/16;
  - bulk defaults from weight, penalties, doubled and −10 when untrained,
    +1 chant through a real cast;
  - `CanWield` table for every class against buckler, shield, tower, mace,
    staff, rod, club, sword; refusals through the real `equip` and
    `company equip`; the load-time unequip once across reload and copyover;
  - real `AttackPlayerVsMob` / `AttackMobVsPlayer` passes with seeded
    rolls showing the skill edge changing hit and defense outcomes;
  - simulator and `ExpectedDamage` agree with the real chances.
- [ ] **Combat edge** (decisions 1–2, 5) in `calculations.go`, Tackle and
  the simulator; config keys and Validate defaults in `internal/configs`;
  `_datafiles/config.yaml` values with comments.
- [ ] **Ratings** (decision 3): characters, archetype overlay fields, mob
  template offsets.
- [ ] **HP** (decision 4): `HPBase`, `HPStart`, new rates, the editor's
  calls, `TestBalanceClassHPShape` update.
- [ ] **Bulk and armor training** (decision 6): item field and defaults,
  shipped armor tags, tempo/dodge penalties, `ArmorChantDelay`.
- [ ] **Shields, weapon classes and items** (decisions 7–9): item fields,
  shipped weapon tags (quarterstaff `staff`, royal scepter `rod`, cudgels
  and great club and tree trunk `club`, sharp stick, crowbar, oar
  `improvised`), the four new items, shop stock for buckler and tower
  shield, the cleric kit, `CanWield`, equip and company refusals, the
  load-time unequip.
- [ ] **Spells** (decision 10): scripts, the edge helper for scripts,
  Opening Strike, holy symbol bonus; update the 35b plan's numbers to
  match and remove its "pending 35a2" note.
- [ ] **Display** (decision 11) and **admin editor** (decision 12),
  with a rendered level-up and `status` test through the real paths and a
  browser check of the editor.
- [ ] **Harness and tuning:** add the design's acceptance rows to the
  opt-in suite (hit size, small HP, skill wins, equal-fight rounds, tank
  shape) and keep the 30g6 mismatch rows. Run at 100 fights a cell;
  tune only the config values, within the design's ranges. Record the
  tables and every tuned value in
  `docs/plans/2026-10-05-phase-35a2-measurements.md`. A change outside the
  design's ranges goes to the owner first.
- [ ] **Help and tutorial:**
  - new `help evasion` (aliases `attack skill`, `combat skill`) and `help
    shields` (aliases `shield`, `buckler`, `block`);
  - update `attack`, `armor` (bulk, training; aliases `bulk`, `heavy
    armor`), `defense`, `speed`, `perception`, `strength`, `health`,
    `progression`, `combatpace`, `combat` (hub links), `abilities`,
    `experience`, `equip`, `company-inventory`, `warrior`, `ranger`,
    `archetype` and the cleric's class text; spell pages for the new
    numbers;
  - `keywords.yaml` entries and aliases;
  - tutorial: combat lesson → `help evasion`, gear lesson (or Departure)
    → `help armor` and `help shields`; fix stale HP and damage hints in
    `modules/tutorial/stages.go`;
  - render tests in the `help_combat_test.go` pattern;
    `TestTutorialHelpPointersExist` passes.
- [ ] **Independent full-diff reviewer** (default model, reports only).
  Verify and fix findings with regression tests; record accepted and
  rejected findings in Project Status.
- [ ] **Final checks:** `make generate`, `make validate`, JS lint (and Lua
  lint if available), `go test -race ./...`. Project Status entry; commit
  and open a PR.

## Acceptance

The design's acceptance tests 1–9 hold, in particular:
- the middle 80% of 1d10 hits between equal unarmored warriors land within
  13–27% of max HP (average 17–23%) at levels 1, 10, 30 and 60;
- level-60 HP is at most 1.6× level-1 HP per archetype, and a warrior has
  at least 1.25× a wizard's HP at level 60;
- a level-20 company beats a level-10 group of 3 in ≥ 99% of fights, and a
  level-10 company loses ≥ 70% against a level-20 group of 2;
- warriors in trained heavy armor with a shield are the hardest targets;
  untrained heavy armor costs a rogue at least 20% of its turns;
- clerics can't hold shields or non-mace/staff/rod weapons through any
  path, and existing clerics are fixed once on load;
- no world-time change; saves, copyover and the tutorial replay are intact.
