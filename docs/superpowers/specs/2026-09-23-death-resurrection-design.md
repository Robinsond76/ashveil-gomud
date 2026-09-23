# Player Death, Company Return, and Companion Resurrection

**Status:** Design direction approved 2026-09-23; written spec awaiting owner
review. Part of the [roadmap](2026-09-23-company-life-onboarding-roadmap.md).

## Existing behavior and intended rules

GoMud currently moves a dead player to a generic death recovery room. Its
default config has no XP death penalty, protects levels 1–5 from configured
penalties, and has permadeath off. Company `onMobDeath` currently clears only
the live instance; a saved companion is otherwise spawned again on login.
Ashveil needs a separate, coherent death path:

1. A player character who dies loses one level **immediately** and respawns
   at the church in the last visited eligible city. All living, present
   company companions arrive alongside the player. Dead companions do not.
2. A companion who dies remains dead on the roster. The leader can travel to
   a city church or village shaman and resurrect that companion within a
   three-game-day-equivalent allowance of the leader's **online** time.
   Resurrection costs that companion one level.

## Player death and safe destination

- Author a settlement service registry: city churches and village shamans
  have explicit room/service tags and a valid resurrection NPC. A city
  church alone can be a **player respawn checkpoint**. Visiting any room in
  a registered city records that city's church room ID as the player's last
  checkpoint. A city without a valid church does not replace it. Persist the
  checkpoint on the player. Frostfang Sanctuary (room 18) is the initial
  fallback; add a Dunmar church before Dunmar becomes a checkpoint. Villages
  never replace the player checkpoint.
- The player death operation applies one level loss, sets experience to the
  floor of the new level, recalculates derived stats, and clamps current
  health/mana to the new limits. Allocated skills/stat training remain
  allocated. At level 1, keep level 1 and reset progress within that level.
  This Ashveil path supersedes the ordinary XP penalty, protection levels,
  and optional engine permadeath; it must not run a second time on login or
  event replay.
- Cancel or resolve an active travel/camp session through its own durable
  lifecycle before moving the leader. Disengage surviving companions and move
  their live instances to the church without re-summoning them or losing
  their formation/gear. If a companion is not present but is alive in the
  durable roster, restore it at the church when its normal attachment path
  runs. Existing corpse and item-drop rules remain governed by the death
  config; death does not mint replacement items. Company cargo and mount
  ownership remain attached to the leader record, without duplication.
- If a saved checkpoint is invalid or its room fails to load, use the
  configured Frostfang fallback. If no valid church can load, retain the
  pending death operation and surface an error for repair rather than moving
  the player to an arbitrary room.

## Companion death and rescue allowance

- Persist `Dead` on the stable companion ID, its level/gear snapshot, the
  remaining online-time allowance, and its current online-session anchor.
  Remove its live attachment and formation occupancy; exclude it from travel, camp
  recovery, combat, and chemistry accrual. A dead record must never be
  auto-spawned by login/copyover restoration.
- At death, snapshot three game days as `3 × RoundsPerDay × RoundSeconds`
  seconds of **online real time** (10,800 seconds, or three hours, under
  current defaults). While the leader is signed in, subtract elapsed real
  time once. At logout or copyover, persist the remaining seconds and clear
  the online-session anchor. On login, start a new anchor at the current
  time; never subtract the offline interval. Checkpoint periodically so a
  crash can conservatively refund at most one interval, but cannot expire a
  companion early. Server downtime does not spend the allowance. Changing
  calendar settings does not retroactively shorten an existing allowance.
- At a tagged city church or village shaman, `resurrect <member>` identifies
  the dead stable companion ID, confirms remaining time, applies one level
  loss (floor level 1), restores the same member ID and gear, and spawns it
  with the company once. The service has no additional gold fee in this
  first slice; level loss and the journey are its costs. The command cannot
  be used in a field camp or from an untagged NPC.
- When the allowance reaches zero, mark the member permanently lost, archive
  its identity/death record for display, and free its roster slot. Its ID is
  never reused. A command at zero cannot resurrect it. Existing corpse and
  item-drop rules determine what physical gear remains recoverable; the
  saved member gear is restored only if those rules did not drop it.

## Recovery and acceptance criteria

Use durable operation IDs for player level loss, company relocation, and
each companion resurrection or expiry. Multi-store writes need an explicit
pending state and idempotent reconciliation; a failed step must not create
a living clone, double the level loss, or remove a recoverable member.

- A player death from a room, route encounter, or camp returns the player
  and every living companion to the last valid city church exactly once.
- Levels 1 and above obey the specified penalty even though the current
  engine config protects early levels; no other XP penalty is added.
- Two dead companions carry independent paused allowances. Logout, restart,
  copyover, and server downtime do not spend rescue time.
- A city church and village shaman can resurrect a timely companion; a
  village cannot become the player's respawn checkpoint.
- Expired or already resurrected members cannot be revived again, and no
  death or resurrection advances global game time.
