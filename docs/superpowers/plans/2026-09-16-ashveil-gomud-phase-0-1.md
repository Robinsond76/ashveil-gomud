# Ashveil on GoMud — Phase 0–1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish a separate Ashveil GoMud project and deliver an evidence-backed integration map before adding gameplay.

**Architecture:** Keep the Python prototype as a read-only, Git-excluded reference inside the Go project. Create a GoMud-derived repository for Ashveil work and retain the original GoMud only as a read-only `upstream` remote. Audit both trees and document actual extension seams.

**Tech Stack:** Git, GitHub CLI (if authenticated), Go 1.24+, GoMud Make targets, Markdown.

**Spec:** `/Users/robinsondesouza/Documents/Codex/ashveil-gomud/docs/ASHVEIL_GOMUD_AGENT_HANDOFF.md`

## Global Constraints

- Never modify, delete, or push to `GoMudEngine/GoMud`; it is read-only `upstream`.
- Never modify the Python source at `/Users/robinsondesouza/Documents/vibing/openCode/projects/ashveil-mud`.
- Do not add Ashveil gameplay or remove/change default-world content in this phase.
- Do not push any commit during this phase.
- Future travel/rest must never fast-forward the shared multiplayer world clock.
- GPT-5.6 Terra (medium) orchestrates and reviews; GPT-5.6 Luna implements only narrow coding tasks.

---

### Task 1: Establish reference and project repositories

**Files:**
- Create: `/Users/robinsondesouza/Documents/Codex/ashveil-gomud/reference/ashveil-mud/`
- Create: `/Users/robinsondesouza/Documents/Codex/ashveil-gomud/`
- Modify: `ashveil-gomud/.git/config` through Git commands only.

**Interfaces:**
- Consumes: the Python source and `https://github.com/GoMudEngine/GoMud`.
- Produces: read-only prototype reference; separate checkout with `origin` = `Robinsond76/ashveil-gomud`, `upstream` = `GoMudEngine/GoMud`.

- [x] **Step 1: Verify inputs and empty destinations**

Run: `test -d /Users/robinsondesouza/Documents/vibing/openCode/projects/ashveil-mud && test ! -e /Users/robinsondesouza/Documents/Codex/ashveil-gomud/reference/ashveil-mud && test ! -e /Users/robinsondesouza/Documents/Codex/ashveil-gomud`

Expected: success before a copy or clone.

- [x] **Step 2: Create the read-only local prototype reference**

Run: `cp -a /Users/robinsondesouza/Documents/vibing/openCode/projects/ashveil-mud /Users/robinsondesouza/Documents/Codex/ashveil-gomud/reference/ashveil-mud && chmod -R a-w /Users/robinsondesouza/Documents/Codex/ashveil-gomud/reference/ashveil-mud`

Expected: the reference copy exists at `reference/ashveil-mud/`, is excluded through `.git/info/exclude`, and the source remains untouched.

- [x] **Step 3: Create/confirm the personal GoMud fork, then clone it**

Run when authenticated: `gh repo fork GoMudEngine/GoMud --fork-name ashveil-gomud --clone=false`, then `git clone https://github.com/Robinsond76/ashveil-gomud.git /Users/robinsondesouza/Documents/Codex/ashveil-gomud`.

Expected: full GoMud history with a personal `origin`. If the fork exists, omit the fork command. If authentication is unavailable, stop and request owner action rather than inventing a remote policy.

- [x] **Step 4: Add and verify read-only upstream**

Run: `git -C /Users/robinsondesouza/Documents/Codex/ashveil-gomud remote add upstream https://github.com/GoMudEngine/GoMud.git && git -C /Users/robinsondesouza/Documents/Codex/ashveil-gomud remote -v`

Expected: both remotes appear. Never execute `git push upstream`.

- [x] **Step 5: Capture baseline identity**

Run: `git -C /Users/robinsondesouza/Documents/Codex/ashveil-gomud status --short && git -C /Users/robinsondesouza/Documents/Codex/ashveil-gomud log -1 --oneline`

Expected: clean worktree and recorded baseline commit.

### Task 2: Validate vanilla GoMud

**Files:**
- Inspect: `ashveil-gomud/AGENTS.md`, `CLAUDE.md`, `README.md`, and `Makefile`.
- Create: `ashveil-gomud/docs/BASELINE_VERIFICATION.md`.

**Interfaces:**
- Consumes: untouched checkout from Task 1.
- Produces: build/test/runtime evidence for the exact baseline revision.

- [x] **Step 1: Read project instructions and supported commands**

Run: `sed -n '1,240p' AGENTS.md; sed -n '1,240p' CLAUDE.md; sed -n '1,260p' README.md; make help`

Expected: record the repository’s actual Go version, required build/test commands, and run procedure before executing them.

- [x] **Step 2: Verify toolchain compatibility**

Run: `go version && grep '^go ' go.mod`

Expected: installed Go meets `go.mod`; otherwise record the blocker and stop without dependency changes.

- [x] **Step 3: Run the unchanged validation and test suite**

Run: `make validate && make test`

Expected: success, or a precise baseline failure record with command, package, and output.

- [x] **Step 4: Build and smoke-test the server**

Run: `make build`, then the documented startup command. Verify local startup and configured web client/admin endpoints; stop cleanly.

Expected: build and startup evidence, plus ordinary movement and availability of default combat/party/mercenary systems each marked as exercised, unavailable, or blocked.

- [x] **Step 5: Write baseline evidence**

Create `docs/BASELINE_VERIFICATION.md` containing exact commit, Go version, commands, results, endpoints, and unresolved environment blockers. Do not claim a feature was exercised merely because documentation mentions it.

### Task 3: Audit mechanics and integration seams

**Files:**
- Inspect: `reference/ashveil-mud/AGENTS.md`, `server/`, and `tests/`.
- Inspect: discovered GoMud room, exit/movement, area/map, party/mercenary, item/equipment, combat/AI, time/timer, event/module, persistence, and disconnect/restart source.
- Create: `ashveil-gomud/docs/ASHVEIL_GOMUD_INTEGRATION.md`.
- Create: `ashveil-gomud/docs/ASHVEIL_GOMUD_AGENT_HANDOFF.md`.

**Interfaces:**
- Consumes: both local trees and Task 2 baseline evidence.
- Produces: exact source-cited reuse/extend/add decisions for every planned Ashveil concept.

- [x] **Step 1: Inventory Python mechanics from source and tests**

Run targeted `rg` searches for `party`, `merc`, `formation`, `travel`, `camp`, `hunger`, `thirst`, `fatigue`, `weather`, `mount`, `weight`, and `combat`. Read all implementation and relevant test matches.

Expected: record file paths, symbols, ownership/persistence, and each feature’s complete/partial/prototype-only status.

- [x] **Step 2: Map GoMud source seams**

Use `rg --files`, `rg`, and focused reads to find exact packages/types/functions for rooms, exits/directions/movement parsing, maps/pathfinding, players/mobs, parties/mercs, items/containers/equipment/weight, combat/AI, shared time/timers, events/hooks/modules, commands, persistence, and disconnect/restart.

Expected: each documented row has a path and symbol or explicitly says `Not found`.

- [x] **Step 3: Write the integration map**

Create `docs/ASHVEIL_GOMUD_INTEGRATION.md` with columns: Ashveil concept; GoMud path and symbol; decision (`Reuse`, `Extend`, `Add`, `Defer`, or `Not found`); Python path and symbol; evidence/migration note. Include an explicit TravelSession section covering server ownership, state transitions, no global-time mutation, persistence, reconnect, and exactly-once completion risks.

Expected: verified facts are distinguished from future additions.

- [x] **Step 4: Move project-local guidance into `docs/`, update the project layout policy, and commit the Phase 0–1 artifacts locally**

Move the canonical handoff into `docs/ASHVEIL_GOMUD_AGENT_HANDOFF.md`, update it and this plan for the project-contained reference layout, run `git diff --check`, commit the four documentation files with message `docs: establish Ashveil GoMud integration baseline`, and leave the worktree clean. Do not push.

## Plan Self-Review

- Spec coverage: Tasks 1–3 cover Phase 0 fork/reference setup, vanilla baseline, exact integration map, and project-local guidance; gameplay is deliberately excluded.
- Placeholder scan: implementation symbols are discovered from the checkout rather than invented before audit.
- Type consistency: this phase introduces no Go runtime types.
