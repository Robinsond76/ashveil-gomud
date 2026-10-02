# Phase 33i2: Coordinated Enemies — Plan

Design: [33i design](../designs/2026-10-01-phase-33i-company-assessment-enemy-roles-design.md),
"Final implementation decisions: 33i2". Implementation worktree
`.worktrees/phase-33i2-coordinated-enemies`, branch
`phase-33i2-coordinated-enemies`, from master after this plan merges.

## Task 1: tiers and template fields

- `internal/mobs`: template fields `role`, `coordination`, `wounds`
  (`none` or empty), validated on load with a warning for unknown values.
- New pure `internal/coordination`: `TierFor(levels []int, explicit []int)`,
  the tier table (word, focus share, noise floor, heal threshold, guards a
  battle, heal-chanter switch), table-driven tests on the band edges
  (9/10, 24/25, 44/45) and explicit overrides.
- The tier is fixed when the battle begins and kept on the enemy group for
  that battle (runtime only, game loop).

## Task 2: enemy wounds and recovery

- `combat.go`: an enemy target is woundable unless its template says
  `wounds: none`; a crit on an enemy leaves `Light: true`. Bleed expiry
  (`statusHolder.woundable`) follows the same rule.
- `NewRound_AutoHeal`: hostile mobs out of battle and not aggressive
  regain `ceil(max/188)` health and mana a round (applied ×3 in the pass),
  never above the wound limit. Companions and players unchanged.
- Regression tests: crit and crushing blow wound an enemy lightly; light
  wounds cap enemy healing mid-fight and close at fight end; `wounds:
  none` takes none; a hurt survivor recovers fully in 188 rounds and not
  while in battle; companions still take lasting wounds.

## Task 3: enemy roles and focus

- `hooks/combat_strategy.go`: an enemy side pass beside `strategyPass`,
  building enemy actors and the tier's heal threshold. Reuse `Decide`,
  `markPendingHeals`, `autoSpellTargets` and `startCast`; tier 1 allows one
  enemy heal a round.
- Focus: the enemy aim pass applies the tier's share and noise floor; tier
  3+ picks by `casters` when the company has a caster, and tier 4 switches
  to a company member chanting a heal.
- Enemy guardians use `combat_guard.go`'s rules with the tier's battle
  limit; surrender, flight and morale cancel chants as for companions.
- Narration: the focus line for tier 2+, companion heal/cast/guard lines,
  `combatstream` events.
- Content: give `role:` to a healer group and a guardian group in each of
  tiers 1–3, with spellbooks, and audit the hostile templates for
  `wounds: none`.

## Task 4: assessment

- `internal/assessment`: the group's tier word and visible roles in
  `Lines` and `Outlook`; the "not counted" line names healing and guards.
- Real `scout`/`consider` and `Company.Battle` outlook tests: the tier word
  by level and override, roles hidden for hidden members, nothing in the
  dark, risk words unchanged by roles.

## Task 5: balance harness

- 30g1 harness cases: one tier-1, tier-2 and tier-3 role group against the
  even no-focus 5v5 at levels 1, 5 and 10. Assert median length at most
  125% of the baseline and the same winner. Record the numbers in Project
  Status.

## Task 6: help and tutorial

New `coordination.template` (aliases `coordinated`, `enemy roles`,
`rabble`), indexed in `keywords.yaml` and linked from `help combat`.
Updates to `assessment`, `scout`, `consider`, `combat`, `wounds`,
`guardian`, `tactics`, `webclient`. The Combat lesson's scout hint names
the coordination word. Render tests and `TestTutorialHelpPointersExist`.

## Task 7: integration tests

A real battle through the game loop: an enemy healer heals a hurt ally
within its tier's threshold and spends mana; an enemy guardian steps in
up to its limit; a tier-2 group's focus line and aim; an interrupted
enemy chant; a surrendering or fleeing enemy healer stops chanting;
multiplayer: two players' battles keep separate tiers and focus.

## Task 8: migration and recovery

No durable state: tiers, focus and enemy wounds are runtime, mobs aren't
saved, and new template fields default to the old behaviour (fighter,
level tier, woundable). Restart and copyover need nothing. World time
never advances.

## Task 9: review and integration

Independent reviewer on the full diff; fixes with regression tests;
Project Status and roadmap; `make generate`, `make validate`,
`go test -race ./...`, `make js-lint`, a dock-windows check if the Battle
view changes; merge and push.
