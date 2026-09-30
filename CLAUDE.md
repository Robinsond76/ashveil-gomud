@AGENTS.md

## Project workflow

Follow [Agent Implementation Workflow](docs/AGENT_IMPLEMENTATION_WORKFLOW.md).
No plugin or external skill is required to work on this repository.
Use the current task's default agent and reasoning settings; implement directly
unless the owner requests delegation. Independent phase review remains required.

Read `docs/PROJECT_STATUS.md` first, then relevant designs in `docs/designs/`,
plans in `docs/plans/`, and package-specific `AGENTS.md` instructions. The
handoff document preserves design intent; current status and code describe
shipped behavior. Retired documents are available in git history.

## Testing and review gate

- Write regression and integration tests for changed behavior, including the
  real entry points a phase wires. Pure helper tests alone are insufficient.
- Ship indexed player help and tutorial pointers with player-facing changes,
  as specified in `AGENTS.md`.
- Before merging a phase, have an independent reviewer subagent inspect its
  complete diff for bugs, design gaps, missing coverage, and inaccurate help.
  Use the task's default model; the reviewer reports findings without editing.
- Verify each finding, fix real issues with regression tests, and record both
  accepted and rejected findings in `docs/PROJECT_STATUS.md`.
- Run focused checks during implementation. After review fixes, run the full
  required checks once: `make generate`, `make validate`, and
  `go test -race ./...`; run applicable JavaScript/Lua lint checks too. Repeat
  only if code changes or a concrete unresolved concern requires it.
- Documentation-only edits need link/reference and diff checks, not Go tests.
  Help templates or world data read by tests require their relevant checks.

Never commit directly to `master`; use the isolated feature workspace required
by `AGENTS.md`. Preserve shared world time, durable state, concurrency safety,
and the read-only Python reference and upstream remote.
