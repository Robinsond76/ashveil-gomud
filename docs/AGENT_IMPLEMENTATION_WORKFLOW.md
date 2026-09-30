# Agent Implementation Workflow

## Default execution

Use the current task's default agent and reasoning settings for implementation.
No named implementation model is required. An explicit owner request for a
model takes precedence for that task; a later request to use the default
agent supersedes the earlier selection.

The lead agent owns discovery, design decisions, task decomposition,
implementation, review, verification, integration, project status, and commits.
Execute the approved plan directly, task by task, by default.

## Design and implementation

No external plugin or skill is required. For a new gameplay phase, inspect
current code and write a design in `docs/designs/` covering scope, prior art,
state ownership, persistence/recovery, integration points, player help, and
acceptance tests. Separate owner decisions from proposed balance defaults.
Obtain owner approval of the phase design before implementation (handoff
rule 20); do not infer approval of new gameplay choices from a request to
start the phase. Then write an actionable plan in `docs/plans/` and implement
it directly. Routine fixes and explicitly requested documentation maintenance
do not need a new phase-design approval.

## Optional delegation

Delegate only when the owner or applicable workflow explicitly calls for it.

1. Read root and nested `AGENTS.md`, activate the feature worktree, and supply
   a narrow brief with acceptance criteria, invariants, exact file/package
   boundaries, and focused checks.
2. Inherit the task's model and reasoning settings unless the owner requests
   an override. Workers must not commit, push, merge, create a worktree,
   alter credentials, modify project status, or broaden scope.
3. Do not edit the worktree while a worker runs. After it exits, inspect its
   complete diff and independently run proportionate checks; worker-reported
   results alone do not establish verification.
4. Resolve findings, update project status, and commit accepted changes as
   the lead. Never delegate two concurrent writers into one worktree.

## Completion gate

Keep the existing tests-first, player-help, persistence, multiplayer, and
worktree requirements. Run focused checks during implementation and the
required full checks after fixes. Before merging a phase, obtain the
independent full-phase reviewer required by `AGENTS.md`; use the task's
configured default model unless the owner explicitly chooses another.
Verify each finding, fix confirmed issues with regression tests, and record
the review outcome and exact verification in `docs/PROJECT_STATUS.md`.
