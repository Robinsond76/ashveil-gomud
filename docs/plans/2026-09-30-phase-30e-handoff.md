# Phase 30e preparation handoff (historical)

Superseded on 2026-10-01 by the completed
[implementation record](2026-09-30-phase-30e-implementation.md) and
[project status](../PROJECT_STATUS.md). The instructions below record the
pre-approval preparation state; they are not current work instructions.

Requested model: **6.1 Sol** (`gpt-6.1-sol`). Use the ashveil-gomud cloud
environment and the latest `master` of `Robinsond76/ashveil-gomud`.

## Task brief

Continue work on Phase 30e. The owner selected this phase, requested removal
of mounted combat from 30f, approved documentation cleanup, and asked for the
project's external skill/plugin workflow to be removed. Those maintenance
changes are complete. Do not reinstall or require that workflow.

Read `AGENTS.md`, `docs/AGENT_IMPLEMENTATION_WORKFLOW.md`, and
`docs/PROJECT_STATUS.md`, then the
[detailed 30e draft](../designs/2026-09-30-phase-30e-morale-mercy-design.md)
and [original direction](../designs/2026-09-26-morale-mercy-design.md).

The draft recommends 30e1 (enemy morale and mercy) followed by 30e2 (company
nerve). It covers code ownership, surrender protection, post-battle prompts,
rewards, loyalty, multiplayer ownership, restart/copyover, and acceptance tests.
The draft's numeric defaults and lifecycle choices are proposals. The owner
has authorized starting 30e and pushing the preparation, but has not yet
approved these detailed gameplay choices. Review the draft against current
code, present the important decisions concisely for approval under handoff
rule 20, then write the implementation plan and execute. Do not repeat the
completed documentation cleanup or claim 30e has been implemented.

## Relevant code

- `internal/battle/battle.go`: transient battles, membership, names.
- `internal/hooks/NewRound_DoCombat.go`: round ordering.
- `internal/hooks/combat_battle.go`: battle end and immediate next-group start.
- `internal/hooks/combat_stream.go`: active-enemy counting and reporting.
- `internal/combatstream`: Yield/Mercy event kinds exist; yielded summary
  folding and gameplay behavior do not.
- `internal/prompt`: command-backed questions; handle coexistence and expiry.
- `internal/mobcommands/suicide.go`: ordinary rewards, alignment, vanish.
- `modules/company/alignment.go`: company average and saved disposition.
- `modules/company/chemistry.go`: saved service and band chemistry.

Preserve shared world time, persisted company state, and existing combat
cadence. A yielded foe must be protected across every damage and retargeting
path. Resolve shared enemies once across players. Unanswered mercy must not
stall another battle. Do not claim crash-atomic rewards across separate save
files. The draft calls out these boundaries explicitly.

## Development and delivery

Use the cloud environment's existing setup. If present, activate tooling with
`. /workspace/ashveil-env/activate.sh` (Go 1.24.13 and local linters/caches).
Use a feature branch and the isolation policy applicable to the new cloud
workspace; never commit directly to `master`.

Follow the plugin-independent repository workflow: focused tests, player help
and tutorial pointers, independent full-phase review, final generation,
format/vet and race checks, applicable linting, and a truthful status update.
Do not rerun an unrelated full baseline suite or reimplement shipped phases.
