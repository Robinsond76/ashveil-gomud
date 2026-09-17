# Phase 5 Terrain and Travel Profiles Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver data-driven profiles that turn selected exits into durable real-time company travel while retaining instant normal movement and the global-clock invariant.

**Architecture:** `internal/expedition` owns profile/session math and native provider seams. `modules/expedition` owns configured profiles, durable sessions, timers, recovery, copyover, rendering, and survival checkpoint application.

**Tech Stack:** Go 1.24, `time`, `sync`, testify, GoMud plugins/YAML persistence/copyover.

**Spec:** `docs/superpowers/specs/2026-09-17-phase-5-terrain-travel-profiles-design.md`

## Global Constraints

- An empty `RoomExit.TravelProfile` preserves current instant-exit behavior.
- Persist stable leader/origin/destination/exit/profile/UTC-start/checkpoint/state values only; never sockets or mob instances.
- Use real UTC time; never mutate GoMud global time or rounds.
- Charge proportional survival exertion at durable checkpoints, never up front or twice.
- Needs are informational: no speed, movement, combat, health, or arrival effects.
- Disconnect/restart/copyover continue travel; terminal arrival is retry-safe and exactly-once to the player.

---

## File Structure

- Create `internal/expedition/expedition.go` and `internal/expedition/expedition_test.go`: pure session model, profile validation, progress/checkpoint cost math, provider seams.
- Modify `internal/exit/exit.go` and tests: YAML `travel_profile`.
- Modify `internal/usercommands/go.go`, `look.go`, and tests: start/view seams.
- Create `modules/expedition/expedition.go`, tests, and config: persistence, profiles, status, scheduler, recovery/copyover.
- Modify existing room YAML discovered under `_datafiles/` or module overlays: Oak Road proving route.
- Generate `modules/all-modules.go`; update `docs/PROJECT_STATUS.md` only after all checks pass.

### Task 1: Create the pure travel model

**Files:** Create `internal/expedition/expedition.go`, `internal/expedition/expedition_test.go`.

**Interfaces:** Export `TravelProfile{Name string; Duration time.Duration; Exertion survival.Exertion}`, `TravelSession`, states `Traveling, Interrupted, Completed, Cancelled`, `ProgressAt`, `CheckpointAt`, `ExertionDue`, and synchronized `StartProvider`/ `ViewProvider` registrations.

- [ ] **Step 1: Write failing tests** for invalid name/duration, clamped pre-start/completed progress, ten checkpoints, exact final total, invalid transitions, and nil providers.
- [ ] **Step 2: Run** `go test ./internal/expedition -count=1`; expect FAIL because the package is absent.
- [ ] **Step 3: Implement** clamped elapsed UTC math, deterministic integer cost rounding with checkpoint 10 equal to the profile total, typed unavailable errors, and mutex-protected provider registration. Do not import rooms, users, plugins, or timers.
- [ ] **Step 4: Run** `go test -race ./internal/expedition -count=1`; expect PASS.
- [ ] **Step 5: Commit:** `git add internal/expedition && git commit -m "feat(expedition): add travel session domain"`.

### Task 2: Add YAML metadata and native command adapters

**Files:** Modify `internal/exit/exit.go`, `internal/exit/exit_test.go`, `internal/usercommands/go.go`, `go_test.go`, `look.go`, and `look_test.go`.

**Interfaces:** Add `TravelProfile string \`yaml:"travel_profile,omitempty"\`` to `RoomExit`. Consume only Task 1 `Start` and `TravelView`.

- [ ] **Step 1: Write failing tests** showing marked/unmarked YAML compatibility; a fake start provider handles marked movement before AP deduction/`MoveToRoom`; an unmarked exit never calls it; a fake travel view suppresses normal room/destination output.
- [ ] **Step 2: Run** `go test ./internal/exit ./internal/usercommands -run 'Test.*(TravelProfile|TravelStart|TravelView)' -count=1`; expect FAIL.
- [ ] **Step 3: Implement** the field; in `Go`, call start only after existing combat/disabled/lock/exit-message/destination/script admission and before AP/move; in `Look`, call the view provider at entry and return when active.
- [ ] **Step 4: Run** `go test ./internal/exit ./internal/usercommands -count=1`; expect PASS.
- [ ] **Step 5: Commit:** `git add internal/exit internal/usercommands/go.go internal/usercommands/go_test.go internal/usercommands/look.go internal/usercommands/look_test.go && git commit -m "feat(expedition): intercept profiled exits"`.

### Task 3: Implement durable module, profile registry, and views

**Files:** Create `modules/expedition/expedition.go`, `modules/expedition/expedition_test.go`, `modules/expedition/files/data-overlays/config.yaml`.

**Interfaces:** `ExpeditionModule` implements Task 1 providers. Inject `Store`, `Clock`, `Scheduler`, `Mover`, and survival/roster adapters for testing. Register `travel status`.

- [ ] **Step 1: Write failing module tests** for unique profile lookup, save-before-schedule, duplicate-start refusal, `travel status`/view output, missing profile/store/survival failure without a session, and an incremental seven-of-ten cost checkpoint.
- [ ] **Step 2: Run** `go test ./modules/expedition -count=1`; expect FAIL.
- [ ] **Step 3: Implement** `ReadBytes` + explicit YAML decode + `WriteStruct` following `modules/survival`; block mutation after load error; parse validated profile config; render origin, destination, profile, percent, remaining time, and roster needs; register providers in `init`.
- [ ] **Step 4: Run** `go test -race ./modules/expedition -count=1`; expect PASS.
- [ ] **Step 5: Commit:** `git add modules/expedition && git commit -m "feat(expedition): persist travel sessions"`.

### Task 4: Implement timers, checkpoint sync, and recovery

**Files:** Modify `modules/expedition/expedition.go` and tests.

**Interfaces:** Add `Sync(leaderID int)`, scheduled completion, load recovery, and a registered `copyover.Contributor`.

- [ ] **Step 1: Write failing tests** for remaining-delay reload, overdue completion, disconnect no-op, copyover restoration, final cost equality, write/survival failure, duplicate timer, and completed recovery when user is destination/origin/another room.
- [ ] **Step 2: Run** `go test ./modules/expedition -run 'Test.*(Checkpoint|Recovery|Copyover|Completion|Duplicate)' -count=1`; expect FAIL.
- [ ] **Step 3: Implement** synchronization before status/recovery/timer handling; apply survival then persist checkpoint; persist `Completed`, move once, then remove only after destination verification. Recovery removes terminal sessions at destination, retries from origin, and retains/logs any other location. Derive restored delay from current clock; never persist timer handles. Guard registry/timer ownership with one mutex.
- [ ] **Step 4: Run** `go test -race ./modules/expedition -count=1`; expect PASS.
- [ ] **Step 5: Commit:** `git add modules/expedition && git commit -m "feat(expedition): recover real-time travel"`.

### Task 5: Add Oak Road content and module wiring

**Files:** Modify exact existing room files found by `rg --files _datafiles modules | rg 'rooms|room'`, module config, generated `modules/all-modules.go`, and module tests.

- [ ] **Step 1: Write `TestDunmarOakRoute`** that loads Dunmar West Gate/Fork at the Black Oak, asserts an `oak-road` profiled directional exit, valid destination, positive duration, and safe exertion (full needs remain above critical).
- [ ] **Step 2: Run** `go test ./modules/expedition -run TestDunmarOakRoute -count=1`; expect FAIL.
- [ ] **Step 3: Implement** only the one/two proving routes using the discovered shipped convention; configure short positive `oak-road`; run `make generate` rather than editing imports.
- [ ] **Step 4: Run** `make generate && go test ./modules/expedition -count=1`; expect PASS.
- [ ] **Step 5: Commit:** `git add modules/expedition modules/all-modules.go _datafiles && git commit -m "feat(expedition): add oak road travel route"`.

### Task 6: Verify and record completion

**Files:** Modify `docs/PROJECT_STATUS.md`.

- [ ] **Step 1: Run** `go test -race ./internal/expedition ./internal/exit ./internal/usercommands ./modules/expedition -count=1`, `make generate`, and `make validate`; expect PASS.
- [ ] **Step 2: Run** `go test -race ./...`; document any environmental blocker rather than claiming a pass.
- [ ] **Step 3: Update status only after success:** mark Phase 5 complete, set Phase 6 next, refresh HEAD/origin facts, and record profile metadata, real-time exactly-once recovery, proportional informational exertion, Oak Road, and actual commands.
- [ ] **Step 4: Commit:** `git add docs/PROJECT_STATUS.md && git commit -m "docs: record phase 5 travel completion"`.

## Plan Self-Review

- **Spec coverage:** Tasks 1–2 cover model/metadata/movement/look; Tasks 3–4 cover persistence, status, timers, survival, recovery, disconnect, restart, and copyover; Task 5 adds only the proving route; Task 6 verifies and records the phase.
- **Placeholder scan:** No TODO/TBD entries; route discovery is constrained to the project’s actual existing datafile format.
- **Type consistency:** Native packages consume only Task 1 seams; the expedition module is their sole implementation.
