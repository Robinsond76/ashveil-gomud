# Phase 16: Walking Fatigue, Inns, and Travel/Rest Modifiers

Part of the roadmap in
`docs/superpowers/specs/2026-09-23-environment-skills-economy-roadmap.md`
(source request: "Walking causes fatigue; rest at an inn (bonus) or a camp
to recover"). This phase also switches on the travel and rest multipliers
that Phases 8, 9, and 10 computed but left unwired (their "Option A"
deferrals). Phase 15 moved that work here.

**Status:** Design written 2026-09-23, not implemented. The open decisions
below carry recommendations and still need the user's confirmation. See
"Open decisions".

## Prior-art check

- **Travel (Phases 5–6, 12).** `expedition.TravelSession` stores only
  `ProfileName`, and every read resolves the static `TravelProfile` again
  by name: `profile.Duration` drives progress, checkpoints, timers, and
  interruption timing, and `profile.Exertion` drives the proportional,
  ledgered survival charge (`PrepareExertion` → `ApplyCompanyExertion`,
  with operation IDs built from session identity). A per-journey
  multiplier therefore needs a new persisted field. All three earlier
  designs said so.
- **Weather (Phase 8).** `weather.Condition` already carries
  `TravelDurationPct`, `ExertionPct`, and `RestRecoveryPct` (25–300,
  validated). Nothing reads them. The `weather.CurrentCondition(zone)`
  provider seam exists. Only zones whose biome has a table are tracked
  (forest/Dunmar today).
- **Encumbrance (Phase 9).** `encumbrance.LoadBand{MinRatio,
  TravelDurationPct, FatiguePct}` bands are resolved by
  `encumbrance.ResolveBand` from `modules/encumbrance` config. The
  `encumbrance.CurrentLoad(leader)` seam returns the load but not the
  band, so callers can't read the band's modifiers.
- **Mounts (Phase 10).** `MountSpec.TravelDurationPct` (pack-horse: 90) is
  config data that nothing reads. `mount.Provider` exposes only
  `CapacityBonusGrams`.
- **Camp rest (Phase 7).** A camp gets exactly one 60-second rest.
  Completion applies `camping.FatigueRecovery` (a constant 20) through
  `ApplyCompanyRestRecovery`. That call is idempotent by operation ID and
  **rejects a replay whose amount differs** (`ErrRestConflict`). Any
  weather-scaled amount must therefore be fixed when the rest starts and
  persisted with it, or a weather change between completion and a
  post-crash retry would conflict.
- **Survival (Phase 4, 15).** Needs run 0–100, where 100 means rested.
  Phase 15 added `ApplyMemberDrain`, a non-ledgered call that marks
  survival dirty for the next save and is guarded by a leaf mutex. No
  survival band has a gameplay effect yet, so fatigue is invisible except
  in status output.
- **Native movement.** `usercommands.Go` spends action points, checks
  locks and scripts, then calls `rooms.MoveToRoom`. Party members and
  charmed mobs (company companions) follow by re-running the command
  themselves. `events.RoomChange` fires for *every* relocation: teleport,
  recall, death, travel completion, and instance moves. Charging walking
  fatigue on it would double-charge journeys and charge teleports, so the
  hook belongs in `Go` right after a successful `MoveToRoom`.
- **Upstream inn.** Frostfang's Frostfire Inn (room 61) sells item 102
  *room rental* (5 gold) from innkeeper mob 7. Buying it moves the player
  to an instanced copy of room 432, whose `onEnter` applies buff 15
  *Sleeping* (50 rounds of HP heal). When that ends it grants buff 16
  *Well Rested* (+1 stats, +5% XP, 75 rounds). None of this touches
  Ashveil survival fatigue.
- **Exposure (Phase 15)** is the proven pattern for a per-member,
  round-ticked band buff. Its lessons carry over: penalty buffs never
  touch vitality; buffs are short and refreshed each tick so permabuff
  reconciliation can't strip them; buffs are applied only on the game
  loop. It also exposes the cold/heat exposure band per member.

## Scope

1. **Travel modifiers.** Weather, load, and mount scale a journey's
   duration and exertion. The multipliers are fixed at departure and
   persisted on the session.
2. **Camp rest modifier.** Weather `RestRecoveryPct` scales camp rest
   recovery. It is fixed at rest start and persisted.
3. **Walking fatigue.** Each ordinary step costs the walker's whole
   company some fatigue, scaled by terrain, weather, load, mount, cold, and
   Well Rested.
4. **Fatigue effects.** Penalty buffs for low fatigue, so walking and
   resting actually matter.
5. **Inns.** Paid, real-time rest in `inn`-tagged rooms, built on camping.
   It recovers more than a camp and grants *Well Rested*.

Out of scope: hunger/thirst from walking, mount fatigue, passive recovery
while idle, sleep-until-dawn, inn prices by settlement standing (Phase 21),
inn meals, and new route content.

## Model

All numbers are module config, shown here as defaults.

### 1. Travel modifiers (fixed at departure)

At `StartTravel`, the module resolves three sources. A source with no
provider, no tracked zone, or no mount counts as 100 (unchanged). Nothing
is guessed.

| Source | Duration | Exertion (all needs) | Fatigue only |
|---|---|---|---|
| Weather in the origin room's zone | `TravelDurationPct` | `ExertionPct` | — |
| Company load band | `TravelDurationPct` | — | `FatiguePct` |
| Leader's mount | `TravelDurationPct` | — | `FatiguePct` (new) |

- `DurationPct` = weather × load × mount / 10⁴, clamped to 25–300.
- `ExertionPct` = weather, 25–300.
- `FatiguePct` = load × mount / 10², clamped to 25–300.
- The effective profile is a pure function,
  `profile.WithModifiers(mods)`:
  - `Duration × DurationPct / 100`
  - hunger and thirst `× ExertionPct / 100`
  - fatigue `× ExertionPct × FatiguePct / 10⁴`

  All values round half-up, and a positive base never rounds to 0.
- The session persists `Modifiers *TravelModifiers` (`duration_pct`,
  `exertion_pct`, `fatigue_pct`). **nil means 100/100/100**, so sessions
  persisted before this phase keep their exact behaviour after upgrade.
- Every place in `modules/expedition` that resolves a profile for a
  session goes through one helper, `sessionProfileLocked(session)`, which
  returns `profile.WithModifiers(session.Modifiers)`. Progress, checkpoints,
  exertion, interruption timing, timers, status, and recovery then agree.
  Interruption checkpoints are fractions of the route, so they scale for
  free. The exertion operation ID doesn't change, and each checkpoint's
  cost is derived from the fixed modifiers, so replays match.
- The departure and status texts name the modifiers, for example "Rain and
  a heavy load slow the going (+37%)." The weather, encumbrance, and mount
  status texts stop saying their modifiers are "not yet applied".

### 2. Camp rest modifier (fixed at rest start)

`camping.RestSession` gains `Recovery int` (`recovery`). `startRest` sets
it to `FatigueRecovery × RestRecoveryPct / 100` (at least 1) for the weather
in the camp room's zone, or 100 if the zone is untracked. Completion and
every retry pass `Recovery` to `ApplyCompanyRestRecovery`. A
**zero `Recovery` means the legacy `FatigueRecovery`**, so a rest persisted
mid-flight before upgrade finishes with 20.

### 3. Walking fatigue (per step)

A pure `internal/fatigue` package computes the cost of one step. The new
`modules/fatigue` charges it.

- **Hook:** after `rooms.MoveToRoom` succeeds in `usercommands.Go`, a
  seam call `fatigue.StepTaken(userID, fromRoomID, toRoomID)` fires, with
  the same shape as `expedition.MovementBlocked`. It is a no-op without a
  provider. It does not fire for failed moves, profiled travel exits
  (travel charges its own exertion), flee, teleport, recall, or other
  relocations.
- **Who pays:** the walker's company, with each member charged
  separately:
  - the player always
  - each live, spawned roster companion that was in the origin room, since
    it follows the step

  Companions left elsewhere or not spawned walk nowhere and pay nothing.
  A charmed companion following by its own `go` is a mob, not a user, so
  it isn't charged twice. A party member following the party leader walks
  with their own `go` and pays for their own company.
- **Cost per member,** in hundredths of a fatigue point:

  `StepFatigueCenti (20) × terrain × weather × load × mount × cold × rested`

  - **terrain:** the destination room's percentage from a biome table.
    A room tagged `indoor` is 0.

    | Biome | % | Biome | % |
    |---|---|---|---|
    | city, house | 0 | shore | 110 |
    | slums, fort | 25 | forest, cave | 125 |
    | road | 50 | desert, spiderweb | 150 |
    | default (no biome) | 50 | snow, swamp | 175 |
    | land, dungeon | 100 | mountains, cliffs | 200 |

    Towns are free on purpose: "rooms represent interesting places;
    distance represents travel."
  - **weather:** the destination zone's `ExertionPct`, outdoors only.
  - **load:** the company load band's `FatiguePct`.
  - **mount:** the leader's mount `FatiguePct`.
  - **cold:** the member's Phase 15 cold band: chilled 110, frostbitten
    125, hypothermic or worse 150. Heat is ignored, because it already
    drains thirst.
  - **rested:** `WellRestedPct` (50) if the member has the `well-rested`
    buff flag.
  - The product is clamped to 0–400%.
- **Accumulator:** each member keeps a remainder in hundredths. Whole
  points go out through `ApplyMemberDrain(Fatigue: n)`, which is
  non-ledgered: a lost step after a crash is harmless. The remainders are
  persisted in the module's plugin-store registry. It is marked dirty on
  each step and flushed on the next save, like Phase 15's drains.
  Members are pruned against the roster.
- **Scale check:** 100 road steps cost 10 fatigue; 100 forest steps cost
  25; 100 mountain steps cost 40. A long wilderness walk tires the party
  out. A day in town does not.
- **Never blocks movement.** Fatigue 0 still lets you walk (the Spent
  buff is the penalty). Nobody gets trapped.

### 4. Fatigue effects (band buffs)

Every `TickRounds` (5), `modules/fatigue` syncs one penalty buff per
member (the online leader and its live, spawned companions), following
Phase 15's exposure pattern. The buffs are short and non-permanent,
refreshed each tick, and applied only on the game loop:

| Fatigue | Buff | Effect |
|---|---|---|
| 51–100 | — | none |
| 26–50 (Low) | 1040 Weary | −5 speed |
| 1–25 (Critical) | 1041 Exhausted | −10 speed, −10 strength, −5 perception |
| 0 (Depleted) | 1042 Spent | −20 speed, −20 strength, −10 smarts, −10 perception |

The buffs never touch vitality or max HP, and never kill. The shipped buff
files are tested with real config (Phase 15 finding 1).

### 5. Inns

Inns are built into `modules/camping`, which already owns real-time rest,
its timers, the recovery protocol, the movement block, and the rest view.

- **Where:** rooms tagged `inn` (`InnRoomTag`). The proving room is
  Frostfang's Frostfire Inn (room 61).
- **Commands:**
  - `inn` or `inn status` shows the price and your stay state.
  - `inn rest` pays and starts a stay.
- **Price:** `InnPrice` of 10 gold, from `Character.Gold`.
- **Stay:**
  - It lasts `InnRestDuration` (60 s) and recovers `InnRecovery` (50) for
    the whole company. It happens indoors, so weather doesn't apply.
  - Resting blocks movement and shows a rest view on `look`, like camp
    rest.
  - Unlike a camp, you can take another stay as soon as one finishes.
- **Refused when:**
  - the room isn't an inn
  - you are in combat or downed
  - you are travelling
  - your camp is mid-rest
  - you are already staying
  - you can't pay
- **Durable model:** the camping registry gains
  `Stays map[int]camping.InnStay`, keyed by leader:

  ```go
  type InnStay struct {
      LeaderUserID    int
      RoomID          int
      PricePaid       int
      Rest            RestSession // StartedAtUTC, State, Recovery
      RecoveryApplied bool
  }
  ```

  Lifecycle: Resting → Completed (recovery applied) → Well Rested granted
  → record deleted. Stay timers are separate from camp timers, with their
  own generation map, so a stale timer can't complete the wrong record.
- **Recovery protocol:** the same two-phase protocol as camp rest.
  1. Persist Completed.
  2. Apply `ApplyCompanyRestRecovery` with the operation ID
     `inn-rest-<leader>-<room>-<startNanos>`.
  3. Persist `RecoveryApplied`.

  A crash at any step retries without recovering twice. An overdue stay
  after restart or copyover completes once, based on the UTC time that has
  passed.
- **Payment:**
  1. Deduct gold on the game loop (the command).
  2. Persist the stay.
  3. If the save fails, refund the gold.

  Crash window: the user record saves on its own schedule. If the server
  dies after the stay is saved but before the user is, the stay could be
  free. This is accepted and documented, just as upstream shops accept it.
- ***Well Rested*** (buff 1030, shipped by `modules/camping`): flag
  `well-rested`, +5% XP, +1 speed and +1 perception, for 450 rounds
  (about 30 minutes).
  - Its main effect is halving walking fatigue (§3).
  - **It is granted on the game loop, not in the timer:** a `NewRound`
    listener in `modules/camping` grants it to an online leader and its
    live companions for any completed, recovered stay, then deletes the
    record. Camping's timer callbacks run off the game loop, and they must
    never mutate a `Character`.
  - An offline leader gets it on their first round back online.
- **Upstream room rental** (item 102 → room 432 → buffs 15/16) is left in
  place. It's content, and it doesn't touch Ashveil survival. A builder can
  retire it later, like Phase 15's Freezing Snow buff. This is noted in the
  status log.

## Durable model summary

| Record | Change | Upgrade safety |
|---|---|---|
| `expedition.TravelSession` | `+ Modifiers *TravelModifiers` | nil = 100/100/100 |
| `camping.RestSession` | `+ Recovery int` | 0 = legacy 20 |
| camping registry | `+ Stays map[int]InnStay` | absent = empty |
| `modules/fatigue` registry | new, leader → member → remainder (centi) | new file |
| `mount.MountSpec` config | `+ FatiguePct` | 0 = 100 |

The design adds no timers beyond camping's existing real-UTC rest timer
pattern. Ticks are `events.NewRound` listeners that read the clock and never
advance it. Every new record validates strictly. An invalid persisted
record is kept and logged, never guessed, the same as every earlier
registry.

## Packages

- `internal/expedition`: `TravelModifiers` with `Validate`, and
  `TravelProfile.WithModifiers`. `TravelSession.Validate` checks
  `Modifiers`.
- `modules/expedition`:
  - resolves modifiers at `StartTravel`, *before* taking `m.mu`, so no
    provider lock is held under the expedition lock
  - the `sessionProfileLocked` helper
  - the departure and status text
- `internal/encumbrance`: `Provider` gains
  `CurrentModifiers(leader) (LoadBand, bool)`, or a package helper that
  resolves the band. `modules/encumbrance` implements it and updates its
  status text.
- `internal/mount` / `modules/mount`: `MountSpec.FatiguePct`, and a
  `Provider.Modifiers(leader) (durationPct, fatiguePct int)` query.
- `internal/camping`: `RestSession.Recovery`, `InnStay` and its
  transitions (validation, start, due, complete), and `RecoveryFor(rest)`.
- `modules/camping`:
  - weather-scaled camp recovery
  - the inn registry, commands, stay timers, recovery, the movement
    block, and the view
  - the Well Rested `NewRound` grant
  - buff 1030
  - config keys `InnRoomTag`, `InnPrice`, `InnRecovery`,
    `InnRestDuration`, `InnBuffID`
- `internal/fatigue` (new, pure): step cost from its inputs (terrain,
  weather, load, mount, cold band, rested), accumulator arithmetic, the
  band → buff mapping, and the `StepTaken` seam.
- `modules/fatigue` (new): config, the remainder registry and persistence,
  the step charge, the band-buff tick, and buffs 1040–1042.
  `make generate` wires it in.
- `internal/usercommands/go.go`: one `fatigue.StepTaken` call after a
  successful `MoveToRoom`.
- `internal/climate` (or `modules/exposure`): a read-only
  `ColdBandFor(leader, key)` query seam, so fatigue doesn't import the
  exposure module.
- Data: the `inn` tag on room 61, and mount `FatiguePct` config.

## Constraints and invariants

- **Never advance the world clock or round count.** Travel and rest stay
  real-time UTC sessions. Ticks only read the clock.
- **Survive restart and copyover.** Every new field and registry is
  persisted, and the upgrade defaults are listed above. Two cases get
  explicit tests: an overdue inn stay completes once after reload, and a
  pre-upgrade travel session or rest is unchanged.
- **Concurrency.**
  - Survival stays a leaf lock.
  - The expedition, camping, and fatigue module mutexes each call into
    survival only while holding their own lock. None calls another of
    these modules while holding its lock.
  - Provider reads (weather, encumbrance, mount, exposure) happen outside
    the expedition lock at `StartTravel`, and on the game loop for steps.
  - `Character` mutation (buffs, gold) happens only on the game loop:
    commands and `NewRound` listeners. It never happens in a timer
    callback.
  - Camping's lit-fire snapshot lock is unchanged.
- Data-driven balance: every number above is module config.
- Travel's existing guarantees still hold: telescoping exertion,
  idempotent checkpoints, the interruption one-shot, and combat encounters.

## Open decisions (recommendations, awaiting confirmation)

1. **Fix travel modifiers at departure** rather than re-reading them
   mid-journey. This is simpler, deterministic, and replay-safe. Phase 8's
   design already leaned this way.
2. **Which sources:** weather scales duration and all exertion; load and
   mount scale duration and fatigue. (Handoff §7.3: fatigue × encumbrance
   × weather × mount.)
3. **Mounts gain `FatiguePct`,** pack-horse 90, as well as wiring the
   existing `TravelDurationPct`. This delivers the roadmap's "mounts
   reduce". The alternative is load reduction only.
4. **Walking fatigue covers ordinary `go` steps only, for the whole
   company, and is free in towns and indoors.** Base 0.20 per step at
   100%, with the terrain table above.
5. **Fatigue gets penalty buffs** (Weary, Exhausted, Spent), which are
   never lethal and never block movement. Without them, walking fatigue and
   inns have no gameplay effect. The alternative is to defer effects to a
   later phase.
6. **Inns live in `modules/camping`** under an `inn` command: 10 gold,
   +50 fatigue, 60 s, and Well Rested for about 30 minutes, which halves
   walking fatigue. The alternative is a separate `modules/inn`, which
   duplicates the timers and recovery protocol.
7. **Walking remainders are persisted** (dirty-flag flush) rather than
   kept only in memory.
8. **Upstream room rental and its buffs 15/16 stay as they are.**

## Acceptance criteria

- **Pure tests:**
  - `TravelModifiers` validation, and `WithModifiers` rounding, clamping,
    and nil identity
  - `RestSession.Recovery` legacy default
  - `InnStay` transitions
  - step cost with each factor alone and combined, the clamps, and indoor
    and town steps costing zero
  - accumulator carry
  - fatigue band mapping
- **Module tests (fake store, clock, and scheduler):**
  - travel with modifiers: duration, exertion totals, reload, and a
    legacy session unchanged
  - camp rest recovery under weather, and a replay after reload with no
    conflict
  - inn: each refusal, payment and refund on save failure, completion
    exactly once, a stale timer as a no-op, overdue completion after
    reload, the movement block, the view, and the Well Rested grant on
    `NewRound` only
  - fatigue: step charge and carry, band-buff sync and refresh,
    persistence and reload, and roster pruning
- **Wiring tests through real entry points:**
  - `usercommands.Go` on real rooms charges a forest step, doesn't charge
    a city step or a refused move, and doesn't charge a profiled travel
    exit
  - `StartTravel` with real weather, encumbrance, and mount providers
    registered persists the modifiers
  - the `inn` command via the module's command dispatcher, with movement
    refused through the real `Go`
  - the shipped buffs 1030 and 1040–1042 load with real config, and max
    HP is unchanged
  - the Well Rested flag halves a real step
- **Review gate:** an independent reviewer subagent over the phase diff,
  with every finding verified and recorded.
- `go test -race ./...`, `make generate`, and `make validate` pass.
