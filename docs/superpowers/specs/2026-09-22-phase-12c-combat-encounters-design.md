# Phase 12c: Combat Encounters During Travel

## Prior-art check

12a/12b (`docs/superpowers/specs/2026-09-22-phase-12{a,b}-*-design.md`) built
the encounter-kind abstraction and weighted rolling on top of Phase 6's
travel-interruption mechanism (`internal/expedition.InterruptionKind`/
`InterruptionProfile`, fire-once-at-a-checkpoint, resolved via `travel
resume`/`travel return`). Both explicitly deferred every subsystem-backed
encounter type (combat, merchant, social, etc.) named in the Phase 12
overview (`docs/ASHVEIL_GOMUD_AGENT_HANDOFF.md` section 37). Per the user's
direction, this pass builds **only** combat encounters — the other types
(merchants, injured NPCs, route choices, camp opportunities, ruined sites,
resources, social encounters) stay future/deferred ideas, not committed
scope.

Two load-bearing facts checked against the actual code before designing
this:

1. **A traveling company never physically changes rooms until arrival.**
   `modules/expedition.go`'s `mover.MoveToRoom` is only called from
   `completeLocked` (line ~766, on arrival); nothing moves the leader at
   departure or mid-route. The leader (and any companions, confirmed via
   `commandCompanionsToFollow`'s `origin.GetMobs(rooms.FindCharmed)`) sit in
   `session.OriginRoomID` for the entire travel duration. This means a
   combat encounter needs no new "encounter room" concept — a hostile mob
   spawned into `OriginRoomID` while the party is there is a normal room
   encounter to the rest of the engine.
2. **Spawning a mob that fights back needs no new combat-initiation code.**
   `internal/mobcommands/lookfortrouble.go` already makes any mob with
   `Hostile: true` attack a present player — but only when a mob's own idle
   commands invoke it, which isn't guaranteed for a freshly spawned instance
   this pass creates outside the normal room `SpawnInfo` config. Rather than
   depend on that, this pass has the spawn itself issue `mob.Command("attack
   @<leaderUserID>")` immediately (the same `mob.Command(...)` dispatch
   pattern already used by the charmed-mob-assist loops in
   `internal/usercommands/attack.go` and `internal/hooks/NewRound_DoCombat.go`)
   — deterministic, no dependency on idle-tick timing. Once Aggro is set,
   the entire existing round-based combat loop (including all of Phase 11's
   formation-combat gating, if the leader has a company) runs unchanged.

## Scope

**In scope:**
- A new `Combat` `InterruptionKind`, usable as a profile's singular `Kind`
  or as one entry in a 12b weighted `Kinds` table.
- `InterruptionProfile.CombatMobID` — the mob template a route's combat
  encounter spawns, required whenever `Combat` is a reachable kind for that
  profile.
- On firing: spawn one hostile mob instance from that template into the
  travel session's origin room, immediately command it to attack the
  leader, and record its instance id in the persisted `TravelInterruption`
  so `travel resume`/`travel return` can check on it later (including after
  a restart).
- Resolution stays exactly `travel resume`/`travel return`, same as every
  other interruption kind — except both now refuse while the spawned
  encounter mob is still alive and present ("you can't do that while
  you're fighting").

**Explicitly deferred / future ideas, not this pass:**
- Multi-mob or formation-shaped encounters (reusing 11a's `mobparty.Groups`
  tag for a whole ambush party instead of one mob) — a natural follow-up
  once a single mob proves the wiring, not needed to satisfy "combat
  encounters during travel."
- Every other Phase 12 encounter type named in the overview doc (merchants,
  injured NPCs, route choices, camp opportunities, ruined sites, resources,
  social encounters) — explicitly out of scope per this session's direction,
  kept only as named future ideas in `docs/PROJECT_STATUS.md`, not planned
  work.
- Terrain/danger-aware encounter selection (e.g. picking a mob template by
  zone or route difficulty automatically) — `CombatMobID` is authored
  per-route by hand, same granularity 12a/12b already established for
  interruption content.
- Fleeing an active encounter via a room exit rather than through
  `travel return` — out of scope; existing flee/exit commands are unrelated
  engine behavior this pass doesn't touch either way.
- Shipping new route config that actually uses `Combat` — same deferral
  12a and 12b both made for their own new kinds: `oak-road` stays
  `fallen-tree` until there's a real reason to author new route content.
  This pass proves the mechanism through unit and module-level tests.

## Durable model

`internal/expedition` additions:

```go
const Combat InterruptionKind = "combat"
```

`InterruptionText` gains a `Combat` case (generic ambush framing — the
mob's own combat messages announce specifics once fighting starts, so this
doesn't need to name the mob).

```go
type InterruptionProfile struct {
    Kind         InterruptionKind
    Kinds        []WeightedInterruptionKind
    Checkpoint   uint8
    CombatMobID  int `yaml:"combat_mob_id,omitempty"`
}
```

`Validate()` additionally requires `CombatMobID > 0` whenever `Combat` is a
reachable kind for the profile (`Kind == Combat`, or any `Kinds` entry has
`Kind == Combat`) — the domain package can't check whether that id actually
resolves to a mob spec (it imports no `internal/mobs`), so this is a
structural check only; an unresolvable template id is handled by the engine
layer's spawn call failing open (below).

```go
type TravelInterruption struct {
    Kind                InterruptionKind
    Checkpoint          uint8
    CombatMobInstanceId int `yaml:"combat_mob_instance_id,omitempty"`
}
```

`CombatMobInstanceId` is engine-set (never by the pure `Interrupt` function,
which knows nothing about mob spawning) after a successful spawn, and left
`0` for every non-`Combat` kind and for a `Combat` firing whose spawn
failed.

## Integration

`modules/expedition` gains a small seam, mirroring the existing
`Mover`/`Survival`/`Scheduler` pattern:

```go
type MobSpawner interface {
    // SpawnHostileEncounter creates mobTemplateID in roomID, immediately
    // commands it to attack leaderUserID, and returns its instance id.
    SpawnHostileEncounter(roomID, mobTemplateID, leaderUserID int) (int, error)
    // EncounterActive reports whether instanceID is still alive and still
    // in roomID. False (including "no such instance," e.g. after a
    // restart that didn't persist it) means the encounter is resolved.
    EncounterActive(instanceID, roomID int) bool
}
```

`nativeMobSpawner` implements it via `mobs.NewMobById` + `room.AddMob` +
`mob.Command(...)` (spawn) and `mobs.GetInstance` (liveness check) — the
same primitives `modules/company/runtime.go`'s `Spawn`/`IsLive` already use
for a different purpose, confirming this is the established pattern for
"a module spawns and tracks its own mob instance."

`interruptLocked` (unchanged for every non-`Combat` kind): after
`session.Interrupt(now, resolvedProfile)` builds `candidate`, if
`candidate.Interruption.Kind == expedition.Combat`, call
`m.mobSpawner.SpawnHostileEncounter(candidate.OriginRoomID,
resolvedProfile.Interruption.CombatMobID, candidate.LeaderUserID)`. On
success, set `candidate.Interruption.CombatMobInstanceId` before persisting.
On failure (bad template id, room unavailable), log a warning and leave
`CombatMobInstanceId` at `0` — **the interruption still fires as a pause**
(fails open to "an uneventful pause," never to "travel silently
un-interrupts itself" or a stuck/broken session).

`resume`/`returnToOrigin`: immediately after confirming
`session.State == Interrupted`, when `session.Interruption.Kind ==
expedition.Combat && session.Interruption.CombatMobInstanceId != 0`, check
`m.mobSpawner.EncounterActive(...)`; if true, refuse with a "you're still
fighting" message and leave the session untouched (same shape as every
other refusal branch already in those two functions). This is the only
change to the resume/return functions; every other interruption kind is
unaffected.

## Constraints and deferrals

- Never advances GoMud's global clock/round count — untouched; the spawn
  and the resume/return gate are ordinary room/mob operations, not travel
  timing.
- Must survive restart/copyover: `CombatMobInstanceId` is a plain
  `omitempty` int field on the already-persisted `TravelInterruption`, so
  it round-trips like every other field. Dynamically spawned mob instances
  (no `SpawnInfo` entry) do **not** themselves survive a restart in this
  engine (confirmed against `internal/rooms/rooms.go`'s spawn/respawn
  model, which only restores mobs from static `SpawnInfo`) — so after a
  restart, `EncounterActive` correctly returns false (`mobs.GetInstance`
  finds nothing) and `travel resume`/`travel return` work again
  immediately. This is the same fail-open shape 11b's reassignment and
  12a/12b's fallback text already use: a restart never leaves a player
  stuck.
- No new session states, commands, or exit points — still exactly
  `Traveling`/`Interrupted`/`Completed`/`Cancelled` and `travel
  resume`/`travel return`.

## Acceptance criteria

- `InterruptionKind.Valid()` accepts `combat`; `InterruptionText` returns
  distinct, non-empty text for it.
- `InterruptionProfile.Validate()` rejects `Kind: Combat` (or a `Kinds`
  table containing `Combat`) with `CombatMobID <= 0`, and accepts it with a
  positive `CombatMobID`.
- Firing a `Combat` interruption calls the injected `MobSpawner` with the
  right room/template/leader, and persists the returned instance id onto
  `session.Interruption.CombatMobInstanceId`.
- A spawn failure still transitions the session to `Interrupted` (fails
  open to a plain pause) rather than aborting the interruption.
- `travel resume` and `travel return` both refuse while
  `EncounterActive` reports true for the tracked instance, and both work
  normally once it reports false.
- `go test -race ./...`, `make generate`, `make validate` all pass.
