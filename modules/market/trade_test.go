package market

import (
	"errors"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/market"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tradeSpecs(t *testing.T) {
	t.Helper()
	for _, spec := range []*items.ItemSpec{
		{ItemId: 28, Name: "wolf hide", NameSimple: "hide", Type: items.Commodity},
		{ItemId: 29, Name: "raw game meat", NameSimple: "meat", Type: items.Commodity},
		{ItemId: 30, Name: "wild thyme", NameSimple: "thyme", Type: items.Botanical},
	} {
		items.SetTestItemSpec(spec)
		id := spec.ItemId
		t.Cleanup(func() { items.RemoveTestItemSpec(id) })
	}
}

type tradeWorld struct {
	module   *MarketModule
	store    *fakeStore
	user     *users.UserRecord
	messages *[]string
}

func newTradeWorld(t *testing.T, gold int) *tradeWorld {
	t.Helper()
	tradeSpecs(t)
	store := &fakeStore{}
	module := newTestModule(store)
	module.load()
	user := marketUser(t)
	user.Character.Gold = gold
	return &tradeWorld{module: module, store: store, user: user, messages: captureMessages(t)}
}

func (w *tradeWorld) run(t *testing.T, rest string) string {
	t.Helper()
	*w.messages = nil
	handled, err := w.module.userCommand(rest, w.user, marketRoom(), 0)
	require.NoError(t, err)
	require.True(t, handled)
	events.ProcessEvents()
	return stripTags(strings.Join(*w.messages, "\n"))
}

func countItem(c []items.Item, itemID int) int {
	n := 0
	for _, it := range c {
		if it.ItemId == itemID {
			n++
		}
	}
	return n
}

func TestMarketBuyPaysAskTakesStockAndPersists(t *testing.T) {
	w := newTradeWorld(t, 100)
	saves := w.store.saveCalls

	out := w.run(t, "buy hide")

	assert.Contains(t, out, "You buy the wolf hide at the market for 27 gold.")
	assert.Equal(t, 73, w.user.Character.Gold)
	assert.Equal(t, 1, countItem(w.user.Character.Items, 28))
	assert.Equal(t, 3, stockOf(t, w.store.saved, "Dunmar", 28), "one unit leaves the ledger")
	assert.Equal(t, saves+1, w.store.saveCalls)
	assert.Regexp(t, `wolf hide\s+28 gold`, w.run(t, ""), "scarcer stock, higher price")
}

func TestMarketSellPaysBidAddsStockAndPersists(t *testing.T) {
	w := newTradeWorld(t, 0)
	w.user.Character.StoreItem(items.New(28))

	out := w.run(t, "sell wolf hide")

	assert.Contains(t, out, "You sell the wolf hide at the market for 22 gold.")
	assert.Equal(t, 22, w.user.Character.Gold)
	assert.Zero(t, countItem(w.user.Character.Items, 28))
	assert.Equal(t, 5, stockOf(t, w.store.saved, "Dunmar", 28))
}

func TestMarketBuyThenSellLosesGold(t *testing.T) {
	w := newTradeWorld(t, 100)
	w.run(t, "buy hide")
	w.run(t, "sell hide")
	assert.Less(t, w.user.Character.Gold, 100)
	assert.Equal(t, 4, stockOf(t, w.store.saved, "Dunmar", 28), "stock back where it started")
}

func TestMarketBuyMatchesNames(t *testing.T) {
	for _, query := range []string{"wolf hide", "HIDE", "wolf", "hid"} {
		t.Run(query, func(t *testing.T) {
			w := newTradeWorld(t, 100)
			w.run(t, "buy "+query)
			assert.Equal(t, 1, countItem(w.user.Character.Items, 28))
		})
	}
}

// assertUnchanged checks a refused trade moved no gold, items, or stock.
func (w *tradeWorld) assertUnchanged(t *testing.T, gold int, itemCount map[int]int, stock map[int]int) {
	t.Helper()
	assert.Equal(t, gold, w.user.Character.Gold)
	for id, n := range itemCount {
		assert.Equal(t, n, countItem(w.user.Character.Items, id), "item %d", id)
	}
	for id, n := range stock {
		got, _ := w.module.zones["Dunmar"].Stock(id)
		assert.Equal(t, n, got, "stock %d", id)
	}
}

func TestMarketTradeRefusals(t *testing.T) {
	t.Run("unknown good", func(t *testing.T) {
		w := newTradeWorld(t, 100)
		assert.Contains(t, w.run(t, "buy sword"), `doesn't trade in "sword"`)
		w.assertUnchanged(t, 100, map[int]int{28: 0}, map[int]int{28: 4})
	})
	t.Run("not enough gold", func(t *testing.T) {
		w := newTradeWorld(t, 26)
		assert.Contains(t, w.run(t, "buy hide"), "costs 27 gold here, and you don't have enough")
		w.assertUnchanged(t, 26, map[int]int{28: 0}, map[int]int{28: 4})
	})
	t.Run("sold out", func(t *testing.T) {
		w := newTradeWorld(t, 100)
		w.module.zones["Dunmar"].Goods[0].Stock = 0
		assert.Contains(t, w.run(t, "buy hide"), "No one at the market has any wolf hide")
		w.assertUnchanged(t, 100, map[int]int{28: 0}, map[int]int{28: 0})
	})
	t.Run("market full", func(t *testing.T) {
		w := newTradeWorld(t, 0)
		w.user.Character.StoreItem(items.New(28))
		w.module.zones["Dunmar"].Goods[0].Stock = 40
		assert.Contains(t, w.run(t, "sell hide"), "glutted with wolf hide")
		w.assertUnchanged(t, 0, map[int]int{28: 1}, map[int]int{28: 40})
	})
	t.Run("item not carried", func(t *testing.T) {
		w := newTradeWorld(t, 0)
		assert.Contains(t, w.run(t, "sell hide"), "You don't have that item.")
		w.assertUnchanged(t, 0, nil, map[int]int{28: 4})
	})
	t.Run("good not traded here", func(t *testing.T) {
		w := newTradeWorld(t, 0)
		w.user.Character.StoreItem(items.New(30))
		assert.Contains(t, w.run(t, "sell thyme"), "doesn't trade in wild thyme")
		w.assertUnchanged(t, 0, map[int]int{30: 1}, nil)
	})
	t.Run("quest item", func(t *testing.T) {
		w := newTradeWorld(t, 0)
		items.SetTestItemSpec(&items.ItemSpec{ItemId: 28, Name: "wolf hide", NameSimple: "hide", Type: items.Commodity, QuestToken: "hunt-1"})
		w.user.Character.StoreItem(items.New(28))
		assert.Contains(t, w.run(t, "sell hide"), "Quest items cannot be sold!")
		w.assertUnchanged(t, 0, map[int]int{28: 1}, map[int]int{28: 4})
	})
	t.Run("ledgers unavailable", func(t *testing.T) {
		w := newTradeWorld(t, 100)
		w.module.loadErr = errors.New("corrupt")
		assert.Contains(t, w.run(t, "buy hide"), "market ledgers are unavailable")
		w.assertUnchanged(t, 100, map[int]int{28: 0}, map[int]int{28: 4})
	})
	t.Run("missing argument and bad verb", func(t *testing.T) {
		w := newTradeWorld(t, 100)
		assert.Contains(t, w.run(t, "buy"), "Buy what?")
		assert.Contains(t, w.run(t, "sell"), "Sell what?")
		assert.Contains(t, w.run(t, "haggle hide"), "Usage: market")
		w.assertUnchanged(t, 100, map[int]int{28: 0}, map[int]int{28: 4})
	})
}

func TestMarketTradeSaveFailureKeepsTheTrade(t *testing.T) {
	w := newTradeWorld(t, 100)
	w.store.saveErr = errors.New("disk full")

	w.run(t, "buy hide")

	assert.Equal(t, 73, w.user.Character.Gold)
	stock, _ := w.module.zones["Dunmar"].Stock(28)
	assert.Equal(t, 3, stock, "the in-memory ledger keeps the trade; the next save persists it")
	w.store.saveErr = nil
	require.NoError(t, w.module.save())
	assert.Equal(t, 3, stockOf(t, w.store.saved, "Dunmar", 28))
}

func TestMarketBuyAmbiguousNameAsksWhich(t *testing.T) {
	w := newTradeWorld(t, 100)
	w.module.markets["Dunmar"] = append(w.module.markets["Dunmar"], func() market.Good {
		g := hide()
		g.ItemID = 30
		return g
	}())
	w.module.load()

	out := w.run(t, "buy w") // prefix of both "wolf hide" and "wild thyme"

	assert.Contains(t, out, "Which do you mean: wolf hide, wild thyme?")
	w.assertUnchanged(t, 100, map[int]int{28: 0, 30: 0}, map[int]int{28: 4})
	w.run(t, "buy wolf")
	assert.Equal(t, 1, countItem(w.user.Character.Items, 28), "a unique prefix still buys")
}

func TestMarketBuyMissingItemSpecCostsNothing(t *testing.T) {
	w := newTradeWorld(t, 100)
	items.RemoveTestItemSpec(28) // e.g. an item-spec reload dropped it after boot

	w.run(t, "buy hide")

	w.assertUnchanged(t, 100, map[int]int{28: 0}, map[int]int{28: 4})
}

func TestMarketSellPrefersATradeableItem(t *testing.T) {
	w := newTradeWorld(t, 0)
	items.SetTestItemSpec(&items.ItemSpec{ItemId: 31, Name: "hide armor", NameSimple: "armor", Type: items.Body})
	t.Cleanup(func() { items.RemoveTestItemSpec(31) })
	special := items.New(28)
	special.Blob = "stitched with a name"
	w.user.Character.StoreItem(items.New(31))
	w.user.Character.StoreItem(special)
	plain := items.New(28)
	w.user.Character.StoreItem(plain)

	out := w.run(t, "sell hide")

	assert.Contains(t, out, "You sell the wolf hide at the market for 22 gold.")
	assert.Equal(t, 1, countItem(w.user.Character.Items, 31), "the armor is kept")
	require.Equal(t, 1, countItem(w.user.Character.Items, 28))
	for _, it := range w.user.Character.Items {
		if it.ItemId == 28 {
			assert.True(t, it.Equals(special), "the plain hide was sold, the special one kept")
		}
	}
}

func TestMarketSellRefusesSpecialItem(t *testing.T) {
	w := newTradeWorld(t, 0)
	special := items.New(28)
	special.Blob = "stitched with a name"
	w.user.Character.StoreItem(special)

	assert.Contains(t, w.run(t, "sell hide"), "not the ordinary article")
	w.assertUnchanged(t, 0, map[int]int{28: 1}, map[int]int{28: 4})
}

func TestMarketTradesQueueOwnershipGoldAndPurchaseEvents(t *testing.T) {
	w := newTradeWorld(t, 100)
	type seen struct {
		gold      []int
		gained    []bool
		purchases []events.Purchase
	}
	got := &seen{}
	ids := []events.ListenerId{
		events.RegisterListener(events.EquipmentChange{}, func(e events.Event) events.ListenerReturn {
			got.gold = append(got.gold, e.(events.EquipmentChange).GoldChange)
			return events.Continue
		}),
		events.RegisterListener(events.ItemOwnership{}, func(e events.Event) events.ListenerReturn {
			got.gained = append(got.gained, e.(events.ItemOwnership).Gained)
			return events.Continue
		}),
		events.RegisterListener(events.Purchase{}, func(e events.Event) events.ListenerReturn {
			got.purchases = append(got.purchases, e.(events.Purchase))
			return events.Continue
		}),
	}
	t.Cleanup(func() {
		events.UnregisterListener(events.EquipmentChange{}, ids[0])
		events.UnregisterListener(events.ItemOwnership{}, ids[1])
		events.UnregisterListener(events.Purchase{}, ids[2])
	})

	out := w.run(t, "buy hide")
	w.run(t, "sell hide")

	assert.Equal(t, []int{-27, 23}, got.gold, "sold back at stock 3: 28 less 20% is 23, still below the 27 paid")
	assert.Equal(t, []bool{true, false}, got.gained)
	require.Len(t, got.purchases, 1, "only buys are purchases")
	assert.Equal(t, events.Purchase{UserId: w.user.UserId, RoomId: 2004, Cost: 27, ItemId: 28}, got.purchases[0])
	assert.Contains(t, out, "buys the wolf hide at the market.", "the room hears the trade")
}
