# Phase 24: Company Chemistry — Implementation Plan

Design: [24 spec](../specs/2026-09-24-phase-24-company-chemistry-design.md).
Each task is tests first.

**Rework (2026-09-24, owner amendment):** Tasks 1–4, 6, and 7 were redone
for the company-wide, dilute model on the same branch; the pair-bond test
names below were replaced by `TestBandAverageDilutesAndUsesSavedRounds`,
`TestGetCopiesService`, `TestPutPrunesServiceOfRemovedMembers`,
`TestPutNormalizesService`, `TestServiceYAMLRoundTrip`,
`TestChemistryServiceAccrues`, `TestChemistryPausesWhenNotEligible`,
`TestChemistryCompanionsServeWhileLeaderElsewhere`,
`TestChemistryDismissEndsService`, `TestChemistryResumesAfterRespawn`,
`TestChemistryTierCrossingSavesAndAnnounces`,
`TestChemistryFailedSaveKeepsTierUntilSaved`,
`TestChemistryRecruitDilutesTheBand`, `TestChemistryCompanionBandAnnounced`,
`TestChemistryChargeKeptWhenDriftSaveFails`, `TestChemistrySurvivesReload`,
`TestDecodeCompaniesReadsService`, `TestChemistryViewShowsBandAndService`,
and the wiring test's dilution step. Task 5 (combat) was unchanged.

- [x] **Task 1: pure bonds** (`internal/company/chemistry.go`,
  `chemistry_test.go`). Tests first: `TestTierBoundaries`,
  `TestRulesValid`, `TestBondPairSorted`, `TestChargeOncePerRound`,
  `TestChargeResumesAfterCounterReset`, `TestBestBondHighestPresentOnly`,
  `TestProgressPercent`.
- [x] **Task 2: record storage** (`internal/company/company.go`). Tests
  first: `TestGetCopiesBonds`, `TestPutPrunesBondsOfRemovedMembers`
  (dismissal ends exactly that companion's bonds),
  `TestBondsYAMLRoundTrip`. Then the provider seam in `provider.go` with
  `TestChemistryProviderNoneRegistered`.
- [x] **Task 3: module accrual** (`modules/company/chemistry.go`). Tests
  first: `TestChemistryIndependentBonds`, `TestChemistryPausesWhenNotEligible`
  (separated, dead, detached, offline leader),
  `TestChemistryCompanionPairWhileLeaderElsewhere`,
  `TestChemistryDismissEndsBonds`, `TestChemistryResumesAfterRespawn`,
  `TestChemistryTierCrossingSavesAndAnnounces`,
  `TestChemistryFailedCrossingSaveHoldsShort`,
  `TestChemistrySurvivesReload`, `TestParseChemistryConfig`. (Also
  `TestDecodeCompaniesReadsBonds`, `TestChemistryUnavailableWhileCompanyDataFailed`.)
- [x] **Task 4: provider and bonus** (`modules/company/chemistry.go`).
  Tests first: `TestChemistryBonusLeaderAloneNone`,
  `TestChemistryBonusCapNeverStacks`, `TestChemistryBonusAbsentPartnerNone`,
  `TestChemistryBonusForInstance`.
- [x] **Task 5: combat** (`internal/combat`). Tests first:
  `TestHitRollBonusDecides`, `TestHitRollNoBonusMatchesHits`,
  `TestAttackPlayerVsMobChemistryRaisesHits` and
  `TestAttackMobVsMobChemistryRaisesHits` (a fake provider, real
  `Attack*` functions), `TestChemistryLineOncePerRound`. (Landed as
  `TestHitRollNoBonusNeverByBonus`; the once-a-round check is inside both
  `Attack*` tests.)
- [x] **Task 6: surfaces** (`company chemistry`,
  `internal/usercommands/status.panels.go`). Tests first:
  `TestChemistryViewShowsTierPartnerProgress`,
  `TestStatusBonusesShowsChemistry`.
- [x] **Task 7: wiring** (`modules/company/wiring_chemistry_test.go`).
  Through `plugins.Load`, `usercommands.TryCommand`, and real `NewRound`
  events via `events.ProcessEvents`: accrual, a tier crossing, `company
  chemistry`, `status bonuses`, `plugins.Save` and reload, and the
  registered provider's bonus at the real combat entry point.
- [x] **Task 8: docs and verification.** Config comments, module
  `AGENTS.md`, `go test -race ./...`, `make generate`, `make validate`,
  independent review, `docs/PROJECT_STATUS.md`.
