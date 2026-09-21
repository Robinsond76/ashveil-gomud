# Phase 5 Travel Durability Corrections Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Phase 5 survival checkpoints crash-idempotent and reconcile completed travel automatically when a leader returns.

**Architecture:** `internal/expedition` persists a pending checkpoint with a deterministic operation ID. `internal/survival` and `modules/survival` durably deduplicate that ID in the same write as needs. `modules/expedition` prepares, applies, and finalizes each checkpoint, then reconciles sessions from `events.PlayerSpawn`.

**Tech Stack:** Go 1.24, GoMud plugin YAML persistence, `sync`, `time`, testify.

**Spec:** `docs/superpowers/specs/2026-09-21-phase-5-travel-durability-corrections-design.md`

## Global Constraints

- Do not change route duration, metadata, or Phase 5's informational-only needs.
- Persist the operation before survival mutates needs; a retry uses the same ID.
- Replaying an operation must never decrement needs again.
- Never advance GoMud global game time or rounds.
- Reconcile from `events.PlayerSpawn`, not a socket or a core copyover contributor.
- Update `docs/PROJECT_STATUS.md` only after every verification command passes.

## Review Focus

- Crash after survival's write but before expedition finalization must replay with no second cost (Task 3).
- Reusing an operation ID with another cost must fail without mutation (Task 2).
- Returning in the origin with a completed session must move once and unblock input (Task 4).
- Returning at the destination must only clean up, with no duplicate arrival (Task 4).
- A timer interrupted by a checkpoint write failure must complete after a later successful synchronization (Task 3).

## File Structure

- `internal/expedition/expedition.go` and tests: pending checkpoint value, stable operation ID, validation, and math.
- `internal/survival/survival.go` and tests: operation-aware service interface and durable ledger type.
- `modules/survival/survival.go` and tests: atomic needs-plus-ledger mutation and replay/conflict behavior.
- `modules/expedition/expedition.go` and tests: prepare/apply/finalize recovery plus player-spawn reconciliation.
- `docs/PROJECT_STATUS.md`: corrected completion record after verification.

### Task 1: Define a persistable pending checkpoint

**Files:** Modify `internal/expedition/expedition.go`, `internal/expedition/expedition_test.go`.

**Produces:** `PendingExertion{OperationID string; Checkpoint uint8; Cost survival.Exertion}`, `TravelSession.PendingExertion *PendingExertion`, and `PrepareExertion(now, profile) (PendingExertion, bool, error)`.

- [ ] **Step 1: Write failing tests** for a seven-of-ten pending charge, deterministic IDs, existing-pending reuse, invalid pending values, and YAML round-trip.

```go
pending, ok, err := session.PrepareExertion(now.Add(7*time.Second), profile)
require.NoError(t, err)
require.True(t, ok)
assert.Equal(t, uint8(7), pending.Checkpoint)
assert.Equal(t, pending.OperationID, repeated.OperationID)
```

- [ ] **Step 2: Run** `go test ./internal/expedition -run TestPrepareExertion -count=1` and confirm it fails because the type and method do not exist.

- [ ] **Step 3: Implement** a YAML-tagged pending value. Derive its ID from leader/origin/destination/profile/UTC-start/target-checkpoint values; return it unchanged if already pending; return no value if no nonzero cost is due. Reject an empty ID, out-of-range target, or negative cost in session validation.

- [ ] **Step 4: Run** `go test -race ./internal/expedition -count=1`; expect PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/expedition
git commit -m "fix(expedition): persist pending checkpoint operations"
```

### Task 2: Add survival-side operation deduplication

**Files:** Modify `internal/survival/survival.go`, `internal/survival/survival_test.go`, `modules/survival/survival.go`, and `modules/survival/survival_test.go`.

**Produces:** `Registry.AppliedExertion map[int]map[string]Exertion`; `CompanyService.ApplyCompanyExertion(leaderID int, operationID string, cost Exertion)` and matching exported forwarder.

- [ ] **Step 1: Write failing tests** for new-ID application, same-ID/same-cost replay, and same-ID/different-cost rejection.

```go
_, err := module.ApplyCompanyExertion(7, "expedition-7", domain.Exertion{Hunger: 4})
require.NoError(t, err)
_, err = module.ApplyCompanyExertion(7, "expedition-7", domain.Exertion{Hunger: 4})
require.NoError(t, err)
assert.Equal(t, 96, module.registry.MustNeedsFor(7, domain.LeaderMemberKey).Hunger)
```

- [ ] **Step 2: Run** `go test ./modules/survival -run 'TestApplyCompanyExertion.*Operation' -count=1` and confirm it fails because the operation ID and ledger do not exist.

- [ ] **Step 3: Implement** non-empty-ID validation and a typed conflicting-operation error. Check the ledger before mutation: return current company results for matching cost; reject a mismatched cost. For a new ID, mutate needs, record it, and save needs plus ledger in the current one-write `m.save()` path; restore the cloned registry on failure.

- [ ] **Step 4: Update all service callers/tests** for the three-argument API, including unavailable-provider forwarding.

- [ ] **Step 5: Run** `go test -race ./internal/survival ./modules/survival -count=1`; expect PASS.

- [ ] **Step 6: Commit.**

```bash
git add internal/survival modules/survival
git commit -m "fix(survival): deduplicate exertion operations"
```

### Task 3: Safely synchronize expedition checkpoints

**Files:** Modify `modules/expedition/expedition.go`, `modules/expedition/expedition_test.go`.

**Consumes:** Task 1 pending values and Task 2's `ApplyCompanyExertion(leaderID, operationID, cost)`.

- [ ] **Step 1: Write failing tests** simulating a finalization-save failure after survival success, restart from both stores, and recovery to exact total cost plus one destination move. Add a test that a timer failure can later complete through `Sync`.

```go
require.Error(t, first.Sync(7))
restarted := newTestModule(expeditionStore, scheduler, mover, survival, now, profiles)
restarted.load()
assert.Equal(t, 90, survival.needsFor(7).Hunger)
assert.Equal(t, []int{200}, mover.moves)
```

- [ ] **Step 2: Run** `go test ./modules/expedition -run 'Test.*(Pending|Crash|ExactlyOnce)' -count=1` and confirm it fails against apply-then-save behavior.

- [ ] **Step 3: Implement prepare/apply/finalize.** Persist pending first, call survival with the stable ID, then set `LastExertionCheckpoint`, clear pending, and persist. On failure retain pending and do not complete. Route load, timer, `look`, status, and `Sync` through this one path.

- [ ] **Step 4: When synchronization succeeds at elapsed completion, invoke completion** so a previously failed timer cannot leave an active route permanently stuck.

- [ ] **Step 5: Run** `go test -race ./modules/expedition -count=1`; expect PASS.

- [ ] **Step 6: Commit.**

```bash
git add modules/expedition/expedition.go modules/expedition/expedition_test.go
git commit -m "fix(expedition): finalize checkpoints idempotently"
```

### Task 4: Reconcile travel at player spawn

**Files:** Modify `modules/expedition/expedition.go`, `modules/expedition/expedition_test.go`.

**Produces:** `onPlayerSpawn(events.Event) events.ListenerReturn`, registered by module initialization.

- [ ] **Step 1: Write failing lifecycle tests** based on `modules/company/lifecycle_test.go`: a normal leave, time advancing beyond duration, new origin login, then `PlayerSpawn` must move once to destination, remove the session, and permit ordinary movement. Add an already-at-destination test that performs cleanup only.

```go
require.Equal(t, events.Continue, module.onPlayerSpawn(events.PlayerSpawn{UserId: 7}))
assert.Equal(t, destination.RoomId, user.Character.RoomId)
assert.NotContains(t, module.sessions, 7)
blocked, _ := module.MovementBlocked(7)
assert.False(t, blocked)
```

- [ ] **Step 2: Run** `go test ./modules/expedition -run 'Test.*(PlayerSpawn|Reconnect|Completed)' -count=1` and confirm it fails because the module does not reconcile player spawns.

- [ ] **Step 3: Implement** listener registration and leader-only reconciliation under the module mutex. Synchronize traveling sessions, complete overdue sessions, and reuse existing origin/destination/unexpected-room handling. Make `MovementBlocked` block only `Traveling` state.

- [ ] **Step 4: Run** `go test -race ./internal/usercommands ./modules/expedition -count=1`; expect PASS.

- [ ] **Step 5: Commit.**

```bash
git add modules/expedition/expedition.go modules/expedition/expedition_test.go
git commit -m "fix(expedition): reconcile travel on player spawn"
```

### Task 5: Verify and amend Phase 5 status

**Files:** Modify `docs/PROJECT_STATUS.md`.

- [ ] **Step 1: Run** `make generate && make validate`; expect PASS with no generated-file drift.

- [ ] **Step 2: Run** `go test -race ./...`; expect PASS, or document the exact environmental blocker.

- [ ] **Step 3: Update** the Phase 5 status record to remove the double-charge limitation and document pending-operation deduplication plus player-spawn recovery while retaining the plugin save/load copyover explanation.

- [ ] **Step 4: Commit.**

```bash
git add docs/PROJECT_STATUS.md
git commit -m "docs: record phase 5 durability corrections"
```

## Plan Self-Review

- **Spec coverage:** Tasks 1–3 cover durable identity and crash boundaries; Task 4 covers reconnect recovery; Task 5 verifies and records the correction.
- **Placeholder scan:** No TODO/TBD or deferred implementation steps are present.
- **Type consistency:** Task 1 defines `PendingExertion`; Task 2 defines the operation-aware survival API; Tasks 3–4 consume both exact names.
- **Review focus coverage:** Every listed risk has a concrete regression test in Tasks 2–4.
