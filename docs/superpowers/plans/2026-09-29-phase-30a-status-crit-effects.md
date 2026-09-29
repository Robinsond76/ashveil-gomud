# Phase 30a: Status Effects and Critical-Hit Effects — Plan

Design: [phase-30a design](../specs/2026-09-29-phase-30a-status-crit-effects-design.md).
Branch: `claude/next-phase-7ekckc` (the session's designated branch; no
worktree). Each task writes its tests first and runs only its own packages.

- [ ] **1. Buff stacks (`internal/buffs`).** Test: `AddBuff` on a live buff
  with `MaxStacks: 3` reaches 3 stacks and refreshes the count; `MaxStacks`
  unset stays at 1. Add `Buff.Stacks`, `BuffSpec.MaxStacks`; update the package
  `AGENTS.md`.
- [ ] **2. Data.** Buff YAMLs 1100–1108, flag files (`combat-status`,
  `lose-actions`, `lose-first-action`, `armor-broken`, `exposed`, `bleeding`,
  `burning`). Test: every buff loads, validates, and its flags exist.
- [ ] **3. `internal/status` (pure logic).** Ids, crit table by subtype
  (weapon override wins), tick rules (bleed damage by stacks, alive-after-tick,
  lose-action rules), line text. Tests: table by subtype, cleaving weights,
  tick and skip rules, every crit id names a shipped buff.
- [ ] **4. Combat resolver.** `calculateCombat` puts the crit effect in
  `BuffTarget`; `damageSuffix` names it; `GetDefense` halves for
  `armor-broken`; `Crits` +25 against `exposed`. Wiring tests through
  `AttackPlayerVsMob`: a forced crit per subtype yields the buff and the text.
- [ ] **5. Combat loop.** `combatstream.StatusTick`; `combat_status.go`
  `statusPass`, action skip in player/mob loops and strategy pass, clear at
  fight end and with no open fight, death by bleeding. Wiring tests through
  `DoCombat`: bleed ticks and stacks and ends at fight end and kills;
  knockdown loses one action; stun two; expiry event; stale clear.
- [ ] **6. Spells.** Sparks applies Overloaded; a scripted test through a real
  cast.
- [ ] **7. Player help and tutorial.** `help statuses` page, `keywords.yaml`
  entries and aliases, links from `combat` and `narration` pages, a pointer in
  the practice-fight tutorial lesson, render test, `TestTutorialHelpPointersExist`.
- [ ] **8. Review and verification.** Independent reviewer over
  `git diff master..HEAD`; verify findings, fix with regression tests;
  `make generate`, `make validate`, `go test -race ./...` once; update
  `docs/PROJECT_STATUS.md` with the **Review:** line; remove the superseded
  proposal spec.
