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
  persistence failure paths), and a `NewRound` listener that nudges price
  toward a baseline.
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
  trades, each with a baseline price and a stock band (min/max), loaded
  from the zone's own config or a new market datafile — whichever the
  existing `ZoneConfig` loader can absorb with the least new plumbing
  (again, a plan-task decision after reading the real loader).
- `market` command: shows the current zone's tracked goods, their price,
  and a coarse stock descriptor (e.g. "plentiful"/"scarce"), read-only.
- Buying/selling *through* a market in this phase reuses the existing
  `buy`/`sell`/vendor-mob flow if a vendor mob's `Shop` is configured to
  reference the zone's market stock for pricing — **or**, if that
  integration proves non-trivial once the real `buy`/`sell` code is read,
  defer live trading to Phase 20/21 and ship Phase 19 as read-only
  (`market` command + the registry + price drift), which still fully
  satisfies the roadmap's "stock-driven prices, round-driven drift,
  `market` command" phase description on its own. **This is an explicit
  plan-time decision point, not pre-decided here** — the design doc
  intentionally leaves it open because it depends on how entangled
  `shop.go`'s pricing is with `ShopItem.Price` once actually read for the
  plan.
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
    BasePrice    int `yaml:"baseprice"`    // price at MidStock
    MinPrice     int `yaml:"minprice"`     // floor, reached at MaxStock
    MaxPrice     int `yaml:"maxprice"`     // ceiling, reached at MinStock (0)
    MaxStock     int `yaml:"maxstock"`     // stock level at which price bottoms out
    StartStock   int `yaml:"startstock"`   // initial stock on first load
}

func (g Good) Validate() error

// PriceForStock is pure: linearly (or step-banded — pick the simpler one
// in the plan) interpolates between MaxPrice (at stock=0) and MinPrice
// (at stock>=MaxStock).
func (g Good) PriceForStock(stock int) int

// DriftStock nudges stock one round-tick toward a config-driven target,
// bounded so it can never runaway or go negative — same "clamped step,
// never advances anything but its own local counter" shape Phase 8's
// weather transitions and Phase 16's walking multipliers already use.
func (g Good) DriftStock(currentStock int, roll uint64) int
```

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
`BasePrice`/`MinPrice`/`MaxPrice`/`MaxStock`) is data, loaded at boot —
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
  loaded at boot; a missing/corrupt store fails open to "no market
  effects this zone" rather than panicking (mirror whatever
  `modules/weather`'s own load-failure handling already does).
- Data-driven balance: `BasePrice`/`MinPrice`/`MaxPrice`/`MaxStock`/drift
  step size all live in config, no hardcoded numbers in Go.
- Bounded drift only: `DriftStock` must be provably incapable of pushing
  stock negative or past a configured ceiling, however many rounds run —
  a property test (fixed seed, many iterations) enforces this rather than
  trusting the formula by inspection.

## Acceptance criteria

- `internal/market.Good.PriceForStock` returns `MaxPrice` at stock 0,
  `MinPrice` at or above `MaxStock`, and a monotonically-decreasing value
  in between (property-tested, not just point-tested).
- `internal/market.Good.DriftStock` never returns a negative stock or a
  stock exceeding a sane configured bound, across many iterations from
  varied starting stocks (property test).
- A configured market zone's stock and derived price actually move over
  successive `events.NewRound` events, through the real module listener
  (wiring test, not just the pure package), and persist across a
  save/reload of the registry.
- An unconfigured zone's `market` command reports no market present; a
  configured zone's `market` command shows real current prices matching
  `PriceForStock(currentStock)` for its stored stock.
- `go test -race ./...`, `make generate` (new module), `make validate`
  all pass.
