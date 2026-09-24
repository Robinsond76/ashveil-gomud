# Phase 22b: Durable Companion Level and Equipment — implementation plan

See the design doc
(`docs/superpowers/specs/2026-09-24-phase-22b-durable-companion-state-design.md`).

## Tasks

- [ ] **`internal/company`: `MemberState`, `Companion.State`, deep copies,
      `SetState`** (`state.go`, `company.go`).
  - Tests first (`state_test.go`): `TestMemberStateCloneIsDeep`,
    `TestRegistryGetDeepCopiesState`, `TestRegistryCloneDeepCopiesState`,
    `TestSetState`, `TestStateYAMLRoundTrip`.
  - Then implement.
- [ ] **`modules/company`: runtime seam** (`runtime.go`).
  - `Spawn` takes a `*MemberState`; add `Snapshot` and `TemplateState`.
    Update the test fake (records the state it was spawned with and serves
    snapshots).
- [ ] **`modules/company`: initialize, restore, snapshot seams**
      (`state.go`, `company.go`).
  - Tests first (`state_test.go`): `TestSummonRecordsTemplateState`,
    `TestSummonSaveFailureLeavesNoState`, `TestRestoreSpawnsWithSavedState`,
    `TestLegacyCompanionUpgradedBeforeSpawn`,
    `TestLegacyUpgradeSaveFailureDoesNotSpawn`,
    `TestItemOwnershipRefreshesAndSaves`, `TestOnSaveRefreshesLiveCompanions`,
    `TestLeaderDespawnSnapshotsAndRemovesCompanions`,
    `TestCompanionDeathClearsGearKeepsLevel`, `TestDismissDropsState`,
    `TestStatusShowsLevelAndGearView`.
  - Then implement and register the listeners in `init`.
- [ ] **Wiring test** (`modules/company/wiring_state_test.go`): see the
      design's acceptance criteria.
- [ ] `modules/company/AGENTS.md` note.
- [ ] `go test -race ./...`, `make generate`, `make validate`.
- [ ] **Testing and review gate:** independent reviewer; verify, fix,
      record.
- [ ] `docs/PROJECT_STATUS.md` Phase 22b entry with **Review:** line.
