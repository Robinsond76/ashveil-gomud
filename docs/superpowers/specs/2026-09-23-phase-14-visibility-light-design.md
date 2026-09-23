# Phase 14: Visibility and Light

Part of the roadmap in
`docs/superpowers/specs/2026-09-23-environment-skills-economy-roadmap.md`.
Builds on Phase 13 (moon, cloud cover, fog, indoor rooms).

## Prior-art check

- `rooms.Room.GetVisibility()` (upstream) returns a room-wide 0–2:
  night −1, dark biome −2, lit biome +1, mutator `LightMod`, and **+1 if any
  player or mob in the room has the `lightsource` buff flag**. Its only
  gameplay consumer is `look` (0 = "You can't see anything!", <2 = exits
  can't be looked through unless the biome is lit). The `nightvision` flag
  bypasses both checks.
- Carried light already exists as equipment: the lantern (item 20036,
  offhand) grants buff 1 (`Illumination`, flag `lightsource`) while worn.
  The `illum` spell gives the same buff.
- Combat hit rolls happen in `internal/combat.calculateCombat` via
  `Hits(atkSpd, defSpd, penalty)`. Darkness has no effect there today.
- Companions are charmed mobs (`Character.Charmed.UserId` = leader); human
  groups use `internal/parties`.
- Camps (`internal/camping.Camp`) persist `RoomID` and `FireLit`.
- Plugins can ship buffs (`buffs/`) and buff flags (`buffs-flags/`).

## Scope (user decision 1)

1. **Ambient light is per room, but only fixtures count.** Sun, moon and
   cloud cover, fog, indoor/dark/lit biome, mutators, and room **fixtures**:
   a `lit` room tag, or a lit campfire from a registered fixture provider.
   A carried light no longer lights the room for everyone.
   `GetVisibility()` now returns this ambient level.
   - Outdoors by day: 2. Fog applies its `VisibilityMod`, but daylight never
     drops below 1.
   - Outdoors at night: 1 if moonlight ≥ 1 (see `sky.Moonlight` with the
     zone's cloud cover), otherwise 0 (moonless or overcast is pitch black).
     Fog applies. A lit biome (street lanterns) adds +1.
   - Indoors: 2 if the biome is lit, otherwise 0. The sky doesn't matter.
   - Dark biome (cave/dungeon): 0.
   - Then mutator `LightMod`, then a fixture adds +1. Clamp to 0..2.
2. **Per-viewer visibility** is `VisibilityForUser`/`VisibilityForMob`:
   ambient, +1 if the viewer has their own `lightsource`, or +1 if an ally
   in the room has `partylight`. `nightvision` gives 2. Allies are:
   - the same leader (a user and their charmed companions)
   - members of the same GoMud party
3. **Party light.** A new `partylight` flag and a *Floating Light* buff are
   shipped by a new `modules/light`. A `floatinglight` spell in the default
   world gives the caster that buff. Gating it to wizards belongs to
   Phase 17 (archetypes).
4. **Combat penalty.** An attacker's to-hit roll is reduced by
   `DarkHitPenalty` (default 40) at visibility 0, or `DimHitPenalty`
   (default 10) at visibility 1. A target carrying its own light counts as
   at least dim, because you can see a torch. Both values are config in
   `modules/light`.
5. **`light` command.** Reports the ambient light level and why, your own
   light, any party light, and what you can see.
6. **`look`** uses per-viewer visibility.

## Durable model

No new persisted state. Ambient light is derived from the clock, weather
(already persisted), data, and camp records (already persisted). Buffs
persist on characters as they do today.

## Concurrency

The camping fixture query takes the camping module's mutex. The camping
module never computes visibility while holding it, so this cannot deadlock.
The fixture registry in `rooms` is guarded by an RWMutex. No timers are
added, and nothing touches the clock.

## Open decisions (resolved by recommendation; user said "continue")

- Pitch black only on moonless or overcast nights. Any visible moon gives
  dim light (1).
- Daylight fog never fully blinds (floor 1).
- A fixture gives +1, the same as a torch, not full light.
- The penalty applies to all attacks, whether by players or mobs. Mobs with
  `nightvision` ignore it.

## Deferrals

- Wizard-only gating of `floatinglight` (Phase 17).
- Race darkvision (for example, wolves): a mob can carry the
  `nightvision` flag through a buff today; a race-level trait comes later.
- Hiding in darkness (stealth) is not in scope.

## Acceptance criteria

- The ambient-level function covers day, night with and without moon,
  overcast, fog by day and night, indoor lit/unlit, dark biome, fixture,
  and mutator clamping.
- Personal light helps only its bearer. Party light helps allies but not
  strangers. Nightvision gives 2.
- The hit penalty is 40, 10, or 0 for visibility 0, 1, and 2, and a lit
  target raises visibility 0 to 1.
- A lit campfire lights its room for everyone.
- `go test -race ./...`, `make generate`, `make validate` pass.
