# Phase 36c — loot economy

Implements slice 4 of the owner-approved
[loot system design](../designs/2026-10-05-loot-system-design.md) and the
36a deferrals (merchants buying rolled gear, GMCP rolled names), on top of
36a's item model, 36b's goods and 37's drops. Branch
`claude/phase-36c-loot-economy-xaj6l8`.

## What ships

- **Merchants buy rolled gear.** `Item.IsSpecialForSale` replaces `IsSpecial`
  in merchant sale and offer: a blob, spent uses and a hand-edited override
  still refuse, but a roll's own override does not. `Item.SaleBaseValue` is
  the quality-and-affix value for read gear (the spec already holds both). An
  unread Rare or better is priced by its rarity alone (Rare 180%, Epic 260%,
  Legendary and Set 400% of the quality value) at a further 60%, so the price
  never leaks hidden affixes and reading first pays. A shop's stocked
  `price:` no longer overrides a rolled item's own worth. Merchant stock
  saturation (the payout falls as the shop stocks more of the item, recovering
  with world time) already existed and now applies to rolled sales, which
  stock the base item.
- **`mark [item] junk|keep` and `sell junk`.** `Item.Junk` saves with the
  item and shows a `j` flag. `sell junk` sells every marked item and every
  plain junk-type item to the merchants in the room in one transaction with a
  summary; items nobody wants stay and are named.
- **`salvage [item]` at a smith.** A smith is a merchant whose own wares
  (not player-sold temporary stock) include a weapon or armor
  (`Mob.IsSmith`). `loot.SalvageYield`: metal gear gives scrap iron (tier 1),
  iron ore (tier 2) or steel ingots (tier 3+), staves and bows ashwood,
  leather armor tanned leather, a cloth robe silk (scraps give nothing),
  rings, packs and unaudited armor nothing. Weight sets the count, Superior
  and Exquisite add 1 or 2, rarity adds 1/2/3 (Rare/Epic/Legendary or Set),
  tier 4+ or Epic+ adds a runestone shard. A test holds every catalog piece's
  yield worth no more than the piece, so buying gear to salvage never pays.
  Refused when the materials would overload the company.
- **Identification fees.** `appraise [item]` at a merchant reads an
  unidentified rolled item for good: 60 gold (Rare), 150 (Epic), 400
  (Legendary or Set); everything else stays the 20 gold appraisal.
- **Goods in markets, with saturation.** Dunmar's market now tracks all 24
  trade goods (it pays most for materials and valuables), the Trappers' Post
  tracks trophies, salvage and a few materials at lower prices and glutted
  start stocks, so hauling from the post to town pays. The existing
  stock-driven price is the saturation: each unit sold lowers the price, a
  glutted market stops buying, and stock drifts back each round. Bare "hide"
  is now ambiguous (wolf hide, bear hide); the market asks which.
- **GMCP.** `Char.Inventory` (backpack and worn) and `Room.Info.Contents.Items`
  carry `label` (the name as players see it), `rarity` and `unidentified` for
  rolled items, and the web gear and room windows show the label. `name`
  stays the plain command name. Company.Inventory already carried labels.
- **Help and tutorial.** New `help salvage` and `help mark`; updated `sell`,
  `appraise`, `identify`, `goods`, `market` and `loot`; `salvage` is no
  longer an alias of `goods`; the Departure lesson's loot hints cover selling,
  junk, reading for a fee, saturation and salvage.

## Decisions (made under the owner's delegation)

| Question | Decision | Reason |
|---|---|---|
| Where does saturation live? | The market ledger's stock-based price, plus the merchant's per-item stock scaling | Both already persist and recover with world time; no second system |
| Identification scrolls | Not built; the fee and the Scribe cover reading | The roadmap's 36c scope lists fees, not scrolls; a scroll is a later consumable |
| Salvage cost | Free, final | The gear is the sink; a fee would only punish clearing a pack |
| Unread gear's price | By rarity at a 40% discount, never by its affixes | Prices must not leak what a Scribe would find |
| Sage as an identifier | Merchants only; there are no sage mobs yet | Keeps scope to existing actors |
| `sell junk` at a market | Merchants only; `market sell` stays one good at a time | The market is its own one-unit trader |

## Accepted and deferred

- Frostfang's trader (Brynja) sold iron ore and tanned leather under
  Dunmar's market price, a small buy-there, sell-at-market loop. Fixed in
  review: her prices sit above every market's target-stock price, held by
  `TestShopkeepersNeverUndercutMarketsOnTradedGoods`.
- Rarity colours in web windows (the label carries the words only).
- Scrolls, smiths raising quality (Field Smith) and crafting from materials.
