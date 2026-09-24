package market

import (
	"errors"
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/market"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/standing"
	"github.com/GoMudEngine/GoMud/internal/users"
)

var (
	errSoldOut     = errors.New("sold out")
	errMarketFull  = errors.New("market full")
	errNotAfford   = errors.New("not enough gold")
	errNotTracked  = errors.New("not traded here")
	errUnavailable = errors.New("ledgers unavailable")
)

// errAmbiguous reports a name matching more than one good; names lists
// the display names of the candidates.
type errAmbiguous struct{ names []string }

func (e errAmbiguous) Error() string { return "ambiguous: " + strings.Join(e.names, ", ") }

// goodsFor returns a copy of the zone's configured goods.
func (m *MarketModule) goodsFor(zone string) []market.Good {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]market.Good(nil), m.markets[zone]...)
}

// matchGood finds the tracked good a player named: an exact name first,
// then a name prefix, then a substring, case-insensitively. A stage that
// matches more than one good is ambiguous rather than a guess.
func (m *MarketModule) matchGood(goods []market.Good, query string) (market.Good, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return market.Good{}, errNotTracked
	}
	for _, test := range []func(name string) bool{
		func(name string) bool { return name == query },
		func(name string) bool { return strings.HasPrefix(name, query) },
		func(name string) bool { return strings.Contains(name, query) },
	} {
		matched := []market.Good{}
		for _, g := range goods {
			for _, name := range m.itemNames(g.ItemID) {
				if test(strings.ToLower(name)) {
					matched = append(matched, g)
					break
				}
			}
		}
		switch len(matched) {
		case 0:
			continue
		case 1:
			return matched[0], nil
		}
		names := make([]string, len(matched))
		for i, g := range matched {
			names[i] = m.itemNames(g.ItemID)[0]
		}
		return market.Good{}, errAmbiguous{names: names}
	}
	return market.Good{}, errNotTracked
}

// commitBuy prices and removes one unit from the zone's stock in a single
// critical section, refusing when sold out or when the price exceeds gold.
// It returns the price paid, after the company's settlement standing.
func (m *MarketModule) commitBuy(zone string, itemID, gold int, pricing standing.Standing) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.commitLocked(zone, itemID, func(g market.Good, stock int) (int, int, error) {
		price, ok := g.AskForStock(stock)
		if !ok {
			return 0, 0, errSoldOut
		}
		price = pricing.BuyPrice(price)
		if price > gold {
			return price, 0, errNotAfford
		}
		return price, -1, nil
	})
}

// commitSell prices and adds one unit to the zone's stock in a single
// critical section, refusing when the market is full. It returns the
// price paid to the seller, after the company's settlement standing.
func (m *MarketModule) commitSell(zone string, itemID int, pricing standing.Standing) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.commitLocked(zone, itemID, func(g market.Good, stock int) (int, int, error) {
		price, ok := g.BidForStock(stock, m.spreadPct)
		if !ok {
			return 0, 0, errMarketFull
		}
		return pricing.SellPrice(price), 1, nil
	})
}

// commitLocked looks up the good and its stock, lets decide price the
// trade and return the stock delta, then applies and saves it. A save
// failure is logged; the in-memory ledger keeps the trade and the next
// save persists it.
func (m *MarketModule) commitLocked(zone string, itemID int, decide func(market.Good, int) (int, int, error)) (int, error) {
	if m.persistenceAvailable() != nil {
		return 0, errUnavailable
	}
	var good market.Good
	found := false
	for _, g := range m.markets[zone] {
		if g.ItemID == itemID {
			good, found = g, true
			break
		}
	}
	zm, tracked := m.zones[zone]
	if !found || !tracked {
		return 0, errNotTracked
	}
	for i := range zm.Goods {
		if zm.Goods[i].ItemID != itemID {
			continue
		}
		stock := good.ClampStock(zm.Goods[i].Stock)
		price, delta, err := decide(good, stock)
		if err != nil {
			return price, err
		}
		zm.Goods[i].Stock = good.ClampStock(stock + delta)
		if err := m.saveLocked(); err != nil {
			mudlog.Error("market: trade save", "zone", zone, "itemid", itemID, "error", err)
		}
		return price, nil
	}
	return 0, errNotTracked
}

func (m *MarketModule) buy(user *users.UserRecord, room *rooms.Room, what string, pricing standing.Standing) {
	if what == "" {
		user.SendText("Buy what? Try: market buy <good>.")
		return
	}
	good, err := m.matchGood(m.goodsFor(room.Zone), what)
	var ambiguous errAmbiguous
	if errors.As(err, &ambiguous) {
		user.SendText(fmt.Sprintf(`Which do you mean: %s?`, strings.Join(ambiguous.names, ", ")))
		return
	}
	if err != nil {
		user.SendText(fmt.Sprintf(`The market here doesn't trade in "%s".`, what))
		return
	}
	name := m.itemNames(good.ItemID)[0]
	// Make the item before touching the ledger, so a missing item spec can
	// never cost stock or gold.
	newItem := items.New(good.ItemID)
	if newItem.ItemId == 0 {
		mudlog.Error("market: tracked good has no item spec", "zone", room.Zone, "itemid", good.ItemID)
		user.SendText(fmt.Sprintf(`No one at the market has any <ansi fg="itemname">%s</ansi> to sell right now.`, name))
		return
	}
	price, err := m.commitBuy(room.Zone, good.ItemID, user.Character.Gold, pricing)
	switch {
	case errors.Is(err, errSoldOut):
		user.SendText(fmt.Sprintf(`No one at the market has any <ansi fg="itemname">%s</ansi> to sell right now.`, name))
		return
	case errors.Is(err, errNotAfford):
		user.SendText(fmt.Sprintf(`The <ansi fg="itemname">%s</ansi> costs <ansi fg="gold">%d gold</ansi> here, and you don't have enough.`, name, price))
		return
	case err != nil:
		user.SendText("The market ledgers are unavailable right now.")
		return
	}

	user.Character.Gold -= price
	user.Character.StoreItem(newItem)
	user.Character.CancelBuffsWithFlag("hidden")
	user.PlaySound(`purchase`, `other`)

	events.AddToQueue(events.EquipmentChange{UserId: user.UserId, GoldChange: -price})
	events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: newItem, Gained: true})
	events.AddToQueue(events.Purchase{UserId: user.UserId, RoomId: room.RoomId, Cost: price, ItemId: good.ItemID})

	user.EventLog.Add(`shop`, fmt.Sprintf(`Bought the <ansi fg="itemname">%s</ansi> at the %s market for <ansi fg="gold">%d gold</ansi>`, newItem.DisplayName(), room.Zone, price))
	user.SendText(fmt.Sprintf(`You buy the <ansi fg="itemname">%s</ansi> at the market for <ansi fg="gold">%d gold</ansi>.`, newItem.DisplayName(), price))
	room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> buys the <ansi fg="itemname">%s</ansi> at the market.`, user.Character.Name, newItem.DisplayName()), user.UserId)
}

func (m *MarketModule) sell(user *users.UserRecord, room *rooms.Room, what string, pricing standing.Standing) {
	if what == "" {
		user.SendText("Sell what? Try: market sell <good>.")
		return
	}
	item, found, refusal := m.pickSaleItem(user, room.Zone, what)
	if !found {
		user.SendText(refusal)
		return
	}

	price, err := m.commitSell(room.Zone, item.ItemId, pricing)
	switch {
	case errors.Is(err, errMarketFull):
		user.SendText(fmt.Sprintf(`The market is glutted with <ansi fg="itemname">%s</ansi>; no one will buy more right now.`, item.DisplayName()))
		return
	case err != nil:
		user.SendText("The market ledgers are unavailable right now.")
		return
	}

	user.Character.RemoveItem(item)
	user.Character.Gold += price
	user.Character.CancelBuffsWithFlag("hidden")

	events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: item, Gained: false})
	events.AddToQueue(events.EquipmentChange{UserId: user.UserId, GoldChange: price})

	user.EventLog.Add(`shop`, fmt.Sprintf(`Sold your <ansi fg="itemname">%s</ansi> at the %s market for <ansi fg="gold">%d gold</ansi>`, item.DisplayName(), room.Zone, price))
	user.SendText(fmt.Sprintf(`You sell the <ansi fg="itemname">%s</ansi> at the market for <ansi fg="gold">%d gold</ansi>.`, item.DisplayName(), price))
	room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> sells the <ansi fg="itemname">%s</ansi> at the market.`, user.Character.Name, item.DisplayName()), user.UserId)
}

// partlyUsed reports whether an item with uses (a whetstone) has spent
// any: the market only buys the ordinary article, at full uses, so a
// stone can't be bought, mostly worn down, and sold back at full price.
func partlyUsed(it items.Item) bool {
	spec := it.GetSpec()
	return spec.Uses > 0 && it.Uses < spec.Uses
}

// pickSaleItem chooses which carried item a sale refers to. It prefers an
// ordinary item of a good this market trades, so "hide" sells a plain wolf
// hide even when hide armor or a special wolf hide is also carried. When
// none qualifies it explains why, judged on the engine's own best match.
func (m *MarketModule) pickSaleItem(user *users.UserRecord, zone, what string) (items.Item, bool, string) {
	tracked := map[int]bool{}
	for _, g := range m.goodsFor(zone) {
		tracked[g.ItemID] = true
	}
	candidates := []items.Item{}
	for _, it := range user.Character.Items {
		if tracked[it.ItemId] && it.GetSpec().QuestToken == `` && !it.IsSpecial() && !partlyUsed(it) {
			candidates = append(candidates, it)
		}
	}
	partial, full := items.FindMatchIn(what, candidates...)
	if full.ItemId != 0 {
		return full, true, ""
	}
	if partial.ItemId != 0 {
		return partial, true, ""
	}

	item, found := user.Character.FindInBackpack(what)
	switch {
	case !found:
		return items.Item{}, false, "You don't have that item."
	case !tracked[item.ItemId]:
		return items.Item{}, false, fmt.Sprintf(`The market here doesn't trade in <ansi fg="itemname">%s</ansi>.`, item.DisplayName())
	case item.GetSpec().QuestToken != ``:
		return items.Item{}, false, "Quest items cannot be sold!"
	}
	return items.Item{}, false, fmt.Sprintf(`The traders won't take that <ansi fg="itemname">%s</ansi>; it's not the ordinary article.`, item.DisplayName())
}
