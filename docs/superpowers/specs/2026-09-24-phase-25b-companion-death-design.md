# Phase 25b: Companion Death and Resurrection

Implements the second half of the
[death and resurrection spec](2026-09-23-death-resurrection-design.md),
after [Phase 25a](2026-09-24-phase-25a-player-death-design.md) (player death
and the church return).

A companion who dies stays dead on the roster. Its leader has a rescue
allowance of three game days of **their own online time** to bring the
company to a city church or village shaman and `resurrect` it, at the cost
of one of the companion's levels. When the allowance runs out, the companion
is lost for good: its name is kept on a roll of the fallen and its roster
slot is freed.

The open decisions below were settled by applying this design's
recommendations, as in Phases 19b–25a, under the owner's "implement phase
25b" instruction (2026-09-24).

## Prior-art check

- **Companion death today** (`modules/company`): `onMobDeath` untracks the
  instance and `recordCompanionDeath` clears the record's gear and gold,
  keeping the level. The next `PlayerSpawn` (`restoreForLeader`) respawns
  the companion from its record.
- **Engine mob death** (`internal/mobcommands/suicide.go`): queues
  `events.MobDeath` first, then drops the mob's carried items and gold, and
  each worn item with `ItemDropChance` percent, into the corpse or room. A
  worn item that doesn't drop is destroyed with the mob. `perma-gear` mobs
  drop nothing. Listeners run after the command, when the instance is gone,
  so today nothing can tell which worn items survived.
- **Roster consumers** (`survival.CurrentRoster`): survival travel exertion
  and rest recovery apply to every rostered member, present or not;
  walking and exposure drain every member; camp tiers grant every rostered
  companion durably; the inn charges per rostered member; survival
  provisioning resolves members by roster.
- **Alignment** (Phase 21a): drift, desertion, and the company average use
  every companion in the record.
- **Chemistry** (Phase 24): only present members (a tracked, live, attached
  mob) accrue; a dead companion's service pauses.
- **Formation**: cells are keyed by member; a companion with no live mob is
  simply absent from combat.
- **Settlement registry** (Phase 25a, `internal/death`): zone, kind
  (`city`/`village`), service room with the `church`/`shaman` tag. Villages
  never set the checkpoint. Room 18's clergyman is mob 4; the chapel's
  Sister Maren is mob 65.
- **Time**: `gametime.GetDate().RoundsPerDay` (900) and
  `configs.GetTimingConfig().RoundSeconds` (4) make three game days 10,800
  seconds. Company saves happen on commands, on the leader's logout, and on
  every plugin `OnSave` (autosave every `RoundsPerAutoSave`, shutdown, and
  copyover).

## Decisions

1. **A durable death on the companion.** `Companion.Death`
   (`*CompanionDeath`, nil when alive) holds the operation ID
   (`cdeath-<leader>-<companion>-<round>`, which only reads the round), the
   allowance granted and the seconds remaining, and the formation cell the
   companion held. The level and gear stay in `State`, which already is the
   companion's snapshot. A dead companion keeps its roster slot, ID,
   disposition, archetype, service, and survival record.
2. **What the body keeps.** `events.MobDeath` gains `KeptWorn`,
   `KeptItems`, and `KeptGold`: what the engine's own drop rules left on the
   body. `mobcommands.Suicide` rolls the worn-item drops before queueing the
   event and then drops exactly those, so the rolls and the event agree
   (an upstream-shaped change: same rolls, same odds). A dead companion's
   record keeps only the kept worn items (by slot), the kept carried items,
   and the kept gold; everything that dropped is in the corpse or room, and
   is never restored. Recruit templates use `itemdropchance: 0`, so their
   worn gear survives death and returns with them.
3. **Marking the death.** `onMobDeath` for a tracked companion: untrack,
   set the kept gear and the event's level, clear its formation cell
   (remembered in the death), snapshot the allowance
   (`AllowanceDays × RoundsPerDay × RoundSeconds`, default 3 days; later
   calendar changes don't touch it), save once, and tell the leader how long
   they have. An already-dead companion is left alone. A failed save leaves
   the death in memory for the next save, as 22b did for the gear.
4. **Dead means absent everywhere.**
   - `restoreForLeader` never spawns a dead companion (login, copyover).
   - `survival.MemberRef` gains `Dead`; the company's `Roster` sets it.
     Survival travel exertion and rest recovery skip dead members;
     provisioning refuses them; `survival status` marks them. Walking and
     exposure drain skip them; camp tiers don't grant them; the inn doesn't
     charge for them.
   - Alignment drift, desertion, and the company average skip them.
   - Chemistry already needs a live mob; relocation (25a) already needs one.
   - A dead companion still counts toward the company cap until it is
     resurrected, lost, or dismissed. Dismissing one is allowed and
     archives nothing.
5. **The allowance counts only online time.** The company module keeps, in
   memory only, an anchor per online leader with a dead companion. Once a
   round (`NewRound`), each such leader's dead companions are charged the
   real time since the anchor (monotonic clock, injected in tests), capped
   at 60 seconds per step so a stalled game loop isn't spent, and the anchor
   moves on. `PlayerSpawn` (login and copyover) starts a fresh anchor;
   `PlayerDespawn` charges up to the logout and drops it, and its save
   records the remaining time. Offline time, restarts, and downtime are
   never charged. The remaining time reaches disk with every company save,
   so a crash refunds at most the time since the last save (at most one
   autosave interval) and can never expire a companion early. A warning is
   sent once when the remaining time crosses 30 minutes; the leader is
   reminded of each dead companion at login.
6. **Expiry.** When a charge reaches zero, the companion is lost: it goes
   through the dismissal path (record, survival, formation, with rollback
   on a failed save; its ID stays spent through the high-water mark), and
   in the same save its name, template, level, and death operation ID are
   added to `Record.Lost` (the ten most recent). The leader is told.
   `company status` lists the lost. A lost companion can't be resurrected,
   because it is no longer on the roster.
7. **`resurrect` lives in the death module**, which owns the settlement
   registry. `resurrect` alone lists the dead with their time left.
   `resurrect <member>` needs the room to be a registered settlement's
   service room with its tag (a city's church or a village's shaman), and
   that settlement's service keeper (new config `ServiceMobId`: 4, 65, 66)
   present and alive in the room; a settlement with no keeper configured
   can't resurrect (warned at load). Not while fighting. It then calls a
   new optional provider, `company.ResurrectionProvider`.
8. **Resurrection is one company save.** `ResurrectCompanion(leader,
   selector, room)`: charge the allowance first; a companion with no time
   left is lost now (decision 6) and the command says so. Otherwise take one
   level (floor 1; experience 0), clear the death, return the companion to
   its old formation cell if it is still free, and save. A failed save
   restores the record and refuses. Only then is the mob spawned into the
   room, from the record, attached to the leader; if that spawn fails the
   companion is alive but awaiting restoration and rejoins at the leader's
   next login. So a crash at any point leaves exactly one of: dead with its
   time, or alive once (never a clone, never a second level). No gold fee.
9. **Village shaman content.** A new village zone, **Fernhollow**: the
   hamlet green (room 2008, west of the Fork at the Black Oak, 2002) and the
   shaman's lodge (2009, `shaman` and `indoor` tags) with **Old Wenna**
   (mob 66: non-hostile, no gear, no gold, no drops). The death config lists
   Fernhollow as a village with keeper 66, and adds keepers to the two
   cities (4 at the Sanctuary, 65 at the chapel). Fernhollow never becomes a
   checkpoint.
10. **No new locks.** Everything runs on the game loop; the company module
    has no mutex and the death module's `mu` stays a leaf.

## Scope

**In scope:** `events.MobDeath` kept-gear fields and the pre-rolled drops;
`internal/company` death record, `Lost`, and `ResurrectionProvider`;
`modules/company` death marking, allowance clock, expiry, resurrection,
exclusions, status and login text; `internal/survival.MemberRef.Dead` and
its consumers in `modules/survival`, `modules/walking`, `modules/exposure`,
`modules/camping`; `modules/death` keeper config and the `resurrect`
command; Fernhollow content; docs.

**Deferred:** a gold fee; a browser/company-panel view of the dead (the
information-surfaces phase); corpse-carrying or body location.

## Constraints

- Never advances or writes the world clock; operation IDs only read it.
- Survives restart and copyover: the death, allowance, and loss are in the
  company file; the anchor is deliberately in memory only.
- Exactly once: resurrection and expiry each remove the death in one
  company save; the survival record follows through the existing rollback
  and `ReconcileCompanyRosters` on load.
- No new locks.

## Acceptance criteria

- Engine: `MobDeath` reports exactly the worn items that didn't drop
  (drop chance 0 keeps all, 100 keeps none), and the carried items and gold
  only for `perma-gear`.
- Domain: `Put` keeps a record with only `Lost`; `Get`/`Clone` deep-copy
  `Death` and `Lost`.
- Company: a companion's death marks it dead with the allowance, kept gear,
  and cleared cell, and saves; login doesn't spawn it; drift, the average,
  and the roster flags exclude it; the allowance is charged only while the
  leader is online, capped per step, not across logout, and warned at 30
  minutes; reaching zero loses it (archived, slot freed, ID not reused,
  survival removed) with rollback on a failed save; resurrection costs one
  level (floor 1), restores the cell when free, spawns once, rolls back on a
  failed save, and refuses the lost, the living, and the unknown.
- Survival and consumers: exertion, rest, drains, tiers, and inn price skip
  the dead; provisioning refuses them.
- Death module: `resurrect` refuses outside a service room, without the
  keeper, and while fighting; a church and a shaman both work; the listing
  shows time left.
- Wiring: through `plugins.Load` with the shipped config and rooms, a
  companion dies through the real mob `suicide`, isn't restored on relog,
  loses no time offline, is refused at a room that isn't a service room, and
  is resurrected by the real `resurrect` command before Sister Maren, one
  level lower with its kept gear; a second companion is raised by Old Wenna
  at the Fernhollow lodge; a third expires and is archived. Fernhollow never
  sets the checkpoint. The clock never moves.
- `go test -race ./...`, `make generate`, `make validate` pass.
