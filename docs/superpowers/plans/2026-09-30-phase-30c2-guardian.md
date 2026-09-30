# Phase 30c2: Guardian Role and Guards — Plan

Design: [phase-30c2 design](../specs/2026-09-30-phase-30c2-guardian-design.md)
(owner decisions 9–12), slice two of the
[phase-30c design](../specs/2026-09-29-phase-30c-company-tactics-design.md).
Branch: `phase-30c2-guardian`, worktree `.worktrees/phase-30c2-guardian`.

Each task writes its tests first and runs only its own packages.

- [x] **1. `internal/strategy` and `internal/formationcombat` (pure).**
  - Tests first:
    - `ParseRole` reads `guardian`, `guard`, `protector`; `ParseRule`
      no longer reads `guard` (`protect`, `defend` still do);
    - `Strategy.Ward` survives `Resolve`; `Default` never a guardian;
      `Decide` swings for a guardian;
    - `GuardWard`: a set ward (only when in reach and not itself), the
      most hurt by fraction in reach, ties by order, nobody hurt → none;
    - `formationcombat.GuardReach`: same column, next column, two away,
      unplaced (fails open).
  - Add them; update `internal/strategy/AGENTS.md`.
- [x] **2. Durable ward (`modules/strategy`).**
  - Tests first (`guardian_test.go`, fake env as `strategy_test.go`):
    `strategy tamsin guard` (guardian, most hurt); `strategy tamsin guard
    me` and `guard oswin`; `guardian <other>`; self refused; unknown
    ward; another role or `default` clears the ward; the reach warning
    (fake members with cells); refused in a battle; listing and describe
    show the ward; a save/load round trip keeps the ward, a ward on a
    non-guardian or a blank-key ward is dropped on load; prune drops a
    ward naming a companion no longer on the record.
  - Add the forms, warnings, listing, cleaning; update `AGENTS.md`.
- [x] **3. Guard counts (`internal/battle`).**
  - Tests first (`guard_test.go`): `GuardsLeft` starts at 2 for a key in
    a battle, 0 with no battle; `SpendGuard` to 0 and refused at 0;
    `TickGuards` refills one per 2 ticks, capped at 2, not when full;
    `End` and a new `Begin` reset; copies out (no shared maps).
  - Add `Battle.Guards`, the functions; update `AGENTS.md` if present.
- [x] **4. The guard in combat (`internal/hooks`).**
  - `combat_guard.go`: `guardFor(leader, struck key)` picks the guardian
    (formation order, able, not down or stunned, in reach, a guard left),
    spends it, narrates, emits `guard-used`/`guard-exhausted`; called in
    `gateMobVsPlayerAttack` and `gateEnemyAttacksCompanion` after
    `resolveAttackTarget`, resolving through the existing
    `resolveInterceptedMobAttack` / `resolveInterceptedAttackOnLeader` /
    plain mob redirect. `guardPass` (the round's refill) after
    `battlePass` in `DoCombat`. The strategy pass skips guardians.
  - Wiring tests (`modules/company/wiring_guardian_test.go`, brawl world,
    real commands and `DoCombat`, `hooks.UseAimRollForTest`):
    - `TestGuardianGuardsWard` (companion guards the player: line, event,
      a guard spent, the player untouched by the redirected blow);
    - `TestGuardianPlayerGuardsCompanion`;
    - `TestGuardianCompanionGuardsCompanion`;
    - `TestGuardsExhaustAndRefill`;
    - `TestGuardianOutOfReach`;
    - `TestGuardianKnockedDown`;
    - `TestGuardianMostHurt`;
    - `TestGuardianRefusedInBattle`.
- [x] **5. `formation` warning (`modules/company`).**
  - Tests first: a `formation move` that puts a guardian two columns from
    its ward prints the warning; moving back clears it (in
    `wiring_guardian_test.go`).
  - Add the warning to the formation command's output.
- [x] **6. Web (`modules/gmcp`, `window-combat.js`).**
  - Tests first: `Company` member `strategy.ward`/`ward_reach`
    (`gmcp_company_test.go`); `Company.Battle` `guards`
    (`gmcp_battle_test.go`).
  - Add `guardian` to `ROLES`, "Guard: …" menu items, the Setup row note,
    the battle view's guard sub-line; extend
    `scripts/browser/dock-windows-check.mjs`.
- [ ] **7. Player help and tutorial.**
  - New `help guardian` (`guardian.template`), in `keywords.yaml` under
    combat with aliases `guardians`, `guard`, `guards`, `ward`, linked
    from `help combat`.
  - Update `help strategy` (the role, ward, the `guard` word), `help
    tactics` (guards beside the focus), `help formation` (a guardian's
    reach), `help webclient` (Setup and battle view), `help
    battle-summary` (the Guards line), `help combat` (hub).
  - The Practice Yard (Combat) lesson points to `help guardian`.
  - Tests: `help guardian` and the updated pages render through `help`
    (`internal/usercommands`); `TestTutorialHelpPointersExist`.
- [ ] **8. Review, fixes, full verification, status, merge.**
  - Independent reviewer over `git diff master..HEAD`; verify each
    finding, fix with regression tests.
  - `go test -race ./...`, `make generate`, `make validate`,
    `make js-lint`, the Playwright checks in `scripts/browser`.
  - `docs/PROJECT_STATUS.md` (What/Why/Verification/Review); merge
    `origin/master` in first, then to `master`, push, remove the
    worktree.
