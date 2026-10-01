# Zone room encounters and wandering parties

Status: owner-requested design and implementation plan, 2026-10-01. No gameplay
implemented or phase number assigned. Owner intent: zones contain rooms with a
chance of a surprise enemy battle, in the Final Fantasy style; this may become
the main source of ordinary enemy encounters while some wandering parties remain.
Numeric defaults and rules below are proposals. Implementation follows design
approval and the existing roadmap rather than silently reordering Phase 33.

## Player experience

Entering an encounter-enabled room sometimes reveals an enemy party and begins
a normal Ashveil battle. The room remains the battlefield: existing formation,
tactics, allies, guardians, pacing, loot, death, fleeing and company retreat
apply. There is no separate combat screen or arena teleport.

Example: “Shapes stir in the bracken. A pack of wolves closes around your company.”
Then the ordinary battle opener and combat lines follow. Arrival/room description
appears before the encounter reveal. Surprise means unexpected appearance, not
an automatic first strike, stun or bypass of defenses. Mechanical ambush
advantages would need a later design.

Risky rooms advertise danger in their descriptions and `scout`/zone assessment;
they do not list a randomly selected party before the roll. Some enemies remain
visible as wandering parties, so players can still inspect, pursue or avoid them.
Ordinary random foes grant normal XP and loot; bosses, recruiters, quest actors,
practice enemies and mandatory encounters remain deliberately placed.

## Trigger and frequency

- Roll once after a successful ordinary player move or completed expedition
  arrival into an enabled room. A solo leader is eligible too. Companion and
  allied followers do not independently generate extra encounters for one
  coordinated movement; designate the initiating leader as roll owner.
- Do not roll on `look`, `scout`, waiting, login, reconnect, spawn, resurrection,
  tutorial placement, admin relocation, scripted teleport or failed movement.
  Re-entering a room through ordinary movement can roll again after grace.
- Suppress rolls while any participating company is already in a battle, has
  an unresolved encounter, is retreating, or is completing a flee/return to
  safety. A visible party already engaging the entrant takes precedence;
  do not create a random group to wait behind it.
- Default chance: **15% per eligible entry** (roughly one battle per 6–7 rolls).
  Rooms may override it; recommended initial risk bands are 10%, 15% and 25%.
  No forced encounter after a fixed number of misses in the first release.
- After any battle ends, suppress new random encounters for **two otherwise
  eligible room entries and at least 30 real seconds**. Both must be satisfied
  before rolling again. Consume the two entry allowances while suppressing
  eligible entries. Persist this grace across zones and reconnects so it cannot
  reset by crossing a boundary. Fresh characters start with the same grace.
- No room-global cooldown: one player's fight must not clear danger for every
  other player. Apply per-owner and encounter capacity limits instead.
- Towns, inns, shops, churches, tutorials and designated safe rooms default to
  disabled. Wilderness/dungeon zones have explicit eligible rooms; never enable
  every room by default merely because its zone has a table.

Travel interruption combat remains available. Completing or returning from a
travel ambush receives the same battle grace, preventing immediate double fights
on arrival. Camp raids are a separate rest trigger and never arise from idle
room-encounter rolls.

## Content model

Extend zone configuration with named weighted encounter tables and a default
entry chance. Extend room configuration with an explicit encounter setting:
disabled, or enabled with a table reference and optional chance override.
A room chance of zero must override inheritance; absence and zero cannot be
represented by the same value. Zone tables alone do not enable rooms.

Illustrative schema (not existing YAML):

```yaml
# zone-config.yaml
encounters:
  entrychance: 15
  tables:
    woodland:
      - id: wolf-pack
        kind: beast
        weight: 60
        members:
          - mobid: 101 # illustrative only; bind real templates during content work
            count: 3
      - id: woodland-raiders
        kind: humanoid
        weight: 40
        members:
          - mobid: 102
            count: 2
          - mobid: 103
            count: 1
# room YAML
encounter:
  enabled: true
  table: woodland
  chance: 15
```

Each composition defines actual member templates/counts and optional bounded
level overrides. Default to parties of 2–5; solitary templates stand alone and
cannot be mixed into ordinary groups. Support mixed roles, such as front-line
raiders with an archer, using existing enemy formation rules. Set levels by zone
content and existing bounded zone scaling, not by silently matching the player.
Validate probabilities, positive weights/counts, table references, templates,
level bounds, group-size limits and solitary constraints at load/reload.
Unknown content disables the affected encounter with a clear diagnostic rather
than crashing or spawning an incomplete party.

A successful entry-chance roll chooses one composition by weight. Future
scent-masking paste reduces beast outcomes and moves their probability to
no encounter without increasing humanoid outcomes. Keep encounter kind explicit
so that consumable can integrate without matching names or retuning other foes.

## Shared world and targeting

Random foes appear in the actual shared room, visible to nearby players, as
one uniquely identified group. Reserve their aggression to the initiating
leader/company and explicitly participating allies using the existing allied
battle rules. A nonparticipant cannot steal the encounter, its XP or loot,
accidentally draw aggro, or have it appended to their battle queue. Shared-room
visibility alone does not imply private targeting: implementation must enforce
eligibility in commands, enemy targeting, damage and reward paths.

Spawned random groups have no wandering or normal respawn entries and never
merge with ordinary room spawn groups. Limit each leader to one unresolved
random group; propose a configurable limit of four active random groups per
room. Capacity exhaustion skips the encounter without stockpiling a later fight.
Reserve capacity before spawning. Either create the full composition and register
its ownership or roll back every mob; never leave half a group or consumed
ownership slot after an error.

On victory, reward through normal exactly-once death/loot paths. On successful
flee, retreat, leader death or abandonment, clear surviving random enemies once
all participating allies have left the battle; do not delete enemies while an
ally is still fighting. Survivors never become public wandering parties.
Corpses/rewards retain normal lifetimes and authorized recipient rules. Offline
leaders follow existing combat/disconnect policy; logout is not a new free flee.
An abandoned group with no participants expires after a bounded cleanup timeout.

## Wandering parties and content migration

Keep a smaller explicit set of public roaming groups per combat zone. They use
ordinary public hostility, rewards and spawn rates. Preserve party identity and
formation while moving: followers move with the party leader, rather than each
member's independent idle wander splitting the group. Respect room exits, zone
boundaries, safe-room exclusions, battle engagement and destination capacity.
No movement mid-battle; solitary roaming creatures can retain their own rules.

Audit existing spawn lists before reducing density. Convert only ordinary
combat fodder represented in encounter tables. Preserve quest requirements,
boss rooms, named enemies, faction inhabitants and intentionally placed fights.
Pilot one wilderness zone first, then migrate every suitable combat zone with
at least one eligible room/table and documented exceptions for safe-only zones.
Target random battles as the majority of routine encounters through playtest
measurement; do not implement an arbitrary encounter quota. Keep a visible
roaming group where suitable so exploration still offers deliberate targets.

## Inspected integration points and architecture

- `internal/rooms/zoneconfig.go` owns `ZoneConfig`; `internal/rooms/rooms.go`
  owns room configuration and existing spawn passes. Add opt-in content here.
- `internal/rooms/roommanager.go` emits `events.RoomChange` on movement.
  Its current contract lacks a movement reason. Do not treat every RoomChange
  as an eligible step. Add an explicit post-arrival encounter-eligible event
  or movement metadata for ordinary moves and expedition arrivals only.
- `internal/usercommands/go.go` and `modules/expedition/expedition.go` are
  inspected ordinary-move and completed-arrival producers. Coordinate arrival
  text, company placement and hostile engagement before the encounter decision.
- The expedition's `nativeMobSpawner.SpawnHostileEncounter` makes a pair (or
  solitary mob), assigns a SpawnGroup and issues attacks. Reuse/refactor its
  construction principles into a shared encounter service, but do not assume
  it already supports mixed compositions, private ownership or durable mobs.
- `internal/enemyparty` and `internal/battle` own group/battle boundaries;
  `internal/hooks/MobIdle_HandleIdleMobs.go` is an inspected wandering hook.
  Audit full aggression, action, ally and reward paths before implementing
  reserved targeting and party-coherent wandering.
- Prefer a pure `internal/encounters` policy/content layer and an auto-wired
  module for event integration and persistence. Narrow engine changes are
  necessary for trustworthy movement reasons, group ownership and eligibility;
  a generic RoomChange listener alone cannot enforce these invariants.

All state changes occur on the authoritative game loop. Never advance global
world time, block with sleeps or mutate the world from a timer goroutine.

## Persistence and recovery

Persist per-leader grace (UTC deadline and remaining skipped entries), movement
transaction IDs for deduplication, and unresolved encounter records with stable
logical group/member IDs. Runtime mob instance IDs are only bindings, not durable
identity. Save the selected composition, ownership/participants, room, living
members' current combat-relevant state and reward/terminal markers so reconnect
or reload does not reroll, heal foes or duplicate rewards.

`internal/battle` explicitly documents runtime-only battles and reconstruction
after restart/copyover; `mobs.SpawnGroup` is runtime-only too. Encounter durability
must therefore be new work, not an assertion that existing battles already save
it. Rebuild the same logical group from the durable record and reconcile into
the existing battle model. Review how HP, statuses and queued actions restore;
restore encounter state without replaying already-resolved strikes or loot.

Use prepare/commit/reconcile transitions for movement decision and encounter
creation. Duplicate events cannot roll twice. A recovered pending decision uses
its recorded outcome; partial construction is reconciled, never rerolled. Save
terminal/reward progress under existing durable reward/death patterns. Invalid
or removed content on recovery causes explicit safe abandonment and logged
cleanup, not an invented replacement party or reward.

## Help and acceptance

Ship `help encounters`, update `help combat`, travel, scout and flee/retreat
help where affected, index keywords/aliases, and add a Departure tutorial hint.
Explain danger rooms, randomness, grace, public wanderers, normal battle rules
and encounter ownership. Test help rendering and tutorial pointer resolution.

Acceptance requires real move/arrival, spawn, combat and reward integration
tests, not only random-policy unit tests. Verify safe rooms, all excluded
movement reasons, duplicate arrivals, followers/allies, existing hostility,
limits, full-spawn rollback, deterministic weighted selection, no double fights,
all cleanup outcomes, save/load/copyover and exactly-once rewards. Verify public
wandering parties move together without crossing safe boundaries or detaching
engaged members. Playtest frequency and party difficulty at representative levels.
