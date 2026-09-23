# Phase 19: Commodities and Markets

## Prior-art check

- **Shops today** (`internal/characters/shop.go`, `internal/usercommands/
  {buy,sell,offer}.go`): `Shop []ShopItem`, each a flat authored `Price`
  (not derived from `ItemSpec.Value`), `Quantity`/`QuantityMax` (0 =
  unlimited, -1 = temporary one-off), and a `RestockRate` game-time period
  restored via `Shop.Restock()`. Shops are attached to a mob (vendor NPC),
  not to a zone/settlement — there is no settlement-as-economic-entity
  concept anywhere in the shop system.
- **Gold** is a flat per-`Character.Gold int` (`internal/characters/
  character.go:65`), unaffected by this phase's design (still just added/
  subtracted at trade time, same as `buy`/`sell` do today).
- **Items** (`internal/items/itemspec.go`): `Value` and `Weight` already
  exist; Phase 18 adds a `Commodity` `ItemType` for animal trade goods.
  `Botanical`/`Junk`/`Object` already exist for other tradeable non-
  equipment items. Markets in this phase trade *any* item with a `Value`,
  not just the new `Commodity` type — a settlement's stock ledger is
  keyed by `ItemId`, type-agnostic.
- **Zones** (`internal/rooms/roommanager.go`): the only existing grouping
  of rooms into a named economic-shaped unit is `Zone string` (already on
  `Character`, `Mob`, rooms) plus `ZoneConfig`/`GetZoneConfig`/
  `GetAllZoneNames`. There is no separate "settlement" entity; per the
  roadmap's own engine-facts table this phase must introduce its own
  settlement concept, and the cheapest one that needs no new content
  authoring is **a settlement = a zone name**, reusing what every room
  already carries.
- **Cargo** (`internal/encumbrance`, Phase 9): commodities carried by
  players are ordinary items in `Cargo` stacks — no new mechanism needed
  there; confirmed in the Phase 18 research pass and the handoff doc's
  section 17 ("trade goods" was already an anticipated cargo category).
- **The pattern this phase mirrors**: `modules/weather`
  (`modules/weather/weather.go:1-60`) is the closest existing precedent
  for "a durable, zone-keyed registry, ticked by `events.NewRound` (never
  real time, never advancing the clock), with a read-only query command."
  Phase 19's market registry follows the same shape: `Registry{Zones
  map[string]ZoneMarket}`, a `Store` interface for injectable
  load/save (weather's own `Store` abstraction, same reason: testable
  persistence failure paths), and a `NewRound` listener that nudges stock
  toward a configured target; price is always derived from current stock.
- **Restock's game-time period** (`shop.go`'s `RestockRate`) is a
  precedent for "periodic economic change keyed off elapsed game time,"
  but this phase's price drift is simpler: round-driven, not time-period
  driven, matching weather's own `events.NewRound` cadence rather than
  inventing a second timing mechanism.

## Decisions confirmed (user, 2026-09-23, via AskUserQuestion)

- **Engine-first, narrow scope for the first slice.** A per-settlement
  (zone-scoped) stock ledger for commodity-shaped items, with **prices
  drifting via a bounded round-driven walk toward a config baseline based
  on stock level** (more stock = cheaper, less stock = pricier) — not a
  live authored production/consumption simulation. A `market` command to
  inspect current stock and prices. No shipped commodity trade routes or
  "buy low here, sell high there" content beyond proving the mechanism —
  same deferral pattern 12a/12b/16/18 have each already used.

## Scope

**Reconnaissance outcome (plan task 1, 2026-09-23; applied under the
user's "begin next phase" instruction, recommended options):**

1. **Config location: the module's own config overlay**
   (`modules/market/files/data-overlays/config.yaml`, read as
   `Modules.market.Markets` via `plug.Config.Get`), a list of
   `{Zone, Goods: [...]}` entries. This is exactly how `modules/weather`
   ships its biome tables. A `ZoneConfig` field would need new
   `internal/rooms` plumbing, zone-config save round-trips, and admin
   editor support for no gain; the overlay needs none. Item specs load
   (`main.go` `loadAllDataFiles`) before `plugins.Load`, so the module
   can verify item IDs at load.
2. **Vendor trading: deferred to a dedicated slice, Phase 19b
   (market-backed trading), which must land before Phase 20 rumours.**
   Price reaches a transaction through at least five independent paths:
   `buy.go`'s `tryPurchase` (its own item/merc/buff/pet price maps and
   `Shop.Destock`), `list.go` (a separate price display), `sell.go` and
   `offer.go` via `mobs.Mob.GetSellPrice` (which scales a sell quote by
   the vendor's current `Quantity/20` and prices never-stocked items by
   type/subtype heuristics), player-owned shops (`shopUser`), and
   `shophooks.go` script hooks. `Shop.StockItem` also creates temporary
   vendor entries on every sale, and `Shop.Restock` refills vendor
   quantity on a game-time period independent of any zone ledger.
   Making quote, transaction, vendor quantity, and zone stock agree
   means changing all of these together, which is a phase of its own,
   not the confirmed narrow slice. Phase 19 ships read-only.

**In scope:**
- A new pure package, `internal/market` (mirrors `internal/climate`,
  `internal/loot`): the price-drift math and stock-band pricing formula,
  fully unit-testable with no engine imports.
- A new module, `modules/market` (mirrors `modules/weather`'s shape): a
  durable `Registry{Zones map[string]ZoneMarket}`, a `Store` interface
  (real file-backed implementation + test fake), boot-time load, a
  `events.NewRound`-driven per-zone price/stock update (bounded — see
  Durable model), and the `market` command.
- Settlement = zone. Only zones an admin explicitly configures as a market
  (a new `market` entry in that zone's config, or a curated allow-list —
  decide the exact shape in the plan's first task after re-reading
  `ZoneConfig`'s actual fields) get a `ZoneMarket` record; every other zone
  is untouched, so this never silently turns every zone in the game into a
  shop.
- A market's tracked goods are data-driven per zone: which `ItemId`s it
  trades, each with minimum/base/maximum prices, a stock ceiling,
  starting and target stock, and a per-round drift step, loaded from
  the zone's own config or a new market datafile — whichever the
  existing `ZoneConfig` loader can absorb with the least new plumbing
  (again, a plan-task decision after reading the real loader).
- `market` command: shows the current zone's tracked goods, their price,
  and a coarse stock descriptor (e.g. "plentiful"/"scarce"), read-only.
- Buying/selling *through* a market is a plan-time decision. The
  existing price paths include `internal/usercommands/buy.go`,
  `sell.go`, `offer.go`, `internal/mobs.Mob.GetSellPrice`, and
  `ShopItem.Price`; changing that field alone cannot guarantee a
  consistent quote or transaction. Integrate this phase only if those
  paths can share a market price source and each completed trade can
  update the zone ledger and vendor stock coherently. Otherwise ship
  the confirmed narrow read-only Phase 19 and make market-backed trading
  a prerequisite in Phase 20 (or an explicit intervening trade slice)
  before trade rumours depend on actionable prices. Record the choice
  in this section after the plan's first reconnaissance task.
- Content: 2-3 zones tagged as markets (reusing zones that already exist,
  e.g. wherever the Waymark Inn or Dunmar West Gate sit), each with a
  small tracked-goods list including at least one Phase 18 `Commodity`
  item, enough to prove the mechanism end-to-end.

**Explicitly deferred:**
- Authored production/consumption simulation (declined in the confirmed
  decision above) — Phase 19 only does the bounded price-toward-baseline
  walk.
- Trade rumours (Phase 20 — explicitly builds on this phase).
- Cross-settlement price divergence driven by anything other than each
  zone's own stock level (no simulated trade routes moving goods between
  zones).
- Bandits dropping commodities (that's Phase 18's loot-table mechanism,
  already covers it once a bandit mob gets `LootCategory: bandit` with
  `Commodity` items in its table — no new work needed here).
- Any new UI beyond the `market` command.

## Durable model

```go
// internal/market/market.go (new package, pure)

// Good is one tracked commodity in a settlement's market.
type Good struct {
    ItemID       int `yaml:"itemid"`
    BasePrice    int `yaml:"baseprice"`   // price at TargetStock
    MinPrice     int `yaml:"minprice"`    // floor, reached at MaxStock
    MaxPrice     int `yaml:"maxprice"`    // ceiling, reached at stock 0
    MaxStock     int `yaml:"maxstock"`    // stock ceiling
    TargetStock  int `yaml:"targetstock"` // baseline stock, strictly inside (0, MaxStock)
    StartStock   int `yaml:"startstock"`  // initial stock on first load
    DriftStep    int `yaml:"driftstep"`   // maximum stock movement per NewRound
}

func (g Good) Validate() error

// PriceForStock clamps stock to [0, MaxStock] and uses two monotone
// linear segments: (0, MaxPrice) -> (TargetStock, BasePrice), then
// (TargetStock, BasePrice) -> (MaxStock, MinPrice). Subtract the
// floor of each segment's proportional price decrease; endpoints are
// exact. Avoid overflow when multiplying price difference by stock.
func (g Good) PriceForStock(stock int) int

// DriftStock clamps currentStock to [0, MaxStock], then moves toward
// TargetStock by min(distance, 1 + roll % uint64(DriftStep)). At target it
// stays put. It changes only this good's stock, never the world clock.
func (g Good) DriftStock(currentStock int, roll uint64) int
```

`Good.Validate` requires `ItemID > 0`,
`0 < MinPrice <= BasePrice <= MaxPrice`,
`0 < TargetStock < MaxStock`,
`0 <= StartStock <= MaxStock`, and `1 <= DriftStep <= MaxStock`.
The config loader additionally verifies each item ID has a loaded item
spec. It rejects a zone containing duplicate item IDs so no ambiguous
stock record is created. Invalid goods are warned about and omitted; a
zone with no valid goods has no market.
Interpolation must remain safe for the largest validated integer values.

```go
// modules/market/market.go (new module, mirrors modules/weather's shape)

type GoodStock struct {
    ItemID int `yaml:"itemid"`
    Stock  int `yaml:"stock"`
}

type ZoneMarket struct {
    Goods []GoodStock `yaml:"goods"` // current stock per tracked good
}

type Registry struct {
    Zones map[string]ZoneMarket `yaml:"zones"` // keyed by zone name
}

type Store interface {
    Load(*Registry) error
    Save(Registry) error
}
```

Config (which zones are markets, and each market's `Good` list with
`BasePrice`/`MinPrice`/`MaxPrice`/`MaxStock`/`TargetStock`/
`StartStock`/`DriftStep`) is data, loaded at boot —
exact file location decided in the plan after reading how `ZoneConfig`
already loads its own data, to avoid inventing a second zone-config
loading path if one already fits.

## Module / integration

- Boot: `modules/market` loads its `Store`-backed `Registry`; any
  configured market zone missing from the loaded registry gets seeded at
  each `Good.StartStock` (first-boot / newly-added-market case), mirroring
  `modules/weather`'s own "missing zone gets a fresh record" pattern.
- Round tick: a `events.NewRound` listener (module-registered, same
  registration style as `modules/weather`) walks every configured
  `ZoneMarket`, calling `Good.DriftStock` per tracked good with a
  module-owned RNG seam (mirroring `modules/expedition`'s `rollUint64`
  seam from Phase 12b, for the same reason: deterministic tests). This
  never reads or advances the shared round counter itself — it only reacts
  to the event, exactly as weather's own doc comment states
  (`modules/weather/weather.go:5-7`).
- `market` command (new, in `internal/usercommands` or `modules/market`
  wherever the existing command-registration convention for a
  module-owned command lives — check `modules/light`'s `light` command
  or `modules/exposure`'s `temperature` command for the precedent, both
  recent module-owned read-only commands): looks up the caller's current
  room's zone, and if it's a configured market, renders each tracked
  good's name, current price (via `Good.PriceForStock(stock)`), and a
  coarse stock descriptor. Otherwise: "There's no market here."
- Persistence: `Registry` saves on every stock change (or batched per
  round tick — decide based on how `modules/weather` batches its own
  saves, to stay consistent) via the `Store` interface, so a
  restart/copyover restores the exact same stock levels and derived
  prices — no re-seeding, no lost drift.

## Constraints and deferrals

- Never advances GoMud's global clock/round count: the market tick is a
  reactive `events.NewRound` listener, same invariant `modules/weather`
  already documents and tests for itself.
- Must survive restart/copyover: `Registry` is durable, `Store`-backed,
  loaded at boot. A missing store file is normal first boot and seeds
  configured markets at `StartStock`; a corrupt or unreadable store
  disables market effects rather than panicking (compare
  `modules/weather`'s load-failure handling).
- Data-driven balance: `BasePrice`/`MinPrice`/`MaxPrice`/`MaxStock`/drift
  step size, target stock, and starting stock all live in config, no
  hardcoded balance numbers in Go.
- Bounded drift only: `DriftStock` must be provably incapable of pushing
  stock negative or past `MaxStock`, however many rounds run. It converges
  to `TargetStock`; with no trading or production simulation, it then
  stays there. A fixed-seed property test enforces the bounds and
  convergence. Use `StartStock != TargetStock` in proving content so
  round ticks visibly change stock and price; choose a price spread
  large enough for an integer price change during the proving slice.

## Acceptance criteria

- `internal/market.Good.PriceForStock` returns `MaxPrice` at stock 0,
  `BasePrice` at `TargetStock`, `MinPrice` at or above `MaxStock`,
  and a monotonically non-increasing value in between (including extreme
  valid integers without overflow). `Good.Validate` and config loading
  reject invalid bounds, missing item specs, and duplicate zone goods.
- `internal/market.Good.DriftStock` never returns stock outside
  `[0, MaxStock]`, moves toward `TargetStock` for any off-target
  starting value, and stays there after convergence (property tests).
- A configured market zone's stock and derived price actually move over
  successive `events.NewRound` events, through the real module listener
  (wiring test, not just the pure package), and persist across a
  save/reload of the registry.
- An unconfigured zone's `market` command reports no market present; a
  configured zone's `market` command shows real current prices matching
  `PriceForStock(currentStock)` for its stored stock.
- `go test -race ./...`, `make generate` (new module), `make validate`
  all pass.
