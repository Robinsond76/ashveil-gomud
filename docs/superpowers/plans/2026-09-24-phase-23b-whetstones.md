# Phase 23b: Whetstones — Implementation Plan

Design: [23b spec](../specs/2026-09-24-phase-23b-whetstones-design.md).
Each task is tests first.

- [x] **Task 1: the item edge** (`internal/items/items.go`,
  `internal/items/edge_test.go`). Tests first: `TestSharpenNeverStacksOrResets`,
  `TestSpendEdgeEndsAtZero`, `TestEdgeSurvivesYAMLRoundTrip`,
  `TestEdgeShownInNameComplexAndDescription`. Then `SharpBonus`,
  `SharpStrikes`, `Sharpened`, `Sharpen`, `SpendEdge`, `EdgeLabel`.
- [x] **Task 2: pure sharpening plan** (`internal/camping/sharpen.go`,
  `sharpen_test.go`). Tests first: `TestBladedSubtypes`,
  `TestPlanOneUsePerMemberEvenWithTwoBlades`,
  `TestPlanSkipsSharpAndBladeless`, `TestPlanLeaderFirstThenByNumberAndLeftOut`,
  `TestPlanAbsentMembersCostNothing`.
- [x] **Task 3: combat edges** (`internal/combat`). Tests first:
  `TestEdgeAddsBonusAndSpendsOnHit` (calculateCombat with a fixed 1d1
  weapon: every hit spends one strike, every miss none, and a non-crit
  hit deals exactly the bonus more), `TestEdgeStopsWhenStrikesRunOut`,
  `TestOffhandEdgeTrackedBySlot`, and wiring
  `TestAttackPlayerVsMobSpendsLeaderEdge` and
  `TestAttackMobVsMobSpendsCompanionEdge` through the real `Attack*`
  functions.
- [x] **Task 4: camping config, registry, and the pass**
  (`modules/camping/sharpen.go`, `inn.go`, `camping.go`). Tests first:
  `TestParseInnSettings` extended; `TestSharpenCommandSharpensCompanyOneUseEach`,
  `TestSharpenTwoBladesOneUse`, `TestSharpenAlreadySharpFree`,
  `TestSharpenStoneRunsOutLeavesOut`, `TestSharpenCarriesOntoSecondStone`,
  `TestSpentStoneQueuesOwnershipLoss`, `TestSharpenWithoutStone`,
  `TestSharpenRefusedInCombat` (leader and companion),
  `TestSharpenAbsentCompanionNotHere`, `TestSharpenStatusPreviewSpendsNothing`,
  `TestAutoSharpenSettingDurable`, `TestLegacyRegistryWithoutAutoLoads`,
  `TestCampSharpenAndStatusShareTheCommand`. (Landed as
  `TestParseSharpenSettings` and `TestSharpenAlreadySharpAndBladelessFree`.)
- [x] **Task 5: auto at camp rest completion** (`modules/camping/tiers.go`).
  Tests first: `TestAutoSharpenAtCampRestCompletion` (through `camp`
  commands, the timer, and `onNewRound`), `TestAutoOffSpendsNothing`,
  `TestAutoSkippedInCombatSaysSo`, `TestManualSharpenBeforeRestEndSpendsNoMoreOnAuto`
  (`sharpen_wiring_test.go`, native roster/formation seams, real buffs).
- [x] **Task 6: surfaces** (`internal/usercommands` inventory and
  conditions, `modules/company` gear view). Tests first:
  `TestInventoryShowsWhetstoneUsesAndEdge`, `TestConditionsShowsSharpened`;
  the gear view is covered in Task 8's wiring test.
- [x] **Task 7: whetstone content and markets** (`30-whetstone.yaml`,
  `modules/market` config and `pickSaleItem`). Tests first:
  `TestShippedWhetstoneMatchesConfig` (camping), and in the market's
  end-to-end test through `plugins.Load` and `usercommands.TryCommand`:
  the shipped spec, both markets listing it, `market buy` giving a 10-use
  stone, and `market sell` refusing a used one but selling an unused one.
- [x] **Task 8: companion edge durability** (`modules/company`). Test
  first: extends `TestCompanionGearSurvivesLogoutRestartAndDeath` (a
  partly spent edge on the live companion's weapon shows in `company
  gear`, is written by `plugins.Save`, and is back on the mob restored
  after logout and restart).
- [x] **Task 9: docs and verification.** Config comments, module
  `AGENTS.md` notes, `go test -race ./...`, `make generate`,
  `make validate`, independent review, `docs/PROJECT_STATUS.md`.
