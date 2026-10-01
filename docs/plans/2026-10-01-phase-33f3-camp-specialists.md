# Phase 33f3: Camp Specialists — Plan

Design: [33f design](../designs/2026-10-01-phase-33f-company-specialists-design.md),
"33f3: Camp specialists". Worktree `.worktrees/phase-33f3-camp-specialists`,
branch `phase-33f3-camp-specialists`.

## Task 1: utilities and seams

- `internal/archetypes`: watch, fieldsmith, vigil, forage; archetype
  config (warrior watch/fieldsmith on brawling, cleric vigil on
  protection, ranger forage on track); the specialists view lists them.
- `enemyparty.SpawnAmbush` (moved from the expedition module).
- `encumbrance.DepositCargo`/`WithdrawCargo` with `Cargo.Applied`
  operation IDs; `company.RaiseLoyaltyOnce` with `Record.AppliedOps`.

## Task 2: camping

- Field Smith in sharpening. Raids planned at rest start, resolved in the
  round pass (watch roll, broken rest, spawn). Rewards owed at completion,
  paid in the round pass (Forage, Vigil). `camp cook`. Config, road
  brigand mob 86, purge of the new debt.

## Task 3: help and tutorial

`campwatch`, `fieldsmith`, `vigil`, `forage` pages, indexed with aliases;
`camp`, `cooking`, `sharpen`, `specialists`, `autoskill` updated; a Camp
lesson hint.

## Task 4: integration tests

Raid planning and durability, unspotted/spotted/lapsed raids through the
round pass and rest completion, restart; forage and vigil through rest
completion and the round pass, once on retry, capacity; camp cook from
pack and cargo with skill gates and battle refusal; Field Smith through
`sharpen`; cargo and loyalty operations in their modules (reload,
rollback); shipped config; help render.

## Task 5: migration and recovery

All new fields are omitempty; old saves load with no raid, no debt, no
applied operations.

## Task 6: review and integration

Independent reviewer; fixes; Project Status; full checks; merge and push.
