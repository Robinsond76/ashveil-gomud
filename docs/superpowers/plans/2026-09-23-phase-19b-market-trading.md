# Phase 19b: Market Trading — implementation plan

See the design doc
(`docs/superpowers/specs/2026-09-23-phase-19b-market-trading-design.md`).
Decisions confirmed by the owner in conversation on 2026-09-23 (tagged
market room; distinct buy and sell prices, both listed); remaining
defaults chosen under their "carry on".

## Tasks

- [x] **`internal/market`: `AskForStock`, `BidForStock`.**
  - Tests first (`internal/market/market_test.go`):
    `TestAskForStockClosedWhenEmpty`, `TestBidForStockClosedWhenFull`,
    `TestBidForStockAppliesSpread`, `TestBidNeverReachesAsk`,
    `TestNoProfitableRoundTrip` (all stock levels, several curves incl.
    steep, flat, extreme ints).
  - Then implement.
- [x] **`modules/market`: market room tag, two-price listing.**
  - Tests first: listing shows buy and sell columns in a tagged room;
    outside a tagged room in a market zone names the market room;
    outside a market zone says no market; `SpreadPct` and `RoomTag`
    config parsing with defaults.
  - Then implement.
- [x] **`modules/market`: `market buy` / `market sell`.**
  - Tests first (handler level): gold, item, stock, and save changes for
    each; every refusal in the design doc's acceptance criteria leaves
    gold, items, and stock unchanged.
  - Then implement (single critical section for quote, stock, save).
- [x] **Content:** rooms 2004 (Dunmar Market Square) and 2005 (Trappers'
      Post), tagged `market`; West Gate east exit and description line;
      Fork east exit.
- [x] **Wiring test** (extend `TestMarketEndToEndThroughPluginsLoad`):
      through `usercommands.TryCommand` in the shipped market room, list,
      buy, and sell; assert gold, items, stock, and the reloaded real
      store; assert the West Gate points players to the square.
- [x] `gofmt -l`, `go vet`, `go build ./...`, `go test -race ./...`,
      `make generate`, `make validate` (clean worktree).
- [x] **Testing and review gate:** independent reviewer over the phase
      diff; verify findings, fix with regression tests, record.
- [x] Update `docs/PROJECT_STATUS.md` (Phase 19b entry with **Review:**
      line; correct Phase 19's deferral wording).
