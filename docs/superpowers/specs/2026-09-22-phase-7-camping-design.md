# Phase 7 Camping Design

**Status:** Approved for autonomous implementation by the project owner.

## Goal

Add the first durable, multiplayer-safe campsite loop: a company leader can
establish an eligible-room camp, light its fire, begin a real-time rest, inspect
status, and break camp. Rest restores only fatigue and never advances global
game time or rounds.

## Scope

Phase 7 is deliberately a single-company, origin-room context. A camp is not a
temporary room, map object, shared public object, or new party system. It is a
leader-keyed durable record associated with the leader's current room.

Included commands are:

```text
camp
camp status
camp fire
camp rest
camp break
```

`camp` establishes a camp in a room whose `camping` tag is present. `camp fire`
marks that camp lit. `camp rest` requires a lit camp and starts one configured
real-time rest session. `camp break` is refused while resting and otherwise
removes the camp. No command moves the leader.

## Durable Model

`internal/camping` is GoMud-free and owns validation plus real-UTC rest math.
It defines a `Camp` with stable leader ID, room ID, `FireLit`, and an optional
`RestSession{StartedAtUTC, Completed}`. A rest session calculates progress from
UTC elapsed time clamped to the configured duration. Its completed state is
durable so a crash between recovery and cleanup cannot apply recovery twice.

`modules/camping` owns YAML persistence, timers, player commands, room-tag
admission, recovery, and the survival seam. It uses the same leader-keyed
registry, injected clock/scheduler/store, mutex, and copyover `SetOnLoad`/
`SetOnSave` pattern as expedition. Timer handles and runtime character IDs are
never persisted.

## Rest and Survival

The configured MVP rest duration is 60 seconds and the fatigue benefit is 20.
At completion, the module persists `Completed` before calling a new survival
company seam that restores fatigue to every current roster member in one durable
survival write. It then persists final camp state/cleans the rest subrecord.

The recovery operation uses a deterministic rest operation ID derived from
leader ID, room ID, and UTC start time. The survival module persists an applied
rest-operation ledger, mirroring Phase 5 exertion deduplication: restart or
copyover replay cannot restore fatigue twice. A failed survival or camp write
retains a recoverable pending completed record and reports no false success.

## Player and Recovery Behavior

- Camp establishment requires an eligible room and no existing camp for that
  leader. A returning logged-in leader must still be in the camp room to rest,
  light a fire, or break it; otherwise the durable camp is retained for repair.
- A resting leader is blocked from ordinary movement, with remaining real-time
  rest duration. `look` and `camp status` render the camp/rest view; chat and
  normal non-movement commands remain responsive.
- Load/copyover derives completion from UTC. An active rest reschedules its
  remaining duration; an overdue rest completes once; a completed durable rest
  retries only its idempotent finalization. Invalid records are retained and
  logged, never guessed, moved, or deleted.
- `camp break` removes only an idle camp. It does not refund recovery, alter
  hunger/thirst, remove a fire item, advance time, or affect any other player.

## Constraints and Deferrals

No Phase 7 code may mutate `gametime`, round count, weather, combat, room
contents, items, food/water, cargo, mounts, encounters, formation, or travel
duration. Weather, shelter, fire fuel/items, cooking, watches, camp encounters,
temporary/discoverable camp rooms, multi-player camps, and sleep-until-dawn are
deferred to later dedicated phases.

## Acceptance Criteria

- An eligible room permits one durable camp per leader; an ineligible room and
  duplicate camp are refused without persistence changes.
- Fire, status, break, and rest obey the legal camp/rest state graph.
- A lit camp starts a 60-second real-time rest; no global time changes; fatigue
  rises exactly 20 for every current company member once on successful finish.
- Timers, rest completion, restart/copyover recovery, and repeated status/look
  never double-apply recovery or duplicate messages.
- A resting leader cannot use ordinary exits; idle/broken camps do not block
  movement. Invalid/foreign-room durable records are retained for repair.
- Focused domain/module/survival/command race tests, `make generate`,
  `make validate`, and `go test -race ./...` provide verification. Live Telnet
  acceptance is recorded only when an interactive client is available.
