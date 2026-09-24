# Phase 26a: Company Summary and Text Surfaces — Implementation Plan

Design: [26a spec](../specs/2026-09-24-phase-26a-company-summary-text-design.md).
Draft: tasks assume the recommended answers to open decisions A–E; revise
this plan if the owner chooses otherwise. Each task is tests first.

- [ ] **Task 1: labels** (`internal/companyview/labels.go`, `labels_test.go`).
  Tests first: `TestNeedShortForms`, `TestLoadBandLabel`,
  `TestActivityLabel`, `TestCompanyCount`, `TestUnknownFieldsRenderUnknown`.
- [ ] **Task 2: seams** (`internal/company/provider.go`,
  `internal/expedition`, `internal/camping`). Tests first:
  `TestMemberViewProviderNone`, `TestActivityProviderNone` (each seam),
  then the interfaces and pass-throughs.
- [ ] **Task 3: module implementations** (`modules/company`,
  `modules/expedition`, `modules/camping`). Tests first:
  `TestCompanyMembersStates` (present, awaiting, dead with time),
  `TestTravelActivity`, `TestCampAndInnActivity`.
- [ ] **Task 4: the read model** (`internal/companyview/summary.go`). Tests
  first, with fake providers: `TestSummaryLeaderAndCompanions`,
  `TestSummaryMissingProvidersAreUnknown`, `TestSummaryKeysByMemberID`
  (two companions with the same name).
- [ ] **Task 5: level-loss note** (`modules/death`). Tests first:
  `TestRespawnRecordsLastLoss`, `TestLastLossSurvivesReload`.
- [ ] **Task 6: `status` and `experience`** (`internal/usercommands`).
  Tests first: the Ashveil block renders, engine training and `experience
  chart` output unchanged.
- [ ] **Task 7: `inventory` and `conditions`.** Tests first: company load
  and cargo lines, whetstone uses, item-count note; conditions groups
  (rest time, edge strikes, chemistry tier not shown as expiring).
- [ ] **Task 8: prompt tokens** (`internal/companyview/prompt.go` registered
  from a module init). Tests first: each token, unknown stays literal, a
  quiet default prompt unchanged (decision B).
- [ ] **Task 9: wiring** (`modules/company/wiring_summary_test.go`). Through
  `plugins.Load`: recruitment, journey, camp rest, a death and a
  resurrection, save/reload; every surface checked at each step through
  `TryCommand` and `GetCommandPrompt`; clock unchanged.
- [ ] **Task 10: docs and verification.** Module guides,
  `go test -race ./...`, `make generate`, `make validate`, independent
  review, `docs/PROJECT_STATUS.md`.
