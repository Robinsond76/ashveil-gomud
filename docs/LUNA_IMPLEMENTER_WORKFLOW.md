# Terra + Luna Implementation Workflow

## Roles

Terra is the lead engineer for Ashveil. It owns discovery, design decisions,
task decomposition, review, verification, integration, and commits.

GPT-5.6 Luna is the implementation-only subagent. It may make scoped code
changes and run focused checks, but its output is always a proposal for Terra
to review.

## Delegation lifecycle

1. Terra reads the applicable root and nested `AGENTS.md` files, chooses or
   creates the required feature worktree, and writes a narrow implementation
   brief. The brief includes acceptance criteria, relevant invariants, files or
   packages in scope, and the focused checks to run.
2. Terra launches one native Codex subagent using `gpt-5.6-luna` with medium
   reasoning effort. The brief must tell Luna not to commit, push, merge,
   create a worktree, alter credentials, modify project-status documentation,
   or broaden the task.
3. No other agent edits that worktree until Luna exits. Luna reports its
   changes and focused-check results to Terra.
4. Terra reviews the entire resulting diff, checks the phase invariants and
   nested instructions, and independently runs proportionate verification.
   Luna-reported results are not sufficient evidence.
5. If needed, Terra launches a new, focused Luna fix round. Terra makes any
   final integration changes, updates `docs/PROJECT_STATUS.md`, and is the
   only agent permitted to commit, merge, push, or report completion.

## Prompt quality

Keep implementation prompts self-contained. State the desired behavior,
non-negotiable invariants, exact package boundaries, tests to add or update,
and explicit exclusions. Do not ask Luna to make design choices that belong to
Terra, and never delegate two concurrent writers into one worktree.
