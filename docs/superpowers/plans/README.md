# Ashveil Implementation Plans

Plans in this directory are executed on an isolated git worktree and feature
branch. Never implement a plan, phase, or feature directly on `master`.

`master` is an integration branch and must stay clean. Before executing any
plan:

1. Read the plan and its linked spec.
2. Create an isolated worktree with a dedicated branch:

   ```bash
   git worktree add .worktrees/<branch-name> -b <branch-name>
   ```

   `.worktrees/` is gitignored and is the project convention (for example
   `.worktrees/phase-6-interruptions`).
3. Run the baseline checks in the worktree before changing code
   (`make generate`, `make validate`, `go test -race ./...`).
4. Execute the plan task-by-task with `superpowers:executing-plans` or
   `superpowers:subagent-driven-development`, committing each task on the
   feature branch.
5. Every plan that changes what a player can do or see has a **player help and
   tutorial** task: help pages for its commands and mechanics (indexed in
   `keywords.yaml`, linked from their hub page), pointers from the tutorial
   lesson that covers them, and tests that they render and resolve. See the
   root `AGENTS.md` ("Testing Guidelines").
6. Before merging, run the testing and review gate (`CLAUDE.md`, "Testing
   and review gate"): wiring tests for each integration point, then an
   independent reviewer subagent over the full phase diff, with every finding
   verified and its outcome recorded in `docs/PROJECT_STATUS.md`.
7. After every task passes its verification and the review is resolved, finish with
   `superpowers:finishing-a-development-branch`: merge the branch to `master`
   locally or open a PR against `origin`. Remove the worktree when done.

If a worktree is unavailable, create and check out a feature branch before
making any commit. See the root `AGENTS.md` ("Branching & Worktrees") and
`docs/ASHVEIL_GOMUD_AGENT_HANDOFF.md` rule 22.
