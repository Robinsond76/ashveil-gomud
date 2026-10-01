# Phase 30g2: Active Defense and Armor — Plan

Design: [phase-30g design](../designs/2026-09-30-phase-30g-tempo-defense-design.md),
slice 30g2 (owner decisions 4–6, 10, 15–17). Branch:
`claude/phase-30g2-defense`.

This plan was written after a first implementation was reviewed and found
short of the design (wrong parry modifiers, a shield ×1.5 left in
`armor_rank.go`, no stream or summary support, stale help, and failing
counter tests reported as expected). It records the tasks as rebuilt.

## Decisions made while building

- **Per-strike defenses.** One attack event covers a round, which can hold
  several strikes, so `AttackResult.Defenses` and the event's `Defenses`
  list each defended strike's outcome rather than carrying one `Defense`
  value. A bash follows any blocked strike of the round (still once a round
  per bearer).
- **Parry modifiers by item.** Subtypes don't separate a staff from a club,
  so swords (`slashing`) +5, daggers (`stabbing`) −5, reach weapons
  (polearms) +5, and an item's own `parry` field adds to that (the ash
  quarterstaff has `parry: 5`). Axes, maces, clubs, and whips are 0;
  claws, ranged weapons, and bare hands can't parry. The modifier moves
  the whole 5–30% range (a sword 10–35%, a dagger 0–25%), so a dagger at
  even Speed falls under the dodge and the dodge is rolled.
- **Block chance:** `BlockChanceMin` + the shield's own armor + the
  defender's share of the two Strengths (±half the range), held to
  15–45%. Even Strength: wooden shield 20%, iron shield 25%.
- **Lines** go through `buildCombatMessages` (names, articles, 29d
  pronouns, lines for those watching), with a fixed seed so picking the
  one line never spends a die. The bash line drops "turns the blow" (the
  block line already says it) for "drives her shield back into".
- **Rankings:** `ArmorRank.AdjDefense` stays (the admin rankings and web
  API read it) and now equals `Defense`.

## Tasks

1. **Mechanics** (`internal/combat`, `internal/items`, `internal/hooks`,
   `internal/interrupt`): `activeDefense`, `blockChance`,
   `parryModifier`, `parryChance`, `BashChance`; stun via
   `status.FlagNoDodge`; bash on `AttackResult.Blocked()`; remove the
   shield ×1.5 from `GetDefense` and `armor_rank.go`; drop the dead
   `interrupt.BashChance` and `Counter.Missed`. Tests through
   `calculateCombat` (`defense_test.go`); every test that forces hits
   zeroes parry and block too.
2. **Stream, summary, narration:** `combatstream.Event.Defenses`; a
   summary `Defenses` line per side; the armor suffix reads `absorbed`;
   the 30g1 harness reports blocks, parries, and dodges.
3. **Integration:** the 30d1 counter tests force blocks (`forceBlocks`);
   new `TestShieldCounterOnlyOnBlock`; the bow case uses the company
   ranger's blocked shot (the old one was vacuous: a bandit given a sling
   never attacked).
4. **Help and tutorial:** new `help defense` (aliases block, parry, dodge,
   shield, ...); `armor` rewritten and indexed under combat; `interrupts`,
   `combat` (hub link and Commands), `narration`, `statuses`,
   `battle-summary`, `perception`, `speed`, `strength`, and `stats`
   updated; the Combat lesson's interrupt hint corrected and a defense
   hint added. Tested by `TestDefenseHelp`, `TestInterruptsHelp`,
   `TestCombatHelpTopics`, and `TestTutorialHelpPointersExist`.
5. **Verify and measure:** `make generate`, `make validate`,
   `go test -race ./...`; the 30g1 table re-run and recorded against the
   baseline; an independent review; `docs/PROJECT_STATUS.md` corrected.
