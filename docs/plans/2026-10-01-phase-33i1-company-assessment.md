# Phase 33i1: Company Encounter Assessment — Plan

Design: [33i design](../designs/2026-10-01-phase-33i-company-assessment-enemy-roles-design.md),
"Final implementation decisions: 33i1". Worktree
`.worktrees/phase-33i1-assessment`, branch `phase-33i1-assessment`.

## Task 1: read model

- `internal/combat`: export `ExpectedDamage` (the existing deterministic
  `expectedDPS`).
- New `internal/assessment`: pure `Estimate(own, foes Side, damage)` →
  risk, closeness, members without reach, foes out of reach; `Gather`
  reads the live company and a visible enemy group on the game loop;
  `Lines` renders text, `Outlook` the browser summary.

## Task 2: entry points

- `scout [group]` appends the assessment.
- `consider [enemy]` assesses the enemy's group; player and dark refusals;
  usage with no argument.
- `Company.Battle` gains `outlook` (`modules/gmcp`); `window-combat.js`
  shows it.

## Task 3: help and tutorial

New `assessment.template` (aliases `assess`, `odds`, `chances`, `risk`),
`consider.template` replacing GoMud's `consider.md`, updated `scout`,
`combat`, and `webclient`; keywords index; Combat lesson scout hint.

## Task 4: integration tests

Pure estimate tables (bands, closeness, reach, fail-open, zero damage);
real `scout`/`consider` through `TryCommand` in the company brawl:
counted and missing companions, wounds and burden changing the words,
formation changing reach, darkness and hidden foes, no round spent and no
state change, player refusal; `Company.Battle` outlook built from a live
battle and absent in the dark; help render and alias tests; tutorial
pointers; dock-windows browser check.

## Task 5: migration and recovery

No durable state: the assessment is recomputed from live state on each
call; nothing to migrate, and restart/copyover needs nothing.

## Task 6: review and integration

Independent reviewer on the full diff; fixes with regression tests;
Project Status and roadmap; `make generate`, `make validate`,
`go test -race ./...`, `make js-lint`, dock-windows check; merge and push.
