# Phase 32g2: Live Battle View — Plan

Design: [32g2 design](../specs/2026-09-29-phase-32g2-battle-view-design.md).
Decisions A–G were accepted as recommended by the owner on 2026-09-29
(health words; every player sees their own battle; the fallen listed off
the grid; an "others" chip; a tab marker, never a switch; the feed's
usual beat; watching plus the member menu and Flee). The owner asked for
the phase branch to be pushed as it goes and merged to `master` once
everything is green.

Server tasks (1–3) come first, so the browser renders real payload
shapes.

## Task 1: Engine helpers (`internal/enemyparty`, `internal/battle`, `internal/hooks`)

- [x] Tests first:
  - `internal/enemyparty`: `BattleParty(b, parties)` finds the party
    sharing an enemy with the battle, and none when no party does.
  - `internal/battle`: `Waiting(userId)` lists the groups set on the
    player other than the battle's own, by first-set round then party id;
    empty with no battle; `End` and `KeepSet` drop from it.
- [x] `enemyparty.BattleParty` (moved from `hooks.battleParty`, which now
  calls it); `battle.Waiting`.

## Task 2: The `Company.Battle` payload (`modules/gmcp`)

- [x] Tests first (`gmcp_battle_test.go`), the pure builder from fixed
  inputs:
  - enemies with label, cell, health word, reach, and a target on the
    player (`leader`) and on a companion (`companion:<id>`);
  - a fallen or departed enemy in `fallen`, not in `enemies`;
  - an enemy striking an outside player fills `others` once;
  - `waiting` names in order; an unplaced player has no `reach`;
  - no battle builds `{}`; the company's targets by member key, only on
    listed enemies.
  - Change detection through the feed: an unchanged battle sends nothing;
    a health word change sends once.
- [x] `gmcp.CompanyBattle.go`: payload types, `buildBattle(battleFacts)`,
  `gatherBattle(user)` (reads `battle.Current`, the room's groups through
  `enemyparty`, `formationcombat.Legal` with `combat.ResolveReach` as
  `scout` does, `company.LeaderAndKeyForInstance` for companions),
  `battleExtra()` registered in `newCompanyFeed`.

## Task 3: Wiring (`modules/company`)

- [x] Wiring test first (`wiring_battle_view_test.go`, the brawl world,
  real `hooks.DoCombat` rounds, the gmcp module loaded, `GMCPOut`
  captured, `companyview.RefreshUser` standing in for the game loop's
  refresh):
  - a battle begins: `Company.Battle` carries the group, enemies with
    cells and words, and the company's targets by member key;
  - a companion's target changes and a bandit falls: an update, the
    bandit in `fallen`;
  - the group is beaten: `{}` is sent;
  - Brom in his own battle gets his own view (his group, no company
    targets); a `PlayerSpawn` (login/copyover) re-sends it; the clock
    never moves.
  - A connection without GMCP gets nothing: unit-tested in
    `modules/gmcp` (the feed's `accepting`).
  - As built: the brawl world has no one placed, so the test places Aria
    (`formation move me 1 1`) to check `reach` (front row only, bare
    hands); the test connection accepts GMCP (`gmcp.AcceptGMCPForTest`),
    as a web client's does. A cutthroat can fall in the first round, so
    "every bandit once" counts the grid and `fallen` together.

## Task 4: The Battle view (`window-combat.js`)

- [x] Browser check first (Task 5's fixtures).
- [x] While `Company.Battle` has enemies: the enemy grid (front row toward
  the middle), the company grid (from the `Company` snapshot; "You" alone
  without a company), SVG target lines, the "others" chips, the fallen
  and waiting lines, the text list, a polite live region (a new target on
  the player, a new fall), a **Flee** button, and Setup under a
  disclosure. Hover or focus highlights a fighter's target and attackers.
  A company member's click opens Setup's member menu.
- [x] The Combat tab's marker (`setBadge`) while a battle runs and the tab
  isn't active; cleared on opening it or when the battle ends.
- [x] As built: the grids always stack, enemy above company (the dock
  is one narrow column, so the sides face up and down), and each grid
  shows rows only as deep as anyone stands. `setBadge` takes an optional
  spoken label ("Combat, a battle is under way").

## Task 5: Browser check (`scripts/browser/dock-windows-check.mjs`)

- [x] Fixtures: a battle, an update (a fall, a new target on the player),
  and `{}`. Checks: the grids and labels, lines per target (to the chip
  for `others`), the text list, the live region's text, the marker, Flee
  and the member menu's commands, markup rendered as text, 280 px and
  360 px widths with no horizontal scroll, keyboard focus on fighters.

## Task 6: Player help and tutorial

- [x] `help webclient`: the Battle view (grids, lines, words, the fallen
  and waiting lines, the marker, Flee).
- [x] `help combat`: a line pointing web client players to the Combat
  tab; `help scout`: the same words appear in the Battle view.
- [x] The practice-fight lesson's hint (`modules/tutorial/stages.go`)
  mentions the Battle view for web client players.
- [x] Tests: `internal/usercommands` renders each page through `help`
  (extend `help_webclient_test.go`); `TestTutorialHelpPointersExist`
  passes. As built: `TestBattleViewHelp` and
  `TestCombatLessonPointsToBattleView`.

## Task 7: Docs, review, verification, merge

- [x] `modules/gmcp/AGENTS.md`: `Company.Battle` in the extras bullet.
- [x] Independent review of `git diff master..HEAD`; verify each finding;
  fix real ones with regression tests.
- [x] Full verification once: `go test -race ./...`, `make generate`,
  `make validate`, `make js-lint`, every `scripts/browser/` check.
- [x] `docs/PROJECT_STATUS.md`: the 32g2 row, a work-log entry with
  **Review:**, the next phase named.
- [ ] Merge to `master` and push; remove the worktree.
