# Phase 27c: Tutorial Practice Fight

The third slice of the [Ashveil tutorial spec](2026-09-23-ashveil-tutorial-design.md),
on the [27a framework](2026-09-25-phase-27a-tutorial-framework-design.md) and
[27b](2026-09-25-phase-27b-tutorial-survival-camp-design.md).

**Split again.** 27a planned 27c as the practice fight, the Alignment
lesson, and the browser tutorial panel together. Each is its own system
(combat, alignment and standing, GMCP and web client), so:

- **27c (this document):** the practice fight (stage 6), and the
  death-in-course decision 27a deferred.
- **27d:** the Alignment lesson and the browser tutorial panel.

Open decisions were settled by applying this design's recommendations under
the owner's "carry on to 27c" instruction (2026-09-25).

## Prior-art check

- **Formation combat (11a–11c):** hostile mobs in a room sharing a `Groups`
  tag assemble into a party (`mobparty.Assemble`). The party's 3×3
  formation is filled front row first by descending effective HP. A
  player's or companion's attack is gated each round
  (`internal/hooks/combat_formation.go`):
  - front-row interception redirects an attack aimed behind the front row;
  - an attack out of reach is skipped, and the player sees "You can't
    reach that target from here.";
  - when a target falls, the attacker is re-targeted to a legal enemy
    (`reassignPlayerTarget`, `reassignCompanionTarget`).
- **Mob death** (`internal/mobcommands/suicide.go`) grants XP (split by
  party), shifts alignment, tracks kills, may teach taming, drops items,
  loot-table rolls, and gold, and fires `MobDeath` (kill counters,
  telemetry, companion death). `Suicide("vanish")` removes a mob with none
  of that.
- **Harmless foes:** the shipped `dummy` race (19) has `0d0` unarmed damage
  and no weapon slot. The old tutorial's training dummy (mob 58) used it,
  and gave XP on death.
- **Leaving the course** (27a): any move out ends it. Before Departure,
  that counts as a skip, including a death respawn. 27a left this for 27c
  to settle.

## Decisions

1. **A Combat stage before Departure**, in a new room 906 (the Practice
   Yard), appended to `TutorialRooms` (index 6). The order is Character,
   Company, Formation, Survival, Camp, Combat, Departure.
2. **A squad of straw soldiers.** Four practice foes share the Groups tag
   `practice-squad`, so they form one enemy party:
   - three straw footmen (mob 67, level 2) fill the front row;
   - a straw archer (mob 68, level 1, lower HP) stands behind them.

   They use the `dummy` race, so they deal no damage. No injury, death, or
   rescue can come from the fight.
3. **Practice foes give nothing.** A new mob field, `practice: true`,
   makes a beaten foe:
   - announce "is beaten and yields the field";
   - fire a new `mobcommands.OnPracticeBeaten` hook;
   - leave as `vanish` does.

   So there is no XP, alignment shift, kill count, taming, drop, loot, gold,
   or `MobDeath`. Others aiming at a beaten foe keep their aim, so the next
   round's reassignment (11b) turns them to a new legal target: the
   lesson's "target reassignment when an enemy falls", from real combat.
   The one who struck the killing blow ends their aim, as in any fight, and
   attacks again.
4. **The tutorial owns the squad.** The foes are spawned by the module, not
   by room spawns:
   - when the player enters the Combat room, or is placed there;
   - only if the player has none standing;
   - in the player's own copy, never elite.

   The foes a player has are held in memory, like the room copies. A
   resume spawns a fresh squad, and the old one is removed. Leaving the
   course or skipping removes any foes left standing.
5. **The gate is the result:** every foe of the player's squad was beaten
   (`OnPracticeBeaten`), by the player or a companion. The hints explain:
   - attacking (`attack footman`), and that the archer is shielded, so an
     attack at it is intercepted by the front row;
   - "You can't reach that target from here." and `formation reach
     <member>`;
   - moving a member between fights (the formation is set before
     combat);
   - automatic re-targeting when a foe falls;
   - HP and `conditions`;
   - that a sharpened edge is spent per strike, whatever the damage.
6. **Waiver:** `tutorial next` passes Combat when no squad could be spawned
   (the templates are missing).
7. **Death in the course is decided:** nothing in the course deals damage
   quickly. The practice foes are harmless, and only lingering in the open
   Weather Yard for a very long time could hurt. A death there is an
   ordinary death (Phase 25a): the player wakes at the church, and leaving
   the course that way counts as a skip, as before. The course can't be
   re-entered, and nothing it grants is lost that a graduate would keep.
   This is recorded in the module and the help.

## Constraints

- Never advances the world clock.
- No XP, gold, item, kill, or alignment gain from the fight; the squad is
  once per placement.
- Progress survives logout, restart, and copyover. The squad is rebuilt on
  resume.
- `tutorial next` and `tutorial skip` always get a player out.

## Acceptance criteria

- **Engine:** a practice mob's death grants no XP, alignment, kills, drops,
  or gold, fires no `MobDeath`, fires `OnPracticeBeaten`, and removes the
  mob. A non-practice mob is unchanged.
- **Module:**
  - the squad spawns once on entering or being placed at Combat;
  - beating every foe passes Combat, and beating another player's foe
    doesn't count;
  - a resume replaces the squad;
  - leave and skip remove it;
  - the waiver;
  - the view's checklist (foes beaten, n of 4).
- **Wiring** (through `plugins.Load` and the real combat round,
  `NewRound_DoCombat`):
  - the squad forms one party with the archer behind the front row;
  - an attack at the archer is intercepted by a footman;
  - beaten foes give no XP or gold;
  - the player is re-targeted when a foe falls;
  - beating all four passes Combat;
  - the clock is unchanged.
- **Shipped content:** seven rooms in stage order; the Practice Yard has no
  spawns; mobs 67 and 68 are practice, `dummy` race, in `practice-squad`.
- `go test -race ./...`, `make generate`, `make validate` pass.

## Implementation notes (2026-09-25)

- **The leader must stand in the grid.** Formation rules cover a leader's
  own attacks only once the leader is placed (`formation move me <row>
  <col>`); unplaced, they strike whoever they choose. The Combat hints lead
  with this, and the wiring test places the leader before attacking.
- **The squad's levels** are 3 (footmen, 16 HP) and 2 (archer, 11 HP): a
  level-1 `dummy` has 1 HP, too fragile to show anything.
