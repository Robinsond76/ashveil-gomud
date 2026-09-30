# Phase 30d1: Broken Chants, Shield Counters, and Enemy Casters — Plan

Design: [phase-30d1 design](../specs/2026-09-30-phase-30d1-chant-interrupts-design.md)
(owner decisions 1–5), slice one of the
[telegraphs and interrupts proposal](../specs/2026-09-26-telegraphs-interrupts-design.md).
Branch: `phase-30d-interrupts`, worktree `.worktrees/phase-30d-interrupts`.

Each task writes its tests first and runs only its own packages.

- [ ] **1. `internal/interrupt` (pure).**
  - Tests first (`interrupt_test.go`):
    - `Breaks`: a hit with 1+ damage on a chanter breaks; a miss, a hit
      of 0, or a target not chanting doesn't;
    - `Refund`: half, rounded down (0, 1, 3, 8);
    - `CanCounter`: true only when missed, melee, same room, a shield,
      able, not chanting, not yet this round (one table row per
      condition);
    - `RollCounter` with a scripted roll: no bash at 50+, a bash at <50
      with 1d4 damage, a stun at <25 on the stun roll; chances as package
      values a test can set.
  - Add the package and its `AGENTS.md`.
- [ ] **2. `Character.HasShield` (`internal/characters`) and the
  stream's outcome and summary (`internal/combatstream`).**
  - Tests first: `HasShield` (a shield; a held weapon; a holdable with no
    reduction; stunned `no-block` → false); `GetDefense` unchanged
    (existing `TestStunnedCantDodgeOrBlock` still passes); the summary's
    Interrupts line without "failed" when 0, with it when not.
  - Add `HasShield`, use it in `GetDefense`; `OutcomeInterrupted`; the
    summary line.
- [ ] **3. Broken chants in the round (`internal/hooks`).**
  - `combat_interrupt.go`: `afterBlow(attacker, defender statusHolder,
    r, melee)` called after the lines at the six blow sites (four in
    `NewRound_DoCombat.go`, two in `combat_formation.go`); `breakChant`
    (company or player: cancel, refund, `endCast`, lines, events; any
    other mob: stop, mark for restart); `restartChant` at the head of a
    mob's magic turn; pruning each round.
  - Wiring tests (`modules/company/wiring_interrupts_test.go`, brawl
    world, real commands and `DoCombat`):
    - `TestCompanionHealBrokenByBlow` (events, line, half mana back, no
      heal lands);
    - `TestPlayerCastBrokenByBlow` (the player's own `cast`);
    - `TestMissLeavesChantWhole` (every blow misses; the heal lands);
    - `TestEnemyChantBreaksAndRestarts` (a bandit with `hex`: broken by
      a company blow, restarts next turn with the line and `CastStart`,
      lands after its full chant with no more blows);
    - `TestGuardedBlowBreaksGuardianChant` (30c2 redirect: the
      guardian's chant breaks, the ward's doesn't).
- [ ] **4. Shield counters (`internal/hooks`).**
  - In `afterBlow`: on a miss, `CanCounter`, `RollCounter`, damage,
    stun, lines, events, fall resolved this round; the round's counter
    set reset at the top of `DoCombat`.
  - Wiring tests (same file, forced rolls):
    - `TestShieldCounterOnMiss` (companion with a shield: line, damage,
      `Attack` event `shield-bash`);
    - `TestShieldCounterStuns` (forced stun: Stunned on the attacker,
      `StatusApplied`);
    - `TestShieldCounterOncePerRound`;
    - `TestNoShieldCounterAgainstBow`; `TestNoShieldCounterWhileStunned`;
    - `TestEnemyShieldCountersPlayer`;
    - `TestShieldCounterKills` (death resolved that round).
- [ ] **5. Enemy caster content (`_datafiles`).**
  - Tests first: `hex` loads with its cost and chant; mob 70 loads
    (goblin, spellbook `hex`, `cast hex`); rooms 402 and 531 list it
    (`modules/company` or a data test beside other shipped-data tests);
    a wiring test: a hexer in the brawl casts `hex` at the company's
    healer (`casters`), its damage reported as a `spell-hit`.
  - Add `spells/hex.yaml`, `hex.js`, `mobs/dark_forest/70-goblin_hexer.yaml`,
    the spawn entries.
- [ ] **6. Player help and tutorial.**
  - Tests first: `help interrupts` and an alias render
    (`internal/usercommands/help_interrupts_test.go`, as
    `help_combat_test.go`); the tutorial pointer
    (`TestTutorialHelpPointersExist`).
  - New `interrupts.template`; update `combat`, `cast`, `strategy`,
    `tactics`, `statuses`, `guardian`, `battle-summary`, `formation`
    pages; `keywords.yaml`; the Practice Yard hint in
    `modules/tutorial/stages.go`.
- [ ] **7. Review, verification, record.**
  - Independent reviewer subagent over `git diff master..HEAD` with the
    design and invariants; verify each finding, fix with regression
    tests (each in its package).
  - Full verification once: `go test -race ./...`, `make generate`,
    `make validate`, `git diff --check`.
  - `docs/PROJECT_STATUS.md`: work-log entry with **Review:**, phase
    table row, current position, known issues (the counter numbers for
    the owner, 30d2 open question).
  - Merge to `master` and push.
