# Phase 23b: Whetstones — Implementation Plan

Design: [23b spec](../specs/2026-09-24-phase-23b-whetstones-design.md).
Each task is tests first.

- [ ] **Task 1: the item edge** (`internal/items/items.go`,
  `internal/items/edge_test.go`). Tests first: `TestSharpenNeverStacksOrResets`,
  `TestSpendEdgeEndsAtZero`, `TestEdgeSurvivesYAMLRoundTrip`,
  `TestEdgeShownInNameComplexAndDescription`. Then `SharpBonus`,
  `SharpStrikes`, `Sharpened`, `Sharpen`, `SpendEdge`, `EdgeLabel`.
- [ ] **Task 2: pure sharpening plan** (`internal/camping/sharpen.go`,
  `sharpen_test.go`). Tests first: `TestBladedSubtypes`,
  `TestPlanOneUsePerMemberEvenWithTwoBlades`,
  `TestPlanSkipsSharpAndBladeless`, `TestPlanLeaderFirstThenByNumberAndLeftOut`,
  `TestPlanAbsentMembersCostNothing`.
- [ ] **Task 3: combat edges** (`internal/combat`). Tests first:
  `TestEdgeAddsBonusAndSpendsOnHit` (calculateCombat with a forced hit),
  `TestEdgeNotSpentOnMiss`, `TestOffhandEdgeTrackedBySlot`, and wiring
  `TestAttackPlayerVsMobSpendsLeaderEdge` and
  `TestAttackMobVsMobSpendsCompanionEdge` through the real `Attack*`
  functions.
- [ ] **Task 4: camping config, registry, and the pass**
  (`modules/camping/sharpen.go`, `inn.go`, `camping.go`). Tests first:
  `TestParseInnSettings` extended; `TestSharpenCommandSharpensCompanyOneUseEach`,
  `TestSharpenTwoBladesOneUse`, `TestSharpenAlreadySharpFree`,
  `TestSharpenStoneRunsOutLeavesOut`, `TestSharpenCarriesOntoSecondStone`,
  `TestSpentStoneRemoved`, `TestSharpenRefusedInCombat` (leader and
  companion), `TestSharpenAbsentCompanionNotHere`, `TestSharpenStatusPreviewSpendsNothing`,
  `TestAutoSharpenSettingDurable`, `TestLegacyRegistryWithoutAutoLoads`.
- [ ] **Task 5: auto at camp rest completion** (`modules/camping/tiers.go`).
  Tests first: `TestAutoSharpenAtCampRestCompletion` (through `camp`
  commands, the timer, and `onNewRound`), `TestAutoOffSpendsNothing`,
  `TestAutoSkippedInCombatSaysSo`.
- [ ] **Task 6: surfaces** (`internal/usercommands` inventory and
  conditions, `modules/company` gear view). Tests first:
  `TestInventoryShowsWhetstoneUsesAndEdge`, `TestConditionsShowsSharpened`,
  `TestCompanyGearShowsEdge`.
- [ ] **Task 7: whetstone content and markets** (`30-whetstone.yaml`,
  `modules/market` config and `pickSaleItem`). Tests first:
  `TestShippedWhetstoneHasTenUses`, `TestShippedMarketsSellWhetstones`,
  `TestMarketRefusesUsedWhetstone`, and a `market buy` wiring test that
  gets a 10-use stone.
- [ ] **Task 8: companion edge durability** (`modules/company`). Test
  first: `TestCompanionEdgeSnapshotsToRecord` (a sharpened companion
  weapon snapshots into `MemberState` and restores onto a respawned mob).
- [ ] **Task 9: docs and verification.** Config comments, module
  `AGENTS.md` notes, `go test -race ./...`, `make generate`,
  `make validate`, independent review, `docs/PROJECT_STATUS.md`.
