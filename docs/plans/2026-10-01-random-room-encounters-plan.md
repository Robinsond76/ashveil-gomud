# Zone room encounters — implementation plan

Design: [zone room encounters and wandering parties](../designs/2026-10-01-random-room-encounters-design.md).
Status: planned only; implementation awaits design approval and scheduling.
Work from an isolated feature worktree. Read root and owning package/module
AGENTS.md before edits. Keep each slice reviewable; do not convert the whole
world before the pilot demonstrates acceptable frequency and persistence.

## 1. Confirm decisions and audit consumers

- [ ] Settle proposed 15% chance, two-entry/30-second grace, four-groups-per-room
  cap, reserved random-group targeting and no free surprise strike.
- [ ] Trace ordinary moves, arrivals, followers, allied moves, scripted moves,
  death relocation, login and retreat ordering through real handlers.
- [ ] Trace hostility, grouping, ally joining, commands, damage, kill rewards,
  corpse access and disconnect/restart policies; record enforcement seams.
- [ ] Inventory each combat zone's ordinary spawns, roamers, quests and safe
  rooms. Select the pilot zone and actual enemy compositions/levels.

## 2. Content schema and pure policy

- [ ] Add zone tables, typed composition/kind, explicit room opt-in and optional
  chance override preserving zero versus absence.
- [ ] Implement validation/reload diagnostics and deterministic RNG injection.
- [ ] Implement eligibility, grace, weighted selection, limits and future
  beast-probability reduction without increasing other encounters.
- [ ] Test boundaries, missing references, invalid groups, chance 0/100 and
  weighted outcomes without flaky statistical assertions.

## 3. Durable state before spawning

- [ ] Add persisted owner grace, movement IDs and encounter prepare/commit/
  terminal records with stable logical identities and recovery strategy.
- [ ] Save composition, participant ownership and living enemy state; reconcile
  runtime mob IDs rather than persisting them as primary identity.
- [ ] Integrate reward/death markers so failure or restart cannot duplicate XP,
  loot, groups or decisions. Test recovery at each transaction boundary.

## 4. Movement trigger and shared encounter service

- [ ] Add a narrowly typed eligible-arrival signal after movement/placement and
  arrival text; emit only from authorized ordinary-move and expedition paths.
- [ ] Deduplicate one roll per coordinated leader movement; exclude followers,
  spawn/login/teleport/recovery/retreat and failed movement.
- [ ] Reserve capacity and spawn full mixed parties atomically, with unique
  group identity, stable name, formation and no ordinary respawn/wander.
- [ ] Check existing battle/hostility before rolling and again before commit.
- [ ] Preserve travel ambush behavior if sharing construction code; add
  regression tests for existing pair/solitary spawns and arrival grace.

## 5. Battle ownership, rewards and cleanup

- [ ] Enforce encounter participant eligibility in aggression, target changes,
  attack/cast/action commands, damage, allied participation and reward access.
- [ ] Use ordinary battles, pacing, formation and death accounting without a
  free surprise turn. Verify all attacker/victim directions and guardians.
- [ ] Wire victory/flee/retreat/death/disconnect/abandonment cleanup, waiting for
  participating allies before deleting survivors; release capacity once.
- [ ] Restore unresolved groups through save/load and copyover without rerolls,
  healed enemies, repeated attacks or duplicate payouts.
- [ ] Add multiplayer integration scenarios: two independent entrants in one
  room, an ally joining, a bystander, capacity exhaustion and owner departure.

## 6. Public wandering parties

- [ ] Add leader-driven group movement with shared eligibility and preserved
  identity/formation; do not merely enable independent MaxWander per mob.
- [ ] Enforce zone/safe-room boundaries and no wandering during engagement.
- [ ] Verify partially blocked exits, leader death, survivors and nearby random
  groups cannot fragment or merge parties incorrectly.

## 7. Pilot content, help and tutorial

- [ ] Author one wilderness table and opt-in room set with level-appropriate
  mixed compositions and clear danger descriptions; retain public roamers.
- [ ] Remove only superseded routine static spawns after quest/boss audit.
- [ ] Add encounter risk to scout/assessment and relevant browser presentation.
- [ ] Ship `help encounters`, hub/related help updates, keyword aliases and
  Departure tutorial pointer. Test help rendering and pointer resolution.
- [ ] Playtest ordinary walking and expedition arrivals; measure battles per
  eligible entry, grace feel, difficulty, supply usage and random/visible mix.

## 8. Remaining zones and completion gate

- [ ] Expand proven tables/rooms to suitable combat zones; document safe-only
  exceptions, retained quest/boss encounters and deliberately sparse roamers.
- [ ] Run focused integration/persistence tests during work, then the required
  make generate, make validate and full race suite at the final code state.
- [ ] Obtain independent full-phase review; verify findings and fix real issues
  with regression tests, including multiplayer and transaction failures.
- [ ] Record measured tuning, review and exact checks in PROJECT_STATUS.md;
  integrate only the finished reviewed branch and update player help together.
