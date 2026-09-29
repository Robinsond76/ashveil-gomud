# Phase 32e: Company Experience — Plan

Design: [32e design](../specs/2026-09-28-phase-32e-company-experience-design.md).
Decisions are the owner's (roadmap decision 1) plus recommendations applied
2026-09-28. Implemented directly by the lead: it touches the kill path and
persistent companion state.

## Task 1: Award to the company (`internal/mobcommands`)

- [x] Tests first (`companyxp_test.go`): companions in the room, alive and
  attached get the leader's figure; absent room, dead, and unattached
  get none; multi-level from one award; level-up line text.
- [x] `companyxp.ashveil.go`: `awardCompanyXP`, `company.LevelLine`.
- [x] Wire into both `killMob` branches. **Wiring test** through the real
  kill (`suicide` command on a mob damaged by the leader's companion).

## Task 2: `experience` lists the company (`internal/usercommands`, `internal/companyview`)

- [x] Tests first: rows for each companion; solo output unchanged.
- [x] Rows from the company view; append after the player's block.
- [x] **Wiring test** through `Experience` with a company.

## Task 3: Durability (`modules/company`)

- [x] Wiring test: earned XP and level survive snapshot → despawn →
  respawn (`wiring_state_test.go` pattern).

## Task 4: Player help and tutorial

- [x] Update `help experience`, `help company`; tutorial inspection hint;
  render test; `TestTutorialHelpPointersExist`.

## Task 5: Review and verify

- [ ] Independent Opus reviewer over the specs and the diff; verify each
  finding; fix with regression tests.
- [ ] Full checks once; status log with **Review:** line.
