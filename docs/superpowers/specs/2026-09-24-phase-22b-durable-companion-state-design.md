# Phase 22b: Durable Companion Level and Equipment

## Prior-art check

- **Spec:** the "Durable model and boundaries" section of
  [recruitment and creation](2026-09-23-recruitment-character-creation-design.md),
  split out of it as described in the
  [22a design](2026-09-24-phase-22a-creation-starter-kits-design.md).
  Recruiters (22c) and later sharpening and resurrection work need a
  companion's level and actual gear to be durable.
- **Company record** (`internal/company`): each companion stores a stable
  `ID` (never reused), `MobTemplateID`, `Archetype`, and `Disposition`
  (Phase 21a). The live mob's `InstanceId` is runtime only
  (`modules/company` keeps an instance map).
- **Restoration** (`modules/company.restoreForLeader`, run on every
  `PlayerSpawn`) spawns a fresh mob from the template, and
  `mobs.NewMobById` mints the template's `Items` and `Equipment` each time.
  So today:
  - gear given to a companion is lost on every relog, restart, and
    copyover;
  - template gear is minted again each time;
  - a dead companion comes back at the next login with fresh template gear.
- **Logout** (`hooks/PlayerDespawn_HandleLeave`) expires the leader's charm.
  The companion reverts to an ordinary mob and stays in the world until the
  leader's next login, when restoration destroys it. While it lingers it
  can be killed for its gear.
- **Copyover and restart** keep no mob instances; only connections carry
  over. Plugin `OnSave` runs on autosave, shutdown, and copyover.
- **Companion death** (`mobcommands/suicide.go`) queues `MobDeath` after the
  charm is removed. It then drops every carried item and each worn item
  that passes the `ItemDropChance` roll, and destroys the instance and the
  rest of its gear. By the time the `MobDeath` listener runs, the instance
  is gone.
- **Gear changes** on a mob (give, get, drop, put, throw, the Phase 11
  interceptor's offhand swap) queue `events.ItemOwnership` with
  `MobInstanceId`. The mob's own `gearup` (equip from its backpack) queues
  nothing.

## Decisions

Applied under the owner's instruction "Merge, then start Phase 22b",
using this design's recommendations. Each one is recorded here.

1. **Durable state on the companion record.**
   - `Companion.State *MemberState` holds `Level`, `Experience`,
     `Equipment` (`characters.Worn`), and `Items` (`[]items.Item`). The
     engine's own types keep each item's UUID, uses, enchantments, and
     adjectives, and the equipment slot it was in.
   - `nil` means a companion saved before this phase.
   - `Registry.Get` and `Clone` deep-copy it.
   - `internal/company` gains imports of `internal/characters` and
     `internal/items`; neither imports it back.
2. **Initialized exactly once.**
   - **New recruit:** `company summon` snapshots the freshly spawned mob
     (template gear minted once) into the record before the summon's
     save. A failed save rolls the summon back as it does today, and the
     mob is destroyed along with its gear.
   - **Legacy companion:** on restore, the state is derived from the
     template spec without spawning, then saved, and only then spawned.
     If that save fails, the record keeps `State: nil`, the companion is
     not spawned (it shows "awaiting restoration"), and the next spawn
     retries. No template gear leaves the template until a record of it
     is durable.
3. **Restoration applies the state.**
   - The mob is spawned from its template at the saved level. The
     template's minted items and equipment are then replaced with copies
     of the saved ones.
   - Experience is restored, stats are recalculated, and the mob starts at
     full health and mana.
   - If an elite roll changed the level, the saved level is put back.
4. **Snapshot seams** (live mob to record, then save):
   - **`ItemOwnership`** on a tracked companion instance: after a give,
     get, drop, and so on.
   - **Plugin `OnSave`** (autosave, shutdown, copyover): every live
     tracked companion is refreshed before the store is written. This also
     catches changes that queue no event: the mob's own `gearup`, a broken
     item, or experience.
   - **Leader `PlayerDespawn`** (leaving the game): every companion is
     snapshotted and saved, then its live mob is removed from the world.
     A reverted, lingering companion could otherwise be killed for gear
     that its record would restore again, which would duplicate it.
   - **Companion `MobDeath`:** gear is cleared from the state (it dropped
     into the corpse or was lost with the body) and the level is kept.
     This stops restoration from bringing back gear that is now on the
     ground. The death penalty and resurrection come in the
     death/resurrection phase.
5. **Dismissal and desertion** remove the record and its state. The
   companion leaves with its gear, as it does today. Dropping the gear
   would let a player summon, dismiss, and repeat to farm template gear.
   Stable IDs are already never reused.
6. **Lifecycle state is deferred** to the death/resurrection phase, which
   defines the states. Until then a dead companion still returns at the
   next login, now without gear.
7. **Surfaces:**
   - `company status` shows each companion's level.
   - The new `company gear <member>` lists what a companion is wearing and
     carrying, read from the record.

## Durable model

`companies` store, per companion (all `omitempty`):

```yaml
state:
  level: 3
  experience: 1200
  equipment: { weapon: {itemid: 10002, uuid: ...}, ... }
  items: [ {itemid: 30004, ...} ]
```

It is decoded by the existing registry decoder (legacy records have no
`state`). No clock access: nothing reads or advances rounds.

## Module and seams

- `internal/company`: `MemberState` (with `Clone`), `Companion.State`,
  deep copies in `Get`/`Clone`, and `Registry.SetState(leader, id, state)`.
- `modules/company`:
  - `Runtime.Spawn(leader, room, template, *MemberState)`,
    `Runtime.Snapshot(instance)`, and `Runtime.TemplateState(template)`;
  - `ensureState` (the legacy upgrade);
  - `refreshSnapshot`/`refreshAll`;
  - the `ItemOwnership`, `PlayerDespawn`, and extended `MobDeath`
    listeners;
  - `OnSave` refresh;
  - the `company gear` view.
- The company module stays on the game loop and has no lock, as today.
  Nothing new crosses module locks.

## Constraints and deferrals

- **Crash window:** a gear change is durable at the next seam. A crash
  between a give (not followed by a save) and the next autosave restores
  the last saved state. That is the same class as the engine's
  give-then-crash window; a give triggers an immediate save, which keeps
  the window small.
- **Mob gear placement:** the mob's own `gearup` after a give is caught
  at the next `OnSave` or logout. Until then a restore puts the item in the
  backpack rather than on the body. Nothing is lost or duplicated.
- **Out of scope:**
  - a way to take gear back from a companion;
  - levelling companions from experience;
  - death penalties;
  - recruiters (22c).

## Acceptance criteria

- **Domain:** deep copies (mutating a returned state never changes the
  registry), and `SetState` for unknown members.
- **Module:** each of these:
  - summon records template gear in the same save;
  - a failed summon save leaves no state and destroys the mob;
  - restore passes the saved state;
  - the legacy upgrade saves before spawning, and a failed save neither
    spawns nor keeps state;
  - `ItemOwnership` refreshes and saves;
  - `OnSave` refreshes;
  - `PlayerDespawn` snapshots, saves, and removes live mobs;
  - `MobDeath` clears gear and keeps the level;
  - dismissal drops the state;
  - a YAML round trip of the store.
- **Wiring,** with real mobs from a fixture world with template gear,
  through `usercommands.TryCommand` and `events.ProcessEvents`:
  - summon;
  - `give` an item to the companion;
  - leader `PlayerDespawn` (the mob is gone, the record holds the item);
  - a simulated restart or copyover (the registry reloaded from the real
    store, instances cleared), then `PlayerSpawn`. The restored mob has
    the given item and the saved level, and there is exactly one of each
    template item;
  - a companion death, after which the next restore has no gear.
- `go test -race ./...`, `make generate`, `make validate`.
