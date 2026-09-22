# Repository Guidelines

## Ashveil Context

This fork is becoming the Ashveil game, with GoMud as its engine foundation. Use game-domain names such as `company`, `companion`, and `expedition`, not `ashveil*` prefixes. Persist multiplayer state across restart/copyover; never advance global game time for travel or rest. Before gameplay, consult `docs/ASHVEIL_GOMUD_AGENT_HANDOFF.md`, the Phase 0–1 plan, and `docs/PROJECT_STATUS.md`.

## Project Status

`docs/PROJECT_STATUS.md` is the living status log for the migration. Update it whenever a commit lands or a phase completes, recording what was done, why, and which step or phase completed. Keep it short and current, and link to detailed docs instead of duplicating them.

## Terra + Luna Implementation Workflow

Terra is the lead engineer for Ashveil: it owns architecture, task scoping,
code review, verification, follow-up work, and commits. For a scoped
implementation task, Terra may delegate the first implementation pass to one
native Codex subagent using `gpt-5.6-luna` at medium reasoning effort. Read
`docs/LUNA_IMPLEMENTER_WORKFLOW.md` before doing so.

Delegate only after the change is understood and the appropriate feature
worktree is active. Give Luna a narrow, self-contained brief with acceptance
criteria, relevant invariants, package or file boundaries, and focused checks.
Do not edit the same worktree while Luna is running.

Treat Luna's output as an untrusted implementation proposal: after it exits,
Terra must inspect the complete diff, verify the phase invariants and nested
`AGENTS.md` instructions, run proportionate tests itself, and either request a
focused Luna fix round or make the final corrections. Only Terra may update
project status, commit, merge, push, or claim verification passed.

## Project Structure & Module Organization

- `internal/` contains the engine. Keep gameplay work in its owning package. Read its nested `AGENTS.md` before changing it.
- `_datafiles/` holds worlds, configuration, scripts, and browser assets. Treat the default world as shipped content.
- `modules/` contains optional auto-wired extensions. Register a module with its Go `init()` flow and refresh generated imports with `make generate`.
- `cmd/` contains CLI tools; `scripts/` contains helpers; `docs/` holds Ashveil planning and verification notes.
- `reference/ashveil-mud/` is the read-only Python prototype. Research only; do not edit, commit, or move.

## Build, Test, and Development Commands

- `make build` formats/checks the project and builds `./go-mud-server`.
- `make run` regenerates module wiring and starts the server locally. Use `make run-docker` for Docker Compose.
- `make validate` runs `gofmt` checks and `go vet`; run it for Go changes.
- `make test` runs generation, JavaScript/Lua linting, then `go test -race ./...`. Docker Desktop may be required for Lua linting.
- `make js-lint`, `make lua-lint`, and `go test ./internal/rooms -run TestName` are focused checks. Run `make help` to list targets.

## Coding Style & Naming Conventions

Use idiomatic Go, `gofmt`, tabs, exported `PascalCase`, and unexported `camelCase`. Keep tests in `*_test.go`; prefer table-driven cases. JavaScript, CSS, HTML, YAML, and Markdown use two spaces per `.editorconfig`. Regenerate generated files through the Makefile.

## Testing Guidelines

Use Go's standard `testing` package. Name tests `TestBehavior` and benchmarks `BenchmarkBehavior`, beside covered code. Add a regression test for each bug fix. Start targeted, then run `make validate`; run `make test` before requesting review when practical. Do not claim checks not run.

## Commit & Pull Request Guidelines

Use concise imperative commits with an optional scope, such as `fix(telnet): stop input masking` or `docs: establish Ashveil baseline`. Keep commits narrow. Pull requests should state user effect, approach, verification, and linked issue; attach screenshots for web/admin UI changes.

### Branching & Worktrees

Never commit directly to `master`; `master` is an integration branch and must
stay clean. This applies to **every** commit, not only code — a design doc, a
`docs/PROJECT_STATUS.md` update, or any other docs-only change gets its own
worktree and branch too. `master` only ever receives a merge (or an
`origin` PR) from a finished branch; it is never a workspace itself. Before
starting *any* work, including drafting a phase design doc, create an
isolated git worktree with a dedicated feature branch:

```bash
git worktree add .worktrees/<branch-name> -b <branch-name>
```

`.worktrees/` is gitignored and is the project convention (for example
`.worktrees/phase-5-travel`). Run the baseline checks in the worktree, make
every commit for that unit of work — design doc included — on the feature
branch, and only then integrate back to `master` (merge locally or open a PR
against `origin`). Remove the worktree when the branch is finished. If a
worktree is unavailable, create and check out a feature branch before making
any commit; working directly on `master`'s checkout is never acceptable, even
for a single-file change.

`origin` is the Ashveil fork. `upstream` is the read-only GoMud source: fetch from it if needed, but never push to it. Preserve the multiplayer invariant: travel and rest must not advance global game time.
