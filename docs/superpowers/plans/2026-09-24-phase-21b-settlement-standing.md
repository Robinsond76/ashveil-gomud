# Phase 21b: Settlement Standing — implementation plan

See the design doc
(`docs/superpowers/specs/2026-09-24-phase-21b-settlement-standing-design.md`).

## Tasks

- [x] **`internal/standing`: pure rules and seam** (`standing.go`).
  - Tests first (`internal/standing/standing_test.go`):
    `TestAssessTierBoundaries`, `TestPricesRoundAgainstPlayer`,
    `TestNoProfitableRoundTripUnderAnyTier`, `TestRefusalsAndBlackMarket`,
    `TestProviderSeam`.
  - Then implement.
- [x] **`internal/company` + `modules/company`: `AlignmentProvider` seam.**
  - Tests first: `TestCompanyAlignmentSeam` (internal),
    `TestCompanyAlignmentProviderUsesCompanyAverage` (module).
  - Then implement and register in `init`.
- [x] **`modules/standing`: config, provider, `standing` command.**
  - Tests first (`modules/standing/standing_test.go`):
    `TestParseConfigDefaultsAndBounds`, `TestParseSettlements`,
    `TestProviderForZone`, `TestStandingCommandInSettlement`,
    `TestStandingCommandOutsideSettlement`.
  - Then implement; config overlay; `make generate`.
- [x] **`modules/market`: standing prices, refusals, black market.**
  - Tests first (`modules/market/standing_test.go`):
    `TestListingAppliesDistrustedMarkup`, `TestBuySellApplyMarkup`,
    `TestShunnedRefusedAtMarket`, `TestBlackMarketServesOnlyOutlaws`,
    `TestNoStandingProviderUnchanged`, `TestMarketHintSkipsBlackMarket`.
  - Then implement.
- [x] **`modules/camping`: inn markup and refusal.**
  - Tests first (`modules/camping/standing_test.go`):
    `TestInnStatusShowsDistrustedPrice`, `TestInnRestChargesMarkup`,
    `TestInnRestShunnedRefusedWithoutGold`.
  - Then implement (standing read before `m.mu`).
- [x] **Content:** room 2006 (Tanner's Back Alley, `blackmarket`), exit
      south from 2004.
- [x] **Wiring test** (`modules/standing/wiring_test.go`): see the
      design's acceptance criteria.
- [x] `AGENTS.md` notes for `modules/standing` and `modules/company` (market and camping have none).
- [x] `go test -race ./...`, `make generate`, `make validate`.
- [ ] **Testing and review gate:** independent reviewer; verify, fix,
      record.
- [ ] `docs/PROJECT_STATUS.md` Phase 21b entry with **Review:** line.
