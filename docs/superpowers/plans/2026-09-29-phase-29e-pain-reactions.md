# Phase 29e: Pain Reactions — Implementation Plan

Design: [Phase 29e](../specs/2026-09-29-phase-29e-pain-reactions-design.md).
Base: `1fd7afb0`. Implement in this isolated worktree. Do not change combat
odds, damage, buffs, event records, or world time. Run focused tests as each
task lands; run the full verification once after independent review fixes.

## Task 1 — Reaction data and selection

Files: `internal/races/races.go`, `internal/mobs/mobs.go`, default-world race
YAML, `internal/combat/pain_reactions.go` and its tests.

- [x] Test pair validation (both viewpoints, allowed tokens) and selection
  order: mob override, race, generic humanoid/creature. Watch it fail.
- [x] Add paired reaction fields and validation to race and mob templates;
  add the selection and rendering helpers. Make the tests pass.
- [x] Author distinct sets for shipped canine, rodent, insect, reptilian,
  giant spider, lagomorph, and reptile races. Test shipped data loading.

## Task 2 — Critical-strike wiring

Files: `internal/combat/combat.go`, `internal/combat/pain_reactions_test.go`,
`modules/company/wiring_pain_reactions_test.go`.

- [x] Write failing tests for one reaction directly after a surviving
  critical strike, second/third-person variants, stable battle name and
  pronouns, NPC override, and separate-room delivery.
- [x] Cover no reaction after a normal hit, fully blocked critical, lethal
  critical, or a later lethal strike in a multi-hit round. Drive both the
  real attack calculation and `DoCombat`.
- [x] Append reactions per strike in each recipient's ordered message list;
  pass the live NPC victim through the attack wrappers. Pick line variants
  without using combat RNG. Make tests pass.
- [x] Compare a seeded before/after fight's damage and combat events to
  confirm narration did not alter mechanics.

## Task 3 — Help and tutorial

Files: `help/narration.template`, `help/combat.template`,
`modules/tutorial/stages.go`, `internal/usercommands/help_combat_test.go`,
and tutorial tests.

- [x] Write failing assertions that both help pages explain the reaction
  rule, the Combat lesson points to `help narration`, and all pointers
  resolve. Then update the content and pass the focused tests.
- [x] Check the `help:` index and aliases still resolve `narration` under
  Combat and `critical`/`crit` to it; no new command or help topic is needed.

## Task 4 — Review, verification, integration

- [x] Inspect the whole diff and obtain an independent reviewer report on
  bugs, design gaps, missing integration coverage, and inaccurate help.
- [x] Reproduce each finding and fix confirmed issues with regression tests;
  record the result in `docs/PROJECT_STATUS.md`.
- [x] Run `make generate`, `make validate`, `go test -race ./...`,
  `make js-lint`, and `git diff --check` after the fixes; record exact results.
- [x] Commit narrowly on `codex/phase-29e-pain-reactions`, merge into clean
  `master`, push `origin/master`, and remove this worktree.

## Review focus

- A critical flag describes the entire attack result, while the reaction
  must use each strike's own flag.
- `calculateCombat` accumulates damage before the wrapper applies it; the
  survival check must subtract earlier strikes too.
- Same-room and separate-room dispatch exclude different viewers; every
  viewer must see one line and the victim must get second person.
- A mob's frozen battle label and authored pronouns must survive rendering.
- A missing race or malformed reaction pair must use a safe fallback or
  fail data load clearly.
