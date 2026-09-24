# Phase 21b: Settlement Standing — implementation plan

See the design doc
(`docs/superpowers/specs/2026-09-24-phase-21b-settlement-standing-design.md`).

## Tasks

- [ ] **`internal/standing`: pure rules and seam** (`standing.go`).
  - Tests first (`internal/standing/standing_test.go`):
    `TestAssessTierBoundaries`, `TestPricesRoundAgainstPlayer`,
    `TestNoProfitableRoundTripUnderAnyTier`, `TestRefusalsAndBlackMarket`,
    `TestProviderSeam`.
  - Then implement.
- [ ] **`internal/company` + `modules/company`: `AlignmentProvider` seam.**
  - Tests first: `TestCompanyAlignmentSeam` (internal),
    `TestCompanyAlignmentProviderUsesCompanyAverage` (module).
  - Then implement and register in `init`.
- [ ] **`modules/standing`: config, provider, `standing` command.**
  - Tests first (`modules/standing/standing_test.go`):
    `TestParseConfigDefaultsAndBounds`, `TestParseSettlements`,
    `TestProviderForZone`, `TestStandingCommandInSettlement`,
    `TestStandingCommandOutsideSettlement`.
  - Then implement; config overlay; `make generate`.
- [ ] **`modules/market`: standing prices, refusals, black market.**
  - Tests first (`modules/market/standing_test.go`):
    `TestListingAppliesDistrustedMarkup`, `TestBuySellApplyMarkup`,
    `TestShunnedRefusedAtMarket`, `TestBlackMarketServesOnlyOutlaws`,
    `TestNoStandingProviderUnchanged`, `TestMarketHintSkipsBlackMarket`.
  - Then implement.
- [ ] **`modules/camping`: inn markup and refusal.**
  - Tests first (`modules/camping/standing_test.go`):
    `TestInnStatusShowsDistrustedPrice`, `TestInnRestChargesMarkup`,
    `TestInnRestShunnedRefusedWithoutGold`.
  - Then implement (standing read before `m.mu`).
- [ ] **Content:** room 2006 (Tanner's Back Alley, `blackmarket`), exit
      south from 2004.
- [ ] **Wiring test** (`modules/standing/wiring_test.go`): see the
      design's acceptance criteria.
- [ ] `AGENTS.md` notes for `modules/standing`, market, camping.
- [ ] `go test -race ./...`, `make generate`, `make validate`.
- [ ] **Testing and review gate:** independent reviewer; verify, fix,
      record.
- [ ] `docs/PROJECT_STATUS.md` Phase 21b entry with **Review:** line.
