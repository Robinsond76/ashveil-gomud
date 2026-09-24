# Phase 22c: Settlement Recruiters and `company recruit` — implementation plan

See the design doc
(`docs/superpowers/specs/2026-09-24-phase-22c-recruiters-design.md`).

## Tasks

- [ ] **`internal/company`: tutorial claims** (`company.go`).
  - Tests first (`company_test.go`): `TestClaimRecordsTemplateOnce`,
    `TestGetDeepCopiesClaims`, `TestPutKeepsClaimsOnlyRecord`,
    `TestClaimsYAMLRoundTrip`.
  - Then implement `Record.Claimed`, `HasClaimed`, `Registry.Claim`.
- [ ] **`modules/company`: extract `enlist` from `summon`** (`company.go`).
  - Existing summon tests must stay green unchanged; `rollbackSummon`
    restores pre-summon claims. `wireRecord.Claimed` decoded.
- [ ] **`modules/company`: recruiters** (`recruit.go`).
  - Tests first (`recruit_test.go`): `TestParseRecruiters`,
    `TestRecruitListShowsCandidates`, `TestRecruitOutsideRecruiterRoom`,
    `TestRecruitFreeTutorialClaimedOnce`, `TestRecruitPricedChargesGold`,
    `TestRecruitRefusalsChangeNothing` (table: gold, full, unknown,
    unavailable template, claimed, alignment),
    `TestRecruitSaveFailureRollsBackClaimAndKeepsGold`,
    `TestSummonRefusesRecruiterTemplate`.
  - Then implement parsing, listing, `recruit`, the `saveUser` seam, and
    the `company recruit` subcommand.
- [ ] **Content:** mobs 61–64, room descriptions for 2003 and 2005, the
      config overlay (`Recruiters`, `CompanionArchetypes`).
  - Shipped-data test (`recruit_shipped_test.go`):
    `TestShippedRecruitersResolve`.
- [ ] **Wiring test** (`wiring_recruit_test.go`): see the design's
      acceptance criteria.
- [ ] `modules/company/AGENTS.md` note.
- [ ] `go test -race ./...`, `make generate`, `make validate`.
- [ ] **Testing and review gate:** independent reviewer; verify, fix,
      record.
- [ ] `docs/PROJECT_STATUS.md` Phase 22c entry with **Review:** line.
