# Phase 30g5: Action meter — delivery plan

Branch: `phase-30g5-action-meter`. Worktree: `.worktrees/phase-30g5-action-meter`.
Base: `b62f1987` (30f and 30g4 merged). Status: complete, independently reviewed and fully verified,
authorized by the owner on 2026-10-03; pet proposals removed.

Design: [agreed 30g roadmap](../designs/2026-09-30-phase-30g-tempo-defense-design.md).
Proposals: [30g5 amendment](../designs/2026-10-02-phase-30g5-action-meter-amendment.md).

1. Done: pull latest master, include shipped 30g4, and remove the pet proposal
   at the owner's request. Use the amendment's documented starting defaults.
2. Add config validation and pure tempo/meter tests first. Cover opening
   credit, changing tempo, decimal boundaries, both caps, and rejected invalid
   values. Implement tempo from effective Speed and personal burden.
3. Add runtime allocation/cleanup in hooks, with regressions for battle end,
   immediate replacement, absent/withdrawn actors, shared enemies, PvP,
   mob-vs-mob, and fresh state after restart-equivalent reset. No saved fields.
4. Refactor physical turns without repeating upkeep. Wire the four attack
   directions and test fast doubles, slow skips, mid-round death/statuses,
   target/reach checks, dual wield/weapon attacks, and unchanged world time.
5. Wire abilities, chants, waits, wind-ups, guards and counters into the
   same round budget. Add real `DoCombat` regressions in hooks and the
   `modules/company` brawl harness; helper tests alone do not complete this step.
6. Remove target-relative unarmed/claw extra turns and update damage estimates,
   weapon ranking and simulation. Replace superseded tests with tests of the
   new per-turn/per-round contract; inspect existing goldens before updating.
7. Add indexed `tempo.template`, hub links and Practice Yard hint; correct
   existing speed/action prose and test help rendering/tutorial pointers.
8. Run focused package checks and the opt-in 30g1 5v5 balance harness. Record
   duration, turn counts, and narration volume against the recorded baseline;
   compare against the post-30g4 baseline.
9. Obtain the required independent reviewer subagent for the complete diff.
   Verify all findings, fix confirmed problems with regressions, then run
   `make generate`, `make validate`, applicable lint and `go test -race ./...`.
   Record actual results and accepted/rejected review findings in
   `docs/PROJECT_STATUS.md`, commit on the feature branch, then integrate only
   after the phase gate passes. Do not push without authorization.

Tasks 1–9 implemented. Final verification and balance measurements are recorded
in [30g5 verification](2026-10-03-phase-30g5-verification.md).
