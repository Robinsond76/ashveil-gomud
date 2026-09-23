# Phase 15: Temperature, Clothing, and Exposure

Part of the roadmap in
`docs/superpowers/specs/2026-09-23-environment-skills-economy-roadmap.md`
(user decision 4: exposure can kill, but only at extremes after a sustained
stretch of escalating penalties).

## Prior-art check

- **Upstream cold.** Snow rooms apply buff 31 *Freezing Snow* (1–2 damage)
  unless the player has the `warmed` flag. Buff 3 *Cold Tolerant* grants
  `warmed`. The `desert_sun` mutator applies buff 33 *Thirsty* (−20 to all
  stats). These are per-room content hacks with no temperature model.
- **Survival (Phase 4)** tracks hunger/thirst/fatigue per company member
  (leader plus companions, keyed by `MemberKey`). Mutation goes through
  `ApplyCompanyExertion`, which keeps a permanent per-operation ledger
  (right for travel, wrong for frequent ticks) and applies one cost to
  every member. No survival band has any gameplay effect yet.
- **Buffs** carry `statmods` and flags and persist on characters. Modules
  can ship buffs (Phase 14 did).
- **Death.** A player below 1 HP is downed and bleeds out to death at −10
  (`NewRound_AutoHeal`). A mob at ≤0 HP dies on the next round tick
  (`NewRound_MobRoundTick`). Exposure damage uses these existing paths.
- **Worn equipment** is `Character.Equipment` (10 slots). Company
  companions are mobs; `company.InstanceFor` maps a roster companion to its
  live instance.
- **Phase 14** gave rooms indoor/lit/fixture concepts. Camping registers
  lit campfires.

## Model

All numbers are module config (`modules/exposure`), shown here as defaults.

1. **Air temperature (°C)**, computed from the clock and data. Nothing is
   stored.
   - Outdoors: biome base, minus the biome's night drop at night, plus the
     zone weather's `TemperatureMod` (new `weather.Condition` field,
     −30..30).
   - Furnished indoors (a lit biome, or a room tagged `indoor` or `lit`):
     `IndoorTemperature` (18).
   - Unlit indoors (cave/dungeon): biome base, with no night or weather.
   - A **heat source** in the room (a lit campfire) adds `FireWarmth` (+10).
   - Defaults: forest 12/−8, snow −12/−10, mountains −2/−10,
     desert 36/−22, swamp 22/−6, cave 10, default 15/−6.
2. **Warmth** from worn items. The new item field `warmth` overrides a
   per-slot default (body 5, legs 3, feet 2, head 2, gloves 2, neck 1,
   belt/weapon/offhand/ring 0). Typical clothing is about 15; naked is 0.
   The upstream `warmed` flag (Cold Tolerant) adds `WarmedBonus` (20).
3. **Comfort band.** A character is comfortable when the air is between
   `ComfortLow − warmth` and `ComfortHigh − warmth × HeatFactor`
   (16, 30, 0.5). Stress is how far outside the band the air is: cold is
   below it, heat above it.
   - Naked, forest noon (12°C): 4 cold, mild.
   - Clothed (15), forest night in rain (1°C): comfortable.
   - Naked on a snowfield at night (−22°C): 38 cold, lethal.
   - Heavy furs (40) in the desert (36°C): 26 heat, lethal. Less clothing
     is better in heat.
4. **Exposure** is a signed durable meter per member: −100 is freezing,
   +100 is heatstroke. Every `TickRounds` (5 rounds ≈ 20 s) it moves toward
   a **ceiling** of `stress × CeilingPerStress` (4), clamped to ±100.
   - It grows by `max(1, stress / 2)` per tick.
   - It recovers toward 0 by `Recovery` (5), or `ShelterRecovery` (12)
     indoors or at a fire.
   - Only stress ≥ 25 can reach ±100. Mild mismatch settles at a
     penalty-only level.
5. **Bands and effects.** Buffs are shipped by the module. The active
   band's buff is kept on the character and the others are removed.

   | \|exposure\| | Cold | Heat | Effect |
   |---|---|---|---|
   | 0–24 | — | — | none |
   | 25–49 | Chilled | Overheated | −5 speed/perception |
   | 50–74 | Frostbitten | Heatstricken | −15 speed/strength/perception |
   | 75–99 | Hypothermic | Heat Exhausted | −30 strength/speed/smarts/perception; 2% max HP per tick, never below 1 HP |
   | 100 | Freezing to Death | Heatstroke | as above; 10% max HP per tick plus the player's regen over the tick |

   Penalty buffs never touch vitality, which would shrink max HP. They are
   short-lived, non-permanent buffs that each tick refreshes, so the engine's
   permabuff reconciliation (on equip, remove, and login) can't strip them.
   Only the critical band can down anyone. A death clears the player's own
   exposure.

   Survival drains per tick: cold stress at moderate or worse costs fatigue
   `stress / 5`. Any heat stress costs thirst `stress / 4` (min 1). This
   uses a new non-ledgered survival call. The call marks survival dirty
   rather than writing to disk; the next save, including the periodic
   autosave, persists it. Survival gained a mutex, because travel and camping
   timers call it off the game loop.

   Timeline, naked on a snowfield at night: lethal band in about 2 minutes, downed
   about 3 minutes later, then the normal bleed-out. A player can escape by
   going indoors, reaching a fire, or dressing.
6. **Who is affected.** Online players and their live company companions.
   A roster companion that isn't spawned right now (e.g. mid-restore after a
   restart) keeps its stored exposure; only members that leave the roster
   are forgotten.
   Offline characters don't tick; that matches the handoff doc's open
   question, recommended "no". Other mobs are unaffected.

## Durable model

A `modules/exposure` registry keyed by leader user ID, then `MemberKey`,
holding exposure values. It is persisted through the plugin store like
weather and camping, and saved when a tick changes it. After a restart the
next tick re-syncs band buffs from the stored exposure: player buffs
persist, and respawned companions get theirs back. No timers are used:
ticks are `events.NewRound` listeners, reading the clock and never
advancing it.

## Packages

- `internal/climate` (new, pure): air-temperature inputs and computation,
  warmth, the comfort band, stress, the exposure step, bands, and a
  heat-source registry.
- `internal/items`: `ItemSpec.Warmth` (`warmth`).
- `internal/weather` / `modules/weather`: `TemperatureMod`. The `weather`
  command shows the air temperature through a climate provider seam.
- `internal/survival` / `modules/survival`: a `MemberDrainService` seam with
  `ApplyMemberDrain(leaderUserID, key, cost)`. It is non-ledgered, persisted,
  and applies to one member.
- `modules/camping`: registers lit campfires as heat sources.
- `modules/exposure` (new): config, registry, tick, buffs, damage, drains,
  and the `temperature` command. Ships buffs 1010–1013 (cold) and
  1020–1023 (heat).
- Data: forest weather gets `TemperatureMod`s, and a few warm items get
  explicit `warmth`.

## Open decisions (resolved by recommendation; user said "continue")

- Celsius. The numbers above are the starting balance, all in config.
- The upstream Freezing Snow and Thirsty room buffs are left in place: they
  are content, and a builder can remove them now that a real model exists.
  Noted in the status log.
- Exposure is per member, and companions' clothing matters.
- Deferred to Phase 16: switching on Phase 8's travel and rest weather
  multipliers. It is a separate wiring concern that fits walking fatigue.

## Acceptance criteria

- Pure tests for temperature (outdoor/indoor/cave/night/weather/fire),
  warmth (explicit vs slot default, `warmed`), stress (the worked examples
  above), the exposure step (growth, ceiling, recovery, sign switch, the
  lethal-only-when-extreme invariant), and bands.
- Module tests for a tick raising exposure and applying the band buff,
  lethal-band damage, recovery at a fire, survival drains, persistence and
  reload, and the `temperature` command.
- Wiring tests: a real room, user, and equipment through the module tick,
  and the `weather` command's temperature line.
- An independent review before merge.
- `go test -race ./...`, `make generate`, `make validate`.
