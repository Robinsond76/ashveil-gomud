# Phase 20: Trade Rumours — implementation plan

See the design doc
(`docs/superpowers/specs/2026-09-24-phase-20-trade-rumours-design.md`).
Defaults applied under the session's "continue with next phase"
instruction and recorded there.

## Tasks

- [x] **`internal/market`: `Rumors`, `PickRumors`** (`rumor.go`).
  - Tests first (`internal/market/rumor_test.go`):
    `TestRumorsScarceAndGlut`, `TestRumorsCheapestAndBestBuyerNeedTwoMarkets`,
    `TestRumorsSkipTiedExtremes`, `TestRumorsSkipClosedSides`,
    `TestRumorsDeterministicOrder`, `TestPickRumorsDistinctAndBounded`.
  - Then implement.
- [x] **`modules/market`: durable news snapshot.**
  - Tests first (`modules/market/rumor_test.go`):
    `TestLoadSeedsNewsWhenMissing`, `TestLoadKeepsStoredNews`,
    `TestNewsRefreshesAfterConfiguredRounds`,
    `TestDecodeRegistryRejectsNewsWithoutStock`,
    `TestParseRumorConfig`.
  - Then implement (`Registry.News`, `Clone`, decode, load seeding,
    `onNewRound` countdown).
- [x] **`modules/market`: `rumors`/`rumours` command.**
  - Tests first: `TestRumorsCommandListsPhrasedNews`,
    `TestRumorsCommandRefusedOutsideRumorRoom`,
    `TestRumorsCommandNoNews`, `TestRumorsCommandPersistenceUnavailable`,
    `TestRumorsUseNewsNotLiveStock`.
  - Then implement (`rumors.go`).
- [x] **Config overlay:** `RumorRoomTag`, `RumorRefreshRounds`,
      `RumorsPerAsk`, documented.
- [x] **Wiring test** (extend `TestMarketEndToEndThroughPluginsLoad`):
      load the shipped Waymark Inn (2003); `rumors` and `rumours` through
      `usercommands.TryCommand` there; refusal at the West Gate; a real
      `NewRound` refreshes news into the real store; the reload
      assertion includes news.
- [x] `gofmt -l`, `go vet`, `go build ./...`, `go test -race ./...`,
      `make generate`, `make validate`.
- [ ] **Testing and review gate:** independent reviewer over the phase
      diff; verify findings, fix with regression tests, record.
- [ ] Update `docs/PROJECT_STATUS.md` (Phase 20 entry with **Review:**
      line).
