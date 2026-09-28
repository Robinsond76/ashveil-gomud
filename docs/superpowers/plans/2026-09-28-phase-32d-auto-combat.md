# Phase 32d: Automatic Combat by Strategy — Plan

Design: [32d design](../specs/2026-09-28-phase-32d-auto-combat-design.md)
(the owner's decisions are recorded there, 2026-09-28).

Branch `phase-32d-auto-combat` (worktree `.worktrees/phase-32d-auto-combat`),
from `claude/confident-pasteur-2n0e2x` at `f418e1d`. Keep out of 32f's
files: `modules/mount`, `modules/encumbrance`, and `modules/company`'s
inventory/provision code.

## Task 1: Strategies, pure (`internal/strategy`)

- [x] Tests first (`internal/strategy/strategy_test.go`): parsing roles
  and rules (and aliases), `assist` refused for the player; defaults by
  archetype (cleric healer, wizard caster, else fighter; weakest); each
  rule's `Pick` over plain `Foe` values: weakest, strongest, wounded
  (fraction), nearest, furthest (front row first, left to right),
  leader, assist, defend (the foe striking our most hurt), ties by
  formation; the fallback (choice out of reach → nearest reachable;
  none reachable → front-most; `spell` picks ignore reach); the role
  decisions (`Decide`): a healer heals the most hurt below half, group
  heal at two or more, swings otherwise; a caster's area spell at two
  foes, else single; mana and known spells gate each; spells outside
  the configured list never cast.
- [x] `strategy.go` (types, parse, describe, defaults), `pick.go`,
  `decide.go`, `provider.go` (the `For` seam with the default).

## Task 2: The module and `strategy` command (`modules/strategy`)

- [x] Tests first (`modules/strategy/strategy_test.go`,
  `purge_test.go`): registry set/clear/default not stored; load/save
  round trip through a fresh module; `UserPurged` drops the user; the
  command: list, one character, set role, set rule (both forms),
  `default`, unknown member or rule, `assist` for `me`, a role it can't
  do yet warns; refused in a battle.
- [x] `modules/strategy`: plugin, `SetOnLoad`/`SetOnSave`, config
  (`AutoSpells`), the command, provider registration; `make generate`.
  Companions resolved through `internal/company` (the record and the
  formation's member keys) and the company's own selector rule.

## Task 3: Companion spells, grants, and mana (`internal/archetypes`, `modules/archetype`, `internal/hooks`)

- [x] Tests first: `CompanionSpells` by level, and a spell outside the
  archetype's schools rejected at load (`internal/archetypes`,
  `modules/archetype`); the shipped config grants wizard `mm` and cleric
  `heal`, and an existing wizard gets it at `PlayerSpawn`
  (`modules/archetype/wiring_test.go`); a companion regains mana every
  third round out of combat and not in it
  (`internal/hooks/companion_mana_test.go`).
- [x] `Archetype.CompanionSpells`, `archetypes.CompanionSpells(id,
  level)`; config (cleric heal 1, healall 5; wizard mm 1, sparks 5);
  `GrantSpells` for wizard and cleric; `regenCompanionMana` in
  `NewRound_AutoHeal.go`. Found and fixed on the way: GoMud's
  `Character.Heal` added the health amount to mana (regression test in
  `internal/characters`).

## Task 4: Aiming by rule (`internal/enemyparty`, `internal/hooks`, `internal/usercommands`)

- [x] Tests first: `enemyparty.Aim` builds foes (reach, leader, what each
  foe strikes) and follows each rule (`groups_test.go`); wiring
  (`modules/company/wiring_strategy_test.go`, real commands and
  `DoCombat`): `attack <group>` starts the player and each companion on
  its own rule's choice; the upkeep re-aims by rule when a target falls;
  `assist` follows the player; `defend` finds the foe on the most hurt;
  a player alone re-aims by their rule.
- [x] `FirstAim` → `Aim(g, attacker, strategy, assistId)`; the upkeep's
  `chooseFromParty` and `retarget` take the member's strategy (re-read
  `assist`/`defend` each round); `turnAlone` uses it; `attack.go` sends
  each companion `attack #<its aim>`; `keepOnBattle` and the mid-round
  reassignment (`reassignWithinLostParty`) aim by rule too.

## Task 5: Healers and casters (`internal/hooks`)

- [x] Tests first (wiring, same file): a cleric companion heals the
  player below half, with mana, chant rounds, the chant line, and a
  `cast-start` event; a healer with no one hurt swings; a wizard player
  on `caster` casts Magic Missile with no command, spends mana, and turns
  back to their aim with no "turns toward"; out of mana, swings; a
  caster's spell keeps to its battle (29b2's hold).
- [x] `combat_strategy.go`: the strategy pass after `closeIdleBattles`
  (player and companions in battles), `startCast` (onCast, mana,
  `SetCast`, `SkillUsed` for players, `cast-start`), the aim-restore map
  used where `DoCombat` ends a cast (success, fizzle, held).

## Task 6: What a player may do in a battle (`internal/usercommands`, `modules/company`)

- [ ] Tests first (`internal/usercommands/battle_refusals_test.go`, and
  the wiring file): in a battle `break`, `eat`, `drink`, `use`, `equip`,
  `remove`, `go`/an exit, `formation move`, and a `strategy` change are
  refused and change nothing; `flee` still works and companions follow;
  out of a battle each works.
- [ ] `usercommands.InBattle(user)` beside `BattleUnderWay`; the checks
  in each command; `formation.go` (changes only).

## Task 7: Peaceful mobs (`internal/mobparty`, `internal/rooms`, `modules/tutorial`)

- [ ] Tests first: `groupKey` ignores the tag of a non-hostile mob, keeps
  its spawn group (`party_test.go`); the tutorial squad (non-hostile)
  is one group with a spawn group, and the Combat lesson's wiring test
  passes.
- [ ] `MobSummary.Hostile` from `rooms.GroupSummary`; `groupKey`; the
  squad's `SpawnGroup` in `modules/tutorial`.

## Task 8: Player help and tutorial

- [ ] Tests first: `help strategy` and its aliases render through `help`
  (`internal/usercommands/help_strategy_test.go`, after
  `help_combat_test.go`); `TestTutorialHelpPointersExist` covers the new
  pointer.
- [ ] `strategy.template` (new); updates to `targeting`, `combat`,
  `cast`, `flee`, `break`, `formation`, `mana`, `attack`, and any
  archetype page listing starting spells; `keywords.yaml` (`strategy`,
  aliases `strategies`, `gambits`, `roles`); the Combat lesson's hint
  (`modules/tutorial/stages.go`).

## Task 9: Review, verification, status

- [ ] Independent reviewer subagent over `git diff f418e1d..HEAD` with the
  design and the invariants; verify each finding, fix the real ones with
  regression tests.
- [ ] `go test -race ./...`, `make generate`, `make validate`, once.
- [ ] `docs/PROJECT_STATUS.md`: the work-log entry with **Review:**; the
  phase table; Known issues (32c's two items resolved; Magic Missile's
  odds for the owner; 32f's `company eat`/`drink` check).
- [ ] Merge into `claude/confident-pasteur-2n0e2x` and push.
