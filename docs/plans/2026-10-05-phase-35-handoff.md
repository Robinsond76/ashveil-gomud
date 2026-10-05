# Phase 35 — implementation handoff

For the agent implementing phases 35a–35c. The owner approved the
[level impact and class power design](../designs/2026-10-05-level-impact-class-power-design.md)
on 2026-10-05, which covers all three phases. You may implement without a
further design approval (handoff rule 20 is met). Any change to an owner
decision listed in that design still needs the owner.

## Plans, in order

| Phase | Plan | Branch | Base |
|---|---|---|---|
| 35a | [Level impact](2026-10-05-phase-35a-level-impact.md) | `phase-35a-level-impact` | master |
| 35b | [Caster power, mana and recovery](2026-10-05-phase-35b-caster-power.md) | `phase-35b-caster-power` | master after 35a |
| 35c | [Companion training](2026-10-05-phase-35c-companion-training.md) | `phase-35c-companion-training` | master after 35a |

Ship 35a first. 35b and 35c both depend on it but not on each other. Do them
one at a time unless you use separate worktrees; never two writers in one
worktree. 35b's harness assertions are the phase's balance gate.

## Before you start

1. Read `AGENTS.md`, `CLAUDE.md`, `docs/AGENT_IMPLEMENTATION_WORKFLOW.md`,
   `docs/PROJECT_STATUS.md` (from the top through "Roadmap priorities"), the
   design, and your phase's plan.
2. Read the nested `AGENTS.md` of every package you change (`internal/*` and
   `modules/*` have their own).
3. Create the worktree:
   `git worktree add .worktrees/<branch> -b <branch> origin/master`.
   Never commit to `master`.

## Invariants (all phases)

- **Never advance global game time.** Regeneration, rest refills, patching
  and training are per-character actions or round-driven timers.
- **Persist across restart and copyover.** No refills, rerolls, duplicate
  grants or minted points on reload. Saved maxima clamp and never refill.
- **Everything runs on the game loop.** The company, survival, tutorial and
  company-view state has no mutexes and assumes main-loop dispatch.
- **Multiplayer ownership.** A player's commands touch only their own
  company. Allies (33d) keep their own members, mana and loot.
- **Rules stay in force:** no item use in battle (33f/33g); no manual ability
  spam in battle (32c/33a); retired skills stay retired, except `scribe`,
  which 36a revives.
- **Help is part of the work.** Every player-visible change ships indexed help
  with tutorial pointers and tests (see `AGENTS.md`).

## Where the plans deviate from the design text

These are recorded implementation decisions; list them again in Project
Status when the phase merges.

- **35a:** smooth stats use a fractional step with `StatStepLevels` kept at
  5, not `StatStepLevels: 1`. Today's `StatStep` returns the raw level for an
  interval of 1, which would roughly quintuple stats.
- **35b:**
  - Aimed Shot is already a guaranteed crit, so its scaling is damage
    (`2 + level/3`), not crit chance.
  - Tackle's chance stays on the stat edge, which smooth stats already raise
    each level.
  - Withering Hex stays direct damage until 38a.
  - New martial level-3 options (Shield Bash, Feint, Pinning Shot) move to
    38b. Only the caster level-3 spells ship now.
- **35c:** none.

## When to stop and ask the owner

- A balance target in 35b can't be met within the plan's tuning bounds
  (±25% on spell power, and the mana rates). Report the table rather than
  widening the bounds.
- `TestBalanceMismatches` fails after 35a's smooth growth.
- Any change would alter an owner decision: mana only from rest or draughts,
  the 50% HP trickle, the encounter contract, companions learning optional
  skills, or a stat point every 2 levels.

## Finishing each phase

1. Tests first, then the implementation, with focused package tests as you
   go.
2. Write a measurements file where the plan asks for one.
3. Run an independent full-diff reviewer subagent (default model, reports
   only). Verify each finding, fix the real ones with regression tests, and
   record accepted and rejected findings.
4. Run once at the end: `make generate`, `make validate`, `make js-lint`
   (plus `make lua-lint` if Docker is available), `go test -race ./...`.
   Don't claim checks you didn't run.
5. Add a Project Status entry under "Current position": what shipped, why,
   the review outcome, the verification, and the deviations. Update the phase
   sequence table's status.
6. Merge to master with a merge commit (the project's convention) and remove
   the worktree.
