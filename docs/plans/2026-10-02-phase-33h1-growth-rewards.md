# Phase 33h1: Companion Growth and Contract Rewards — Plan

Design: [33h design](../designs/2026-10-01-phase-33h-progression-recovery-continuity-design.md),
"Final implementation decisions: 33h1". Worktree
`.worktrees/phase-33h1-growth-rewards`, branch `phase-33h1-growth-rewards`.

## Task 1: growth model

- `internal/characters`: `StatPointsAtLevel(level)`, shared with mob spawns
  (`internal/mobs`).
- `internal/company/growth.go`: pure `GrowthWeights`, `Deal(points, w)`
  (a prefix-stable weighted rotation), stat parsing, `WithFocus`.
- `internal/archetypes`: `Growth` on the archetype, parsed from the module
  config; optional `CompanionGrowthProvider` and `CompanionGrowth(id)`.

## Task 2: entry points

- `Companion.GrowthFocus` (`growth_focus`).
- `Runtime.Spawn` takes the weights and retrains before vitals are set;
  `Runtime.Retrain` re-deals a live mob's points (health and mana only
  clamped, never raised).
- `company.GrowthProvider` (`RetrainCompanion`) replaces `AutoTrain` after
  a live level-up in `mobcommands`.
- `company growth [member] [stat|balanced]`: view and set, refused in a
  battle; saves the record, then retrains the live mob.
- `rewards.companyexperience` on quests; the quest hook pays present
  companions through the exported combat award.

## Task 3: help and tutorial

`help growth` (aliases `specialize`, `specialization`, `companion growth`)
and `help contracts` (aliases `contract`); `company`, `experience`/leveling
and `quests` pages link them; keywords index; Departure lesson hint.

## Task 4: integration tests

Pure deal and weight tables (prefix stability, totals, focus, empty
weights); real `company growth` through the command; live level-up
through a real kill; resurrection and respawn re-derive the same training
(no minting); contract turn-in pays present companions only and the
personal quest leaves companions alone; shipped archetype weights parse;
help render and tutorial pointers.

## Task 5: migration and recovery

Old records load without `growth_focus`; training is never saved, so a
restart or copyover respawn re-derives it. Record that no data migration
is required.

## Task 6: review and verification

Independent reviewer on the full diff; fix findings with tests; record
outcome in `docs/PROJECT_STATUS.md`; `make generate`, `make validate`,
`go test -race ./...`.
