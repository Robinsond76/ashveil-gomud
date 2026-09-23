# Phase 19: Commodities and Markets — implementation plan

See the companion design doc
(`docs/superpowers/specs/2026-09-23-phase-19-commodities-markets-design.md`).
Open decisions (engine-first narrow scope: stock ledger + bounded
round-driven price drift + read-only `market` command, no authored
production/consumption sim) confirmed by the user via `AskUserQuestion`
on 2026-09-23, recommended option. **One decision is explicitly deferred
to this plan's first tasks, not pre-decided in the design doc:** whether
live buy/sell integrates with vendor `Shop` pricing this phase, or ships
read-only with trading following in Phase 20/21 — resolve it once
`shop.go` and `ZoneConfig` are actually read (task 1 below), and record
the outcome at the top of the design doc's "Scope" section before writing
any trading-integration code.

## Tasks

- [ ] **Reconnaissance (no code yet): read `internal/characters/shop.go`
      in full and `internal/rooms/roommanager.go`'s `ZoneConfig` in
      full.** Decide and record (amend the design doc's Scope section):
  1. Where market config (tracked goods, base/min/max price, max stock)
     lives — a new field on `ZoneConfig`, or a separate market datafile
     keyed by zone name. Pick whichever needs the least new loader
     plumbing.
  2. Whether `buy`/`sell` integration is in-scope this phase (only if
     `ShopItem.Price` can cleanly defer to `market.Good.PriceForStock`
     without restructuring `shop.go`) or deferred, per the design doc's
     explicit fallback.
- [ ] **`internal/market` (new pure package): `Good`, `Validate`,
      `PriceForStock`, `DriftStock`.**
  - Tests first, `internal/market/market_test.go`:
    - `TestGoodValidateRejectsNonPositivePrices`
    - `TestGoodValidateRejectsMaxStockZero`
    - `TestPriceForStockAtZeroReturnsMaxPrice`
    - `TestPriceForStockAtOrAboveMaxStockReturnsMinPrice`
    - `TestPriceForStockMonotonicallyDecreasing` (property test: sample
      many stock levels 0..MaxStock*2, assert non-increasing price)
    - `TestDriftStockNeverNegative` (property test: many iterations from
      varied starting stocks and rolls, assert stock stays >= 0)
    - `TestDriftStockNeverExceedsConfiguredCeiling` (same shape, upper
      bound)
    - `TestDriftStockIsDeterministicForFixedRoll`
  - Then implement.
- [ ] **`modules/market` (new module): `Registry`, `ZoneMarket`,
      `GoodStock`, `Store` interface + file-backed implementation +
      test fake.**
  - Mirror `modules/weather`'s `Registry`/`Store`/load-at-boot shape
    (`modules/weather/weather.go`) as closely as the actual code allows;
    read it in full before implementing, don't design from the design
    doc's sketch alone.
  - Tests first: load with an empty store seeds every configured market
    zone at `Good.StartStock`; load with an existing store round-trips
    unchanged; a `Store.Load` failure fails open (no market anywhere,
    documented in a comment, no panic).
  - Then implement.
- [ ] **`modules/market`: `events.NewRound` listener drifting stock.**
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
- [ ] **`market` command (read-only).**
  - Check `modules/light`'s `light` command and/or `modules/exposure`'s
    `temperature` command for the current module-owned-command
    registration convention before writing a new one from scratch.
  - Test first, through the real command dispatch: a player standing in
    a configured market zone sees each tracked good's name, current
    price (matching `PriceForStock` for the stored stock), and a coarse
    stock descriptor; a player in an unconfigured zone sees "no market
    here."
  - Then implement.
- [ ] **If task 1 decided buy/sell integrates this phase:** wire
      `ShopItem.Price` (or the sell-price path) for market-zone vendor
      mobs to consult `market.Good.PriceForStock`, with its own
      wiring test through the real `buy`/`sell` commands. **If deferred:**
      skip this task; the design doc's Scope section already documents
      the fallback, and the work-log entry below should say so
      explicitly.
- [ ] **Content:** configure 2-3 existing zones as markets (reuse zones
      that already exist rather than authoring new ones — check
      Waymark Inn's zone and Dunmar West Gate's zone as candidates), each
      with a small tracked-goods list including at least one Phase 18
      `Commodity` item (only after Phase 18a has landed; if this phase
      starts before 18a merges, use an existing item type instead and
      note the follow-up).
- [ ] `gofmt -l`, `go vet ./internal/market/... ./modules/market/...`,
      `go build ./...`, `go test -race ./...` after each task.
- [ ] `make generate` (registers the new `modules/market` plugin),
      `make validate`.
- [ ] **Testing and review gate:** independent reviewer subagent (most
      capable model tier) over the full phase diff, briefed with the
      design doc, the non-negotiable invariants (never advance the world
      clock, survive restart/copyover, lock ordering — `modules/market`
      needs its own leaf mutex if it holds any in-memory state read from
      outside the game loop, same pattern Phase 15's survival mutex and
      Phase 16's camping-lock fix already set), and asked for bugs,
      design gaps, and missing coverage. Verify each finding, fix real
      ones with a regression test, record rejected ones and why.
- [ ] Update `docs/PROJECT_STATUS.md`: header, Current position/Next,
      phase table row 19, new work-log entry with a **Review:** line and
      an explicit note of the buy/sell-integration decision made in
      task 1.
