# Phase 30d2: Physical Wind-Ups and an Ogre with Crushing Blow — Plan

Design: [phase-30d2 design](../specs/2026-09-30-phase-30d2-windups-design.md)
(owner decisions 1–6). Branch: `phase-30d2-windups`, worktree
`.worktrees/phase-30d2-windups`, from `origin/master` at `66fc6dc`.

Each task writes its tests first and runs only its own packages.

- [x] **1. The rules (`internal/windup`, new; `internal/interrupt`).**
  - Tests first: `windup_test.go` (`Get` finds `crushing-blow` with its
    numbers, unknown is false; `RollStart` never at 0, certain at 100
    with no roll, `chance-1` starts, `chance` doesn't);
    `interrupt_test.go` `TestBreaksWindUp` (heavy damaging hit breaks; an
    ordinary hit, a miss, 0 damage don't).
  - `internal/windup/windup.go`, `AGENTS.md`; `interrupt.BreaksWindUp`,
    and the package comment no longer lists the shield bash as heavy force
    (decision 2); `internal/interrupt/AGENTS.md`.
- [x] **2. Mob data (`internal/mobs`).** `Mob.WindUps` (`windups:`), and a
  note in `internal/mobs/AGENTS.md`.
- [x] **3. The power blow (`internal/combat`).**
  - Tests first (`windup_power_test.go`): with a provider set, a mob's
    `AttackMobVsPlayer` is one strike, its damage doubled before armor (a
    fixed-dice weapon, crits and dodges off), the target's `BuffTarget`
    holds Knocked Down, the lines carry `(Crushing Blow, N damage, knocked
    down)`; a strike the armor takes entirely has no knockdown; with no
    provider or another mob, the blow is ordinary.
  - `Power`, `SetPowerProvider`, `calculateCombatPower` (the old
    `calculateCombat` wraps it); `AttackMobVsPlayer`/`AttackMobVsMob` ask
    the provider; `damageSuffix` takes a leading name.
- [x] **4. The turn (`internal/hooks/combat_windup.go`).**
  - `windUps`, `windUpCooldown`, `landing`; `windUpRoll`,
    `UseWindUpRollForTest`; `windUpTurn` (start, hold, land, waste,
    cooldown), the reach check (gates' legality), the provider, the
    pruning in `interruptRound`, the break in `afterBlow`, the drop in the
    status-lost turn, `WindUpStart`/`WindUpLand`/`Interrupt` events, the
    lines. `counterBlow` no longer breaks anything (decision 2).
  - `newBrawl` pins the wind-up roll (never starts).
  - Wiring tests first (`modules/company/wiring_windups_test.go`, a
    fixture ogre in the brawl world set on a companion, the roll forced):
    - `TestOgreWindsUpThenLands` (telegraph, `WindUpStart`, no attack
      that turn; next turn release line, `Crushing Blow` parentheses,
      knocked down, `WindUpLand` hit);
    - `TestCritBreaksWindUp` (company crits: broken line, `Interrupt`
      succeeded `Crushing Blow`, no landing, no new wind-up for 2 turns);
    - `TestOrdinaryBlowsLeaveWindUpWhole` (hits, no crits: no line, no
      `Interrupt`, it lands);
    - `TestGuardianTakesCrushingBlow` (30c2: the guard steps in and is
      knocked down);
    - `TestWindUpWastedWhenTargetGone` (the named target leaves: wasted
      line, `WindUpLand` wasted);
    - `TestShieldBashBreaksNoWindUp` (a counter on the landing's miss
      breaks nothing; the ogre isn't winding);
    - `TestSummaryListsBrokenWindUp` (Interrupts dealt: Crushing Blow).
- [x] **5. Content.** Race 22 ogre, item 10022 great club, mob 85 forest
  ogre (`windups: {crushing-blow: 35}`), room 530's spawn.
  - Tests first: `TestShippedForestOgre` (`modules/company`): the shipped
    ogre, in a brawl, winds up and lands on a companion through the real
    round; a content check that every shipped mob's `windups` names a
    registered ability (`internal/mobs` or the wiring test).
- [x] **6. Player help and tutorial.**
  - Tests first: `internal/usercommands/help_interrupts_test.go` checks the
    wind-up section (Crushing Blow, the numbers, what breaks it, the
    shield bash doesn't) and the new aliases.
  - `help interrupts`; `combat`, `statuses`, `guardian`,
    `battle-summary`, `formation`; `keywords.yaml`; the Practice Yard hint
    in `modules/tutorial/stages.go`; `TestTutorialHelpPointersExist`.
- [ ] **7. Correct the docs on the shield bash (decision 2).** The 30d1 and
  30d1b designs (amendment notes); `docs/PROJECT_STATUS.md`'s Next line,
  30d table row, and Known issues.
- [ ] **8. Review and verification.** Loop `modules/company`
  (`-count=5`); independent reviewer over `git diff 66fc6dc..HEAD`;
  verify findings, fix with regression tests; then once:
  `go test -race ./...`, `make generate`, `make validate`;
  `docs/PROJECT_STATUS.md` work-log entry with **Review:**; merge to
  `master` and push.
