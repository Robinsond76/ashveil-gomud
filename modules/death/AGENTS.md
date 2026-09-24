# Death Module Guide

Phase 25a: a player's death costs one level, and they wake at the church of
the last city they visited, with their living companions. Design:
`docs/superpowers/specs/2026-09-24-phase-25a-player-death-design.md`.

- **The seam.** `internal/death` holds the settlement registry, the
  `Destination` rule, and the provider seam. `internal/usercommands/suicide.go`
  asks `death.Active()`. Without a provider the engine's path is unchanged
  (XP penalty, permadeath, the Shadow Realm). With one, the announcements,
  kill stats, `PlayerDeath` (never `Permanent`), drops, and corpse stay, the
  engine's XP penalty and permadeath don't run, and the player is handed to
  `Respawn` instead of the Shadow Realm.
- **Exactly once.** `Respawn(userID, newDeath)`: a new death takes the level
  (`Character.LoseLevel`) and sets `MiscData["death-pending"]` to an operation
  ID in the same step, so both are saved together in the user file; a retry
  never takes a level. `suicide` decides which: a player still down (health
  below 1) with the mark is a retry (no announcement, corpse, drop, or level);
  a living player returned this round or the last (`JustReturned`, in memory)
  is ignored, because the combat loop and AutoHeal can both queue a `suicide`
  for one death; anything else is a new death. Clear the mark only once the
  player is at the church.
- **Retry by staying dead.** When no church loads, or ending the journey or
  camp fails, or the move fails, `hold` leaves the player where they fell at
  −10 health. The engine's own AutoHeal (every 3 rounds out of combat) and
  combat loop re-issue `suicide`, which retries. Don't heal a pending player
  from this module.
- **Order.** Destination first (so nothing is ended for nothing), then
  `expedition.AbandonForDeath`, `camping.AbandonForDeath`, the move, vitals,
  `company.RelocateCompany`, then the mark is cleared. The abandons take
  their module's own mutex inside the world lock (the same order as the
  `travel` and `camp` commands).
- **Checkpoint.** `MiscData["death-checkpoint"]` is set on `RoomChange` and
  `PlayerSpawn` when the room's zone is a registered city whose church is
  valid (loads, has the `church` tag, is the church its zone is registered
  with). Villages never set it. Read it with `configInt`: YAML may bring it
  back as another integer type.
- **Config** (`files/data-overlays/config.yaml`): `Settlements` (zone, kind,
  service room), `FallbackRoomId` (18), `RespawnVitalsPct` (50). The module
  reserves the `church` and `shaman` room tags.
- **Level loss.** `Character.LoseLevel` keeps `PeakLevel`; `LevelUp` grants
  training and stat points only above it, so dying and re-levelling can't
  farm points. Any new code that lowers a level should raise `PeakLevel`
  first, the same way.
- **Phase 25b** (companion death, the online-time allowance, `resurrect`,
  village shamans) builds on this registry: a village's service room carries
  the `shaman` tag.
- `wiring_test.go` calls `plugins.Load` with the same
  `SnapshotLoadStateForTest` guard as the company tests, and drives travel
  through the real `go` command (the exit message requeues the command with
  input blocked, so the test unblocks it the way the game loop would).
