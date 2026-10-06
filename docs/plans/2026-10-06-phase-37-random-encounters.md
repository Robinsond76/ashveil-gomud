# Phase 37: random room encounters, zone bands and drop tables

Implements the owner's encounter contract (2026-10-05) from the
[random room encounters design](../designs/2026-10-01-random-room-encounters-design.md)
and loot slice 3 of the [loot design](../designs/2026-10-05-loot-system-design.md),
on 35d's [boss shape](../designs/2026-10-06-phase-35d-combat-feel-design.md).
Branch `claude/37-random-encounters-ekkcx5`. Measurements:
[phase 37 measurements](2026-10-06-phase-37-measurements.md).

## What ships

- **`internal/encounters`** (pure): zone encounter config (`band`,
  `entrychance`, named weighted `tables`), room setting, validation with
  diagnostics, weighted pick, the entry-chance roll, level planning from the
  band, and the post-battle grace. No engine type.
- **Content model.** `zone-config.yaml` gains `encounters:` (band, default
  chance, tables) and `loot:` (item-level range, tier). A room opts in with
  an `encounter:` block (`enabled`, `table`, `chance`); a zone's tables alone
  enable nothing, a room `chance: 0` overrides inheritance (`*int`, so zero
  and absent differ). Validation drops only the affected composition and logs
  it once per zone: unknown or solitary templates, bad weights and counts,
  sizes outside 2-4, a boss without 2-3 escorts, a healer in a four-foe group
  or a boss's escorts, and healer groups above a fifth of a table's weight.
  A solitary template may be a boss composition's boss.
- **Trigger (`modules/encounters`).** Rolls on ordinary steps (the `walking`
  step seam `Go` already calls) and journey arrivals (new
  `walking.AddArrivalListener`, fired by the expedition module after the
  record is gone). Nothing else rolls: not look/scout, login, spawn,
  relocation, a failed move, a party follower's step, or while fighting,
  downed, travelling, camping, already holding a group, or when the room holds
  four groups. The mover's company is the roll owner.
- **Spawn (`enemyparty.SpawnEncounter`).** Builds every foe or none, at the
  planned levels; sets `EncounterOwner`/`EncounterID`; shares `engage` with
  `SpawnAmbush` but rolls **no surprise** (a sudden appearance). Ordinary
  groups: 2-3 foes at band low .. high-1, four foes at band low. Boss: `boss`
  composition, boss at low+2 with +75% HP and `Boss` set (38a's hex resist),
  coordination tier 1 for the whole group (Rabble, no strategy), escorts at
  band low.
- **Ownership.** `parties.ReservedFrom(owner, user)` gates `attack`, mob
  `lookfortrouble` and the "notices you" path: only the owner, their party
  and accepted alliance may fight or draw an encounter's foes.
- **Cleanup.** Won groups retire; when a battle ends or a round passes with
  no participant (owner downed, gone or in another room, no battle involving
  a foe) survivors are removed at once after a battle, else after 120 s. A
  foe is never removed while a battle involves it.
- **Grace.** Two skipped eligible entries and 30 real seconds after any
  battle, saved per leader (plugin file `encounters`), started fresh for new
  characters, dropped on purge.
- **Drop tables (`internal/loot/drops.go`, `mobcommands/drops.go`).** Zones
  with a `loot:` block or an encounter band opt in; others drop exactly as
  before. Per death each contributing company rolls for itself: ordinary
  (12% one item), elite (30%, Rare+ x2), boss (two items, first Rare+, Rare+
  x5, three goods rolls), and the group cache once for the last foe of an
  encounter (20% an item, 1-2 goods from the foe's `lootcategory`, gold by
  level band, much more with a boss). Item level is the source's level held
  to the zone range (band .. band top+2 by default); the tier follows it
  (1 at 1-9, 2 at 10-19). Personal loot: the claimant's rolls join the corpse,
  every other contributing company gets a corpse it alone claims. The battle
  summary gains a `Spoils` line (a bounded in-memory ledger read at fight
  end).
- **Pilot content** (placeholder until world building): Dark Forest (band
  5-7, 15 road rooms, forest-ogre lair) and Catacombs (band 10-12, 12 rooms,
  lich lair, one healer group in five). The forest ogre and lich carry the
  telegraphed Crushing Blow; the five shipped solitary bosses are flagged
  `boss: true`.
- **Help and tutorial.** New `help encounters` (indexed under combat with
  aliases); `combat`, `loot`, `scout`, `retreat`, `party`, `travel`, `ambush`
  and `battle-summary` updated; a Departure hint; `scout` says when a room can
  spring an encounter.

## Decisions taken (defaults, no owner question needed)

- Boss escorts are 2-3 (35d), superseding the contract's "up to 4".
- Healer groups: at most 20% of a table's weight, enforced by validation.
- Random encounters give no surprise round; `help ambush` says so.
- Boss lairs are rooms with a high-chance `lair` table and the same grace.
  There is no per-boss respawn clock yet (see deferred).
- Cache goods come from the last foe's `lootcategory` (no composition kind
  table yet); `kind` is recorded for the scent-masking consumable later.

## Pacing and rest placement

At the default 15% a battle is followed by two skipped entries and then a 15%
roll per entry: about **8.7 entries between battles** (pinned by
`TestPacingGapBetweenBattles`). The 35d mana run leaves a cleric about 6%
mana after the third fight, which is roughly 26 entries into a road. Content
rule: **a rest point (inn, camp or safe room with an inn) at least every ~20
eligible entries**, and no more than a third of a zone's road rooms enabled.
The pilot zones follow it (15 of 135 and 12 of 69). Zone bands are not
retuned: the settled mana result stands and is recorded in the measurements.

## Deferred, with reasons

- **Bad-luck protection and smart loot** (loot slice 3): bad luck guards a
  guaranteed legendary or set piece and none exist until slice 5; smart loot
  needs the company's families. Autoloot filters by rarity/value/type are a
  UI slice better built with 36c's goods economy.
- **Durable unresolved groups.** Mobs and battles are runtime-only in this
  codebase (see `internal/battle`), so a restart ends an unresolved group
  without reward; the grace persists. The design's prepare/commit recovery
  needs durable mobs and is out of scope; recorded in help.
- **Wandering parties moving as one and migrating existing spawn lists** (the
  design's content migration) wait for world building. Existing spawns are
  untouched.
- **Scent-masking paste and a boss respawn clock.**

## Tests

Pure: `internal/encounters` (validation, healer cap, weighted pick, chance
inheritance and zero override, level planning, grace, pacing),
`internal/loot` (profile, tier, base pick, rules by source, rarity boost,
goods, gold, spoils ledger). Integration: `modules/encounters` through the
real `usercommands.Go`, the step and arrival seams, real spawn, ownership
gates in `attack` and `lookfortrouble`, cleanup, grace save/load/purge,
shipped content validation; `modules/expedition` arrival seam;
`internal/mobcommands` real `Suicide` for cache once, double kill, boss,
allied personal loot; `modules/company` a real fight ending in one cache and
the spoils line; `internal/combatstream` and `internal/usercommands` (help,
scout). Balance cells: `TestBalanceEncounterShapes` (opt-in).

## Gates

`make generate`, `make validate`, `go test -race ./...`, `make js-lint`.
