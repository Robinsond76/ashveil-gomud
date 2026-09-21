# Phase 6 Travel Interruptions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (- [ ]) syntax for tracking.
>
> **Workspace:** Implement only after Phase 5 travel-durability corrections are merged. Use an isolated worktree and feature branch (for example, `git worktree add .worktrees/phase-6-interruptions -b phase-6-interruptions`); never work directly on master.

**Goal:** Add one durable, deterministic fallen-tree interruption that pauses real-time travel and lets the leader resume or return safely.

**Architecture:** internal/expedition owns typed interruption values and active-time session math. modules/expedition remains the sole persisted transition owner: it schedules route boundaries, uses Phase 5's pending-exertion protocol, pauses/restarts timers, recovers sessions, and implements the travel commands.

**Tech Stack:** Go 1.24, time, sync, testify, GoMud module YAML persistence, GoMud events/users/rooms, and the Phase 4 survival service.

**Spec:** docs/superpowers/specs/2026-09-21-phase-6-travel-interruptions-design.md

## Global Constraints

- Complete docs/superpowers/plans/2026-09-21-phase-5-travel-durability-corrections.md first; use its PendingExertion operation-ID workflow for every checkpoint.
- An unconfigured profile preserves all Phase 5 behavior.
- Support one fallen-tree interruption per session at profile checkpoint 1..9; do not add random rolls or tick/frame polling.
- Derive progress, remaining duration, exertion, and scheduling from active elapsed UTC time; paused time never counts as route progress.
- Persist stable leader/session/profile/pause/interruption values only. Never persist timer handles, sockets, monotonic clocks, or mob instance IDs.
- Apply survival exertion only at durable checkpoints. A pause has no cost/refund and trigger, resume, restart, and retry cannot duplicate a charge.
- An interrupted leader remains blocked from ordinary movement and cannot interact with origin/destination room content through travel views.
- Do not add combat, ambushes, temporary rooms, event tables, rewards, weather, camping, cargo, mounts, rerouting, or formation mechanics.
- Never advance GoMud global game time, round count, or world-time scheduling.

## Review Focus

- A resumed session must not trigger its profile interruption again; Task 1 pins the durable one-shot marker.
- A restart while paused must add the entire offline pause on resume; Tasks 1 and 4 pin active-time recovery.
- Checkpoint 10 must be rejected rather than race arrival; Tasks 1 and 2 pin it.
- A post-checkpoint interruption-save failure must retain a recoverable travelling record, not report a false interruption; Task 3 pins order and rollback.
- A stale completion callback after interruption or return must not move or announce arrival; Task 3 pins the state guard.

---

## File Structure

- internal/expedition/expedition.go — interruption/profile values, active elapsed-time calculation, validation, and pure state transitions.
- internal/expedition/expedition_test.go — profile/session validity, pause arithmetic, one-shot behavior, checkpoint/exertion, and transition tests.
- modules/expedition/expedition.go — profile parsing, next-boundary scheduling, persisted transitions, recovery, command parsing, and status/refusal output.
- modules/expedition/expedition_test.go — parser, timer, persistence, resolution, recovery, stale-callback, and command/status tests.
- modules/expedition/files/data-overlays/config.yaml — Oak Road's proving interruption.
- docs/PROJECT_STATUS.md — Phase 6 completion record after verification.

### Task 1: Extend the pure session model with active-time interruption semantics

**Files:**
- Modify: internal/expedition/expedition.go
- Modify: internal/expedition/expedition_test.go

**Interfaces:**
- Consumes: existing TravelProfile, TravelSession, SessionState, CheckpointCount, and Phase 5 PendingExertion.
- Produces: InterruptionKind, FallenTree, InterruptionProfile, TravelInterruption, ErrInvalidInterruption, TravelSession.ActiveElapsedAt(now, duration), RemainingAt(now, duration), InterruptionDue(now, profile), Interrupt(now, profile), Resume(now), and Return().

- [ ] **Step 1: Write failing domain tests**

Add a valid fallen-tree profile to validProfile. Table-test empty/unknown kind, checkpoint 0, checkpoint CheckpointCount, and a valid configured interruption. Extend session validation tests for negative PausedDuration; interrupted sessions missing payload/PausedAtUTC; a payload on Traveling; mismatched kind/checkpoint; and invalid persisted interruption data.

Test a 30-second route interrupted at 15 seconds: progress remains 50% after an hour paused; resuming then leaves 15 seconds; and route completion 15 active seconds later reaches 100%. Assert resume retains InterruptionTriggered, clears only the active payload/pause instant, and InterruptionDue stays false.

~~~go
func TestResumePreservesOneShotInterruptionAndActiveProgress(t *testing.T) {
    profile := validProfile()
    profile.Interruption = &InterruptionProfile{Kind: FallenTree, Checkpoint: 5}
    session := validSession()

    interrupted, err := session.Interrupt(baseTime().Add(15*time.Second), profile)
    require.NoError(t, err)
    resumed, err := interrupted.Resume(baseTime().Add(time.Hour))
    require.NoError(t, err)

    assert.True(t, resumed.InterruptionTriggered)
    assert.False(t, resumed.InterruptionDue(baseTime().Add(time.Hour), profile))
    assert.Equal(t, 15*time.Second, resumed.RemainingAt(baseTime().Add(time.Hour), profile.Duration))
}
~~~

- [ ] **Step 2: Run the focused domain tests and verify failure**

Run: go test ./internal/expedition -run 'Test.*(Interruption|ActiveElapsed|Resume|Return)' -count=1

Expected: FAIL because the interruption types, pause fields, and active-time methods do not exist.

- [ ] **Step 3: Implement value types, validation, and pure transitions**

Add YAML-tagged optional Interruption to TravelProfile and PausedAtUTC, PausedDuration, Interruption, and InterruptionTriggered to TravelSession. Keep TravelProfile.Validate the sole profile validator; absence remains valid.

Create ActiveElapsedAt as shared math and route existing ProgressAt, CheckpointAt, and ExertionDue through it without changing their public signatures.

~~~go
func (s TravelSession) ActiveElapsedAt(now time.Time, duration time.Duration) time.Duration {
    elapsed := now.Sub(s.StartedAtUTC) - s.PausedDuration
    if s.State == Interrupted && !s.PausedAtUTC.IsZero() {
        elapsed -= now.Sub(s.PausedAtUTC)
    }
    if elapsed < 0 { return 0 }
    if elapsed > duration { return duration }
    return elapsed
}

func (s TravelSession) InterruptionDue(now time.Time, p TravelProfile) bool {
    return s.State == Traveling && !s.InterruptionTriggered &&
        p.Interruption != nil &&
        s.CheckpointAt(now, p.Duration) >= p.Interruption.Checkpoint
}
~~~

Interrupt validates a due profile, sets Interrupted, PausedAtUTC, immutable kind/checkpoint payload, and InterruptionTriggered in the returned copy. Resume accepts only a valid Interrupted record, adds now - PausedAtUTC to PausedDuration, clears the active payload/pause instant, and returns Traveling. Return transitions only Interrupted to Cancelled. None of these mutate LastExertionCheckpoint or PendingExertion.

- [ ] **Step 4: Run the full domain race suite**

Run: go test -race ./internal/expedition -count=1

Expected: PASS, including existing Phase 5 exact-cost tests for profiles with no interruption.

- [ ] **Step 5: Commit**

~~~bash
git add internal/expedition/expedition.go internal/expedition/expedition_test.go
git commit -m "feat(expedition): model durable travel interruptions"
~~~

### Task 2: Parse and ship the Oak Road interruption

**Files:**
- Modify: modules/expedition/expedition.go
- Modify: modules/expedition/expedition_test.go
- Modify: modules/expedition/files/data-overlays/config.yaml

**Interfaces:**
- Consumes: Task 1 InterruptionProfile and TravelProfile.Validate.
- Produces: parseInterruption(raw any) (*expedition.InterruptionProfile, bool) and parseProfiles with optional valid interruptions.

- [ ] **Step 1: Write failing parser and content tests**

Extend TestParseProfilesRejectsMalformedAndDuplicates with a valid nested Interruption map. Assert Kind equals expedition.FallenTree and Checkpoint is 5. Add malformed cases: unknown kind, non-map interruption, checkpoint 0, checkpoint 10, and otherwise-valid profile with malformed interruption. Extend TestDunmarOakRoute to require the parsed Oak Road profile's fallen-tree checkpoint 5.

~~~go
raw := []any{map[string]any{
    "Name": "oak-road", "Duration": "30s",
    "Interruption": map[string]any{"Kind": "fallen-tree", "Checkpoint": 5},
}}
profile := parseProfiles(raw)["oak-road"]
require.NotNil(t, profile.Interruption)
assert.Equal(t, expedition.FallenTree, profile.Interruption.Kind)
assert.Equal(t, uint8(5), profile.Interruption.Checkpoint)
~~~

- [ ] **Step 2: Run focused tests and verify failure**

Run: go test ./modules/expedition -run 'Test(ParseProfiles|DunmarOakRoute)' -count=1

Expected: FAIL because configuration does not parse or ship an interruption.

- [ ] **Step 3: Parse only the documented shape and update Oak Road**

Use existing stringMap and configInt. parseInterruption returns (nil, true) when absent and (nil, false) when present but malformed. parseProfiles skips a profile containing an invalid interruption; it must not silently downgrade it to no interruption.

Add only this nested data to the existing oak-road profile:

~~~yaml
    Interruption:
      Kind: fallen-tree
      Checkpoint: 5
~~~

Do not add room metadata, a second route, content files, or event tables.

- [ ] **Step 4: Run focused module checks**

Run: go test -race ./modules/expedition -run 'Test(ParseProfiles|DunmarOakRoute)' -count=1

Expected: PASS.

- [ ] **Step 5: Commit**

~~~bash
git add modules/expedition/expedition.go modules/expedition/expedition_test.go modules/expedition/files/data-overlays/config.yaml
git commit -m "feat(expedition): configure oak road interruption"
~~~

### Task 3: Make synchronization and timers interruption-safe

**Files:**
- Modify: modules/expedition/expedition.go
- Modify: modules/expedition/expedition_test.go

**Interfaces:**
- Consumes: Task 1 session operations and Phase 5's prepare/apply/finalize pending-exertion synchronization.
- Produces: nextBoundaryDelayLocked(session, profile), interruptLocked(session), and a syncLocked/onTimer path that reaches exactly one of Interrupted or Completed.

- [ ] **Step 1: Write failing timer, checkpoint, and failure-order tests**

Use a fifth-checkpoint fallen-tree test profile. Assert starting it schedules 5 seconds, not 10. At 5 seconds fire the fake callback: survival receives exactly the fifth-checkpoint cumulative cost; saved and live state is Interrupted; there is no active completion timer; no destination move occurs. Fire the callback again and assert no extra exertion, save, move, or interruption message.

Add a failed-interruption-save test. Make the store succeed for start and Phase 5 finalization but fail the next save. Assert the live/durable session returns to Traveling at the finalized checkpoint, no message is sent, and no move occurs. Clear the failure and Sync; interruption persists without a second survival operation.

~~~go
now = baseTime().Add(5 * time.Second)
scheduler.fire(0)
session := module.sessions[7]
assert.Equal(t, expedition.Interrupted, session.State)
assert.Equal(t, uint8(5), session.LastExertionCheckpoint)
assert.Empty(t, mover.moves)
assert.Equal(t, survival.Exertion{Hunger: 5, Thirst: 5, Fatigue: 5}, surv.applied[0])
~~~

- [ ] **Step 2: Run focused scheduler tests and verify failure**

Run: go test ./modules/expedition -run 'Test.*(Interruption|Boundary|Stale|InterruptionSave)' -count=1

Expected: FAIL because Phase 5 schedules only completion and never creates an interrupted record.

- [ ] **Step 3: Implement next-boundary scheduling and persistence-ordered interruption**

Replace StartedAtUTC.Add(duration) delay calculation. For an untriggered profile interruption, target duration is profile.Duration * checkpoint / expedition.CheckpointCount; otherwise it is profile.Duration. Subtract session.ActiveElapsedAt(now, profile.Duration), clamp at zero, stop the prior timer, and register one callback.

Route timer callbacks, Sync, status/view sync, and recovery through one locked progression path. First call the Phase 5 checkpoint prepare/apply/finalize routine. If InterruptionDue, create the candidate with Interrupt, install it only for save, restore the original map value on save failure, then stop/remove the timer and message only after successful persistence. Complete only when still Traveling and active elapsed reaches profile.Duration.

~~~go
func (m *ExpeditionModule) nextBoundaryDelayLocked(s expedition.TravelSession, p expedition.TravelProfile) time.Duration {
    target := p.Duration
    if !s.InterruptionTriggered && p.Interruption != nil {
        target = p.Duration * time.Duration(p.Interruption.Checkpoint) / expedition.CheckpointCount
    }
    delay := target - s.ActiveElapsedAt(m.clock().UTC(), p.Duration)
    if delay < 0 { return 0 }
    return delay
}
~~~

Keep the module mutex as sole owner of sessions/timers. A stale callback observes current state and is a no-op for Interrupted or Cancelled.

- [ ] **Step 4: Run module race tests**

Run: go test -race ./modules/expedition -count=1

Expected: PASS, including all existing exactly-once completion/recovery coverage for profiles without interruptions.

- [ ] **Step 5: Commit**

~~~bash
git add modules/expedition/expedition.go modules/expedition/expedition_test.go
git commit -m "feat(expedition): pause travel at route interruptions"
~~~

### Task 4: Add resume/return, interrupted rendering, and durable recovery

**Files:**
- Modify: modules/expedition/expedition.go
- Modify: modules/expedition/expedition_test.go

**Interfaces:**
- Consumes: Task 1 Resume/Return, Task 3 scheduleLocked, existing travel command/store/clock/scheduler dependencies.
- Produces: resume(leaderUserID int) string, returnToOrigin(leaderUserID int) string, interruptionTextLocked(session), and travel status|resume|return behavior.

- [ ] **Step 1: Write failing lifecycle, command, and recovery tests**

Create an interrupted session at 50%, with PausedAtUTC equal to baseTime plus 5 seconds, InterruptionTriggered true, and a fallen-tree payload. At one hour, resume it. Assert Traveling state, PausedDuration of 59m55s, cleared active payload/pause instant, one new 5-second timer, normal status/remaining text, and no survival cost while paused. Fire after 5 active seconds and assert one final move and the exact total exertion.

Test return persists Cancelled before cleanup, leaves a fake user at origin, makes MovementBlocked false after cleanup, and never calls mover. Force final cleanup save failure and assert Cancelled remains; load must clean it up without movement. Verify malformed Interrupted persistence stays retained with no timer. Through userCommand, assert unknown/empty arguments preserve usage, resolution outside Interrupted gives clear refusal, and valid resume/return send their results. Assert MovementBlocked, statusTextLocked, and RenderTravelView identify a paused fallen tree and do not render a live countdown.

~~~go
now = baseTime().Add(time.Hour)
text := module.resume(7)
assert.Contains(t, text, "resume")
assert.Equal(t, expedition.Traveling, module.sessions[7].State)
assert.Equal(t, 59*time.Minute+55*time.Second, module.sessions[7].PausedDuration)
assert.Equal(t, 5*time.Second, scheduler.delays[len(scheduler.delays)-1])
~~~

- [ ] **Step 2: Run focused tests and verify failure**

Run: go test ./modules/expedition -run 'Test.*(Resume|Return|Interrupted|Paused|Cancelled)' -count=1

Expected: FAIL because Phase 5 only accepts travel status and retains Interrupted/Cancelled for forward compatibility.

- [ ] **Step 3: Implement resolution, view, recovery, and dispatch**

Under module lock, require persistence and a current Interrupted session. Resume creates session.Resume(clock.UTC), saves before scheduling, restores original on save failure, schedules its next boundary, then confirms. Return creates/persists Cancelled, stops timer, announces return, removes record, and saves cleanup; if cleanup save fails, retain the durable Cancelled record for later cleanup only.

On recovery, a valid Interrupted record schedules no timer/exertion. Cancelled cleans up without movement. Invalid paused/interrupted records log and remain for repair. Use ActiveElapsedAt/RemainingAt in refusal/status rendering; Interrupted output identifies obstruction and travel resume/travel return. Keep look on the existing provider seam.

~~~go
switch args[0] {
case "status":
    user.SendText(m.status(user.UserId))
case "resume":
    user.SendText(m.resume(user.UserId))
case "return":
    user.SendText(m.returnToOrigin(user.UserId))
default:
    user.SendText(travelUsage)
}
~~~

- [ ] **Step 4: Run cross-package focused race verification**

Run: go test -race ./internal/expedition ./internal/usercommands ./modules/expedition -count=1

Expected: PASS; native movement/look provider seams continue to block and render active travel without an engine rewrite.

- [ ] **Step 5: Commit**

~~~bash
git add modules/expedition/expedition.go modules/expedition/expedition_test.go
git commit -m "feat(expedition): resolve interrupted travel"
~~~

### Task 5: Verify and record Phase 6 completion

**Files:**
- Modify: docs/PROJECT_STATUS.md

**Interfaces:**
- Consumes: Tasks 1–4 and existing make generate, make validate, and Go race-test workflows.
- Produces: accurate Phase 6 status/verification record.

- [ ] **Step 1: Run focused checks**

~~~bash
go test -race ./internal/expedition ./internal/usercommands ./modules/expedition -count=1
make generate
make validate
~~~

Expected: all exit 0. Inspect generated-file drift; include modules/all-modules.go only if generation creates a real registration change.

- [ ] **Step 2: Run broad race verification**

Run: go test -race ./...

Expected: PASS. If blocked by the environment, document the exact prerequisite; do not claim a pass.

- [ ] **Step 3: Run local acceptance when prerequisites permit**

Start the server using the documented local procedure. From Dunmar West Gate, start Oak Road, wait for the midway fallen-tree pause, run look, travel status, and travel resume, then verify arrival after only the remaining active duration. Repeat and run travel return; verify the leader stays in origin and ordinary movement works after cleanup. Stop cleanly. Record unavailable prerequisites rather than claiming a live pass.

- [ ] **Step 4: Update project status after actual verification**

Mark Phase 6 complete and Phase 7 next. Record the durable single fallen-tree interruption, active-time resume/return semantics, no-combat boundary, Phase 5 operation-ID prerequisite, Oak Road proof, commits, and actual checks.

- [ ] **Step 5: Commit**

~~~bash
git add docs/PROJECT_STATUS.md modules/all-modules.go
git commit -m "docs: record phase 6 travel interruptions"
~~~

## Plan Self-Review

- **Spec coverage:** Task 1 covers typed durable interruption and active-time math; Task 2 strict configuration/content; Task 3 checkpoint ordering, timer boundaries, and stale callbacks; Task 4 resolution, output, persistence/restart/copyover, and movement refusal; Task 5 verification/status.
- **Completeness scan:** No deferred markers or unnamed implementation work remain. Every code task includes concrete tests, interfaces, and persistence order.
- **Type consistency:** Task 1 defines InterruptionProfile, TravelInterruption, ActiveElapsedAt, RemainingAt, InterruptionDue, Interrupt, Resume, and Return; Tasks 2–4 use those exact names. PendingExertion remains Phase 5's API.
- **Review-focus coverage:** One-shot resume is in Task 1; paused restart and checkpoint-10 rejection in Tasks 1, 2, and 4; post-checkpoint persistence rollback and stale callbacks in Task 3.
