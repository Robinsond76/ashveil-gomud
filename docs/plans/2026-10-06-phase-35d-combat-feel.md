# Phase 35d — combat feel

Implements the [35d combat feel design](../designs/2026-10-06-phase-35d-combat-feel-design.md).
Branch and worktree: `phase-35d-combat-feel`, from master after 35b
(fb89b3f).

Status: plan written 2026-10-06. The design **awaits the owner's approval**;
implementation begins only when the owner asks for it. Phase 37 waits on
this phase.

## Goal

Make a watched fight read well: every swing has a result, a cleric's
one-round heal resolves, the company patches to 80% and fights on mana for
five to six fights, the company's defaults grow smarter with its leader's
level, HP keeps pace after level 20, bosses are short and scary, and the
harness measures seconds and lines beside rounds. Full reasoning in the
[second opinion](2026-10-06-combat-rebalance-second-opinion.md).

Out of scope: boss abilities (38b), hexes (38a), zone encounter tables and
boss rolls (37), any change to spell damage, crits, mana sources or the 50%
HP trickle.

## Code context (master `fb89b3f`)

- **Blow resolution:** `internal/combat/combat.go` `calculateCombatPower`
  (~L489): `hitRoll` → `activeDefense` → dice → `Crits` →
  `applyDefenseReduction`. Chances in `calculations.go` (`hitChanceForEdge`,
  `edgeChance`, `combinedEdge`, `hitEdge`). Predictions in `expectedDPS`
  and `simulate.go` must use the same helpers. Messages:
  `damagePercentOfMax` picks the message tier; `damageSuffix` adds the
  number and statuses.
- **Chant breaks:** `internal/interrupt.BreakChance(damage, maxHP, heavy,
  difficulty)`, called from `internal/hooks/combat_interrupt.go` ~L110 with
  `heavyBlow(r)` and `chantDifficulty(defender)`. The chant's wait rounds
  come from the spell's `waitrounds` (`heal.yaml`: 1; `healall.yaml`: 2).
- **Tactics:** `internal/strategy/tactics.go` (`Tactics{Focus, Healing}`,
  `DefaultHealing` 50), `modules/company/tactics.go` (the command), the
  healing threshold read in `internal/hooks/combat_strategy.go` ~L121 and
  `combat_coordination.go` ~L327. Patch mode in `modules/company/wounds.go`
  and the `events.BattleEnded` handler (35b decision 10).
- **Member defaults:** `internal/strategy/strategy.go` `Default(archetype)`
  (role by class, rule `Weakest`), `Resolve`; guardian wards (`Ward`, 30c2);
  `internal/battle/guard.go` `MaxGuardsFor`. Company focus in
  `internal/battle/focus.go` and `internal/hooks/combat_engagement.go`
  `retarget`. Enemy tiers in `internal/coordination`.
- **HP:** `internal/configs/config.progression.go` `HealthAtLevel`,
  `HPAfterFull` (0.2), `HPFullLevels` (20); `_datafiles/config.yaml` ~L457.
- **Harness:** `modules/company/balance_test.go` (`newBalanceFightWithOptions`,
  `balanceFightOptions{Boss, Coordination}`, `TestBalanceZoneBands` cells
  ~L1121, `zoneMiddleRounds` 12, `balanceRow` columns),
  `balance_mana_test.go` (`TestBalanceManaRun`, 4 fights). `RoundSeconds`
  is 4 (`config.yaml` ~L164).
- **Help:** `_datafiles/world/default/templates/help/`, `keywords.yaml`,
  `modules/tutorial/stages.go`, render tests in
  `internal/usercommands/help_combat_test.go`.

## Implementation decisions

1. **Quality roll.** Add `Combat.ToHitEven` 88 and a `BlowQuality` config
   block (`GlanceEven` 25, `TellingEven` 20, `GlanceFactor` 0.5,
   `TellingFactor` 1.4, with the edge moving glancing toward 5/50 and
   telling toward 50/5). `blowQuality(edge, roll)` in `calculations.go`
   returns the factor and the word; `calculateCombatPower` applies it to the
   dice total before the crit bonus and `applyDefenseReduction`. The message
   tier reads the post-quality percent; a telling blow appends its word in
   the damage suffix. `expectedDPS` multiplies by the expected factor for the
   edge. `AttackResult` gains `Qualities []string` for the battle view and
   the 40e event stream.
2. **Heal exemption.** `interrupt.BreakChance` gains a `resolves bool`
   (true for a one-round chant whose spell is a heal use); when set and the
   blow is not heavy, the chance is 0. `combat_interrupt.go` derives it from
   the defender's chant (`waitrounds` 1) and `strategy.SpellFor(..., UseHeal)`.
   No spell YAML changes.
3. **Patch threshold.** `Tactics.Patch int` (`yaml:"patch,omitempty"`),
   `DefaultPatch` 80, `ParsePatch` (50 to 100), `company tactics patch <n>`,
   shown in `company tactics`. Patch mode and `company patch` read it in
   place of `Healing`; in-battle healing is unchanged. Saved with the
   company's tactics.
4. **Company ladder.** `strategy.DefaultsFor(leaderLevel, roster)` returns
   the default focus (`Weakest` from level 10) and the default ward (the
   first warrior guards the first healer, when both exist and the warrior
   has no ward); `Resolve` and the engagement's tactics read apply them only
   where the player set nothing. `Casters` focus becomes "casters first"
   (casters while any stand, else weakest) in `strategy.EnemyPick`'s
   company counterpart from level 25. `company tactics` prints "(default at
   your level)" beside an inherited value.
5. **HP after 20.** `HPAfterFull` 0.4 in `config.yaml` (code default
   unchanged). Verify the 1.6× cap and the fifth-of-HP solid blow at 30 and
   60; fall back to holding `DamagePerStrength` growth after level 20 if the
   cap breaks.
6. **Harness.** Boss cells: `Coordination: 1` (Rabble), `EnemyCount` 3 or 4
   (boss plus 2 or 3 escorts at `band[0]`), boss HP `+0.75×`. Four-foe cells
   use `band[0]`. The under row uses `band[0] - 5`. `balanceRow` gains
   seconds (median rounds × `RoundSeconds`) and lines per fight. The mirror's
   round assertion is removed (reported). `TestBalanceManaRun` runs 6 fights
   with the design's assertions. Tier probes run the default company (now
   with the ladder) and keep the old no-ladder cells as a report.
7. **Spells** are unchanged. `SpellFactor` already carries the edge.

## Tasks

- [ ] **Owner approval** of the design, recorded in Project Status.
- [ ] **Tests first**, red before the code:
  - `blowQuality` odds at edge −1, 0, +1 and the factors; `expectedDPS`
    agrees with a seeded real pass; the message tier and suffix for each
    quality; the battle view line;
  - `BreakChance` with `resolves` true and heavy false is 0, heavy still
    100, attack spells and two-round heals unchanged; a real chant through
    `AttackMobVsPlayer` with seeded dice;
  - `ParsePatch` bounds; patch mode heals to 80 and stops at the reserve;
    in-battle healing still 50; save and reload of `Patch`;
  - `DefaultsFor` at leader levels 1, 10, 25 with and without a healer,
    with a set ward or focus overriding; casters-first picks;
  - `HealthAtLevel` at 30 and 60 for each archetype under 0.4;
  - harness options for the boss shape and columns.
- [ ] **Quality roll** (decision 1): config, helper, `calculateCombatPower`,
  `expectedDPS`, `simulate.go`, messages, battle view.
- [ ] **Heal exemption** (decision 2).
- [ ] **Patch threshold** (decision 3) and the command.
- [ ] **Company ladder** (decision 4).
- [ ] **HP after 20** (decision 5).
- [ ] **Harness** (decision 6): cells, columns, mana run, then tune within
  the design's ranges under the 10-minute timebox rule; record every table
  and tuned value in `docs/plans/2026-10-06-phase-35d-measurements.md`,
  including the seconds and lines columns and heals finished.
- [ ] **Help and tutorial** per the design: pages, aliases, hints, render
  tests, `TestTutorialHelpPointersExist`.
- [ ] **Independent full-diff reviewer** (default model, reports only);
  verify and fix findings with regression tests; record accepted and
  rejected findings in Project Status.
- [ ] **Final checks:** `make generate`, `make validate`, JS lint (and Lua
  lint if available), `go test -race ./...`. Project Status entry; merge
  master (36a and 38a may have landed); commit and open a PR.

## Acceptance

The design's acceptance tests 1 to 10 hold, in particular:
- at level 10, at most 30% of swings end with no damage, and a cleric
  finishes at least 75% of the heals it begins;
- a band-middle fight is a median of 6 to 9 rounds and 25 to 40 seconds;
- the mana run reaches five fights with wins ≥80%;
- nobody falls in ≥85% of band-middle fights at every band;
- bosses win 70 to 85% in 12 to 18 rounds;
- the kept rows (skill wins, warriors the tanks, the 1.6× HP cap) still
  pass;
- no world-time change; saves and copyover are intact.
