# Phase 23a: Rest Tiers (Camp *Rested*, Inn *Well Rested*)

## Prior-art check

- **Spec:** the "Rest tiers" section of
  [rest and weapon preparation](2026-09-23-rest-weapon-preparation-design.md),
  second on the [onboarding roadmap](2026-09-23-company-life-onboarding-roadmap.md).
  That spec also covers whetstones and *Sharpened* weapons; they touch
  items, combat, and the camp command, and are split out as Phase 23b, as
  19/19b, 21a/21b, and 22a–22c were split.
- **Camp rest** (Phase 7, `modules/camping/camping.go`): a durable 60 s
  real-time rest at a lit camp. The completion timer runs off the game loop,
  so it only persists state and applies the ledgered survival recovery
  (`applyRestRecoveryLocked`). There is no buff today.
- **Inn rest** (Phase 16, `modules/camping/inn.go`): the timer marks
  `WellRestedPending`; the `NewRound` listener (`grantPendingWellRested`)
  grants buff 1030 to the leader and every *spawned* companion once the
  leader is online, then removes the stay. A companion not spawned at that
  moment "misses out".
- **Buff 1030 *Well Rested*** ships in `modules/walking` (flag
  `well-rested`, +2 speed, +2 perception, 450 rounds ≈ 30 min). Walking
  applies `WellRestedPct` (50) while the flag is held.
- **Buff 16**, also named *Well Rested*, is upstream content in
  `_datafiles/world/default/buffs/16-well_rested.yaml` (+1 to six stats,
  +5% XP, 75 rounds). `15-sleeping.js` grants it when *Sleeping* ends; a
  sleeping bag item and Frostfang room 432 grant *Sleeping*. It is not an
  inn stay and does nothing to walking.
- **Buffs** are round-counted (`TriggersLeft`) and tick only while the
  holder is in the world. A player's buffs are saved in the user file; a
  companion mob's buffs die with the mob. `AddBuff` takes an optional
  trigger-count override and refreshes an existing buff in place.
- **Roster and live companions:** `survival.CurrentRoster` lists the
  company's member keys; `company.InstanceFor` maps a companion ID to its
  live mob instance.

## Scope

In: the camp *Rested* tier, tier exclusivity, owed grants for companions
absent at completion (restored while the tier would still be active),
config for durations and percentages, reconciling upstream buff 16, and
walking's use of the new tier.

Out (Phase 23b): whetstones, *Sharpened*, `camp sharpen`, combat changes.
The owner amended the whetstone rules on 2026-09-24 (on-demand use at any
time, 10 uses, one use per member sharpened); the amendment is recorded in
the parent spec for 23b.

## Decisions

Applied under the owner's instruction "merge and begin the next phase",
using this design's recommendations, as in 22a–22c. Each is recorded here.

1. **Rested is a new buff, 1033**, shipped by `modules/walking` beside
   1030 (walking owns rest-tier flags and their strain effect). Flag
   `rested`; no stat mods. Walking applies `RestedPct` (75) while the flag
   is held, and `WellRestedPct` instead when both are somehow held: the
   better tier wins, so walking is correct even before normalization.
2. **Durations are real time, in camping config**: `RestedDuration: 15m`,
   `WellRestedDuration: 30m`, `RestedBuffId: 1033` (beside the existing
   `WellRestedBuffId`). A grant converts the duration to rounds with the
   engine's `RoundSeconds` and passes it as `AddBuff`'s trigger-count
   override, so the buff files' own counts are only fallbacks. As with
   every buff, the count ticks only while the holder is in the world.
3. **Exclusivity, per member, at grant time.** Tiers are ranked
   Rested < Well Rested. Granting a tier to a member that holds a higher
   one grants nothing (a camp rest still restores fatigue). Granting a
   tier removes any lower one, then adds or refreshes its own. So Well
   Rested replaces Rested, and a Rested that was removed cannot reappear
   when Well Rested expires. Pure rule: `camping.Decide`.
4. **Camp completion grants on the game loop** with the inn's pattern:
   the timer's recovery step also marks `RestedPending` in the same save
   as `recovery_applied`; the `NewRound` listener grants once the leader
   is online and clears it. The camp itself stays until broken, as now.
   If both kinds are pending at once, Well Rested is granted and both
   are cleared.
5. **Owed grants.** At grant time, every rostered companion without a
   live mob is owed the tier: `Owed[leader][companionID] = {BuffID, Tier,
   ExpiresAtUTC}` where the expiry is the grant time plus the tier's
   duration. Each round, for leaders with owed entries, an entry is
   dropped once expired; a live owed companion gets the buff for its
   remaining time (at least one round), through the same exclusivity
   rule, and the entry is removed. A later grant replaces a companion's
   owed entry only when its tier is at least as high or the old entry has
   expired, so a camp rest never downgrades an owed Well Rested. Owed
   entries are durable, so restart and copyover keep them. **Each rest
   grants once:** a companion present at the grant, or granted from an
   owed entry, is not granted again when it later respawns. The owed
   window runs on real time, because the absent companion has no buff to
   tick. The leader's own buff, like every player buff, pauses while
   they're offline. Companions only spawn with an online leader, so this
   difference is small, and it is documented.
6. **Upstream buff 16 is renamed *Refreshed*** (same id, same effects,
   new description), so *Well Rested* names only the Ashveil inn tier.
   Buff 16 was never a rest tier (it's the end of a nap and does nothing
   to walking), so this is the smallest change that makes the tier
   unambiguous. Saved copies load unchanged. The alternative, having the
   nap grant 1030, would make a sleeping bag a free, solo Well Rested.
7. **Normalization on spawn.** A `PlayerSpawn` listener removes Rested
   from a player who also holds Well Rested, so a saved character carries
   at most one tier. No other combination can be saved: buff 16 is no
   longer a tier.

## Durable model

`modules/camping` registry (YAML, all new keys `omitempty`, so older files
load unchanged):

```yaml
rested_pending: {7: true}
owed:
  7:
    3: {buff_id: 1033, tier: 1, expires_at_utc: 2026-09-24T12:15:00Z}
```

`internal/camping/tiers.go` (pure): `Tier` (None, Rested, WellRested),
`Decide(held, grant) (grant bool, remove []Tier)`, `OwedGrant`,
`MergeOwed(old, new, now)`, and `RemainingRounds(expires, now, roundSeconds)`.

## Module changes

- `modules/camping`: config parsing, `RestedPending` and `Owed` in the
  registry, the combined grant pass (`grantPendingTiers`) replacing
  `grantPendingWellRested`, an owed-restoration pass, the `PlayerSpawn`
  normalizer, and seams for buff add/remove/has, live companions by ID,
  rostered companion IDs, and rounds per duration.
- `modules/walking`: buff 1033 and flag `rested`, `RestedPct`, precedence,
  and a "rested 75%" note in the walking view.
- `_datafiles/world/default/buffs/16-well_rested.yaml`: renamed.

## Constraints

- Buffs are only touched on the game loop (`NewRound`, `PlayerSpawn`);
  timers only persist markers. Camping never holds `m.mu` while calling
  another module or touching characters; lock order is unchanged.
- No global time or round advance. Expiry is real UTC time from the
  module clock.
- A failed save restores the pending markers and owed entries, so the
  next round retries; a retry only refreshes a buff already granted.

## Acceptance criteria

- A completed camp rest grants Rested (configured duration) once to the
  online leader and every live companion. A member holding Well Rested
  keeps it and does not get Rested.
- A completed inn stay grants Well Rested and removes Rested.
- A companion absent at completion gets the tier on restoration, for the
  remaining time, only before expiry, and only once. This survives
  reload.
- Walking strain is 75% with Rested and 50% with Well Rested (never both).
- Buff 16 is no longer named Well Rested; a saved player holding both
  tiers is normalized on spawn.
- Tests go through the `camp`/`inn` user commands and the real
  `NewRound`/`PlayerSpawn` listeners, not only the pure rules.
