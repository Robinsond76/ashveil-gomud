package market

import (
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/market"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "low", "", false)
	os.Exit(m.Run())
}

type fakeStore struct {
	saved     Registry
	loadErr   error
	saveErr   error
	loadCalls int
	saveCalls int
}

func (f *fakeStore) Load(registry *Registry) error {
	f.loadCalls++
	if f.loadErr != nil {
		return f.loadErr
	}
	*registry = f.saved.Clone()
	return nil
}

func (f *fakeStore) Save(registry Registry) error {
	f.saveCalls++
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = registry.Clone()
	return nil
}

func hide() market.Good {
	return market.Good{ItemID: 28, BasePrice: 12, MinPrice: 6, MaxPrice: 30, MaxStock: 40, TargetStock: 20, StartStock: 4, DriftStep: 2}
}

func meat() market.Good {
	return market.Good{ItemID: 29, BasePrice: 5, MinPrice: 2, MaxPrice: 12, MaxStock: 60, TargetStock: 30, StartStock: 30, DriftStep: 3}
}

func newTestModule(store Store) *MarketModule {
	return &MarketModule{
		store:      store,
		roll:       func() uint64 { return 0 },
		itemExists: func(id int) bool { return id == 28 || id == 29 || id == 30 },
		itemName: func(id int) string {
			return map[int]string{28: "wolf hide", 29: "raw game meat", 30: "wild thyme"}[id]
		},
		zoneExists: func(zone string) bool { return zone == "Dunmar" || zone == "Old Kings Road" },
		markets:    map[string][]market.Good{"Dunmar": {hide(), meat()}},
		zones:      map[string]ZoneMarket{},
	}
}

func stockOf(t *testing.T, registry Registry, zone string, itemID int) int {
	t.Helper()
	zm, ok := registry.Zones[zone]
	require.True(t, ok, "zone %s tracked", zone)
	stock, ok := zm.Stock(itemID)
	require.True(t, ok, "item %d tracked in %s", itemID, zone)
	return stock
}

func TestLoadSeedsMissingStoreAtStartStock(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store)

	module.load()

	require.NoError(t, module.loadErr)
	assert.Equal(t, 1, store.saveCalls, "seeding a first-boot market is persisted")
	assert.Equal(t, 4, stockOf(t, store.saved, "Dunmar", 28))
	assert.Equal(t, 30, stockOf(t, store.saved, "Dunmar", 29))
}

func TestLoadSeedsOnlyGoodsMissingFromExistingStore(t *testing.T) {
	store := &fakeStore{saved: Registry{Zones: map[string]ZoneMarket{
		"Dunmar": {Goods: []GoodStock{{ItemID: 28, Stock: 17}}},
	}}}
	module := newTestModule(store)

	module.load()

	assert.Equal(t, 17, stockOf(t, store.saved, "Dunmar", 28), "existing stock is kept, not re-seeded")
	assert.Equal(t, 30, stockOf(t, store.saved, "Dunmar", 29), "a newly configured good is seeded")
}

func TestLoadRoundTripsExistingStoreUnchanged(t *testing.T) {
	saved := Registry{Zones: map[string]ZoneMarket{
		"Dunmar":  {Goods: []GoodStock{{ItemID: 28, Stock: 11}, {ItemID: 29, Stock: 45}}},
		"Retired": {Goods: []GoodStock{{ItemID: 30, Stock: 3}}},
	}}
	store := &fakeStore{saved: saved.Clone()}
	module := newTestModule(store)

	module.load()

	assert.Zero(t, store.saveCalls, "nothing to seed, nothing to save")
	assert.Equal(t, saved, Registry{Zones: module.zones}, "stock and unconfigured records survive untouched")
}

func TestLoadFailureDisablesMarketsWithoutPanic(t *testing.T) {
	store := &fakeStore{loadErr: errors.New("corrupt")}
	module := newTestModule(store)

	require.NotPanics(t, module.load)
	require.Error(t, module.loadErr)
	assert.Zero(t, store.saveCalls, "a failed load never overwrites the store")

	module.onNewRound(events.NewRound{RoundNumber: 10})
	assert.Zero(t, store.saveCalls, "rounds do nothing while persistence is unavailable")

	user := marketUser(t)
	messages := captureMessages(t)
	module.userCommand("", user, &rooms.Room{RoomId: 2001, Zone: "Dunmar"}, 0)
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*messages, "\n"), "market ledgers are unavailable")
}

func TestDecodeRegistryRejectsCorruptData(t *testing.T) {
	registry := &Registry{}
	require.NoError(t, decodeRegistry([]byte("zones:\n  Dunmar:\n    goods:\n    - itemid: 28\n      stock: 9\n"), registry))
	assert.Equal(t, 9, stockOf(t, *registry, "Dunmar", 28))

	assert.Error(t, decodeRegistry([]byte("zones: [not, a, map"), &Registry{}), "unreadable data is an error, not an empty registry")
}

func TestDecodeRegistryDropsEmptyZonesAndDuplicateGoods(t *testing.T) {
	registry := &Registry{}
	data := "zones:\n  \"\":\n    goods:\n    - itemid: 28\n      stock: 1\n  Dunmar:\n    goods:\n    - itemid: 28\n      stock: 9\n    - itemid: 28\n      stock: 2\n    - itemid: 0\n      stock: 5\n"
	require.NoError(t, decodeRegistry([]byte(data), registry))
	assert.NotContains(t, registry.Zones, "")
	assert.Equal(t, []GoodStock{{ItemID: 28, Stock: 9}}, registry.Zones["Dunmar"].Goods, "first record wins; non-positive ids dropped")
}

func TestOnNewRoundDriftsTowardTargetAndPersists(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store)
	module.load()
	saves := store.saveCalls

	module.onNewRound(events.NewRound{RoundNumber: 1})

	assert.Equal(t, 5, stockOf(t, store.saved, "Dunmar", 28), "roll 0 moves one step toward target 20")
	assert.Equal(t, 30, stockOf(t, store.saved, "Dunmar", 29), "a good at target stays put")
	assert.Equal(t, saves+1, store.saveCalls, "one batched save per round tick")

	for i := 0; i < 100; i++ {
		module.onNewRound(events.NewRound{RoundNumber: uint64(2 + i)})
	}
	assert.Equal(t, 20, stockOf(t, store.saved, "Dunmar", 28), "converges to target")
	converged := store.saveCalls
	module.onNewRound(events.NewRound{RoundNumber: 500})
	assert.Equal(t, converged, store.saveCalls, "no change, no save")
}

func TestOnNewRoundLeavesUnconfiguredRecordsAlone(t *testing.T) {
	store := &fakeStore{saved: Registry{Zones: map[string]ZoneMarket{
		"Retired": {Goods: []GoodStock{{ItemID: 30, Stock: 3}}},
	}}}
	module := newTestModule(store)
	module.load()

	module.onNewRound(events.NewRound{RoundNumber: 1})

	assert.Equal(t, 3, stockOf(t, store.saved, "Retired", 30))
}

func TestOnNewRoundClampsOutOfRangePersistedStock(t *testing.T) {
	store := &fakeStore{saved: Registry{Zones: map[string]ZoneMarket{
		"Dunmar": {Goods: []GoodStock{{ItemID: 28, Stock: 500}, {ItemID: 29, Stock: -7}}},
	}}}
	module := newTestModule(store)
	module.load()

	module.onNewRound(events.NewRound{RoundNumber: 1})

	assert.Equal(t, 39, stockOf(t, store.saved, "Dunmar", 28))
	assert.Equal(t, 1, stockOf(t, store.saved, "Dunmar", 29))
}

func TestOnNewRoundSaveFailureKeepsInMemoryDrift(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store)
	module.load()
	store.saveErr = errors.New("disk full")

	module.onNewRound(events.NewRound{RoundNumber: 1})

	stock, _ := module.zones["Dunmar"].Stock(28)
	assert.Equal(t, 5, stock)
	store.saveErr = nil
	require.NoError(t, module.save())
	assert.Equal(t, 5, stockOf(t, store.saved, "Dunmar", 28), "the next save persists it")
}

var tagPattern = regexp.MustCompile(`<[^>]+>`)

func stripTags(s string) string { return tagPattern.ReplaceAllString(s, "") }

func marketUser(t *testing.T) *users.UserRecord {
	t.Helper()
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 1)
	users.SetTestUser(user)
	return user
}

func captureMessages(t *testing.T) *[]string {
	t.Helper()
	messages := []string{}
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		messages = append(messages, e.(events.Message).Text)
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	return &messages
}

func TestUserCommandShowsPricesAndStockLevels(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store)
	module.load()
	user := marketUser(t)
	messages := captureMessages(t)

	module.userCommand("", user, &rooms.Room{RoomId: 2001, Zone: "Dunmar"}, 0)
	events.ProcessEvents()

	out := stripTags(strings.Join(*messages, "\n"))
	assert.Contains(t, out, "Market prices in Dunmar:")
	assert.Regexp(t, `wolf hide\s+27 gold\s+\(scarce\)`, out, "stock 4 prices at PriceForStock(4)")
	assert.Regexp(t, `raw game meat\s+5 gold\s+\(steady\)`, out)
	assert.Equal(t, 27, hide().PriceForStock(4))
}

func TestUserCommandReportsNoMarketInUnconfiguredZone(t *testing.T) {
	module := newTestModule(&fakeStore{})
	module.load()
	user := marketUser(t)
	messages := captureMessages(t)

	module.userCommand("", user, &rooms.Room{RoomId: 1, Zone: "Frostfang"}, 0)
	events.ProcessEvents()

	assert.Contains(t, strings.Join(*messages, "\n"), "There's no market here.")
}

func TestParseMarketsValidatesEntries(t *testing.T) {
	good := func(id, start int) map[string]any {
		return map[string]any{"itemid": id, "baseprice": 12, "minprice": 6, "maxprice": 30, "maxstock": 40, "targetstock": 20, "startstock": start, "driftstep": 2}
	}
	raw := []any{
		map[string]any{"zone": "Dunmar", "goods": []any{
			good(28, 4),
			good(29, 99), // start above max stock: omitted
			good(404, 4), // no item spec: omitted
			map[any]any{"ItemId": 30, "BasePrice": 3, "MinPrice": 1, "MaxPrice": 9, "MaxStock": 10, "TargetStock": 5, "StartStock": 5, "DriftStep": 1},
		}},
		map[string]any{"zone": "Old Kings Road", "goods": []any{good(28, 4), good(28, 8)}}, // duplicate item: whole zone rejected
		map[string]any{"zone": "Dunmar", "goods": []any{good(29, 4)}},                      // duplicate zone: rejected
		map[string]any{"zone": "Nowhere", "goods": []any{good(28, 4)}},                     // unknown zone: rejected
		map[string]any{"zone": "", "goods": []any{good(28, 4)}},
		map[string]any{"zone": "Old Kings Road", "goods": []any{good(404, 4)}}, // no valid goods: no market
	}
	module := newTestModule(&fakeStore{})

	markets := parseMarkets(raw, module.itemExists, module.zoneExists)

	require.Contains(t, markets, "Dunmar")
	require.Len(t, markets["Dunmar"], 2)
	assert.Equal(t, 28, markets["Dunmar"][0].ItemID)
	assert.Equal(t, market.Good{ItemID: 30, BasePrice: 3, MinPrice: 1, MaxPrice: 9, MaxStock: 10, TargetStock: 5, StartStock: 5, DriftStep: 1}, markets["Dunmar"][1], "mixed-case keys parse")
	assert.Len(t, markets, 1)
}
