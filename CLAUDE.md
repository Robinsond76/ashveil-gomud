@AGENTS.md

## Claude-specific workflow (overrides "Terra + Luna" for Claude sessions)

The "Terra + Luna Implementation Workflow" above, and `docs/LUNA_IMPLEMENTER_WORKFLOW.md`,
are written for ChatGPT/Codex, which delegates to a native Codex subagent running
`gpt-5.6-luna`. This project is also worked on with Claude Code, where the driving
methodology is the **Superpowers** plugin (`obra/superpowers`, enabled for this repo via
`.claude/settings.json` → `enabledPlugins["superpowers@claude-plugins-official"]`). On
first opening this project in Claude Code, approve the one-time plugin-trust prompt if
one appears.

Superpowers' `using-superpowers` skill is mandatory-invocation: before any response or
action, check whether a Superpowers skill applies and use it if so. In practice that
means its own workflow — `brainstorming` → `using-git-worktrees` → `writing-plans` →
`executing-plans`/`subagent-driven-development` → `test-driven-development` →
`requesting-code-review` → `finishing-a-development-branch` — now supersedes the manual
worktree/plan steps this file used to spell out, and **is** what
`docs/superpowers/plans/README.md` and the phase plans in `docs/superpowers/plans/`
already assume; those references now resolve correctly under Claude Code, exactly as
they do under Codex.

Superpowers' `subagent-driven-development` skill dispatches implementer/reviewer
subagents itself (via the `Agent` tool) and asks the driving session to pick a model per
task under its own "Model Selection" tiers. Map those tiers to this project's Terra/Luna
split — Sonnet as the brain, Haiku as the implementer:

- **Terra role → the main Claude session, on Sonnet.** Own architecture, task scoping,
  brainstorming/plan approval, code review, verification, `docs/PROJECT_STATUS.md`
  updates, and all commits/merges. Never hand these off. This is also Superpowers'
  "standard" and "most capable" model tier — use Sonnet for integration/judgment tasks,
  architecture/design tasks, and all review/escalation rounds.
- **Luna role → Superpowers' "cheap, fast model" tier → Haiku.** When dispatching a
  mechanical implementation task (isolated function, clear spec, 1–2 files) via
  `Agent({ ..., model: "haiku" })`, give it the same kind of narrow, self-contained brief
  `docs/LUNA_IMPLEMENTER_WORKFLOW.md` describes: acceptance criteria, relevant
  invariants, exact file/package boundaries, and the focused checks to run. Tell it
  explicitly not to commit, push, merge, create a worktree, or touch
  `docs/PROJECT_STATUS.md`.
- Treat any subagent's diff as an untrusted proposal: read the whole diff yourself,
  independently run proportionate tests, and either request a focused fix round or make
  the final corrections yourself.
- Skip delegation and implement directly yourself (on Sonnet) for anything involving
  concurrency, timers, persistent-state recovery, disconnect/reconnect, or other
  multiplayer invariants — the same threshold the handoff doc uses to escalate past Luna
  (`docs/ASHVEIL_GOMUD_AGENT_HANDOFF.md`, rule 19 in section 50), and matches
  Superpowers' own guidance to escalate architecture/design and fix-loop rounds 4–5 to a
  more capable model.

## Project context for Claude

Ashveil is a multiplayer MUD being rebuilt on the GoMud Go engine
(`github.com/GoMudEngine/GoMud`, forked to `Robinsond76/ashveil-gomud`). The original
Python prototype is preserved read-only at `reference/ashveil-mud/` — research only,
never edit, commit, or move it.

**Non-negotiable invariant:** travel and rest run as durable, real-time, server-side
sessions and must never fast-forward or locally advance GoMud's shared world
clock/round count. Every migrated system must survive restart/copyover.

**Before gameplay work, read in this order:**
1. `docs/PROJECT_STATUS.md` — current phase, what's done, known issues/limitations.
   Check this first; it's the freshest source of truth and gets stale fast.
2. `docs/ASHVEIL_GOMUD_AGENT_HANDOFF.md` — the authoritative design doc (world
   structure, travel/survival mechanics, formation combat, the phased roadmap).
   Section 50 ("Agent Working Rules") holds the hard constraints.
3. The relevant `docs/superpowers/plans/*.md` and `docs/superpowers/specs/*.md` for
   the phase being touched.
4. Any nested `AGENTS.md` in the package being edited (`internal/<pkg>/AGENTS.md`,
   `modules/<pkg>/AGENTS.md`, etc. — nearly every package has one).

**Status snapshot (2026-09-22, verify against `docs/PROJECT_STATUS.md` before relying on
it):** Phases 0–9 complete (fork/baseline, company companion, 3×3 formation, survival
state, terrain/travel profiles, travel interruptions, camping, weather, encumbrance and
cargo). Phase 10 (mounts) is next. `master` is pushed through Phase 9 — check
`git status`/`git log` rather than trusting this snapshot once it ages.

**Branching:** never commit directly to `master`, for any change — code, a design doc, or
a `docs/PROJECT_STATUS.md` update alike. Create
`git worktree add .worktrees/<branch-name> -b <branch-name>` before the *first* commit of
any unit of work, do all of it there (design doc included), verify, then merge locally or
PR back to `origin`, and remove the worktree when done. `master`'s own checkout is never a
workspace — not even for a single docs file.

**Verification:** `go test -race ./...`, `make generate`, and `make validate` before
calling anything done. `make test`'s `js-lint` stage can stall on this host (it shells
out to `npx jshint`) — that's environmental, not a code failure; `go test -race ./...`
is the documented fallback. Never claim a check passed without running it.

**Structure:** `internal/` = engine packages (mostly upstream GoMud plus Ashveil
additions like `internal/company`, `internal/survival`, `internal/expedition`);
`modules/` = auto-wired plugins that own persistence/commands (run `make generate`
after adding one); `_datafiles/` = world/config/script content; `docs/` = all Ashveil
planning docs. `upstream` remote is the read-only GoMud source; `origin` is the fork.
