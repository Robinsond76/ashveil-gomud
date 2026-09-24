# Phase 20: Trade Rumours

## Prior-art check

- **Roadmap** (`2026-09-23-environment-skills-economy-roadmap.md`): "Trade
  rumours: `rumors` at taverns/innkeepers, stale/fuzzy hints. Builds on 19."
  The source request: "trade rumours hint at where goods are cheap or
  wanted."
- **Phase 19/19b** (`internal/market`, `modules/market`) own the zone stock
  ledgers, stock-derived ask/bid prices, round-driven drift, and the
  durable `Registry` store. Prices are only visible by standing in a
  market room, so nothing tells a player about another settlement's
  market.
- **Inns** are rooms tagged `inn` (Phase 16, `modules/camping`): the
  Waymark Inn (2003, Dunmar) and Frostfang room 61. Its description
  already has carters "trade news of the forest track".
- **Periodic work** in modules hangs off `events.NewRound` without reading
  the round counter (`modules/weather`, `modules/market` drift).
- **Weighted/random choice** uses an injected `roll func() uint64` so tests
  are deterministic (`modules/market`, `modules/weather`).

## Decisions

The session instruction was "Continue with next phase implementation";
the defaults below are this design's recommendations, applied without a
separate confirmation round, the same way Phase 19b's "carry on" defaults
were. Each is a config knob or a small follow-up if the owner wants it
changed.

1. **Owner: `modules/market`**, not a new module. Rumours are a view of
   the market ledger; putting them there avoids a cross-module provider
   seam and keeps one leaf lock and one durable file.
2. **Where:** rooms carrying the rumour room tag (`RumorRoomTag`, default
   `inn`), in any zone. News travels; an inn outside a market zone still
   hears it.
3. **Stale:** rumours come from a **news snapshot** of every market's
   stock, not the live ledger. The snapshot is refreshed every
   `RumorRefreshRounds` rounds (default 150, 10 minutes at 4-second
   rounds; valid 1..100000), counted by the module's own round listener.
   Between refreshes the markets keep drifting and trading, so the news
   can be out of date.
4. **Fuzzy:** rumours never quote numbers. They name a good and a market
   and say one of four things:
   - **scarce**: the market is short (stock level `none` or `scarce`);
   - **glut**: the market is overstocked (`plentiful` or `glutted`);
   - **cheapest**: among two or more markets trading the good, this one
     had the unique lowest open buy price (a sold-out market isn't
     selling, so it doesn't compete);
   - **best buyer**: among two or more markets, this one had the unique
     highest open sell price (a full market isn't buying).
   Each ask shows at most `RumorsPerAsk` (default 3, valid 1..10) of the
   current rumours, picked at random, each in one of two phrasings.
5. **Places** are named by the market's room title (the room a player
   must walk to), falling back to the zone name.
6. **Free.** No gold or drink purchase; that can come later with inn
   content.
7. **`rumors` and `rumours`** are both registered commands.

**Open owner decision carried from 19b (unchanged):** whether trading
between markets keeps a small standing profit at equilibrium. The
cheapest/best-buyer rumours point at exactly that route, so this phase
makes it easier to find but does not change its size.

## Scope

**In scope:**

- `internal/market/rumor.go` (pure): `Sighting{Place, Good, Stock}`,
  `RumorKind`, `Rumor{Kind, ItemID, Place}`, `Rumors(sightings,
  spreadPct) []Rumor` (deterministic order: item, kind, place), and
  `PickRumors(rumors, n, roll) []Rumor` (up to n distinct, random).
- `modules/market`:
  - `Registry.News` (`news` in the store): a snapshot of stock per zone
    in the same shape as `Zones`. Decoded with the same corruption rules.
  - On load, if the stored news is empty (a store from before this
    phase, or a first boot), the news is taken from current stock and
    saved. The refresh countdown starts at `RumorRefreshRounds`.
  - `onNewRound` counts down and, at zero, snapshots every configured
    market's stock into `News` and saves (together with drift, one save
    per tick).
  - `rumors`/`rumours` command: refused outside a rumour room; lists up
    to `RumorsPerAsk` phrased rumours, or says there's no news worth
    hearing. Refused while persistence is unavailable.
  - Config: `RumorRoomTag`, `RumorRefreshRounds`, `RumorsPerAsk`.

**Known limitations:**

- The refresh countdown is in memory. A restart or copyover restores the
  persisted news but restarts the countdown, so news can be up to one
  extra interval stale after a restart. It never refreshes early.
- Goods sold in only one market get only scarce/glut rumours.

**Explicitly deferred:** paying the innkeeper or buying a drink for
rumours; wrong or invented rumours; rumours about non-market topics
(bandits, routes); per-inn news that only covers nearby markets.

## Durable model

`Registry` gains `News map[string]ZoneMarket` (`yaml:"news,omitempty"`).
It is written in the same atomic plugin file as the ledger. A store
without `news` loads as empty news and is re-snapshotted, so old stores
upgrade in place. A news record missing its stock is corrupt, as in
`Zones`.

## Constraints

- Never reads, drives, or advances the world clock or round counter; the
  countdown only counts `NewRound` events the module receives.
- Leaf mutex: names and room titles are looked up outside it.
- Rumours never change stock, gold, or items.

## Acceptance criteria

- Pure tests: each rumour kind's thresholds; cheapest/best buyer need two
  or more markets and a unique extreme, and closed sides don't compete; output is
  deterministic; `PickRumors` returns distinct rumours, at most n, all
  when n exceeds the count.
- Module tests: news seeded on load from current stock when missing and
  kept when present; refresh after exactly `RumorRefreshRounds` rounds
  and persisted; news survives a reload; refused outside the rumour
  room; empty news message; persistence-unavailable message; config
  parsing with defaults and bounds; decoding rejects a news record with
  no stock.
- Wiring test (extends `TestMarketEndToEndThroughPluginsLoad`): through
  `usercommands.TryCommand` in the shipped Waymark Inn, `rumors` and
  `rumours` report shipped-market news naming the shipped market rooms;
  the West Gate refuses; a real `NewRound` refreshes the news into the
  real store.
- `go test -race ./...`, `make generate`, `make validate` pass.
