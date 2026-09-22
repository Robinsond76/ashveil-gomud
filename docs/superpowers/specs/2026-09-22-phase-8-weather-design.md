# Phase 8 Weather Design

**Status:** Draft — pending owner review before a plan is written.

## Goal

Add a shared, server-driven, zone-scoped weather system: each zone cycles
through a small set of biome-appropriate conditions on the shared round
clock, persists durably, is queryable by other systems, and colors room
descriptions. No per-player weather, no player-visible round-skipping, no
mutation of global game time.

## Prior-art check (handoff requirement: inspect before building)

GoMud core has no weather engine. What exists:

- `Room.Biome` / `ZoneConfig.DefaultBiome` — a cosmetic string (`forest`,
  `swamp`, ...) used only for map symbol/color and light/dark area rules
  (`internal/rooms/biomes.go`). No gameplay effect.
- `rooms.GetAllZoneNames()`, `rooms.GetZoneBiome(zone)`,
  `rooms.GetZoneForRoom(roomId)` — native, already sufficient to enumerate
  zones and resolve a room to its zone/biome. No new core room/zone fields
  needed.
- `internal/mutators` + `ZoneConfig.Mutators`, ticked once per round by
  `internal/hooks/NewRound_UpdateZoneMutators.go` — a general
  decorate-a-room-or-zone-with-text/buffs/exits system with its own decay
  model. It's the closest existing pattern (round-driven, zone-scoped,
  `events.NewRound`-ticked) but it's aimed at ad hoc room dressing
  (`dusty`, a temporary breach), not a cycling shared condition with
  gameplay multipliers. Weather should follow its *shape* (a
  `NewRound`-ticked, zone-keyed, round-number state machine) without
  reusing the mutator type itself, matching `docs/ASHVEIL_GOMUD_AGENT_HANDOFF.md`
  §33's instruction to build it as an Ashveil module/domain service.

**Conclusion:** no complete engine to integrate — build `internal/weather` +
`modules/weather` as a new vertical slice, following the same
domain/module split as expedition and camping.

## Scope

Included:

- One current `Condition` per zone (biome-tabled, weighted-random,
  round-scheduled transitions).
- A `weather` command and a weather line prepended to `look` for zones whose
  biome has a configured table.
- A read-only query seam other modules can consult.
- Persistence across restart/copyover, driven only by the shared round
  counter (never a per-player or real-time clock).

Excluded (deferred; see Constraints and Deferrals): storms as discrete
"hazard" encounters, weather-driven room/exit changes, player-visible
forecasts, seasons, climate/latitude modeling, mount/encumbrance
interaction (Phases 9–10), and any change to combat, health, or room
contents.

## Durable Model

`internal/weather` is GoMud-free, matching `internal/expedition` and
`internal/camping`.

```go
type Condition struct {
    Name                string // e.g. "clear", "rain", "storm"
    Description         string // appended to room look / weather command
    TravelDurationPct    int   // 100 = unchanged; multiplies travel duration
    ExertionPct          int   // 100 = unchanged; multiplies survival exertion cost
    RestRecoveryPct      int   // 100 = unchanged; multiplies camp rest fatigue recovery
}

type ZoneWeather struct {
    Zone            string
    Current         string // Condition.Name, validated against the zone's table at apply time
    NextChangeRound uint64
}
```

- `Due(currentRound) bool`, `Advance(currentRound uint64, next Condition, nextChangeRound uint64) (ZoneWeather, error)` mirror `camping.RestDue`/`CompleteRest`'s shape: pure, validated, copy-returning.
- Percentages are clamped to a sane validated range (e.g. 25–300) at config parse time; a malformed condition is rejected (logged, dropped) rather than guessed, same as `expedition.parseProfiles`.
- Round numbers only — no `time.Duration` anywhere in this package. This is the one durable-timing package in the codebase that is intentionally *not* real-UTC-based, because weather must track the shared world clock, not wall time (handoff §33: "should not be based on a per-player clock").

## Module

`modules/weather` owns:

- A **biome table** config (embedded default + on-disk overlay, same
  pattern as `modules/expedition/files/data-overlays/config.yaml`):
  weighted `Condition` list plus a change-interval round range (min/max)
  per biome ID. A zone whose `rooms.GetZoneBiome(zone)` has no configured
  table is simply not tracked (no weather line, no multiplier) rather than
  falling back to a guessed table.
- A **durable, zone-keyed registry** (`map[string]weather.ZoneWeather`),
  YAML-persisted like every other module registry in this codebase.
- A single `events.RegisterListener(events.NewRound{}, m.onNewRound)`
  hook: for every tracked zone whose weather `Due(round)`, roll a new
  condition (weighted by the biome table) and a new `NextChangeRound`
  (`round + randInRange(min,max)`), then persist. This mirrors
  `NewRound_UpdateZoneMutators.go`'s shape but lives in the module, not in
  `internal/hooks` (modules don't touch `internal/hooks`).
- **Load/copyover recovery**: on `SetOnLoad`, for every tracked zone with no
  persisted record, roll an initial condition; for a zone whose
  `NextChangeRound` is already `<=` the current round (server was down
  across a scheduled change), roll forward exactly once to the current
  round — never fast-forward through multiple missed transitions'
  side effects, since weather has none beyond the description/multiplier
  (unlike travel/rest there is nothing to "complete" or double-charge, so
  this is simpler than the expedition/camping recovery problem).
- A **read-only query seam** in `internal/weather` (same shape as
  `survival.CompanyService` / `camping.ViewProvider`):
  ```go
  type Provider interface {
      CurrentCondition(zone string) (Condition, bool)
  }
  func SetProvider(p Provider)
  func CurrentCondition(zone string) (Condition, bool)
  ```
- A `weather` player command (`weather` / `weather here`) showing the
  current condition for the player's zone, and `RenderWeatherLine(zone)`
  consulted by `look` to prepend the condition's `Description` to room
  rendering when the zone is tracked (purely additive — never replaces the
  room view the way camping's rest view does).

## Integration with existing systems (open decision — see below)

The handoff wants weather to affect travel duration, fatigue/thirst, and
camp quality, not just render text. Two ways to wire that in:

**Option A — read-only, no schema changes (recommended for this phase).**
`modules/expedition` and `modules/camping` are left untouched. Weather
exposes `CurrentCondition(zone)`; nothing calls it yet except `look` and the
`weather` command. Multiplier wiring becomes a small, separate fast-follow
change once the engine itself is proven, so this phase stays a pure
additive vertical slice (like Phase 5's route/exertion review found value
in landing the domain before the integration).

**Option B — wire multipliers now.** `expedition.StartTravel` would look up
the origin zone's condition once at journey start and need to store an
effective duration/exertion *on the session* (today `TravelSession` only
stores `ProfileName` and re-resolves the static profile by name every
time, so a per-journey multiplier needs a new persisted field, e.g.
`DurationPct`/`ExertionPct` on `TravelSession`, applied wherever
`profile.Duration`/`profile.Exertion` are currently read). `modules/camping`
is lower-risk: `startRest` already passes a literal `camping.FatigueRecovery`
constant into `ApplyCompanyRestRecovery`; multiplying that one int by the
camp's zone condition at rest-start needs no schema change at all.

**Recommendation:** do Option A for camp rest (free, no schema risk) and
Option A-then-B for travel — ship the read-only engine first, and only
touch `TravelSession`'s persisted shape in a reviewed follow-up once the
owner confirms the exact multiplier semantics (does weather change
mid-journey re-roll the effective duration, or lock at departure? locking
at departure is simpler and matches "duration is fixed once travel
starts", but needs an explicit decision since it changes durable schema on
an already-shipped, tested phase).

## Constraints and Deferrals

- Never advances `gametime`, the round counter, or moves any player;
  reacts to rounds, never drives them.
- Never mutates combat, health, room contents, exits, or items.
- A zone with no configured biome table is untracked, not defaulted.
- Weather transitions are round-scheduled and persisted; never derived from
  wall-clock/real time, and never guessed on a missing/corrupt record — an
  invalid persisted `ZoneWeather` is retained and logged, not deleted.
- Deferred: weather hazards/events as encounters (Phase 12 territory),
  seasons/climate, forecasts, mount/encumbrance interaction (Phases 9–10),
  indoor/outdoor room-level overrides beyond "zone has a tracked biome",
  and (per the recommendation above) actual travel-duration/exertion
  multiplier wiring into `modules/expedition`.

## Acceptance Criteria (for this phase's slice)

- Every zone whose biome has a configured table gets an initial condition
  on first load and transitions on schedule, driven only by
  `events.NewRound`.
- `weather` and `look` show the current condition's description for a
  tracked zone; an untracked zone shows neither.
- Restart/copyover never fast-forwards through more than one missed
  transition and never resets a zone to a default condition it didn't
  roll.
- A malformed biome table entry or a corrupt persisted zone record is
  rejected/retained and logged, never guessed.
- Focused domain/module/race tests, `make generate`, `make validate`, and
  `go test -race ./...` provide verification, consistent with every prior
  phase's use of the fake clock/scheduler/store harness (round number
  stands in for the "clock" here — tests inject the round number directly
  rather than a `time.Time`).

## Open questions for the owner

1. Confirm Option A scope (engine + descriptions only) for this phase, with
   travel/camp multiplier wiring as an explicit follow-up — or ask for
   Option B (schema change to `TravelSession`) now.
2. Confirm the biome→condition tables to ship for the proving content
   (forest is the only biome currently in use, at Dunmar 2001/2002).
3. Confirm change-interval cadence (suggested default: 40–120 rounds per
   zone, i.e. minutes of real time at typical round-second configs) is
   reasonable, or specify a preferred range.
