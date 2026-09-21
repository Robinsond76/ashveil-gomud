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
5. After every task passes its verification, finish with
   `superpowers:finishing-a-development-branch`: merge the branch to `master`
   locally or open a PR against `origin`. Remove the worktree when done.

If a worktree is unavailable, create and check out a feature branch before
making any commit. See the root `AGENTS.md` ("Branching & Worktrees") and
`docs/ASHVEIL_GOMUD_AGENT_HANDOFF.md` rule 22.
