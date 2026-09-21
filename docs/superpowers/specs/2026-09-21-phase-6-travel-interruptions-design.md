# Phase 6 Travel Interruptions Design

**Status:** Approved for implementation planning

**Goal:** Make a durable real-time expedition safely interruptible by one
configured, non-combat route obstruction. An interruption pauses travel without
advancing GoMud's global clock, survives restart/copyover, and lets the leader
resume or turn back exactly once.

## Decisions

- Phase 6 introduces a reusable, typed interruption contract and proves it with
  a deterministic `fallen-tree` obstruction. It does not add native combat,
  temporary encounter rooms, random encounter tables, or formation-combat
  effects.
- An interruption is scheduled by the selected travel profile at one configured
  progress checkpoint. It is not rolled every server tick and can occur at most
  once per travel session.
- Entering `Interrupted` pauses elapsed travel time. While paused, progress,
  remaining travel exertion, and arrival timers do not advance. The shared
  GoMud world clock and round count continue normally.
- The leader resolves a fallen tree with `travel resume` or `travel return`.
  `resume` continues from the durable checkpoint; `return` ends the session and
  leaves the company in its origin room. No player-controlled cancellation is
  added for an uninterrupted journey.
- Survival exertion remains proportional to active route progress. A pause does
  not cost survival, and resuming cannot re-charge an already persisted
  checkpoint.
- Combat ambushes are deliberately deferred. A later handler may use the same
  interruption identity and resolve/resume contract after native combat and
  formation ownership have a dedicated design.

## Scope

### Included

- An explicit interruption payload, pause accounting, and legal transition
  rules in `internal/expedition`.
- One deterministic per-profile interruption trigger, using a configured route
  checkpoint and the typed `fallen-tree` kind.
- Durable persistence and recovery of paused sessions in `modules/expedition`.
- Timer cancellation/replacement, checkpoint synchronization, and exactly-once
  handling when an interruption races completion or recovery.
- `travel status`, `look`, and `travel resume|return` player interaction.
- A small Oak Road proving configuration and focused domain/module/command,
  persistence, timer, copyover, and race coverage.

### Excluded

- Native combat, hostile ambushes, mob spawning, combat-room placement,
  rewards, injuries, formation effects, and encounter loot.
- Random rolls, weighted route/region/terrain event tables, weather-driven
  events, multiple simultaneous events, and player decisions other than resume
  or return.
- Camping, rest/sleep, weather, cargo, encumbrance, mounts, multi-leg travel,
  route rerouting, en-route actions, and needs-based travel penalties.
- Changing the current Phase 5 origin-room model: a travelling or interrupted
  company is not represented by an en-route room instance.

## Architecture

`internal/expedition` remains GoMud-free and owns all time arithmetic and
transition validation. It extends `TravelProfile` with an optional,
profile-owned interruption definition, and extends `TravelSession` with the
durable identity and pause fields necessary to calculate *active* elapsed time.
It exposes pure operations to determine a due interruption, enter it, and
resolve it. It imports only `internal/survival` for the existing `Exertion`
value type.

`modules/expedition` remains the sole transition owner for persisted sessions,
timers, survival checkpoint calls, room movement, recovery, and player-facing
output. It injects its clock, scheduler, store, mover, and survival adapter in
tests. The native movement and look seams added in Phase 5 are unchanged; the
module's existing movement refusal covers both `Traveling` and `Interrupted`
sessions. The existing `travel` command gains only the subcommands defined
below.

The implementation must not create an alternate follower, room-movement, item,
combat, or persistence system. Stable leader IDs, company roster projection,
and native arrival movement remain the Phase 5 integration boundaries.

## Domain Model and Active-Time Accounting

The public types are conceptually:

```go
type InterruptionKind string

const FallenTree InterruptionKind = "fallen-tree"

type InterruptionProfile struct {
	Kind       InterruptionKind
	Checkpoint uint8 // 1..CheckpointCount
}

type TravelInterruption struct {
	Kind        InterruptionKind `yaml:"kind"`
	Checkpoint  uint8            `yaml:"checkpoint"`
	InterruptedAtUTC time.Time   `yaml:"interrupted_at_utc"`
}

type TravelSession struct {
	// Existing Phase 5 stable identity, start time, checkpoint, and state.
	PausedAtUTC          time.Time           `yaml:"paused_at_utc,omitempty"`
	PausedDuration       time.Duration       `yaml:"paused_duration,omitempty"`
	Interruption         *TravelInterruption `yaml:"interruption,omitempty"`
	InterruptionTriggered bool               `yaml:"interruption_triggered,omitempty"`
}
```

`TravelProfile.Interruption` is optional. A populated profile requires a known
kind and a checkpoint in `1..CheckpointCount`; Phase 6 supports only
`FallenTree`. A session may have a non-nil interruption only while its state is
`Interrupted`. Its kind/checkpoint must match the profile that started the
session. `InterruptionTriggered` is set in the same durable transition that
enters `Interrupted`, and is never cleared by resume; it prevents a resumed
session from triggering the profile's one event again. A malformed persisted
interruption is unavailable for automatic recovery and must be retained/logged
for operator repair; the module must never guess whether to arrive, return, or
resume it.

Active elapsed duration is not `now - StartedAtUTC` once a pause exists. The
domain calculates it as:

```text
now - StartedAtUTC - PausedDuration - currentPausedInterval
```

where `currentPausedInterval` is `now - PausedAtUTC` only while the state is
`Interrupted`. Clamp the result to `[0, profile.Duration]`. `ProgressAt`,
`CheckpointAt`, `ExertionDue`, remaining-duration rendering, and completion
scheduling all use this active elapsed value. On `resume`, add the current
paused interval to `PausedDuration`, clear `PausedAtUTC` and `Interruption`, and
transition back to `Traveling`. Paused duration is cumulative and serialized;
no runtime timer handle or monotonic-clock value is persisted.

## Trigger and State Transitions

The module synchronizes a travelling session before every status/view command,
timer callback, recovery/copyover pass, and resolution command. When the
profile defines an interruption and the attained durable checkpoint first
reaches its configured checkpoint, it must:

1. Apply and durably persist the incremental survival exertion due through that
   checkpoint.
2. Replace the session with `State: Interrupted`, a `PausedAtUTC` from the
   injected UTC clock, the corresponding immutable interruption payload, and
   `InterruptionTriggered: true`.
3. Persist that transition before reporting the obstruction to the player.
4. Stop and remove the session's travel-completion timer.

The trigger condition is evaluated before normal completion and only when
`InterruptionTriggered` is false. A route whose interruption checkpoint equals
`CheckpointCount` is invalid, avoiding an ambiguous interruption-versus-arrival
race at 100% progress. A travelling session schedules its next *route boundary*:
the earlier of the interruption checkpoint or normal completion. After every
resume/recovery/synchronization, it schedules only that next active-time
boundary. On any callback, synchronization therefore produces exactly one of:
an unchanged travelling session, a durably interrupted session, or a durably
completed session.

The legal state graph is:

```text
Traveling --due configured interruption--> Interrupted
Interrupted --resume--> Traveling
Interrupted --return--> Cancelled --cleanup--> no active session
Traveling --completion--> Completed --arrival verification--> no active session
```

`Completed` and `Cancelled` remain terminal. No callback, retry, duplicate
command, or recovery pass may transition a terminal record back to an active
state. A stale completion callback reads the current record under the module
lock; if it is no longer `Traveling`, it does nothing.

## Player Experience

When the configured checkpoint is reached, the leader sees clear, private
travel text such as:

```text
A fallen tree blocks the Oak Road. Your company pauses before the obstruction.
Use "travel resume" when you are ready to continue, or "travel return" to head back.
```

The `travel` command becomes:

```text
travel status
travel resume
travel return
```

- `travel status` renders the existing origin/destination/profile, active
  progress, remaining active duration, and survival summary. When interrupted,
  it additionally identifies the obstruction and the available resolution
  commands.
- `travel resume` is valid only for the leader's current interrupted session.
  It persistently records elapsed pause time and the resumed `Traveling` state,
  then schedules exactly one timer for the remaining active duration. A failed
  write leaves the durable interruption in place and starts no new timer.
- `travel return` is valid only for an interrupted session. It persistently
  records `Cancelled`, clears any timer, announces that the company returned to
  its origin, then removes the terminal record. It does not move the player,
  refund exertion, or advance global time: the company never left the origin
  room in Phase 5's model.
- `look` uses the existing travel view. For an interrupted session it reports a
  paused journey and the obstruction instead of a live countdown; it must not
  expose origin/destination contents or allow origin interaction.
- Ordinary directional movement remains refused for both travelling and
  interrupted leaders, without changing the session. Chat, status, and manual
  Phase 4 `eat`/`drink` remain responsive according to their existing rules.

## Persistence, Recovery, and Failure Rules

The persisted registry continues to store leader-keyed sessions only. On load,
restart, reconnect, and copyover recovery:

- A valid `Traveling` session derives active elapsed time, synchronizes one due
  checkpoint/interruption if necessary, then schedules only its next remaining
  route boundary.
- A valid `Interrupted` session remains paused. Recovery schedules no
  completion timer, does not accrue survival, and preserves `PausedAtUTC` so
  future resume accounts for the entire real pause across a restart.
- A stale completion callback and an interruption trigger are serialized by the
  module's single mutex. The first successfully persisted transition wins; the
  other observes the new state and is a no-op.
- A persistence, survival, profile, or timer-scheduling failure cannot claim a
  resolved or resumed journey. The last durably known state remains recoverable
  and is logged. No state may be deleted merely because a live timer cannot be
  restored.
- Return cleanup follows Phase 5's terminal-record discipline: persist
  `Cancelled` before cleanup so recovery cannot convert a requested return into
  an arrival. If cleanup write/removal fails, retain the terminal record and
  retry cleanup only; never restore active travel.

No Phase 6 code mutates the global game clock, game round count, or GoMud
world-time scheduling.

## Configuration and Proving Content

Interruption configuration belongs to the expedition module's existing
profile definitions, not room exits. A profile uses an optional nested value:

```yaml
Profiles:
  - Name: oak-road
    Duration: 90s
    Exertion:
      Hunger: 4
      Thirst: 6
      Fatigue: 8
    Interruption:
      Kind: fallen-tree
      Checkpoint: 5
```

The parser accepts only this known kind and requires checkpoint `1..9`. The
Oak Road proving profile intentionally triggers at checkpoint 5 so test and
live validation can demonstrate a mid-route pause. It remains a deliberately
small content slice, not a general event catalog.

## Acceptance Criteria

- A valid profile can define one deterministic fallen-tree interruption before
  completion; profiles without one preserve all Phase 5 travel behavior.
- Reaching its checkpoint applies the exact incremental exertion once, durably
  pauses travel, cancels its completion timer, and presents the resolution.
- While interrupted, active progress, remaining active duration, survival
  exertion, and arrival do not advance—even across restart or copyover.
- `travel resume` resumes from the exact durable checkpoint and schedules only
  the next remaining active-time boundary; it never re-triggers the same event,
  double-charges survival, or creates duplicate timers.
- `travel return` is available only while interrupted, never moves the leader,
  never refunds exertion, and cannot later become an arrival.
- Concurrent/stale completion, trigger, resume, return, persistence failure,
  and recovery paths preserve one legal durable state and never duplicate
  movement, arrival messaging, or survival cost.
- `look`, `travel status`, and ordinary movement accurately represent an
  interrupted leader without exposing room interaction or freezing the
  connection.
- The feature adds no combat, random events, en-route room, global-time change,
  or Phase 7–12 mechanics.
- Focused domain/module/command/persistence/timer/copyover race tests,
  `make generate`, `make validate`, and `go test -race ./...` pass; an
  environmental blocker is documented rather than claimed as passing.

## Deferred Decisions

- A future combat/encounter phase can add `bandit-ambush` and other typed kinds
  only after specifying native-combat entry/exit, formation interaction,
  disconnect/restart recovery, and its explicit resolution outcomes.
- Weighted/random event selection, region/terrain/weather inputs, multiple
  events per journey, loot, discoveries, and NPC choices belong to the rich
  expedition-encounter work, not this checkpoint proof.
- Phase 7 owns camping and fatigue recovery; Phases 8–10 own weather, cargo,
  and mounts; Phase 11 owns formation combat. Those systems may not change the
  Phase 6 active-time and exactly-once transition guarantees without their own
  designs.
