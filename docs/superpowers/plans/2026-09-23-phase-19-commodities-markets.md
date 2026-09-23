# Phase 19: Commodities and Markets — implementation plan

See the companion design doc
(`docs/superpowers/specs/2026-09-23-phase-19-commodities-markets-design.md`).
Open decisions (engine-first narrow scope: stock ledger + bounded
round-driven price drift + read-only `market` command, no authored
production/consumption sim) confirmed by the user via `AskUserQuestion`
on 2026-09-23, recommended option. **One decision is explicitly deferred
to this plan's first tasks, not pre-decided in the design doc:** whether
live buy/sell integrates with vendor `Shop` pricing this phase, or ships
read-only with market-backed trading planned before Phase 20 rumours —
resolve it after tracing every price and transaction path (task 1 below),
and record the outcome at the top of the design doc's "Scope" section
before writing any trading-integration code.

## Tasks

- [x] **Reconnaissance (no code yet):** *(done 2026-09-23: config lives
      in the module's config overlay, `Modules.market.Markets`; vendor
      trading deferred to Phase 19b, before Phase 20. See the design
      doc's "Reconnaissance outcome".)* read
      `internal/characters/shop.go`, `internal/rooms/roommanager.go`'s
      `ZoneConfig`, `internal/usercommands/{buy,sell,offer}.go`, and
      `internal/mobs.Mob.GetSellPrice` in full. Decide and record in the
      design doc's Scope section:
  1. Where market config (tracked item IDs, price bounds, `MaxStock`,
     `TargetStock`, `StartStock`, `DriftStep`) lives — a `ZoneConfig`
     field or a separate zone-keyed datafile. Use the loader with less
     plumbing.
  2. Whether market-backed vendor trading is feasible this phase without
     widening the confirmed narrow scope. Trace quote and transaction
     prices, inventory changes, and stock availability; `ShopItem.Price`
     alone is not the whole transaction. If deferred, put trading in
     Phase 20's prerequisites or a separate intervening slice before
     rumours depend on it.
- [x] **`internal/market` (new pure package): `Good`, `Validate`,
      `PriceForStock`, `DriftStock`.**
  - Tests first, `internal/market/market_test.go`:
    - `TestGoodValidateRejectsNonPositivePrices`
    - `TestGoodValidateRejectsInvalidBounds` (price ordering, target
      strictly between 0 and `MaxStock`, start stock range, drift step
      range, and non-positive item ID)
    - `TestPriceForStockAtZeroReturnsMaxPrice`
    - `TestPriceForStockAtTargetReturnsBasePrice`
    - `TestPriceForStockAtOrAboveMaxStockReturnsMinPrice`
    - `TestPriceForStockMonotonicallyDecreasing` (sample both
      interpolation segments and out-of-range stock)
    - `TestPriceForStockAvoidsOverflow` (extreme valid integers)
    - `TestDriftStockStaysWithinBoundsAndConverges` (many iterations
      from varied starts, including out-of-range values)
    - `TestDriftStockIsDeterministicForFixedRoll`
  - Config-loader tests: reject a zone with duplicate item IDs and
    warn/omit goods whose item spec does not exist.
  - Then implement the two-segment interpolation and target-seeking
    bounded drift specified in the design doc.
- [x] **`modules/market` (new module): `Registry`, `ZoneMarket`,
      `GoodStock`, `Store` interface + file-backed implementation +
      test fake.**
  - Mirror `modules/weather`'s `Registry`/`Store`/load-at-boot shape
    (`modules/weather/weather.go`) as closely as the actual code allows;
    read it in full before implementing, don't design from the design
    doc's sketch alone.
  - Tests first: a missing store seeds every configured market zone at
    `Good.StartStock`; an existing store round-trips unchanged; a
    corrupt/unreadable store disables market effects without a panic.
    Distinguish a missing first-boot file from a load failure.
  - Then implement.
- [x] **`modules/market`: `events.NewRound` listener drifting stock.**
  - Test first: a fixed `rollUint64`-style seam (mirror
    `modules/expedition`'s Phase 12b seam) drives a deterministic
    `DriftStock` call across configured zones on a real
    `events.NewRound` firing (through the actual event dispatch, not a
    bypassed helper) — assert the registry's stock changed as expected
    and got persisted via `Store.Save`.
  - Then implement. Confirm this listener only reacts to the event and
    never reads or advances the round counter itself (grep
    `modules/weather` for its own equivalent assertion/comment and match
    it).
- [x] **`market` command (read-only).**
  - Check `modules/light`'s `light` command and/or `modules/exposure`'s
    `temperature` command for the current module-owned-command
    registration convention before writing a new one from scratch.
  - Test first, through the real command dispatch: a player standing in
    a configured market zone sees each tracked good's name, current
    price (matching `PriceForStock` for the stored stock), and a coarse
    stock descriptor; a player in an unconfigured zone sees "no market
    here."
  - Then implement.
- [x] **Skipped per task 1 (trading deferred to Phase 19b).**
      **If task 1 includes market-backed trading:** route vendor
      `buy`, `sell`, and `offer` through a shared market-price lookup
      for tracked goods in market zones. Define when a completed buy
      decrements zone stock and a completed sell increments it, how
      vendor `Shop` quantity and zone stock stay consistent, and what
      happens at zero/full stock. Test the real commands for quote/
      transaction agreement, gold and item transfer, ledger changes,
      persistence, and unchanged prices outside configured markets.
      **If deferred:** record that Phase 20 (or an intervening trade
      slice) must add this before actionable trade rumours; skip this
      task and state the decision in the work log.
- [x] **Content:** configure 2-3 existing zones as markets (reuse zones
      that already exist rather than authoring new ones — check
      Waymark Inn's zone and Dunmar West Gate's zone as candidates), each
      with a small tracked-goods list including at least one Phase 18
      `Commodity` item (only after Phase 18a has landed; if this phase
      starts before 18a merges, use an existing item type instead and
      note the follow-up). Choose at least one `StartStock` away from
      `TargetStock`, with enough price spread for the round-listener
      wiring test to observe both stock and price drift.
- [x] `gofmt -l`, `go vet ./internal/market/... ./modules/market/...`,
      `go build ./...`, `go test -race ./...` after each task.
- [x] `make generate` (registers the new `modules/market` plugin),
      `make validate`.
- [x] **Testing and review gate:** independent reviewer subagent (most
      capable model tier) over the full phase diff, briefed with the
      design doc, the non-negotiable invariants (never advance the world
      clock, survive restart/copyover, lock ordering — `modules/market`
      needs its own leaf mutex if it holds any in-memory state read from
      outside the game loop, same pattern Phase 15's survival mutex and
      Phase 16's camping-lock fix already set), and asked for bugs,
      design gaps, and missing coverage. Verify each finding, fix real
      ones with a regression test, record rejected ones and why.
- [x] Update `docs/PROJECT_STATUS.md`: header, Current position/Next,
      phase table row 19, new work-log entry with a **Review:** line and
      an explicit note of the buy/sell-integration decision made in
      task 1.
