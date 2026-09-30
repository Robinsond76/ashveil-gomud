# Phase 30g1: Balance Harness and Baseline — Plan

Design: [phase-30g design](../specs/2026-09-30-phase-30g-tempo-defense-design.md),
slice 30g1 (owner decisions 8, 9, 14). Branch: `claude/blissful-ptolemy-2c76l0`
(the session's designated branch; not merged to `master` until the owner
says so).

30g1 is **test-only**: no game behavior, content, or player-visible text
changes, so it ships no player help (the rule covers phases that change
what a player can do or see). Each task writes its tests first and runs
only `modules/company`.

## Decisions made while planning (amended after the first run and review)

- **Distributions, not seeds.** `util.Rand` is the global `math/rand`
  (Go 1.24, auto-seeded), so fights can't be replayed from a seed. The
  harness runs many fights per cell and reports medians and percentiles.
- **Real dice.** `newBrawl` pins the enemy re-aim, shield-counter, and
  chant-break rolls for its deterministic tests; the harness restores
  `util.Rand` for all three, so the baseline is the game as played.
- **Real statuses.** The brawl world has no buffs; the harness loads 30a's
  nine status buffs (`loadStatusBuffs`) and registers the game's `Buff`
  listener (`hooks.ApplyBuffs`), or no crit status ever lands (review).
- **An even fight.** The enemy side mirrors the company: five fixture
  mobs with the companions' kits (Tamsin's sword and shield, Oswin's
  sword and robe, Garrick's broadsword and armor, Ysolde's sling kit, and
  Aria's bare fists), human, in one spawn group. Both sides are set to
  the cell's level the way a spawning mob is (`levelTo`): training
  cleared, the level's stat points given, `AutoTrain`, experience at the
  start of the level (review: template experience levelled Garrick and
  Ysolde on their first kill), full health and mana.
- **Known asymmetries, recorded, not fixed** (they are the game as it
  is, and 30g6's tuning weighs them):
  - the company has 32d strategies (Oswin heals, by archetype); the
    enemies have no healer;
  - a player's `HealthMax` starts with `Base: 1` (`characters.New`), a
    mob's with 0, so Aria gains a little racial HP growth her mirror
    doesn't (+1 at level 5, +3 at level 10; 30g4 replaces the formula);
  - the players' pass of `DoCombat` runs before the mobs', so Aria
    always strikes first;
  - companions and players can be wounded (30b); enemies can't;
  - both sides are **unplaced** (reach and interception fail open), so
    formation plays no part yet.
- **Modes (decision 14).** Company: **spread** (each member a different
  rule: nearest, strongest, furthest, wounded), **default** (the shipped
  strategies: everyone aims at the weakest, so default is already focus
  fire), or **focus** (30c1 `tactics` weakest). Enemies: **spread**
  (targeting `nearest` with noise 100: each new aim is a random foe; an
  aim sticks until its target falls) or **default** (the shipped human
  personality: the weakest, 10% random). The **baseline cell is spread ×
  spread** (no focus on either side).
- **The leader can't die.** A player at −10 dies (25a: a level lost, a
  church). The harness holds Aria at 0 once she drops below 1: she is
  fallen (out of the fight, healable), and never dies.
- **Stalls are reported, not failed.** A fight still going after 200
  rounds (raised from 100 once the first run showed level-10 fights
  reaching it) counts as a stall in the table.
- **What the table counts.** A turn is one attacker's round of blows (one
  `Attack` event); a hit is a turn whose blow passed the to-hit roll and
  wasn't dodged (armor may still take all of it); damage includes
  overkill; shield bashes are counted apart, not as turns (review);
  status ticks count for the side that didn't take them (review).
- **Runtime.** Cells: levels 1, 5, 10 × 3 company modes × 2 enemy modes =
  18. Fights per cell from `ASHVEIL_BALANCE_FIGHTS` (default 50); about
  2–3 minutes for the default table.

## Tasks

- [x] **1. Tally helpers (pure, `modules/company/balance_test.go`).**
  - Tests first (`TestBalanceTally`, `TestBalancePercentile`): a tally
    folds `combatstream` events by side (the company: `UserId` 7 or
    `LeaderUserId` 7; else the enemy): turns, hits, crits, misses,
    counters, damage, tick damage, and healing; percentiles of an odd and
    even list, one value, and none.
  - `balanceTally`, `sideOf`, `percentile`.
- [x] **2. The fight (wiring, same file).**
  - Tests first, always on: `TestBalanceHarnessRunsAFight` (one level-1
    fight through the real `DoCombat` ends, both sides took turns, Aria
    never below 0, the clock never moves); `TestBalanceSidesStayEven`
    (at levels 1, 5, 10 every member matches its mirror on level,
    experience, the combat stats, health (Aria's player HP pinned), and
    armor, and nobody levels during a fight); `TestBalanceStatusesLand`
    (a bleed queued through the game's listener lands and ticks for the
    other side).
  - `newBalanceFight(t, level, companyMode, enemyMode)`: `newBrawl`, 30a
    statuses, real dice, archetypes, the bandits removed, the mirror
    group spawned in the road, everyone levelled, the modes set (the
    strategy commands asserted), a stream subscribed, `attack` on the
    group.
  - `step` (one combat round: the round, the idle-mob pass, regeneration
    for its two game rounds, Aria held at 0) and `run` (steps until a
    side is down or 200 rounds; the result).
- [x] **3. The table (`TestBalance5v5`, gated).**
  - Skipped unless `ASHVEIL_BALANCE=1`; logs one row per cell: company
    wins %, rounds p10/median/p90, stalls, fallen per side, damage and
    healing per side, turns per standing fighter per round, hit and crit
    %, bashes, and tick damage per side.
- [x] **4. Baseline.** Recorded, with a short reading, in the 30g1
  work-log entry in `docs/PROJECT_STATUS.md`.
- [x] **5. Review and verification.** Independent reviewer over
  `git diff 66fc6dc..HEAD`; findings verified and fixed with regression
  tests (see the work log's **Review:** line); `go test -race ./...`,
  `make generate`, `make validate`.

## Baseline (2026-09-30, 50 fights a cell, shipped config)

`ASHVEIL_BALANCE=1 go test ./modules/company -run TestBalance5v5$ -v`,
after the review fixes. Averages are per fight; the baseline cell
(decision 14) is spread × spread.

| level | company | enemy | fights | company wins | rounds p10/median/p90 | stalls | fallen company/enemy | damage company/enemy | healing company | turns per fighter-round company/enemy | hit% company/enemy | crit% company/enemy | bashes company/enemy | tick damage company/enemy |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | spread | spread | 50 | 60% | 6/8/13 | 0 | 3.4/3.4 | 34/34 | 1 | 0.72/0.74 | 42/38 | 16/13 | 0.8/0.9 | 0.2/0.3 |
| 1 | spread | default | 50 | 68% | 6/9/15 | 0 | 3.1/3.9 | 37/36 | 2 | 0.69/0.67 | 43/37 | 14/16 | 1.0/0.8 | 0.1/0.5 |
| 1 | default | spread | 50 | 38% | 7/10/14 | 0 | 4.1/3.8 | 37/42 | 1 | 0.69/0.79 | 41/48 | 15/13 | 0.5/0.7 | 0.1/0.4 |
| 1 | default | default | 50 | 52% | 7/9/13 | 0 | 3.8/4.1 | 39/44 | 1 | 0.63/0.70 | 49/49 | 13/16 | 0.6/0.6 | 0.2/0.6 |
| 1 | focus | spread | 50 | 50% | 7/9/12 | 0 | 3.7/4.2 | 41/39 | 1 | 0.67/0.77 | 50/46 | 15/13 | 0.5/0.8 | 0.1/0.4 |
| 1 | focus | default | 50 | 40% | 7/9/14 | 0 | 4.0/3.9 | 38/46 | 2 | 0.65/0.69 | 49/51 | 14/19 | 0.5/0.6 | 0.1/1.0 |
| 5 | spread | spread | 50 | 66% | 23/32/46 | 0 | 2.9/3.9 | 131/132 | 17 | 0.76/0.83 | 38/36 | 15/16 | 4.0/6.2 | 3.9/2.0 |
| 5 | spread | default | 50 | 50% | 26/34/48 | 0 | 3.5/3.4 | 124/141 | 16 | 0.72/0.83 | 38/36 | 14/15 | 4.9/6.5 | 2.9/1.6 |
| 5 | default | spread | 50 | 80% | 27/32/46 | 0 | 2.2/4.6 | 142/117 | 16 | 0.76/0.84 | 39/37 | 16/15 | 3.7/4.0 | 3.1/3.8 |
| 5 | default | default | 50 | 52% | 26/39/52 | 0 | 3.3/3.9 | 128/132 | 12 | 0.73/0.80 | 38/38 | 16/17 | 4.1/4.1 | 3.3/3.9 |
| 5 | focus | spread | 50 | 80% | 27/33/48 | 0 | 2.3/4.6 | 142/120 | 19 | 0.75/0.85 | 38/38 | 17/16 | 4.1/4.1 | 3.5/2.8 |
| 5 | focus | default | 50 | 74% | 28/39/48 | 0 | 3.0/4.4 | 139/127 | 13 | 0.73/0.80 | 39/39 | 17/14 | 3.7/3.7 | 3.7/4.0 |
| 10 | spread | spread | 50 | 84% | 57/68/92 | 0 | 1.9/4.6 | 300/260 | 56 | 0.77/0.83 | 38/37 | 15/15 | 8.9/15.2 | 8.4/3.6 |
| 10 | spread | default | 50 | 54% | 65/80/97 | 0 | 3.5/4.0 | 281/306 | 35 | 0.73/0.85 | 39/38 | 15/16 | 9.5/14.5 | 11.4/3.3 |
| 10 | default | spread | 50 | 88% | 58/73/92 | 0 | 2.1/4.8 | 304/250 | 43 | 0.78/0.85 | 38/38 | 17/15 | 7.2/8.0 | 8.4/9.5 |
| 10 | default | default | 50 | 66% | 65/82/107 | 0 | 3.0/4.3 | 282/273 | 38 | 0.75/0.80 | 38/38 | 15/15 | 8.8/7.8 | 10.0/11.0 |
| 10 | focus | spread | 50 | 76% | 58/70/90 | 0 | 2.5/4.5 | 292/263 | 42 | 0.78/0.86 | 38/39 | 15/16 | 7.6/8.0 | 7.5/10.6 |
| 10 | focus | default | 50 | 88% | 59/75/104 | 0 | 2.3/4.8 | 301/248 | 34 | 0.75/0.79 | 38/38 | 16/16 | 10.6/7.4 | 10.4/10.1 |
