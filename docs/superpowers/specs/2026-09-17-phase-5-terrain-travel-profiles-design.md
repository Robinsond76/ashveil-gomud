# Phase 5 Terrain and Travel Profiles Design

**Status:** Approved for implementation planning

**Goal:** Add data-driven terrain travel profiles and durable, real-time travel for selected world exits. Ordinary exits remain instant. A travelling company remains responsive but unavailable for ordinary room movement until it arrives, and travel accrues individual survival needs by actual progress without changing GoMud's global clock.

## Decisions

- An optional `TravelProfile` field on `exit.RoomExit` marks an exit as travel-enabled. An empty field preserves current instant movement.
- Profile definitions live in the expedition module, while `RoomExit` only names the applicable profile. Travel sessions, timers, persistence, recovery, and player interaction stay outside GoMud's core exit type.
- A travelling company remains at its origin logically and physically until arrival. It cannot use ordinary movement or interact with either room as though present there.
- `look` renders a travel view while a session is active. It shows route progress and company needs without exposing destination contents or origin-room interactions.
- Travel uses real elapsed UTC time. A disconnect, restart, or copyover does not pause it; recovery derives progress from the persisted start time and completes an overdue session exactly once.
- Profile exertion is accrued proportionally in durable progress checkpoints, never charged as an upfront estimate.
- Hunger, thirst, and fatigue are informational in Phase 5. They do not change travel duration, arrival, movement, combat, health, or any other mechanical outcome. The proving routes must be calibrated not to strand a fully supplied/rested company before Phase 7 camping/recovery exists.
- Travel, survival, camping, and rest must never advance GoMud's shared world clock or round count.

## Scope

### Included

- An optional YAML `travel_profile` field on `exit.RoomExit`.
- Module-owned, data-driven terrain/travel profile definitions.
- A pure expedition session state machine and durable registry keyed by leader user ID.
- Directional-command interception for travel-enabled exits after ordinary exit validation and before movement.
- Real-time progress, status/countdown display, restart/reconnect/copyover recovery, and exactly-once arrival.
- Proportional per-member survival exertion using the Phase 4 service API.
- Travel-aware `look` output and a `travel status` command.
- One or two purpose-built test routes, including the Dunmar West Gate to Fork at the Black Oak example route.
- Unit, module, command, persistence, timer, copyover, and race coverage.

### Excluded

- Travel-speed, health, combat, movement, or completion penalties based on needs.
- Camping, rest, sleep, fatigue recovery, weather, terrain modifiers beyond selected travel-profile data, mounts, cargo, capacity, weight, and automatic provisioning.
- En-route room instances, waypoint interaction, destination preview, player-controlled cancellation, routing across multiple travel legs, diagonal-route work, and player-facing encounter handling.
- Phase 6 interruption behavior. The persisted format reserves its state, but Phase 5 never transitions a live session into it.

## Architecture

`internal/expedition` owns the GoMud-free model: profile validation, session state transitions, UTC elapsed-progress calculation, checkpoint calculation, and idempotent completion/cancellation decisions. It imports `internal/survival` only for the `Exertion` value type; it imports no rooms, users, commands, plugins, timers, or clocks.

`modules/expedition` owns profile loading, durable session storage, scheduling, module-load and copyover recovery, leader/company roster projection, user-facing rendering, and the adapters registered with native commands. It is the only component that schedules a timer or calls `rooms.MoveToRoom`.

`exit.RoomExit` gains only:

```go
TravelProfile string `yaml:"travel_profile,omitempty"`
```

Its presence is declarative world metadata. Existing room files and engine behavior remain compatible because the zero value denotes an ordinary exit.

The native movement layer calls a narrow expedition provider after normal exit lookup, combat/disabled/lock checks, exit-message handling, destination loading, and room script admission checks, but before action-point deduction and `rooms.MoveToRoom`. The provider returns one of: not applicable, travel started, or a handled error. This prevents a partial move, avoids action-point cost for a route that did not start, and leaves unmarked exits untouched.

The native `look` command calls a narrow expedition view provider before ordinary room rendering. It renders a travel view only for an active session; otherwise native `look` behavior remains unchanged.

## Domain Model

The expedition domain defines profile and session values conceptually equivalent to:

```go
type TravelProfile struct {
	Name     string
	Duration time.Duration
	Exertion survival.Exertion
}

type SessionState uint8

const (
	Traveling SessionState = iota
	Interrupted // reserved for Phase 6; not a Phase 5 transition target
	Completed
	Cancelled
)

type TravelSession struct {
	LeaderUserID           int
	OriginRoomID           int
	DestinationRoomID      int
	ExitName               string
	ProfileName            string
	StartedAtUTC           time.Time
	LastExertionCheckpoint uint8
	State                  SessionState
}
```

The durable module record stores all session fields except runtime timer handles. `StartedAtUTC` is an absolute, serialized UTC instant. Progress is derived as `clamp(now - StartedAtUTC, 0, Duration) / Duration`; it is never advanced by a GoMud turn, round, or game-time API.

Profiles require a unique non-empty name, a positive duration, and non-negative exertion totals. Phase 5 rejects a malformed profile rather than applying a guess. The selected profile and every referenced origin/destination room must exist before a session is saved.

The module supports a fixed, documented checkpoint granularity (for example, ten percentage points). At any durable boundary—status, recovery, scheduled completion, or explicit synchronization—it derives the attained checkpoint, invokes `survival.ApplyExertion` for only the incremental proportional cost due since `LastExertionCheckpoint`, persists the updated checkpoint, and announces any threshold crossings according to the survival presentation rules. It never charges the full profile cost at departure. Rounding is deterministic and ensures that a completed route charges exactly its configured total once.

## Commands and Player Experience

For a normal exit, `north` and other normal movement syntax keep their existing instant behavior.

For a travel-enabled exit, the same command performs the usual admission checks and then starts travel. It tells the leader that the company has departed, identifies the route/destination, and reports the duration. The leader and every current charmed company companion stay in the origin while the session is active; no one is placed in the destination early.

While travelling:

- A movement command is handled with a clear refusal and remaining duration/progress rather than attempting ordinary movement.
- `look` shows a travel view with origin, destination, profile, progress, remaining time, and the leader/company survival summary. It does not display the destination's contents or permit origin-room interaction through the view.
- `travel status` presents the same session state explicitly and synchronizes any earned survival checkpoint before rendering.
- Input remains responsive. Phase 5 defines no new en-route actions; commands that require presence in a room must not treat the leader as available in the origin or destination.
- Existing `eat` and `drink` remain usable for manual leader-backpack provisioning of the leader or current companions, subject to their Phase 4 rules.

At completion, the module applies and persists the final earned exertion checkpoint, then persists `Completed` before it calls GoMud's normal `rooms.MoveToRoom` path for the leader. It retains that terminal record until it can verify that the leader is in the destination room, then lets native charm/follow behavior carry current companions through the move, announces arrival once, and removes the record. Recovery of a `Completed` session verifies the leader's room: destination means cleanup only, origin means one retry of the move, and any other location is retained for operator repair rather than guessed. A retry, stale timer, restart recovery, or duplicate callback therefore cannot produce a duplicate successful arrival.

There is no player-facing `travel cancel` command in Phase 5. `Cancelled` exists for internal cleanup and forward-compatible persistence; the first player-visible pause/resume semantics are Phase 6 travel interruptions.

## Persistence, Restart, and Copyover

The expedition module persists a leader-keyed travel-session registry through the established plugin persistence path. It treats persistence unavailability or write failure as a failure to start/advance the relevant transition; it never claims a departure or arrival that was not durably committed.

On ordinary module load, the module validates every saved session and immediately derives its progress from current UTC time. A session still in progress schedules only its remaining delay. An overdue session synchronizes costs and completes once. Invalid records are retained for operator repair or safely marked/cancelled only where the state machine can prove that no arrival occurred; a malformed record must never place a company in a guessed destination.

The module registers a copyover contributor for runtime session/timer continuity. Its durable record remains authoritative: after copyover restoration it derives elapsed progress anew, reschedules a remaining session, or completes an overdue one. Disconnect is not a state transition. A link-dead leader's journey continues and the eventual room placement is visible on their next connection.

Travel state is leader-owned, never keyed by a socket, live mob instance ID, or native in-memory party record. Company composition is projected from the authoritative durable company roster at exertion and arrival boundaries. Phase 5 does not create a second follower system.

## Failure Rules

- A missing/invalid profile, missing destination, unavailable expedition store, or failed initial save leaves the leader and company in the origin with no active session and no exertion charge.
- A second attempted travel start for an already travelling leader does not replace, duplicate, or reset the active session.
- A failed checkpoint write prevents further progression/arrival until recovery can synchronize safely; it never double-charges a previously persisted checkpoint.
- A missing or unavailable survival provider prevents a marked route from starting and prevents a pending session from completing until its due exertion can be applied safely. This preserves the Phase 4 no-loss guarantee.
- If the final normal move cannot succeed, the persisted `Completed` record remains until recovery can verify or retry the destination move; it never guesses a destination or emits a second successful arrival.
- Duplicate timer firings, recovery calls, or copyover callbacks cannot cause duplicate survival cost, repeated room movement, or repeated arrival messaging.
- Travel does not mutate global game time, round count, or any native timer used for the world clock.

## Test World and Content

Phase 5 adds only one or two focused routes, beginning with a travel-enabled link between **Dunmar West Gate** and **Fork at the Black Oak**. Each endpoint retains ordinary room data and can have other instant exits. The link names a single configured terrain/travel profile with a short, test-friendly duration and safe proving-route exertion.

The route metadata is symmetrical only if both directions are intentionally configured. Each directed `RoomExit` is independently authoritative, allowing later terrain/profiles to differ by direction without a schema change.

## Acceptance Criteria

- A YAML-marked exit starts a durable `TravelSession`; every unmarked exit stays instant and preserves native movement behavior.
- A session records stable leader, origin, destination, exit, profile, UTC start, state, and exertion checkpoint; it never stores sockets or live mob IDs.
- Travel blocks ordinary movement but keeps the connection responsive and provides clear `look` and `travel status` views without revealing destination contents.
- A leader and current company companions arrive exactly once after the real profile duration, including after disconnect, module reload, copyover, and server restart.
- Progress and survival exertion are based on real elapsed progress; incremental checkpoints add up deterministically to the configured total at completion.
- Hunger, thirst, and fatigue can display warnings/crossings but do not alter Phase 5 travel duration, arrival, or other gameplay mechanics.
- Failed validation, persistence, survival synchronization, and duplicate completion paths leave state consistent without premature movement, double cost, or duplicate arrival.
- No code path mutates GoMud's global game time or round count for travel.
- Focused unit/module/command/persistence/timer/copyover race tests, `make generate`, `make validate`, and `go test -race ./...` pass; any environmental blocker is documented rather than claimed as passing.

## Deferred Decisions

- Phase 6 owns encounter-triggered interruption, pause/resume, combat handoff, and remaining-duration/cost correctness under an interruption.
- Phase 7 owns camping, rest/sleep, real-time fatigue recovery, and the first complete fatigue consequence/recovery loop.
- Phase 8 owns weather's effect on routes. Phase 9 owns cargo, capacity, weight, and automatic provisioning. Phase 10 owns mounts. None may alter the Phase 5 session identity or its exactly-once transition guarantees.
- Future designs may add player-controlled cancellation, waypoints, multi-leg routes, en-route actions, and travel-speed modifiers only with explicit recovery and persistence rules.
