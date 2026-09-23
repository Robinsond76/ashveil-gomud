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
	"github.com/GoMudEngine/GoMud/internal/users"
)

var (
	errSoldOut     = errors.New("sold out")
	errMarketFull  = errors.New("market full")
	errNotAfford   = errors.New("not enough gold")
	errNotTracked  = errors.New("not traded here")
	errUnavailable = errors.New("ledgers unavailable")
)

// goodsFor returns a copy of the zone's configured goods.
func (m *MarketModule) goodsFor(zone string) []market.Good {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]market.Good(nil), m.markets[zone]...)
}

// matchGood finds the tracked good a player named: an exact name first,
// then a name prefix, then a substring, case-insensitively.
func (m *MarketModule) matchGood(goods []market.Good, query string) (market.Good, bool) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return market.Good{}, false
	}
	for _, test := range []func(name string) bool{
		func(name string) bool { return name == query },
		func(name string) bool { return strings.HasPrefix(name, query) },
		func(name string) bool { return strings.Contains(name, query) },
	} {
		for _, g := range goods {
			for _, name := range m.itemNames(g.ItemID) {
				if test(strings.ToLower(name)) {
					return g, true
				}
			}
		}
	}
	return market.Good{}, false
}

// commitBuy prices and removes one unit from the zone's stock in a single
// critical section, refusing when sold out or when the price exceeds gold.
// It returns the price paid.
func (m *MarketModule) commitBuy(zone string, itemID, gold int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.commitLocked(zone, itemID, func(g market.Good, stock int) (int, int, error) {
		price, ok := g.AskForStock(stock)
		if !ok {
			return 0, 0, errSoldOut
		}
		if price > gold {
			return price, 0, errNotAfford
		}
		return price, -1, nil
	})
}

// commitSell prices and adds one unit to the zone's stock in a single
// critical section, refusing when the market is full. It returns the
// price paid to the seller.
func (m *MarketModule) commitSell(zone string, itemID int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.commitLocked(zone, itemID, func(g market.Good, stock int) (int, int, error) {
		price, ok := g.BidForStock(stock, m.spreadPct)
		if !ok {
			return 0, 0, errMarketFull
		}
		return price, 1, nil
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

func (m *MarketModule) buy(user *users.UserRecord, room *rooms.Room, what string) {
	if what == "" {
		user.SendText("Buy what? Try: market buy <good>.")
		return
	}
	good, ok := m.matchGood(m.goodsFor(room.Zone), what)
	if !ok {
		user.SendText(fmt.Sprintf(`The market here doesn't trade in "%s".`, what))
		return
	}
	name := m.itemNames(good.ItemID)[0]
	price, err := m.commitBuy(room.Zone, good.ItemID, user.Character.Gold)
	switch {
	case errors.Is(err, errSoldOut):
		user.SendText(fmt.Sprintf(`No one at the market has any <ansi fg="itemname">%s</ansi> to sell right now.`, name))
		return
	case errors.Is(err, errNotAfford):
		user.SendText(fmt.Sprintf(`A <ansi fg="itemname">%s</ansi> costs <ansi fg="gold">%d gold</ansi> here, and you don't have enough.`, name, price))
		return
	case err != nil:
		user.SendText("The market ledgers are unavailable right now.")
		return
	}

	newItem := items.New(good.ItemID)
	user.Character.Gold -= price
	user.Character.StoreItem(newItem)
	user.Character.CancelBuffsWithFlag("hidden")
	user.PlaySound(`purchase`, `other`)

	events.AddToQueue(events.EquipmentChange{UserId: user.UserId, GoldChange: -price})
	events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: newItem, Gained: true})

	user.EventLog.Add(`shop`, fmt.Sprintf(`Bought a <ansi fg="itemname">%s</ansi> at the %s market for <ansi fg="gold">%d gold</ansi>`, newItem.DisplayName(), room.Zone, price))
	user.SendText(fmt.Sprintf(`You buy a <ansi fg="itemname">%s</ansi> at the market for <ansi fg="gold">%d gold</ansi>.`, newItem.DisplayName(), price))
	room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> buys a <ansi fg="itemname">%s</ansi> at the market.`, user.Character.Name, newItem.DisplayName()), user.UserId)
}

func (m *MarketModule) sell(user *users.UserRecord, room *rooms.Room, what string) {
	if what == "" {
		user.SendText("Sell what? Try: market sell <good>.")
		return
	}
	item, found := user.Character.FindInBackpack(what)
	if !found {
		user.SendText("You don't have that item.")
		return
	}
	tracked := false
	for _, g := range m.goodsFor(room.Zone) {
		if g.ItemID == item.ItemId {
			tracked = true
			break
		}
	}
	if !tracked {
		user.SendText(fmt.Sprintf(`The market here doesn't trade in <ansi fg="itemname">%s</ansi>.`, item.DisplayName()))
		return
	}
	if item.GetSpec().QuestToken != `` {
		user.SendText("Quest items cannot be sold!")
		return
	}
	if item.IsSpecial() {
		user.SendText(fmt.Sprintf(`The traders won't take that <ansi fg="itemname">%s</ansi>; it's not the ordinary article.`, item.DisplayName()))
		return
	}

	price, err := m.commitSell(room.Zone, item.ItemId)
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
	user.SendText(fmt.Sprintf(`You sell a <ansi fg="itemname">%s</ansi> at the market for <ansi fg="gold">%d gold</ansi>.`, item.DisplayName(), price))
	room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> sells a <ansi fg="itemname">%s</ansi> at the market.`, user.Character.Name, item.DisplayName()), user.UserId)
}
