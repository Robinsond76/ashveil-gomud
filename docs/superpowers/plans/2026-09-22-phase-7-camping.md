# Phase 7 Camping Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a durable, real-time, fatigue-only campsite and rest loop.

**Architecture:** `internal/camping` owns validated camp/rest state and UTC math. `modules/camping` owns persistence, timers, eligible-room command admission, recovery, views, and movement blocking. `modules/survival` owns idempotent company-wide fatigue recovery through a new service seam.

**Tech Stack:** Go 1.24, `time`, `sync`, testify, GoMud modules/plugins/YAML persistence, existing company and survival services.

**Spec:** `docs/superpowers/specs/2026-09-22-phase-7-camping-design.md`

## Global Constraints

- Never advance global game time, round count, weather, combat, room contents, or travel state.
- Persist stable IDs, UTC values, and state only; never timer handles or runtime mob IDs.
- Rest lasts exactly 60 seconds and restores exactly 20 fatigue to every current company member once.
- Invalid/foreign-room durable records remain retained and logged for repair; never guess, move, or delete them.
- Scope excludes supplies, food/water, weather, encounters, temporary rooms, watches, cargo, mounts, and formation effects.

## Review Focus

- Crash between survival recovery and camp finalization must retry without double recovery.
- A stale timer must not complete a replacement/cleared rest.
- Resting blocks only ordinary movement, not status or camp commands.
- An overdue rest after copyover completes once from UTC elapsed time.
- A failed terminal cleanup must remain terminal and never begin a fresh rest.

### Task 1: Model validated camp and real-time rest state

**Files:** Create `internal/camping/camping.go`, `internal/camping/camping_test.go`.

- [ ] Write failing tests for `Camp{LeaderUserID,RoomID,FireLit,Rest}`, `RestSession{StartedAtUTC,State}`, `Established`, `LightFire`, `StartRest`, `ProgressAt`, `RestDue`, `CompleteRest`, and `Break`; reject non-positive IDs, unlit rest, duplicate fire/rest, terminal/invalid transitions, and non-60-second timing.
- [ ] Run `go test ./internal/camping -count=1`; expect missing package/types.
- [ ] Implement pure validation and copy-returning transitions. Define `RestDuration = 60*time.Second`, `FatigueRecovery = 20`, and `Resting`/`Completed` session states. Clamp progress to `0..1`; completed records retain start time and cannot restart.
- [ ] Run `go test -race ./internal/camping -count=1` and commit `feat(camping): model durable campsite rest`.

### Task 2: Add idempotent company rest recovery

**Files:** Modify `internal/survival/survival.go`, `internal/survival/survival_test.go`, `modules/survival/survival.go`, `modules/survival/survival_test.go`.

- [ ] Write failing tests for `ApplyCompanyRestRecovery(leaderID, operationID, fatigue)` applying fatigue to every roster member in one durable write, replaying the identical operation without a second change, rejecting a conflicting operation ID/cost, and rolling in-memory state back on a save failure.
- [ ] Run focused internal/module tests; expect the service API absent.
- [ ] Add an applied-rest-operation ledger to the durable survival registry and an exported `CompanyService` method plus wrapper. Reuse existing roster projection and `ApplyRestRecovery`; persist the ledger and needs atomically in the survival module.
- [ ] Run `go test -race ./internal/survival ./modules/survival -count=1` and commit `feat(survival): apply idempotent company rest recovery`.

### Task 3: Add durable camp module, commands, and eligibility

**Files:** Create `modules/camping/camping.go`, `modules/camping/camping_test.go`, `modules/camping/files/data-overlays/config.yaml`; modify `_datafiles/world/default/rooms/2002-fork-at-black-oak.yaml` only if it is the existing proving room.

- [ ] Write failing tests using fake store/clock/scheduler/survival/room provider: ineligible or duplicate `camp` refuses; eligible camp persists; fire requires camp; rest requires fire and schedules 60 seconds; status shows camp/fire/rest state.
- [ ] Run `go test ./modules/camping -run 'Test.*(Camp|Fire|Rest|Status)' -count=1`; expect module absent.
- [ ] Implement leader-keyed YAML registry, strict decode, injected dependencies, `camp` command dispatcher, room-tag eligibility, and status output. Register the module conventionally and ship only an eligibility configuration/tag needed for the proving room.
- [ ] Run focused module race tests and commit `feat(camping): establish durable camps`.

### Task 4: Finalize rest, recovery, views, and movement blocking

**Files:** Modify `modules/camping/camping.go`, `modules/camping/camping_test.go`; modify the existing expedition/native movement provider seam only if a composable camping movement provider is required.

- [ ] Write failing tests for completion applying one recovery operation, stale callback no-op, failed survival/finalization recovery, restart/copyover overdue completion, invalid/foreign-room retention, `camp break`, paused/rest view, and directional movement block while resting.
- [ ] Run focused tests; expect absent completion/recovery behavior.
- [ ] Implement timer-generation ownership, persistence-before-side-effect protocol, load/copyover recovery, `look` view provider, movement refusal, and terminal cleanup. Ensure idle/broken camps never block movement and terminal records never recover fatigue twice.
- [ ] Run `go test -race ./internal/camping ./internal/survival ./modules/survival ./modules/camping -count=1` and commit `feat(camping): complete real-time camp rest`.

### Task 5: Verify and record Phase 7

**Files:** Modify `docs/PROJECT_STATUS.md`.

- [ ] Run `make generate`, `make validate`, and `go test -race ./...`; inspect generated-file drift.
- [ ] Attempt live acceptance only with an interactive Telnet client: establish camp at the proving room, light fire, rest, verify no world-time skip and one fatigue recovery, then break camp. Record unavailable prerequisites precisely.
- [ ] Update project status with completed scope, commits, actual verification, and Phase 8 as next; commit `docs: record phase 7 camping`.
