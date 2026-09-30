# Phase 30d1b: A Blow Breaks a Chant Only by Chance — Plan

Design: [phase-30d1b design](../specs/2026-09-30-phase-30d1b-chant-break-chance-design.md)
(owner decisions 1–5). Branch: `phase-30d1b-chant-break-chance`, worktree
`.worktrees/phase-30d1b-chant-break-chance`.

Each task writes its tests first and runs only its own packages.

- [ ] **1. The rule (`internal/interrupt`, pure).**
  - Tests first (`interrupt_test.go`): `BreakChance` table (heavy, nick,
    quarter, 10%, zero damage, zero max health, clamp at 90);
    `RollBreak` (100 needs no roll, 0 never, `chance-1` breaks,
    `chance` holds); `CanBreak` (30d1's `Breaks` table, renamed).
  - Add `BreakChance`, `RollBreak`, the min/max constants; rename
    `Breaks` → `CanBreak`; update the package doc and `AGENTS.md`.
- [ ] **2. The roll in the round (`internal/hooks/combat_interrupt.go`).**
  - `breakRoll` and `UseBreakRollForTest`; `heavyBlow`; `afterBlow`
    rolls; `holdChant` (room line, the player's line, `Interrupt`
    `failed`); the bash breaks as heavy.
  - `newBrawl` pins the break roll to 0.
  - Wiring tests first (`modules/company/wiring_interrupts_test.go`):
    - `breakDice` helper; 30d1's chant tests force a break;
    - `TestChantHoldsOnLightBlow` (Oswin's heal, roll forced high:
      line, `Interrupt` failed, still chanting);
    - `TestPlayerChantHolds` (the player's line);
    - `TestHeavyBlowAlwaysBreaks` (a crit; a blow whose statuses
      stagger, both with the roll forced high);
    - `TestSummaryCountsHeldEnemyChant` ("failed 1").
- [ ] **3. Player help and tutorial.**
  - Tests first: `internal/usercommands/help_interrupts_test.go` checks
    the chance, the heavy blows, and the held line.
  - `help interrupts` (numbers, heavy blows, held line); `combat`,
    `cast.md`, `strategy`, `tactics`, `formation`, `guardian`; the
    Practice Yard hint in `modules/tutorial/stages.go`;
    `TestTutorialHelpPointersExist`.
  - Summary comment in `internal/combatstream/summary.go`.
- [ ] **4. Record 30d2's decision.** The 30d1 design's Deferrals and
  `docs/PROJECT_STATUS.md`'s Next line: a physical wind-up breaks only on
  heavier force (crit, stagger, knockdown, stun, shield bash).
- [ ] **5. Review and verification.** Loop `modules/company`
  (`-count=5`); independent reviewer over `git diff 7306212..HEAD`;
  verify findings; `go test -race ./...`, `make generate`,
  `make validate`; `docs/PROJECT_STATUS.md` work-log entry with
  **Review:**; merge to `master` and push.
