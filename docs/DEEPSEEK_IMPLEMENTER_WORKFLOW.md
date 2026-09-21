# Terra + DeepSeek Implementation Workflow

## Roles

Terra is the lead engineer for Ashveil. It owns discovery, design decisions,
task decomposition, review, verification, integration, and commits.

DeepSeek Flash is an implementation-only agent. It may make the scoped code
changes and run focused checks, but its output is always a proposal for Terra
to review.

## Local prerequisite

The local Codex configuration must include the `deepseek` profile, which uses
the DeepSeek Flash provider and the locally stored credential. Confirm it is
available before delegation:

```bash
codex exec --profile deepseek --skip-git-repo-check "Respond with exactly: READY"
```

Do not put an API key in this repository, an `AGENTS.md`, a prompt, a commit,
or command output. If the profile is unavailable, Terra implements the task
itself and reports the missing local setup rather than attempting to recreate
or expose credentials.

## Delegation lifecycle

1. Terra reads the applicable root and nested `AGENTS.md` files, chooses or
   creates the required feature worktree, and writes a narrow implementation
   brief. The brief includes acceptance criteria, relevant invariants, files or
   packages in scope, and the focused checks to run.
2. From that worktree, Terra runs:

   ```bash
   codex exec --profile deepseek --full-auto "Implement <scoped task>. Do not commit, push, create a worktree, change unrelated files, or modify project-status documentation. Run <focused checks> and summarize the changes and results."
   ```

3. No other agent edits that worktree until the command exits. DeepSeek must
   not commit, push, merge, alter credentials, or broaden the task.
4. Terra reviews the entire resulting diff, checks the phase invariants and
   nested instructions, and independently runs proportionate verification.
   DeepSeek-reported results are not sufficient evidence.
5. If needed, Terra launches a new, focused DeepSeek fix round. Terra makes
   any final integration changes, updates `docs/PROJECT_STATUS.md`, and is the
   only agent permitted to commit, merge, push, or report completion.

## Prompt quality

Keep implementation prompts self-contained. State the desired behavior,
non-negotiable invariants, exact package boundaries, tests to add or update,
and explicit exclusions. Do not ask DeepSeek to make design choices that
belong to Terra, and never delegate two concurrent writers into one worktree.
