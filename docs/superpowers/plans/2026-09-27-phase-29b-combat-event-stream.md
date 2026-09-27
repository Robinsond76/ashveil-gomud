# Phase 29b: Combat Event Stream and Battle Summary — Plan

Design: [29b design](../specs/2026-09-27-phase-29b-combat-event-stream-design.md).
Spec: [29b spec](../specs/2026-09-26-combat-event-stream-design.md).

## Task 1: The stream (`internal/combatstream`)

- [ ] Tests first (`internal/combatstream/*_test.go`):
  - `Emit` stamps sequence numbers in order and calls subscribers;
    unsubscribe stops them; a subscriber may emit without deadlock;
  - `OpenFight` opens once, emits `fight-start`, merges new enemies and
    companions, and finds the same fight when the party id changes;
  - an event between the two sides gets the fight id and party id; one
    with an outsider gets 0;
  - a `death` with no source is credited to the victim's last damager;
    a second death for the same actor in a fight is dropped;
  - `EndFight` emits `fight-end` with the summary and forgets the fight.
- [ ] Tests first (`summary_test.go`): the fold (damage per side, spell
  damage, healing, most damage sorted, highest hit with crit, effects,
  kills, enemy endings: slain, fled, still standing) and `Render`
  (heading per outcome, `You` for the viewer, empty lines left out).
- [ ] Implement `event.go`, `stream.go`, `summary.go`.

## Task 2: Producers and fight tracking (`internal/hooks`)

- [ ] Tests first (`internal/hooks/combat_stream_test.go`):
  - `attackEvent` from an `AttackResult` (hit, miss, crit, weapon type);
  - `spellHealthDeltas`: a loss is a spell hit, a gain is a heal;
  - the interception fix: an interceptor dropped under 1 HP by
    `resolveInterceptedMobAttack` is in the round's affected mobs.
- [ ] `combat_stream.go`: `userRef`/`mobRef`, `emitAttack`,
  `emitStatuses`, spell before/after snapshots, `emitTargetChange`,
  `emitDeath`, `trackFightsAfterUpkeep`, `settleFights`, and the summary
  delivery with the `battlesummary` setting.
- [ ] Call them from the six attack sites, the spell paths, the flee
  path, the five reassignment sites, and `handleAffected`; add
  interceptors to the round's affected lists.

## Task 3: The setting (`internal/usercommands/set.go`)

- [ ] `set battlesummary` toggles (on when unset); `set` lists it;
  `help set` documents it.

## Task 4: Wiring through the real round (`modules/company`)

- [ ] `wiring_stream_test.go`, on 29a's `brawl` (shipped config,
  `plugins.Load`, real `company summon`, `attack`, `DoCombat`, idle mobs):
  - the event sequence and ids for the 5v5 to its end;
  - summary totals equal the attack and spell events per side; kills
    equal deaths;
  - the leader gets the summary text once, at the end;
  - `set battlesummary` off: no summary, events still flow;
  - a solo fight: fight id 0, no summary;
  - the clock is unchanged.
- [ ] The 27c practice-fight and 29a wiring tests pass untouched.

## Task 5: Docs, verification, review

- [ ] `internal/hooks` guide mentions the stream producers.
- [ ] `go test -race ./...`, `make generate`, `make validate`.
- [ ] Independent review of `git diff <base>..HEAD`; verify each finding,
  fix the real ones with regression tests.
- [ ] `docs/PROJECT_STATUS.md`: the 29b row and a work-log entry with
  **Review:**.
