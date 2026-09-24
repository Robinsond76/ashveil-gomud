# Phase 24: Company Chemistry — Implementation Plan

Design: [24 spec](../specs/2026-09-24-phase-24-company-chemistry-design.md).
Each task is tests first.

- [ ] **Task 1: pure bonds** (`internal/company/chemistry.go`,
  `chemistry_test.go`). Tests first: `TestTierBoundaries`,
  `TestRulesValid`, `TestBondPairSorted`, `TestChargeOncePerRound`,
  `TestChargeResumesAfterCounterReset`, `TestBestBondHighestPresentOnly`,
  `TestProgressPercent`.
- [ ] **Task 2: record storage** (`internal/company/company.go`). Tests
  first: `TestGetCopiesBonds`, `TestPutPrunesBondsOfRemovedMembers`
  (dismissal ends exactly that companion's bonds),
  `TestBondsYAMLRoundTrip`. Then the provider seam in `provider.go` with
  `TestChemistryProviderNoneRegistered`.
- [ ] **Task 3: module accrual** (`modules/company/chemistry.go`). Tests
  first: `TestChemistryIndependentBonds`, `TestChemistryPausesWhenNotEligible`
  (separated, dead, detached, offline leader),
  `TestChemistryCompanionPairWhileLeaderElsewhere`,
  `TestChemistryDismissEndsBonds`, `TestChemistryResumesAfterRespawn`,
  `TestChemistryTierCrossingSavesAndAnnounces`,
  `TestChemistryFailedCrossingSaveHoldsShort`,
  `TestChemistrySurvivesReload`, `TestParseChemistryConfig`.
- [ ] **Task 4: provider and bonus** (`modules/company/chemistry.go`).
  Tests first: `TestChemistryBonusLeaderAloneNone`,
  `TestChemistryBonusCapNeverStacks`, `TestChemistryBonusAbsentPartnerNone`,
  `TestChemistryBonusForInstance`.
- [ ] **Task 5: combat** (`internal/combat`). Tests first:
  `TestHitRollBonusDecides`, `TestHitRollNoBonusMatchesHits`,
  `TestAttackPlayerVsMobChemistryRaisesHits` and
  `TestAttackMobVsMobChemistryRaisesHits` (a fake provider, real
  `Attack*` functions), `TestChemistryLineOncePerRound`.
- [ ] **Task 6: surfaces** (`company chemistry`,
  `internal/usercommands/status.panels.go`). Tests first:
  `TestChemistryViewShowsTierPartnerProgress`,
  `TestStatusBonusesShowsChemistry`.
- [ ] **Task 7: wiring** (`modules/company/wiring_chemistry_test.go`).
  Through `plugins.Load`, `usercommands.TryCommand`, and real `NewRound`
  events via `events.ProcessEvents`: accrual, a tier crossing, `company
  chemistry`, `status bonuses`, `plugins.Save` and reload, and the
  registered provider's bonus at the real combat entry point.
- [ ] **Task 8: docs and verification.** Config comments, module
  `AGENTS.md`, `go test -race ./...`, `make generate`, `make validate`,
  independent review, `docs/PROJECT_STATUS.md`.
