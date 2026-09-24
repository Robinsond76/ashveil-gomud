# Phase 21b: Settlement Standing

## Prior-art check

- **Roadmap** decision 2(4): alignment affects "settlement standing
  (prices, inn access, black-market access)". Phase 21a
  (`2026-09-24-phase-21a-company-alignment-design.md`) split this out.
- **Company alignment** (Phase 21a): `modules/company` computes the
  company average (the online leader and every companion, engine scale
  −100..100) for the recruit gate. Nothing outside the module can read it
  yet. `internal/company` already has provider seams
  (`FormationProvider`, `ArchetypeProvider`) that modules/hooks use without
  importing `modules/company`.
- **Settlements are zones** (Phase 19): only configured zones have a
  market. The market is in rooms tagged `market` (`RoomTag`); trading is
  `market buy`/`market sell` against the zone ledger, with the invariant
  that no buy-then-sell round trip within a market profits
  (`modules/market`, `commitBuy`/`commitSell`).
- **Inns** (Phase 16, `modules/camping`): rooms tagged `inn`; `inn rest`
  charges `PricePerMember` × members for a durable real-time stay.
  Camping never holds its lock while calling another module. Shipped inns:
  the Waymark Inn (2003, Dunmar) and Frostfang room 61.
- **Shopkeepers** stay untouched (owner decision, Phase 19b).

## Decisions

Applied under the owner's "Yes" to "go on to Phase 21b" after the 21a
report, which proposed this scope; defaults are this design's
recommendations and each is a config knob.

1. **Owner: a new `modules/standing`**, with pure rules in
   `internal/standing`. Standing is derived, not stored: it is recomputed
   from the company's alignment and the settlement's every time, so there
   is no new durable state and nothing to recover after a restart
   (alignments are already durable).
2. **Settlements** are configured zones with an alignment (engine scale):
   `Settlements: [{Zone, Alignment}]`. Shipped: Dunmar 40 (virtuous),
   Old Kings Road 0 (neutral), Frostfang 30 (lawful). A zone that isn't
   configured has no standing effects.
3. **Whose alignment:** the company average from Phase 21a (the leader
   alone when there are no companions). The company walks in together.
4. **Tiers** by the gap between the company and the settlement:
   **welcome** (≤ `WelcomeGap` 40), **tolerated** (≤ `ToleratedGap` 80),
   **distrusted** (≤ `DistrustedGap` 130), **shunned** (beyond).
5. **Effects are penalties only.** Welcome and tolerated trade and rest at
   the normal price. Distrusted pays `DistrustedMarkupPct` (20) more to buy
   at the market, gets `DistrustedMarkupPct` less when selling, and pays
   `DistrustedInnMarkupPct` (50) more for a room. Shunned companies are
   refused by the market and the inn. Penalties only (no welcome
   discount) keep Phase 19b's invariant that buying and selling straight
   back never profits, with no re-capping: buy prices only rise and sell
   prices only fall.
6. **Black market:** rooms tagged `BlackMarketRoomTag` (default
   `blackmarket`) in a market zone trade the same zone ledger at normal
   prices, but only with **distrusted or shunned** companies; anyone in
   better standing is turned away ("no one here will deal with you").
   A black market in a zone that isn't a configured settlement serves no
   one. Shipped: Tanner's Back Alley (2006), south of Dunmar Market
   Square.
7. **`standing` command:** your company's standing in the current
   settlement (both alignments on the 1–100 display, the tier, and its
   effects), then every other settlement's tier, so players can plan.

## Scope

**In scope:**

- `internal/standing` (pure): `Tier`, `Rules` (`DefaultRules`),
  `Assess(company, settlement int, rules) Standing` with the tier and
  effects; `Standing.BuyPrice(p)`, `SellPrice(p)` (rounded against the
  player, sell never below 1), `InnPrice(p)`, `MarketRefused()`,
  `InnRefused()`, `BlackMarketServes()`; a provider seam
  (`SetProvider`, `For(leaderUserID, zone) (Standing, bool)`).
- `internal/company`: an `AlignmentProvider` seam
  (`CompanyAlignment(leaderUserID) (int, bool)`) implemented by
  `modules/company` with its existing company average.
- `modules/standing`: config (settlements, gaps, markups, black market
  tag) with validation and defaults, the provider, the `standing` command.
- `modules/market`: listing, buy, and sell apply the standing (prices,
  refusal); black-market rooms count as market rooms for the listing and
  trades, gated by `BlackMarketServes`; elsewhere in the zone the "market
  is at" hint names only the ordinary market rooms.
- `modules/camping`: `inn`/`inn rest` apply the inn markup or refusal,
  read before taking the camping lock.
- Content: room 2006 and an exit from 2004.

**Explicitly deferred:** discounts for good standing; per-settlement
reputation earned by deeds (standing is alignment only); guards or
encounters reacting to standing; shopkeeper prices; black-market-only
goods.

**Implementation review additions (2026-09-24):** a room tagged both
`market` and the black-market tag is a black market for companies it
serves and an ordinary market for everyone else (so a `BlackMarketRoomTag`
equal to the market tag degrades gracefully instead of refusing welcome
companies everywhere). The `standing` text is worded conditionally ("any
market here", "a black market, if there is one"), since not every
settlement has a market, inn, or black market. The inn shows its markup
beside the per-member price.

**Known limitations (accepted):**

- Standing fails open: if company data is unavailable (a failed load),
  there is no standing and markets and inns behave as before, while
  black markets serve no one. This matches "no provider, no effects".
- The gap is symmetric, so a saintly company is distrusted at a
  low-alignment settlement just as an evil one is at a virtuous town.
  With the shipped alignments only Dunmar can shun anyone (below 5 on the
  1–100 display); Frostfang and Old Kings Road can only distrust.
- An empty or non-string `BlackMarketRoomTag` falls back to
  `blackmarket`; black markets are turned off by not tagging any room.

## Constraints

- Never reads or advances the world clock; no periodic work.
- No new durable state.
- Camping reads standing before taking `m.mu`; market reads it outside
  `m.mu` and passes plain percentages into the locked commit.
- With no standing provider (tests of other modules, or the module
  absent), everything behaves exactly as before.

## Acceptance criteria

- Pure tests: tier boundaries; price rounding in the player's disfavour;
  sell floor; buy-then-sell never profits for any stock level under any
  tier; refusal flags; black market serves only distrusted/shunned.
- Module tests: config parsing (defaults, bounds, unknown zones, gap
  order); `standing` output in and out of a settlement; the provider uses
  the company seam.
- Market tests: distrusted listing and trades use the markup; shunned is
  refused in the market room but trades in the black market; a welcome
  company is refused at the black market; no provider means unchanged
  prices.
- Camping tests: distrusted inn price in `inn` and `inn rest`; shunned
  refused without taking gold.
- Wiring test (`modules/standing`, whose test binary also imports the
  real `modules/company` and `modules/market` so `plugins.Load` wires all
  three): with the shipped world's items and Dunmar rooms, `standing`
  through `usercommands.TryCommand`; a distrusted company's `market`
  listing and `market buy` in Dunmar Market Square carry the markup; a
  shunned company is refused there and trades in Tanner's Back Alley
  (2006), which carries the shipped black-market tag and connects to
  2004; a welcome company is turned away from 2006.
- `go test -race ./...`, `make generate`, `make validate` pass.
