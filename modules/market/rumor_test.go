package market

import (
	"errors"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/market"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// postHide is the Trappers' Post's hide: cheaper than Dunmar's at target.
func postHide() market.Good {
	return market.Good{ItemID: 28, BasePrice: 9, MinPrice: 4, MaxPrice: 20, MaxStock: 30, TargetStock: 15, StartStock: 30, DriftStep: 2}
}

func newRumorModule(store Store) *MarketModule {
	m := newTestModule(store)
	m.markets["Old Kings Road"] = []market.Good{postHide()}
	m.marketRooms = func(zone, tag string) []string {
		if tag != "market" {
			return nil
		}
		return map[string][]string{"Dunmar": {"Dunmar Market Square"}, "Old Kings Road": {"Trappers' Post"}}[zone]
	}
	m.rumorTag = "inn"
	m.rumorRefresh = 3
	m.rumorsPerAsk = 3
	return m
}

func innRoom() *rooms.Room {
	return &rooms.Room{RoomId: 2003, Zone: "Dunmar", Title: "The Waymark Inn", Tags: []string{"inn", "indoor"}}
}

func (m *MarketModule) newsStock(t *testing.T, zone string, itemID int) int {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	stock, ok := m.news[zone].Stock(itemID)
	require.True(t, ok, "news of item %d in %s", itemID, zone)
	return stock
}

func TestLoadSeedsNewsWhenMissing(t *testing.T) {
	store := &fakeStore{}
	module := newRumorModule(store)

	module.load()

	require.NoError(t, module.loadErr)
	assert.Equal(t, 1, store.saveCalls, "seeding stock and news is one save")
	assert.Equal(t, 4, module.newsStock(t, "Dunmar", 28), "news taken from seeded stock")
	assert.Equal(t, 30, module.newsStock(t, "Old Kings Road", 28))
	stock, ok := store.saved.News["Dunmar"].Stock(29)
	require.True(t, ok, "news persisted")
	assert.Equal(t, 30, stock)
}

func TestLoadKeepsStoredNews(t *testing.T) {
	saved := Registry{
		Zones: map[string]ZoneMarket{
			"Dunmar":         {Goods: []GoodStock{{ItemID: 28, Stock: 11}, {ItemID: 29, Stock: 45}}},
			"Old Kings Road": {Goods: []GoodStock{{ItemID: 28, Stock: 15}}},
		},
		News: map[string]ZoneMarket{
			"Dunmar": {Goods: []GoodStock{{ItemID: 28, Stock: 2}}},
		},
	}
	store := &fakeStore{saved: saved.Clone()}
	module := newRumorModule(store)

	module.load()

	assert.Zero(t, store.saveCalls, "stored news is kept, nothing to save")
	assert.Equal(t, 2, module.newsStock(t, "Dunmar", 28), "news is the stored snapshot, not live stock")
	assert.Equal(t, 3, module.newsIn, "the refresh countdown starts over")
}

func TestNewsRefreshesAfterConfiguredRounds(t *testing.T) {
	store := &fakeStore{}
	module := newRumorModule(store)
	module.load()
	saves := store.saveCalls

	module.mu.Lock()
	module.markets["Dunmar"] = []market.Good{hide()} // no meat drift noise
	module.mu.Unlock()
	for round := 1; round <= 2; round++ {
		module.onNewRound(events.NewRound{RoundNumber: uint64(round)})
		assert.Equal(t, 4, module.newsStock(t, "Dunmar", 28), "round %d: news unchanged before the interval", round)
	}
	module.onNewRound(events.NewRound{RoundNumber: 3})
	live, _ := module.zones["Dunmar"].Stock(28)
	assert.Equal(t, 7, live, "stock drifted one step per round")
	assert.Equal(t, 7, module.newsStock(t, "Dunmar", 28), "the third round refreshes the news")
	assert.Equal(t, 3, module.newsIn)
	assert.Equal(t, saves+3, store.saveCalls, "drift and news share one save per round")
	stock, _ := store.saved.News["Dunmar"].Stock(28)
	assert.Equal(t, 7, stock, "refreshed news persisted")
	_, ok := store.saved.News["Dunmar"].Stock(29)
	assert.False(t, ok, "news only covers configured goods")

	// A reload restores the persisted news and restarts the countdown.
	reloaded := newRumorModule(store)
	reloaded.load()
	assert.Equal(t, 7, reloaded.newsStock(t, "Dunmar", 28))
}

func TestNewsRefreshSavesEvenWhenNothingDrifts(t *testing.T) {
	saved := Registry{
		Zones: map[string]ZoneMarket{"Dunmar": {Goods: []GoodStock{{ItemID: 28, Stock: 20}, {ItemID: 29, Stock: 30}}}},
		News:  map[string]ZoneMarket{"Dunmar": {Goods: []GoodStock{{ItemID: 28, Stock: 1}}}},
	}
	store := &fakeStore{saved: saved.Clone()}
	module := newTestModule(store)
	module.rumorRefresh = 1
	module.load()

	module.onNewRound(events.NewRound{RoundNumber: 1})

	assert.Equal(t, 1, store.saveCalls, "markets at target don't drift, but the news refresh saves")
	stock, _ := store.saved.News["Dunmar"].Stock(28)
	assert.Equal(t, 20, stock)
}

func TestDecodeRegistryReadsAndValidatesNews(t *testing.T) {
	registry := &Registry{}
	data := "zones:\n  Dunmar:\n    goods:\n    - itemid: 28\n      stock: 9\nnews:\n  Dunmar:\n    goods:\n    - itemid: 28\n      stock: 3\n"
	require.NoError(t, decodeRegistry([]byte(data), registry))
	stock, ok := registry.News["Dunmar"].Stock(28)
	require.True(t, ok)
	assert.Equal(t, 3, stock)

	truncated := "zones:\n  Dunmar:\n    goods:\n    - itemid: 28\n      stock: 9\nnews:\n  Dunmar:\n    goods:\n    - itemid: 28\n"
	assert.ErrorIs(t, decodeRegistry([]byte(truncated), &Registry{}), ErrCorruptStore)

	old := &Registry{}
	require.NoError(t, decodeRegistry([]byte("zones:\n  Dunmar:\n    goods:\n    - itemid: 28\n      stock: 9\n"), old))
	assert.Empty(t, old.News, "a store from before Phase 20 has no news")
}

func TestParseRumorConfig(t *testing.T) {
	assert.Equal(t, "inn", parseRumorTag(nil))
	assert.Equal(t, "inn", parseRumorTag("  "))
	assert.Equal(t, "tavern", parseRumorTag(" tavern "))

	assert.Equal(t, 150, parseRumorRefresh(nil))
	assert.Equal(t, 150, parseRumorRefresh(0))
	assert.Equal(t, 150, parseRumorRefresh(100001))
	assert.Equal(t, 1, parseRumorRefresh(1))
	assert.Equal(t, 100000, parseRumorRefresh(100000))

	assert.Equal(t, 3, parseRumorsPerAsk(nil))
	assert.Equal(t, 3, parseRumorsPerAsk(0))
	assert.Equal(t, 3, parseRumorsPerAsk(11))
	assert.Equal(t, 10, parseRumorsPerAsk(10))
	assert.Equal(t, 1, parseRumorsPerAsk("1"))
}

func (w *tradeWorld) rumors(t *testing.T, room *rooms.Room) string {
	t.Helper()
	*w.messages = nil
	handled, err := w.module.rumorsCommand("", w.user, room, 0)
	require.NoError(t, err)
	require.True(t, handled)
	events.ProcessEvents()
	return stripTags(strings.Join(*w.messages, "\n"))
}

func newRumorWorld(t *testing.T, store *fakeStore) *tradeWorld {
	t.Helper()
	tradeSpecs(t)
	module := newRumorModule(store)
	module.load()
	return &tradeWorld{module: module, store: store, user: marketUser(t), messages: captureMessages(t)}
}

func TestRumorsCommandListsPhrasedNews(t *testing.T) {
	w := newRumorWorld(t, &fakeStore{})
	w.module.rumorsPerAsk = 10
	gold := w.user.Character.Gold

	out := w.rumors(t, innRoom())

	assert.Contains(t, out, "You listen to the talk of the markets")
	// Seeded news: Dunmar hides scarce (4 of target 20); the post's hides
	// glutted (30 of 30), so the post is cheapest and Dunmar pays best.
	lines := strings.Split(strings.TrimSpace(out), "\n")[1:]
	assert.Len(t, lines, 4, "scarce, glut, cheapest, best buyer")
	assert.Contains(t, out, "Dunmar Market Square is short of wolf hide")
	assert.Contains(t, out, "Trappers' Post is drowning in wolf hide")
	assert.Contains(t, out, "wolf hide comes cheapest at Trappers' Post")
	assert.Contains(t, out, "nobody pays better for wolf hide than Dunmar Market Square")
	assert.NotContains(t, out, "gold", "rumours never quote prices")
	assert.Equal(t, gold, w.user.Character.Gold, "rumours are free")

	w.module.rumorsPerAsk = 2
	lines = strings.Split(strings.TrimSpace(w.rumors(t, innRoom())), "\n")[1:]
	assert.Len(t, lines, 2, "RumorsPerAsk caps the rumours shown")

	// The alternate phrasings are picked by roll.
	w.module.rumorsPerAsk = 10
	w.module.roll = func() uint64 { return 1 }
	out = w.rumors(t, innRoom())
	assert.Contains(t, out, "you can't find wolf hide at Dunmar Market Square for love or money")
}

func TestRumorsUseNewsNotLiveStock(t *testing.T) {
	w := newRumorWorld(t, &fakeStore{})
	w.module.rumorsPerAsk = 10
	w.module.mu.Lock()
	w.module.zones["Dunmar"].Goods[0].Stock = 20 // live Dunmar hides now steady
	w.module.mu.Unlock()

	out := w.rumors(t, innRoom())
	assert.Contains(t, out, "Dunmar Market Square is short of wolf hide", "old news until the next refresh")

	w.module.mu.Lock()
	w.module.newsIn = 1
	w.module.markets["Dunmar"] = []market.Good{hide()}
	w.module.mu.Unlock()
	w.module.onNewRound(events.NewRound{RoundNumber: 1})
	out = w.rumors(t, innRoom())
	assert.NotContains(t, out, "short of wolf hide", "refreshed news")
}

func TestRumorsCommandRefusedOutsideRumorRoom(t *testing.T) {
	w := newRumorWorld(t, &fakeStore{})
	out := w.rumors(t, marketRoom())
	assert.Contains(t, out, "You hear no talk of the markets here. Try an inn.")
	assert.Contains(t, w.rumors(t, &rooms.Room{RoomId: 1, Zone: "Frostfang"}), "Try an inn.")

	// An inn outside any market zone still hears the news.
	out = w.rumors(t, &rooms.Room{RoomId: 61, Zone: "Frostfang", Tags: []string{"inn"}})
	assert.Contains(t, out, "You listen to the talk of the markets")
}

func TestRumorsCommandNoNews(t *testing.T) {
	saved := Registry{
		Zones: map[string]ZoneMarket{"Dunmar": {Goods: []GoodStock{{ItemID: 28, Stock: 20}, {ItemID: 29, Stock: 30}}}},
		News:  map[string]ZoneMarket{"Dunmar": {Goods: []GoodStock{{ItemID: 28, Stock: 20}, {ItemID: 29, Stock: 30}}}},
	}
	tradeSpecs(t)
	store := &fakeStore{saved: saved.Clone()}
	module := newTestModule(store)
	module.rumorTag = "inn"
	module.load()
	w := &tradeWorld{module: module, store: store, user: marketUser(t), messages: captureMessages(t)}

	assert.Contains(t, w.rumors(t, innRoom()), "Nobody here has heard any market news worth repeating.")
}

func TestRumorsCommandPersistenceUnavailable(t *testing.T) {
	store := &fakeStore{loadErr: errors.New("corrupt")}
	tradeSpecs(t)
	module := newRumorModule(store)
	module.load()
	w := &tradeWorld{module: module, store: store, user: marketUser(t), messages: captureMessages(t)}

	assert.Contains(t, w.rumors(t, innRoom()), "Nobody here has any news of the markets right now.")
}
