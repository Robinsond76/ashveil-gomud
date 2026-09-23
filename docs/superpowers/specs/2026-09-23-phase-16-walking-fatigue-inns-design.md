# Phase 16: Walking Fatigue, Inns, and Travel/Rest Multipliers

Part of the roadmap in
`docs/superpowers/specs/2026-09-23-environment-skills-economy-roadmap.md`
("Walking causes fatigue; rest at an inn (bonus) or a camp to recover").
Phase 15 moved the deferred switch-on of Phase 8's weather multipliers here
because it is the same wiring job as walking fatigue.

**Status:** design written 2026-09-23. The user confirmed the decisions the
same day: 1, 2, 4, 5, 6, and 7 as recommended, and 3 changed so mount relief
covers at most two characters (see "Decisions"). Economy tuning (inn price)
will be revisited later. Implemented 2026-09-23; see "Implementation notes"
at the end for where the build differs from this text.

## Prior-art check

- **Local movement** (`internal/usercommands/go.go`) spends 10 action points
  per step, or 50 when the carried item count is over the limit. It has no
  survival cost. Handoff §5.1 says local movement should have "little or no
  survival cost". Anything we add must stay small, and zero inside
  settlements.
- **Party and companion movement.** When a leader moves, each GoMud party
  member runs their own `Go`, so each one charges their own company. Charmed
  mobs (company companions) move through `mob.Command`, not the user `Go`.
  So one hook after the leader's successful move charges the whole company
  once. Route arrivals go through `MoveToRoom` from `modules/expedition`, not
  `Go`, so walking never charges a route step twice. The `RoomChange` event
  can't tell walking from teleport or recall, so we don't use it.
- **Survival (Phase 4, 15).** Needs are ints 0–100. `ApplyCompanyExertion`
  keeps a permanent per-operation ledger, which is right for travel
  checkpoints and wrong for frequent steps. `ApplyMemberDrain` (Phase 15)
  has no ledger, marks the registry dirty, and flushes on the next save. It
  is the right tool for per-step cost. Survival has a leaf mutex. **No
  survival band has any gameplay effect yet.**
- **Travel (Phase 5/6/12).** `TravelSession` stores only `ProfileName`. The
  module looks up the static profile by name at about 15 call sites
  (`m.profile(session.ProfileName)`). Duration and exertion come from the
  profile. Exertion is charged over 10 durable checkpoints, with
  deterministic operation IDs.
- **Deferred multipliers (Option A in Phases 8/9/10).**
  - `weather.Condition` has `TravelDurationPct`, `ExertionPct`, and
    `RestRecoveryPct` (25–300).
  - `encumbrance.LoadBand` has `TravelDurationPct` and `FatiguePct`; the
    band is resolved only inside `modules/encumbrance`'s status output.
  - `mount.MountSpec.TravelDurationPct` is parsed and validated.
  - None of these are applied. Phase 8's design says "lock at departure" is
    simpler, but it needs a durable schema change to `TravelSession`.
- **Camping (Phase 7).** A camp needs a `camping`-tagged room and a lit fire.
  `camp rest` is one real-time 60 s session per camp. On completion it
  restores a constant `camping.FatigueRecovery` (20) through
  `ApplyCompanyRestRecovery` with a deterministic operation ID. Completion
  can run on a `time.AfterFunc` goroutine, **off the game loop**. Camping
  owns the only `camping.MovementBlocked` seam that `Go` consults.
- **Upstream inn.** The Frostfire Inn (room 61) sells a 5-gold "room rental"
  service item (id 102) from the serving wench (mob 7). Buying it moves you
  into an ephemeral copy of room 432. That room's script gives buff 15
  *Sleeping* (heals; cancelled by any action), which ends by giving buff 16
  *Well Rested* (5 minutes, +1 all stats, +5 xpscale, a small heal). This is
  content with no survival interaction. We leave it alone.
- **Exposure (Phase 15)** keeps a signed per-member exposure in
  `modules/exposure`. Only the air temperature is exposed through a seam
  (`climate.AirTemperatureIn`). There is no read seam for a member's
  exposure yet.
- **Weather is zone-scoped.** It uses `rooms.GetZoneBiome(zone)`, the
  zone's `defaultbiome`. The Dunmar zone's default biome is `city`, so
  **Dunmar, including the proving route's origin, has no weather at all**.
  The weather config comment says forest is "in use at Dunmar 2001/2002",
  but that isn't true. See decision 6.

## Scope

1. **Walking fatigue.** Each ordinary step through the user `Go` command
   costs fatigue for every company member. The cost depends on terrain,
   load, weather, cold exposure, the mount, and Well Rested.
2. **Fatigue has teeth.** There are band penalty buffs for Exhausted and
   Collapsed. A company with a Collapsed member can't set out on a route.
   Ordinary movement is **never** refused, so nobody gets stuck.
3. **Inns.** An `inn` room tag and an `inn` command. The company pays to
   rest in a real-time rest session that recovers more fatigue than a camp,
   then everyone gains a module *Well Rested* buff that halves walking
   fatigue.
4. **Multipliers switched on.**
   - Weather, load, and mount multipliers are locked onto a travel session
     at departure and change its duration and exertion.
   - Weather `RestRecoveryPct` is locked onto a camp rest at its start and
     changes its recovery.
5. **Inspection.** A `strain` command shows this room's per-step cost and
   each factor. `inn` shows the price and any stay in progress.

## Model

All numbers are module config, shown here as defaults.

### 1. Per-step strain (`internal/walking`, pure)

Strain is in **centi-fatigue** (100 = 1 fatigue point), so small costs add up
without rounding them away.

```
stepCost = TerrainCost(dest room)
         × load.FatiguePct       / 100   (encumbrance band; 100 when none)
         × weather.ExertionPct   / 100   (outdoor rooms only; 100 untracked)
         × ColdPct(band)         / 100   (per member, from exposure)
         × mount.FatiguePct      / 100   (100 without a mount)
         × WellRestedPct         / 100   (per member, when the buff is on)
```

The product of the multipliers is clamped to [25 %, 400 %]. Integer math
rounds half up.

- **Terrain cost** belongs to the **destination** room: you pay for the
  ground you walk into.
  - Settlements cost **0**: biomes `city`, `slums`, `house`, `fort`, any lit
    biome, and any room tagged `indoor`.
  - Otherwise the biome table: road 25, land/farmland/shore 35, forest 50,
    cave/dungeon 50, water 60, swamp 80, desert 80, snow 90,
    mountains/cliffs 100, default 40.
  - A room can override it with a `strain:<n>` tag. This is optional and
    lets builders tune a room without a code change.
- **Cold** (Phase 15 band, cold side only; heat already drains thirst):
  - Chilled 125
  - Frostbitten 150
  - Hypothermic / Freezing 200
- **Mount:** a new `MountSpec.FatiguePct` field, where 0 means 100. The
  pack-horse is 75. It applies to **at most `Riders` company members** (a
  new `MountSpec.Riders` field; 0 means 2, and the pack-horse is 2).
  - Riders are chosen deterministically, from the durable roster with no
    new state: the leader first, then companions by ascending companion ID.
  - Only members who are present count, so a companion that isn't spawned
    never uses up a seat.
  - Everyone else walks at 100. (Decision 3.)
- **Well Rested:** 50.
- **Accrual.** Each member has a durable **carry** in centi-fatigue. A step
  adds `stepCost` to the carry. Every whole 100 is drained through
  `survival.ApplyMemberDrain(leader, key, Exertion{Fatigue: n})`, and the
  remainder stays in the carry.

Worked examples (fatigue points lost per 10 steps):

| Situation | Per step (centi) | Per 10 steps |
|---|---|---|
| Town streets | 0 | 0 |
| Forest, clear, unladen | 50 | 5 |
| Forest in rain (Exertion 115), 90 % loaded (Fatigue 115) | 66 | ~7 |
| Snowfield, Frostbitten, fully loaded (130) | 176 | ~18 |
| Same, riding the pack-horse, Well Rested | 66 | ~7 |
| Same, Well Rested, a third member walking beside the horse | 88 | ~9 |

Starting from full (100), an unladen company reaches Exhausted (25) after
about 150 forest steps, and after about 43 in the snow example. Roads and
towns stay cheap. That matches handoff §5.1: local movement has "little"
cost, and wilderness is where fatigue bites.

### 2. Fatigue bands (`modules/walking`)

A `NewRound` tick runs every `TickRounds` (5). It visits online leaders and
their spawned companions, the same shape as Phase 15. It keeps one band buff
refreshed and removes the other. The buffs are short, non-permanent, and
refreshed every tick, the pattern Phase 15's review settled on so the
engine's permabuff reconciliation can't strip them.

| Fatigue | Buff | Effect |
|---|---|---|
| 1–25 (Exhausted) | 1031 *Exhausted* | −10 speed, −10 strength |
| 0 (Collapsed) | 1032 *Collapsed* | −25 speed, strength, and smarts |

No buff touches vitality (Phase 15's review finding 1). Starting a route
checks `survival.CompanyNeeds`. If any member is at 0 fatigue, the route
refuses: "Your company is too exhausted to set out. Rest first."
**Ordinary movement is never blocked.** Camps need tagged rooms, so a
refusal could strand a Collapsed company in the wilderness.

### 3. Inns (`modules/camping`, domain in `internal/camping`)

- An **inn room** has the `inn` tag. Content for this phase:
  - a new Dunmar room 2003, *The Waymark Inn*, off the west gate, tagged
    `inn` and `indoor`
  - the upstream Frostfire Inn (61), tagged `inn`
- **`inn`** shows the price (`PricePerMember` × company size, default 5
  gold per member) and any stay in progress. **`inn rest`** does this:
  1. Checks the leader is in an inn room, has no active camp rest or stay,
     and has the gold.
  2. Takes the gold.
  3. Persists a durable `InnStay{LeaderUserID, RoomID, Paid, Rest
     RestSession}`.
  4. Schedules the timer.

  If the save fails, the gold is refunded and nothing changes (the same
  rollback shape as camp commands).
- **`inn status`** is an alias of `inn`.
- **Rest.** `InnRestDuration` is 60 s real time, the same as a camp.
  Recovery is `InnFatigueRecovery` (60) versus the camp's 20. It uses the
  same `ApplyCompanyRestRecovery` ledger with a deterministic
  `inn-rest-<leader>-<room>-<startNanos>` operation ID. While a stay is
  active, ordinary movement is refused through the existing
  `camping.MovementBlocked` seam.
- **Well Rested.** On completion the stay records `WellRestedPending`
  durably. The rest completion can run on a timer goroutine off the game
  loop, and granting buffs there would race character state. So a camping
  `NewRound` listener (on the game loop) applies buff **1030 *Well
  Rested*** and then clears the flag. The buff has the `well-rested` flag,
  +2 perception, and +2 speed, and lasts `450` rounds (about 30 min). It
  goes to the leader and every spawned companion. A companion that isn't
  spawned yet gets the buff when it spawns, as long as the flag is still
  pending, or else it misses out. (Decision 4.)
- A completed stay is removed once its recovery and buff are both applied.
  Stays are repeatable. Gold and 60 s per rest are the only limits.
  Weather never touches inn rests, since you're indoors.
- **Upstream room rental (item 102, rooms 432, buffs 15/16)** is left
  unchanged. It is content, and a builder can remove it later. The buff name
  *Well Rested* now exists twice (upstream 16 and module 1030); the status
  log notes this.

### 4. Multipliers switched on

- **Travel. Locked at departure.** `TravelSession` gains three persisted
  fields: `DurationPct`, `ExertionPct` (hunger, thirst, and fatigue), and
  `FatiguePct` (fatigue only, multiplied on top). All are `omitempty`, and
  **0 means 100**, so every session saved before Phase 16 reloads and runs
  exactly as before. At `StartTravel` the module computes:
  - `DurationPct` = weather.TravelDurationPct × load.TravelDurationPct ×
    mount.TravelDurationPct, clamped to [25, 400]
  - `ExertionPct` = weather.ExertionPct
  - `FatiguePct` = load.FatiguePct, clamped

  The mount's **fatigue** relief does not apply to route travel this phase.
  Travel charges one cost to the whole company through
  `ApplyCompanyExertion`, so relief for only two riders would need a
  per-member charge through the ledger. That change touches the Phase 5
  durability code, so it is deferred. On a route, the mount still speeds up
  the journey (`TravelDurationPct`), which shortens it for everyone.

  Weather is read from the origin room's zone.
  A new pure `TravelSession.EffectiveProfile(profile)` scales `Duration` and
  `Exertion`, and every module call site uses `m.sessionProfile(session)` in
  place of `m.profile(session.ProfileName)`. Interruption checkpoints are
  fractions of the duration, so they scale with it. Operation IDs still
  derive only from session identity and the checkpoint, so recovery replay
  stays idempotent.
  - The journey view and refusal text show the effective duration.
  - When a factor applies, departure prints one line naming it, e.g.
    "Rain and a heavy load slow your pace."
- **Camp rest. Locked at rest start.** `RestSession` gains `Recovery int`,
  where 0 means `camping.FatigueRecovery` (legacy). `camp rest` stores
  `FatigueRecovery × weather.RestRecoveryPct / 100` for the camp room's zone.
  Inn stays store `InnFatigueRecovery`.

### 5. Read seams added

These are all read-only, and each returns a neutral value when no provider
is registered:

- `encumbrance.CurrentBand(leader) (LoadBand, bool)`
- `mount.Relief(leader) (fatiguePct, riders int)` and
  `mount.TravelDurationPct(leader)`. Without a mount these return (100, 0)
  and 100.
- `climate.ExposureOf(leader, key) (int, bool)`, implemented by
  `modules/exposure`
- `walking.Stepped(userID, fromRoomID, toRoomID)`, the provider called from
  `Go` after a successful move. It does nothing without a provider.

## Durable model

- **`modules/walking` registry.** Keyed by leader user ID, then `MemberKey`,
  it holds the carry (0–99 centi-fatigue). It is persisted through the
  plugin store. Steps mark it dirty, and it flushes on the next save
  (the plugin `SetOnSave` callback, including the periodic save, as survival does). A crash loses under 1 fatigue point per
  member, which is harmless. It is pruned against the roster the way
  exposure is.
- **`modules/camping` registry** gains `Stays map[int]camping.InnStay` and
  `WellRestedPending map[int]bool`. Inn recovery reuses the
  `RecoveryApplied` idempotency pattern, keyed separately
  (`InnRecoveryApplied`). The existing `recoverLocked` also reconciles stays
  on load or copyover: it reschedules or completes once.
- **`TravelSession`** gains `DurationPct`, `ExertionPct`, and `FatiguePct`.
  **`RestSession`** gains `Recovery`. All are backward-compatible zero
  values.
- **Nothing reads or writes the world clock.** Ticks are `NewRound`
  listeners that only read the round. Rests and travel stay real-UTC
  sessions.

## Concurrency and lock ordering

- **The walking step** runs on the game loop (the `Go` command). It takes
  the walking mutex, then calls read seams: encumbrance, mount, weather, and
  the exposure read (which takes the exposure mutex as a reader). Then it
  calls `survival.ApplyMemberDrain` (a leaf). None of those call back into
  walking. Order: **walking → {encumbrance, mount, weather, exposure} →
  survival**.
- **`StartTravel`** runs under the expedition mutex and newly reads
  encumbrance, mount, weather, and `survival.CompanyNeeds`. None of them
  call expedition. Order: **expedition → {encumbrance, mount, weather} →
  survival**.
- **Camping timers** run off the loop and must not touch characters or
  buffs. They only persist state and set `WellRestedPending`. The buff is
  granted on the game loop.
- **Gold is taken and refunded** only on the command path (game loop).
- The known pre-existing risk from Phase 15 still stands: expedition and
  camping timers read company state, which has no lock. This phase adds no
  new off-loop company read.

## Packages

- `internal/walking` (new, pure): the terrain-cost table type, `StepCost`,
  the multiplier clamp, carry accrual (`Accrue(carry, cost) (newCarry,
  drain)`), and the `Stepped` provider seam.
- `internal/expedition`: `DurationPct`, `ExertionPct`, `FatiguePct` on
  `TravelSession`; `EffectiveProfile`; `Validate` range checks.
- `internal/camping`: `RestSession.Recovery`; duration-parameterised rest
  progress (`ProgressFor(now, duration)`); an `InnStay` type with the same
  pure transitions (start, due, complete).
- `internal/encumbrance`, `internal/mount`, `internal/climate`: the read
  seams in §5.
- `modules/walking` (new): config, registry, the step handler, the fatigue
  band tick, the `strain` command, and the shipped buffs 1030–1032.
- `modules/expedition`: multipliers locked at departure, the Collapsed
  refusal, the departure line.
- `modules/camping`: the weather rest multiplier, inn stays, the `inn`
  command, and applying Well Rested on the loop.
- `modules/mount`: `FatiguePct` config, provider methods.
- `modules/encumbrance`, `modules/exposure`: provider methods.
- `internal/usercommands/go.go`: one `walking.Stepped` call after a
  successful move.
- Data: Dunmar room 2003 and its exit, the `inn` tag on room 61, pack-horse
  `FatiguePct`, and config comments that no longer say "informational".

## Decisions (confirmed by the user, 2026-09-23)

1. **Settlements cost nothing.** Walking is free in city/slums/house/fort,
   lit biomes, and `indoor` rooms, and cheap on roads. *Recommended*, per
   handoff §5.1.
2. **Fatigue teeth.**
   - The Exhausted and Collapsed penalty buffs, plus refusing route travel
     when any member is Collapsed. Ordinary movement is never refused.
     *Recommended.*
   - Alternative: no penalties this phase. That makes walking fatigue purely
     cosmetic until a later phase.
3. **The mount's fatigue relief** covers **at most two characters**. The
   user changed this from the recommendation (the whole company).
   - It is `MountSpec.Riders` (default 2): the leader first, then
     companions by ascending ID, counting only present members.
   - It applies to walking only. Route-travel relief for riders is deferred
     (see §4).
4. **Well Rested** is a new module buff (1030) that halves walking strain
   for about 30 minutes, for the leader and spawned companions.
   *Recommended* over reusing upstream buff 16, which is 5 minutes and tied
   to the rental script.
5. **Inn price and power:** 5 gold per member, 60 s, +60 fatigue, and
   stays can be repeated. *Recommended.* All of it is config. The user will
   revisit economy tuning later.
6. **The proving zone has no weather** (Dunmar's default biome is `city`).
   - Recommended: move Fork at the Black Oak (2002) into a new zone, *Old
     King's Road* (`defaultbiome: forest`), so the proving route and camp
     actually see weather. This changes only content. The new zone simply
     gets established the first time weather loads.
   - Alternative: leave it, so the multipliers are proven only by tests.
7. **Multipliers lock at departure or rest start** and are not re-rolled
   mid-journey when the weather changes. *Recommended* (Phase 8's own
   recommendation, and simpler for durability).

## Constraints and deferrals

- Never advance or fast-forward the world clock or round count. Sessions
  stay real-time, and ticks only read the round.
- Everything new that is durable survives restart and copyover: carries,
  inn stays, pending Well Rested, and multiplier fields. Legacy records load
  with neutral values.
- All balance numbers live in module config overlays.
- **Deferred:**
  - fatigue affecting travel speed (handoff "slower travel")
  - mount fatigue, health, and feed
  - mount fatigue relief on route travel (needs a per-member charge through
    the ledger)
  - choosing which members ride (e.g. a `mount ride <member>` command)
  - injury modifiers
  - passive fatigue recovery in settlements
  - inn food and drink bundles
  - alignment-gated inn access (Phase 21)
  - camp fuel and burn-out
  - sneaking or running pace
  - mob-only walking strain
  - `flee` movement strain
  - retiring the upstream room rental

## Acceptance criteria

- **Pure tests:**
  - `StepCost`: settlement 0, the biome table, the `strain:` tag, each
    multiplier, the clamp, rounding, and the worked examples
  - `Accrue`: carry and whole-point drain
  - `EffectiveProfile`: legacy zero → unchanged, scaling, and interruption
    checkpoint preserved
  - `TravelSession.Validate` range checks
  - `RestSession.Recovery` legacy default
  - `InnStay` transitions
- **Module tests:**
  - walking: step drains per member with the carry persisted and reloaded;
    Well Rested halves it; cold band raises it; mount relief reaches only the first two present riders (the leader, then the lowest companion IDs), with a third member paying full cost; roster pruning; band buffs
    applied and swapped; `strain` output
  - expedition: multipliers locked at start and survive reload; a legacy
    session reloads unchanged; Collapsed refusal
  - camping: weather rest recovery; inn pay, rest, and complete; refund on
    save failure; movement blocked during a stay; stay recovered after
    reload; Well Rested applied on the loop, not in the timer
- **Wiring tests (through real entry points):**
  - `Go` from a forest room with a real user and a charmed companion charges
    both; `Go` in a city charges nothing; a route arrival charges nothing
    extra
  - `expedition.Start` through `go` on a travel exit with a heavy load and a
    tracked weather zone produces a longer session
  - `camp rest` in a tracked zone recovers the scaled amount
  - `inn rest` in room 2003 with real gold
  - the exposure read seam feeding the walking cold multiplier
- **Concurrency:** a `-race` test calls a step, a camp or inn timer
  completion, and an exposure tick concurrently.
- An independent review before merge (the `CLAUDE.md` gate), recorded with a
  **Review:** line.
- `go test -race ./...`, `make generate` (adds `modules/walking`),
  `make validate`, and the server boots with the new buffs and room.

## Implementation notes (2026-09-23)

Where the implementation departs from, or pins down, the text above:

- **Zone name.** The new zone is **Old Kings Road** (no apostrophe):
  `rooms.ValidateZoneName` allows only letters, digits, spaces, and
  underscores. Folder `rooms/old_kings_road/`.
- **Route weather source.** Weather is read from the origin room's zone,
  **falling back to the destination zone** when the origin is untracked.
  Without the fallback the proving route (Dunmar 2001, a city, to Old Kings
  Road 2002) would never see weather even after decision 6.
- **Who walking charges.** Only members walking with the leader: the leader,
  plus spawned companions standing in the origin or destination room. An
  unspawned or stray companion is neither charged nor seated on the mount.
- **Lit biomes.** The engine's `default` biome is `litarea: true`, so rooms
  with no biome count as settled ground (cost 0). The table's `default 40`
  applies to unknown non-lit biomes (e.g. `spiderweb`).
- **Travel multiplier product.** `DurationPct` is the rounded product of the
  three percentages, clamped to [25, 400]. A neutral multiplier is stored as
  0, so a neutral new session saves byte-for-byte like a legacy one.
- **`InnStay.Duration`.** A stay also stores the rest duration locked at its
  start, so a config change mid-stay can't move its end.
- **Inn timers.** Inn stays have their own timer map and generations,
  separate from camp rests, so breaking an idle camp never stops a stay's
  timer.
- **Completed stay.** Until its Well Rested is granted (the next round with
  the leader online), a completed stay refuses a new `inn rest` ("only just
  woken"). If the leader is offline, the buff stays owed until they return.
- **`inn rest` while travelling** is refused (checked through the
  expedition movement seam before taking the camping lock).
- **Buffs 1030–1032 and the `well-rested` flag** ship from `modules/walking`.
  `modules/camping` grants 1030 by configured id.
