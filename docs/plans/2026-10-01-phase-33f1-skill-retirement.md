# Phase 33f1: Skill and Charm Retirement — Plan

Design: [33f design](../designs/2026-10-01-phase-33f-company-specialists-design.md),
"33f1: Skill and charm retirement". Worktree
`.worktrees/phase-33f1-skill-retirement`, branch `phase-33f1-skill-retirement`.

## Task 1: commands and code

- Unregister and delete the player commands `backstab` (review finding:
  it needed `sneak`'s hidden state), `bump`, `changeform`, `peep`,
  `pickpocket`, `portal`, `pray`, `scribe`, `sneak`, `tame` (and their
  `.md` notes); delete the mob `befriend` command. Keep mob `sneak`,
  `portal`, and `backstab`.
- Remove peep's passive health display in room details, the tame-learning
  roll on kills, the tame training hook, Tame mastery (`MobMasteries`,
  `Character.MobMastery`), `GetMaxCharmedCreatures`, the `tame` stat mod,
  `combat.ChanceToTame`, `Mob.IsTameable`, `actionpolicy.TameTarget`, and
  the `tameskill` battle recheck.
- Scripting API: drop `CharmSet`, `CharmRemove`, `CharmExpire`,
  `GetCharmCount`, `GetMaxCharmCount`, `IsTameable`, `GetTameMastery`,
  `SetTameMastery`, `GetChanceToTame` (functions, schema, TypeScript defs).
- Shops: `list` and `buy` skip `mobid` entries; the hiring branch and the
  charm cap check go.

## Task 2: world data

- Delete skills `changeform`, `peep`, `portal`, `scribe`, `tame`; the
  `tameskill` spell; the Bonecrafter's skeleton shop entry; the long
  whip's `tame` stat mod; the trainer entries in rooms 830 and 160; the
  obelisk's Portal lesson (room 871 script).
- Archetypes: rogue drops `peep`, wizard `portal`, ranger `tame`.
- Professions: Arcane Scholar (enchant, inspect), Merchant (trading,
  inspect), Treasure Hunter without peep; remove Explorer and Monster
  Hunter. Strip retired skills from the shipped admin user 1.
- Remove the `status-lite` peep panel layout and the web pet window's
  `tame` stat label.

## Task 3: migration and recovery

- `skills.Retired`/`TrainingCost`, `Character.RetireSkills`, and the
  archetype module's `retireSkills` on `PlayerSpawn`: refund 1+…+level per
  retired skill, remove it, notify, save; Protection above its new cap of
  3 is lowered with the difference refunded. Idempotent by construction.
- Charms are runtime-only; nothing creates a non-companion charm, so a
  restart/copyover leaves none. No sweep.

## Task 4: help and tutorial

- Delete help for `peep`, `portal`, `tame`, `changeform`, `scribe`,
  `hire`, `guide` (the unshipped newbie guide), `explorer`,
  `monster-hunter`; drop their keywords and the `sneak`, `scribe`, `hire`
  command aliases.
- Correct `skulduggery`, `protection`, `jobs`, `training-schools`,
  `arcane-scholar`, `merchant`, `treasure-hunter`, `ask`, `company`,
  `cargo`, `morale`, `list`.
- No tutorial lesson points to a retired skill; `TestTutorialHelpPointersExist`
  must still pass. No new pointer: the slice removes features.

## Task 5: tests

- `modules/archetype/retire_test.go`: refund through the real spawn
  listener, saved once, never twice; no archetype lists a retired skill.
- `internal/usercommands/retired_skills_test.go`: retired commands and
  help topics gone, corrected pages name none of them; a mercenary shop
  neither lists nor sells (fails on the old code); a world-data guard over
  trainers, professions, item stat mods, quests, and scripts.
- Adjust tests that exercised retired skills; regenerate the 29d combat
  outcome capture (the removed tame-learning roll shifted the seeded RNG;
  confirmed by restoring a dummy roll, which reproduces the old capture).

## Task 6: review and integration

Independent reviewer on the full diff; fix findings with regression tests;
record in Project Status; `make generate`, `make validate`,
`go test -race ./...`, `make js-lint`; merge to master and push.
