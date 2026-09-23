# Phase 19b: Market Trading

## Prior-art check

- **Phase 19** (`internal/market`, `modules/market`) ships zone stock
  ledgers, a stock-derived price (`Good.PriceForStock`), bounded
  round-driven drift, and a read-only `market` command available
  anywhere in a market zone. Nothing a player does changes stock.
- **Phase 19's deferral** assumed market trading would run through
  shopkeeper NPCs (`buy`/`sell`/`offer`/`list`, `GetSellPrice`, player
  shops, shop hooks). The owner clarified (2026-09-23) that market prices
  belong to the local commodity market only. Shopkeepers are untouched;
  the market is its own counterparty. That removes the shop-path
  coupling entirely.
- **Room-tag opt-in** is the established module pattern: `modules/camping`
  gates camping and inns on configurable room tags (`camping`, `inn`),
  read with `room.HasTag`.
- **Item handover** mirrors `internal/usercommands/buy.go` and `sell.go`:
  `Character.StoreItem`/`RemoveItem`, `Gold` adjusted directly,
  `events.ItemOwnership` and `events.EquipmentChange{GoldChange}` queued,
  a `shop` event-log line, and a room message. `buy` has no backpack
  capacity check (weight is Phase 9's encumbrance concern), so neither
  does this.

## Decisions (owner, 2026-09-23, in conversation)

1. **The market lives in a tagged room**, a town square whose
   description mentions the stalls. Listing prices and trading happen
   only in a room carrying the market room tag inside a configured
   market zone.
2. **Buy and sell prices differ**, and the market listing shows both.

Implementation defaults chosen under the owner's "carry on":

- Tag name `market`, configurable as `RoomTag` (mirrors camping).
- Elsewhere in a market zone, `market` says there is no market here and
  names the zone's market room(s), so players can find it.
- One module-wide `SpreadPct` (default 20, valid 1..90) rather than a
  per-good spread: one knob until content needs more.
- One item per `market buy`/`market sell`.

## Scope

**In scope:**

- `internal/market`: `Good.AskForStock(stock)` (the price you pay: the
  stock price, unavailable at stock 0) and `Good.BidForStock(stock,
  spreadPct)` (the price the market pays you, unavailable at `MaxStock`).
  The bid is `price - floor(price*spread/100)`, capped at
  `AskForStock(stock+1) - 1` so no buy-then-sell or sell-then-buy round
  trip can ever profit, however steep the curve. A bid below 1 means the
  market is not buying.
- `modules/market`:
  - `market` (no args) in a market room lists each good with both
    prices and the stock descriptor; `-` marks a side that is closed.
  - `market buy <good>` / `market sell <good>`, matched against the
    zone's tracked goods by item name (exact, then prefix, then
    substring, case-insensitive). Buying needs stock of 1 or more and
    enough gold; selling needs the item in the backpack, not a quest
    item or special item, and stock below `MaxStock`.
  - The price check, stock change, and ledger save happen in one
    critical section under the module's leaf mutex; gold and the item
    change hands only after it commits, on the game loop (commands run
    there, so nothing else mutates the character concurrently). A save
    failure is logged; the in-memory ledger keeps the trade and the next
    save persists it (weather's precedent). Trading is refused while
    persistence is unavailable.
- Content: a new **Dunmar Market Square** (room 2004, east of the West
  Gate) and a new **Trappers' Post** (room 2005, in Old Kings Road, east
  of the Fork at the Black Oak), both tagged `market`, with stalls in
  their descriptions. The West Gate description gains a line pointing
  east.

**Explicitly deferred:** shopkeeper trading at market prices (not wanted);
bulk quantities; per-good spreads; admin ledger reset; market-room
fixtures or NPC traders.

## Durable model

No new durable state. Trades change the Phase 19 `Registry` stock via
the same `Store`; the restart/copyover story is unchanged.

## Constraints

- Never touches the world clock or round counter.
- Leaf mutex: no other package's locking code runs while it is held
  (item names, room lookups, and character changes happen outside it).
- Stock stays within `[0, MaxStock]` under any mix of trades and drift.

## Acceptance criteria

- Pure tests: ask/bid endpoints and closure; bid below ask; no
  profitable round trip across every stock level of several curves,
  including steep and flat ones and extreme integers.
- Through `usercommands.TryCommand` in the shipped market room: listing
  shows both prices; a buy and a sell move gold, items, and stock, and
  persist to the real store; the next listing reflects the new stock.
- Refusals: outside a market room (with the market room named), unknown
  good, not enough gold, stock empty, market full, item not carried,
  quest item, persistence unavailable.
- `go test -race ./...`, `make generate`, `make validate` pass.
