# Phase 29a: Combat Fixes from the 5v5 Simulation — Plan

Design: [29a design](../specs/2026-09-27-phase-29a-combat-fixes-design.md).
Findings: [29a findings](../specs/2026-09-26-combat-fixes-design.md).

## Task 1: Live enemy-party adapters (`internal/enemyparty`)

- [x] Tests first (`internal/enemyparty/enemyparty_test.go`):
  - `Parties` skips charmed mobs and groups by tag;
  - `PartyOf` finds a member's party;
  - `Alive` marks gone and dead members as not alive.
- [x] Move `hostileMobSummaries`, `effectiveHP`, `resolveEnemyParty`, and
  `aliveMapForParty` out of `internal/hooks` into exported
  `Parties`/`PartyOf`/`Alive`/`EffectiveHP`. The hooks call sites use
  them, and the existing hooks tests keep passing.

## Task 2: Engagement upkeep (`internal/hooks/combat_engagement.go`)

- [x] Tests first:
  - **Pure helpers:**
    - `plainAttack`/`retargetable`/`attackType`: a spell, a backstab, or
      a shot through an exit is left alone;
    - `classifyPartyTarget`: idle, living, dead, bystander, or gone;
    - `chooseFromParty`/`legalAgainstParty`: the weakest legal, or fail
      open when unplaced;
    - the leader's turn text;
    - `reassignWithinLostParty`: never a bystander.
  - **Through `hooks.DoCombat`:** these tests live in Task 4.
- [x] `upkeepEngagements()`, called first in `DoCombat`. For each online
  leader's room:
  - assemble the parties there;
  - find the engaged ones;
  - refresh hostility;
  - retarget company members, then party members.
- [x] `reassignEnemyTarget` is replaced by `reassignWithinLostParty`: the
  lost target's party, else a party hostile to the leader, never a
  bystander. Unplaced company attackers fail open.
- [x] Added during implementation (see the design doc):
  - `resolveAttackTarget` fails open for an unplaced target, which could
    never be struck before;
  - a same-room `Shooting` aim counts as an ordinary attack;
  - the upkeep keeps any aim the gate lets through, interception
    included.
- [x] Messages:
  - to the leader: `You can't reach X from here. You turn on Y.` or
    `You turn on Y.`;
  - to the room, for a mob: `X turns on Y.`

## Task 3: `formation reach` in and out of a fight (`modules/company`)

- [x] Tests first (in the wiring test):
  - in a fight, the reachable enemies with the member's own reach;
  - out of a fight, the labelled demonstration.
  - The "can't reach any of the enemy" branch has no test: with the
    party's front row always filled, the wiring world can't produce it.
- [x] Implement it with `enemyparty` and `combat.ResolveReach`.

## Task 4: Wiring through the real round (`modules/company/wiring_combat_test.go`)

- [x] Replace the scratch reproduction with regression tests through
  `plugins.Load`, the shipped config (`configs.ReloadConfig()`), real
  `company summon`, `formation move`, and `attack` commands, and
  `hooks.DoCombat`, `hooks.IdleMobs`, and `HandleIdleMobs` with queued
  mob commands:
  - **F1:** the leader in column 3 swings at a legal bandit in round 1
    and is told of the turn;
  - **the kill case:** the leader and the killer continue after the
    captain dies, and the fight runs until every bandit is dead;
  - **F2:** all five bandits join; none is left idle beside the company
    when the fight ends;
  - **the enemy side:** a bandit turns from an unreachable company target
    to a legal one;
  - **unplaced:** an unplaced company keeps fighting as its targets fall,
    and can be struck (`TestUnplacedCompanyFightsAndCanBeStruck`);
  - **F3:** level-1 companions have more than 1 HP;
  - **F4:** `formation reach me` mid-fight names bandits.
  - **the clock:** it is unchanged throughout.

## Task 5: Docs, verification, review

- [x] Update `internal/hooks` and `modules/company` guides if they
  describe the gates or `formation reach`.
- [x] Run `go test -race ./...`, `make generate`, and `make validate`.
- [ ] Independent review of `git diff <base>..HEAD`; verify each finding
  and fix the real ones with regression tests.
- [ ] `docs/PROJECT_STATUS.md`: the Phase 29a row and a work-log entry
  with **Review:**.
