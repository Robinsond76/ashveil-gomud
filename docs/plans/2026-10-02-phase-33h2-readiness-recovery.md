# Phase 33h2: Company Readiness and Recovery — Plan

Design: [33h design](../designs/2026-10-01-phase-33h-progression-recovery-continuity-design.md),
"Final implementation decisions: 33h2" (owner-approved 2026-10-02).
Worktree `.worktrees/phase-33h2-readiness`, branch `phase-33h2-readiness`.

## Task 1: durable vitals

- `internal/company/state.go`: `Vitals{Health, Mana, Percent}` on
  `MemberState` (`vitals`, omitted when nil), deep-copied by `Clone`; pure
  `Resolve(limit, manaMax)`: nil is full, `Percent` is that share of the
  limit and mana (health at least 1), otherwise absolute points clamped to
  `[1, limit]` and `[0, manaMax]`.
- `modules/company/runtime.go`: `Snapshot` records the live vitals;
  `applyState` resolves them after the wounds and stats are set.
- `death.go` `keptState` clears them; `resurrect.go` sets `Percent: 50`.

## Task 2: recovery entry points

- `internal/hooks/NewRound_AutoHeal.go`: companions regain
  `HealthPerRound` with their mana on the players' beat, out of a battle,
  leader online, capped at the wound limit (`Heal`).
- `modules/exposure`: `regenPerTick` counts companions' regeneration.
- `internal/mobcommands/companyxp.ashveil.go`: a companion's live level-up
  keeps its health and mana (clamped).
- `modules/camping/tiers.go`: a granted Well Rested (inn stay) restores the
  leader and live companions to their wound limit and full mana, after the
  wounds knit.

## Task 3: help and tutorial

New `help readiness` (aliases `recovery`, `companion health`, `vitals`),
indexed in `keywords.yaml`, linked from `help company`. Update `health`,
`inn`, `camp`, `resurrect`, `heal`, and `quit`. Tutorial: the rest lesson
and Departure point to `help readiness`.

## Task 4: integration tests

Pure `Resolve` table (nil, percent, clamps, floors) and YAML round trip
with an old record lacking `vitals`. Wiring: logout/login keeps a hurt
companion's health and mana; the autosave/copyover seam (`OnSave` then
respawn) keeps it; a failed save leaves the previous values; an old record
spawns full; a raised or lowered maximum clamps; resurrection returns at
half; online regeneration out of battle (none in battle, none at the
limit, none with the leader offline); a live level-up keeps vitals; an inn
stay restores leader and companions; help renders and
`TestTutorialHelpPointersExist` passes.

## Task 5: review and verification

Independent reviewer subagent on the full diff; verify and fix findings
with regression tests; record them in `docs/PROJECT_STATUS.md`; run
`make generate`, `make validate`, `go test -race ./...`, and the JS/Lua
lint checks where available.
