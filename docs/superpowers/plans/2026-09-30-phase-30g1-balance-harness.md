# Phase 30g1: Balance Harness and Baseline — Plan

Design: [phase-30g design](../specs/2026-09-30-phase-30g-tempo-defense-design.md),
slice 30g1 (owner decisions 8, 9, 14). Branch: `claude/blissful-ptolemy-2c76l0`
(the session's designated branch; not merged to `master` until the owner
says so).

30g1 is **test-only**: no game behavior, content, or player-visible text
changes, so it ships no player help (the rule covers phases that change
what a player can do or see). Each task writes its tests first and runs
only `modules/company`.

## Decisions made while planning

- **Distributions, not seeds.** `util.Rand` is the global `math/rand`
  (Go 1.24, auto-seeded), so fights can't be replayed from a seed. The
  harness runs many fights per cell and reports medians and percentiles.
- **Real dice.** `newBrawl` pins the enemy re-aim, shield-counter, and
  chant-break rolls for its deterministic tests; the harness restores
  `util.Rand` for all three, so the baseline is the game as played.
- **An even fight.** The enemy side mirrors the company: five fixture
  mobs with the companions' kits (Tamsin's sword and shield, Oswin's
  sword and robe, Garrick's broadsword and armor, Ysolde's sling kit, and
  Aria's bare fists), human, in one spawn group. Both sides are set to
  the cell's level the way a spawning mob is: training cleared, the
  level's stat points given, `AutoTrain`, full health and mana.
  Asymmetry recorded, not fixed: the company has 32d strategies (Oswin
  heals, by archetype), the enemies have no healer.
- **Tactics both ways (decision 14).** Company focus none (each member's
  own strategy) or weakest (30c1 `tactics`). Enemies **spread**
  (targeting `nearest`, noise 100: every re-aim a random foe) or
  **focus** (`weakest`, noise 0). Shipped humans already target the
  weakest, so the spread cell needs the override. The baseline cell is
  company none × enemies spread.
- **The leader can't die.** A player at −10 dies (25a: a level lost, a
  church). The harness holds Aria at 0 once she drops below 1: she is
  fallen (out of the fight, healable), and never dies.
- **Stalls are reported, not failed.** A fight still going after 200
  rounds (raised from 100 once the first run showed level-10 fights
  reaching it) counts as a stall in the table (amending the design's "fails"),
  so a baseline with a stall can still be recorded and looked into.
- **Runtime.** Cells: levels 1, 5, 10 × 2 × 2 = 12. Fights per cell
  from `ASHVEIL_BALANCE_FIGHTS` (default 50).

## Tasks

- [ ] **1. Tally helpers (pure, `modules/company/balance_test.go`).**
  - Tests first (`TestBalanceTally`, `TestBalancePercentile`): a tally
    folds `combatstream` events by side (the company: `UserId` 7 or
    `LeaderUserId` 7; else the enemy): turns (`Attack` events), hits,
    crits, misses, damage, and healing (`Heal` `Amount`); percentiles of
    an odd and even list, one value, and none.
  - `balanceTally`, `sideOf`, `percentile`.
- [ ] **2. The fight (wiring, same file).**
  - Test first, always on (`TestBalanceHarnessRunsAFight`): one level-1
    fight through the real `DoCombat` ends (win, loss, or stall), both
    sides took turns, and Aria is never below 0.
  - `balanceBrawl(t, level, companyFocus, enemyRule)`: `newBrawl`, real
    dice, `withArchetypes`, the mirror group spawned in the road,
    everyone levelled, the tactics saved, a stream subscribed, `attack`
    on the group.
  - `runBalanceFight`: rounds until one side is down or 200 rounds;
    Aria held at 0; the result (rounds, outcome, fallen per side, the
    tally).
- [ ] **3. The table (`TestBalance5v5`, gated).**
  - Skipped unless `ASHVEIL_BALANCE=1`; runs every cell and logs one
    row per cell: company wins %, rounds median/p10/p90, stalls, fallen
    per side, damage and healing per side, turns per living fighter per
    round, hit and crit %.
- [ ] **4. Baseline.** Run it; record the table in the work-log entry
  and a short reading of it (where the baseline sits against 10–15
  rounds).
- [ ] **5. Review and verification.** Independent reviewer over the
  branch diff (design, invariants, the harness's fairness); verify
  findings; `go test -race ./...`, `make generate`, `make validate`;
  `docs/PROJECT_STATUS.md` work-log entry with **Review:**.
