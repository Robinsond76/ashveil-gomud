# Phase 30c1: Company Tactics and Enemy Personalities — Plan

Design: [phase-30c design](../specs/2026-09-29-phase-30c-company-tactics-design.md)
(sections A–D; owner decisions 1–8). 30c2 (guardian, guards) is a later
plan.
Branch: `phase-30c1-company-tactics`, worktree
`.worktrees/phase-30c1-company-tactics`.

Each task writes its tests first and runs only its own packages.

- [x] **1. `internal/strategy` (pure).**
  - Tests first:
    - `Casters` parses, is listed, and chooses the chanting foe, then a
      foe that knows spells, else none (so `Pick` falls to the nearest);
    - `ParseFocus` (the seven values, `none`, aliases; `assist`/`defend`
      refused) and `ParseHealing` (10–90 in tens, with or without `%`);
    - `Tactics` resolve (blank focus is `none`, zero healing is 50);
    - `Decide` heals below `HealBelow` (40 skips an ally at 45%; 60 heals
      them; 0 reads as 50);
    - `EnemyPick`: each rule over plain foes, only reachable foes, a
      seeded noise roll that hits and one that misses;
    - `TacticsFor` falls back to defaults with no provider.
  - Add them, and update `AGENTS.md`.
- [x] **2. Durable tactics (`modules/strategy`).**
  - Tests first: set and read back; save failure rolls back; a
    save/load round trip through the plugin store; bad stored values are
    dropped; `UserPurged` drops the user's tactics.
  - Add `Registry.Tactics`, `TacticsFor`/`SetTactics`, and register the
    `TacticsProvider`.
- [x] **3. Battle focus (`internal/battle`) and the stream.**
  - Tests first: `SetFocus` needs a battle and sets it pending;
    `Focus` reads it; `TakeRefocus` returns the pending players and
    clears them; `ClearFocus` (back to saved); a new battle starts with
    no override; `End` drops it.
  - Add `combatstream.FocusChange` and `Event.Rule`.
- [x] **4. Aiming by focus (`internal/enemyparty`, `internal/hooks`).**
  - `PlayerAttacker`/`CompanionAttacker` use the effective focus (the
    battle's, else the saved one, else the member's own rule).
  - `Foes` fills `Chanting` and `Caster` for the `casters` rule.
  - The upkeep's `retarget` and `turnAlone` re-aim once in a refocus
    round; `DoCombat` takes the refocus set after the battle pass, emits
    `focus-change`, and drops it at the round's end.
  - The strategy pass passes `HealBelow` from the tactics.
  - Wiring tests (`modules/company`, brawl world, real `DoCombat`):
    - `TestTacticsFocusOverridesStrategies`: a saved focus `strongest`
      turns every member onto the captain (reach binding a back-row
      member), and the healer still heals;
    - `TestTacticsHealingThreshold`: a healer heals a member at 60% only
      with the threshold at 70;
    - `TestTacticsFocusMidBattle`: `company tactics focus leader` in a
      battle names the choice, is refused again before the next round,
      turns everyone at the next upkeep (with `focus-change`), allows a
      change the round after, and the next battle is back to the saved
      focus; `healing` and `strategy` are refused in the battle;
    - `TestTacticsFocusSoloPlayer`: a player alone turns by the focus
      (`turnAlone`).
- [x] **5. Enemy personalities (`internal/mobs`, `internal/races`,
  `internal/hooks`).**
  - Tests first:
    - a mob template's `targeting` wins over its race's; none means
      `weakest`, no noise;
    - `keepPartyEngaged` with a `wounded` race turns a joining enemy onto
      the most hurt member (fraction), a `casters` one onto the healer,
      and noise (seeded roll) onto a random reachable member
      (`TestEnemyPersonalities`, brawl world, real `DoCombat`);
    - the shipped races load with decision 7's table
      (`internal/races`).
  - Add the fields, `hooks.UseAimRollForTest`, the enemy path in
    `keepPartyEngaged`, and the race data. The brawl turns noise off
    so the existing fights keep their targets.
- [x] **6. The command (`modules/company`).**
  - Tests first (`tactics_test.go`, fakes for the provider): show;
    focus and healing set and saved; bad values; `default`; refusals in a
    battle; the "still turning" cooldown; the room line.
  - Add `tactics.go`, the `company tactics` case, and the `tactics`
    command.
- [x] **7. Web (`modules/gmcp`, `window-combat.js`).**
  - Tests first: `Company.Battle` carries `focus`, `saved_focus`,
    `focus_ready` (`gmcp_battle_test.go`); `Company` carries `tactics`.
  - Add the focus buttons to the battle view and the tactics row in
    Setup; extend `scripts/browser/dock-windows-check.mjs` (buttons
    render, the current one pressed, disabled when not ready, a click
    sends the command; Setup shows the tactics).
- [x] **8. Player help and tutorial.**
  - New `help tactics` (`tactics.template`), in `keywords.yaml` under
    combat with aliases `focus`, `company-tactics`, `personalities`,
    linked from `help combat`.
  - Update `help strategy` (the `casters` rule; the focus overrides
    target rules; the healing threshold), `help combat` (the focus is
    the one mid-battle change), `help targeting` (enemy personalities),
    `help webclient` (the buttons), and `help company`.
  - The Practice Yard (Combat) lesson points to `company tactics`.
  - Tests: `help tactics` and the updated pages render through `help`
    (`internal/usercommands`, as `help_combat_test.go`);
    `TestTutorialHelpPointersExist`.
- [ ] **9. Review, fixes, full verification, status, merge.**
  - Independent reviewer over `git diff master..HEAD`; verify each
    finding, fix with regression tests.
  - `go test -race ./...`, `make generate`, `make validate`,
    `make js-lint`, the Playwright checks in `scripts/browser`.
  - `docs/PROJECT_STATUS.md` (What/Why/Verification/Review); merge
    `origin/master` in first, then to `master`, push, remove the
    worktree.
