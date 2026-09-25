# Phase 26a: Company Summary and Text Surfaces — Implementation Plan

Design: [26a spec](../specs/2026-09-24-phase-26a-company-summary-text-design.md)
(decisions confirmed 2026-09-25). Each task is tests first.

- [x] **Task 1: labels** (`internal/companyview/labels.go`, `labels_test.go`).
  Tests first: `TestWarmthLabel`, `TestLightLabel`, `TestLoadLabel`,
  `TestActivityLabel`, `TestCompanyCount`, `TestWarnCluster` (order, cap,
  empty when fine), `TestFormatRemaining`.
- [x] **Task 2: seams** (`internal/company/provider.go`,
  `internal/expedition`, `internal/camping`, `internal/templates`). Tests
  first: `TestMemberViewProviderNone`, `TestProgressProviderNone`,
  `TestRestProviderNone`, `TestPanelLayoutHasPanel`. Then the interfaces,
  pass-throughs, and `HasPanel`.
- [x] **Task 3: module implementations** (`modules/company`,
  `modules/expedition`, `modules/camping`, `modules/exposure`). Tests
  first: `TestCompanyMembersStates` (present, awaiting, dead with time),
  `TestExpeditionProgress` (travelling, interrupted, none),
  `TestCampAndInnRestActivity`, `TestRestTier`, buff groups registered.
- [x] **Task 4: the read model** (`internal/companyview/summary.go`). Tests
  first, with fake providers: `TestSummaryLeaderAndCompanions`,
  `TestSummaryMissingProvidersAreUnknown`, `TestSummaryKeysByMemberID`.
- [x] **Task 5: level-loss note** (`modules/death`). Tests first:
  `TestRespawnRecordsLastLoss`, `TestLastLossSurvivesReload`.
- [x] **Task 6: prompt tokens and cache** (`internal/companyview/prompt.go`,
  `internal/usercommands/usercommands.go` deferred refresh,
  `_datafiles/config.yaml` default prompt, `help prompt`). Tests first:
  `TestPromptTokens`, `TestPromptUnknownStaysLiteral`,
  `TestQuietDefaultPromptUnchanged`, `TestPromptCacheRace` (`-race`).
- [x] **Task 7: `status`** (`internal/usercommands/status.ashveil.go`,
  status layout yaml). Tests first: every panel present, the `empty`
  layout still renders, stat training unchanged.
- [x] **Task 8: `conditions`, `inventory`, `experience`.** Tests first:
  grouped conditions (rest time, edge strikes, survival, chemistry, other),
  "None" when empty; inventory load lead and item-count note, filters
  unchanged; experience last-loss line and `experience chart` unchanged.
- [x] **Task 9: wiring** (`modules/death/wiring_summary_test.go`, where the travel, camping, company, survival, and death modules already load). Through
  `plugins.Load` with the company, survival, camping, expedition, death,
  and encumbrance modules: recruitment, a journey, a camp rest, a companion
  death and resurrection, save/reload. At each step it checks `status`,
  `conditions`, `inventory`, `experience`, and `GetCommandPrompt`; the
  clock is unchanged.
- [x] **Task 10: docs and verification.** Module guides,
  `go test -race ./...`, `make generate`, `make validate`, independent
  review, `docs/PROJECT_STATUS.md`.
