# Phase 21a: Company Alignment — implementation plan

See the design doc
(`docs/superpowers/specs/2026-09-24-phase-21a-company-alignment-design.md`).
Defaults applied under the session's "Implement next phase" instruction
and recorded there.

## Tasks

- [x] **`internal/company`: pure alignment rules** (`alignment.go`).
  - Tests first (`internal/company/alignment_test.go`):
    `TestDisplayAlignmentMapsEngineScale`, `TestAlignmentBandNames`,
    `TestAverageAlignmentRounds`, `TestDriftTowardNeverOvershoots`,
    `TestTickAlignmentUsesRestOfCompany`, `TestTickAlignmentLoyalty`,
    `TestTickAlignmentWarnsOnceAndDeserts`, `TestCanRecruitBoundary`.
  - Then implement.
- [x] **`internal/company`: durable disposition and countdown.**
  - Tests first: `TestSetDispositionReplacesAndClamps`,
    `TestGetCopiesDoNotShareDisposition`.
  - Then implement (`Disposition`, `SetDisposition`, `DriftIn`).
- [x] **`modules/company`: seeding and live alignment.**
  - Tests first (`modules/company/alignment_test.go`):
    `TestSummonSeedsDispositionFromTemplate`,
    `TestSummonRefusesFarCandidateWithoutWriting`,
    `TestLoadSeedsLegacyDispositionAndSaves`,
    `TestSpawnAppliesStoredAlignment`,
    `TestDecodeCompaniesReadsDispositionAndDriftIn`.
  - Then implement (`alignmentWorld`, native world, seeding, gate, wire
    decoding).
- [x] **`modules/company`: drift listener, loyalty, desertion.**
  - Tests first: `TestDriftRunsAfterConfiguredRounds`,
    `TestDriftSkipsOfflineLeaders`, `TestDriftInPersistsAndResumes`,
    `TestLoyaltyWarningAndDesertion`, `TestDesertionWaitsForCombat`,
    `TestDriftSaveFailureRollsBack`.
  - Then implement (`onNewRound`, `desert`, shared removal path).
- [x] **`modules/company`: commands.** Tests first:
      `TestCompanyInspectShowsCandidate`, `TestCompanyAlignmentView`,
      `TestCompanyStatusShowsAlignmentAndLoyalty`,
      `TestParseAlignmentRules`. Then implement.
- [x] **Config overlay:** the eight knobs, documented in engine points.
- [x] **Wiring test** (`modules/company/wiring_test.go`, named to run after tests that call `plugins.New`):
      `plugins.Load` with a disposable world, `company summon`/`inspect`/
      `alignment`/`status` through `usercommands.TryCommand`, a real
      `NewRound` through `events.ProcessEvents` drifts the real live mob
      and writes the real store; the gate refuses a far template.
- [x] Update `modules/company/AGENTS.md` for the new state and commands.
- [x] `gofmt -l`, `go vet`, `go build ./...`, `go test -race ./...`,
      `make generate`, `make validate`.
- [ ] **Testing and review gate:** independent reviewer over the phase
      diff; verify findings, fix with regression tests, record.
- [ ] Update `docs/PROJECT_STATUS.md` (Phase 21a entry with **Review:**
      line).
