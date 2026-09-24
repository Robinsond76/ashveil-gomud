package market

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/market"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// TestMarketEndToEndThroughPluginsLoad drives the registered module through
// its real entry points: plugins.Load registers the market command, merges
// the shipped config overlay, and runs OnLoad against a temp plugin-data
// dir (seeding the shipped markets); the command runs through
// usercommands.TryCommand in the shipped rooms (listing and trading only in
// the tagged market rooms); a real NewRound event is dispatched through
// events.ProcessEvents; and fresh store reloads prove drift and trades
// persisted.
func TestMarketEndToEndThroughPluginsLoad(t *testing.T) {
	// Point at the shipped world for item specs and keyword aliases. The
	// shipped room files are decoded and registered in memory (loading the
	// world's rooms would persist a NextRoomId config override into the
	// data dir); market zone names are checked against the shipped rooms'
	// zone fields instead.
	_, thisFile, _, _ := runtime.Caller(0)
	dataDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
	require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": dataDir}))
	keywords.LoadAliases()
	items.LoadDataFiles()
	shippedZones := shippedZoneNames(t, dataDir)
	require.Contains(t, shippedZones, "Dunmar")
	shippedRooms := map[int]*rooms.Room{}
	for id, path := range map[int]string{
		2001: "rooms/dunmar/2001.yaml",
		2003: "rooms/dunmar/2003.yaml",
		2004: "rooms/dunmar/2004.yaml",
		2002: "rooms/old_kings_road/2002.yaml",
		2005: "rooms/old_kings_road/2005.yaml",
		1:    "rooms/frostfang/1.yaml",
	} {
		data, err := os.ReadFile(filepath.Join(dataDir, path))
		require.NoError(t, err)
		room := &rooms.Room{}
		require.NoError(t, yaml.Unmarshal(data, room))
		require.Equal(t, id, room.RoomId)
		shippedRooms[id] = room
		rooms.SetTestZoneRoom(room)
		t.Cleanup(func() { rooms.RemoveTestZoneRoom(room) })
	}
	assert.Contains(t, shippedRooms[2001].Exits, "east", "the West Gate leads to the square")
	assert.Equal(t, 2004, shippedRooms[2001].Exits["east"].RoomId)
	assert.Equal(t, 2005, shippedRooms[2002].Exits["east"].RoomId)

	require.NotNil(t, module, "init registered the module")
	rolls := uint64(0)
	module.roll = func() uint64 { return rolls }
	for _, zone := range []string{"Dunmar", "Old Kings Road"} {
		require.True(t, shippedZones[zone], "%s is a shipped zone", zone)
	}
	// The real zoneExists and marketRooms run against the zone-indexed
	// shipped rooms registered above.
	plugins.Load(t.TempDir())

	require.NoError(t, module.loadErr)
	require.Contains(t, module.markets, "Dunmar")
	require.Contains(t, module.markets, "Old Kings Road")

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 1)
	user.Password = "$2a$test" // not a plaintext password, so commands aren't gated
	users.SetTestUser(user)
	messages := captureMessages(t)

	run := func(roomId int, rest ...string) string {
		t.Helper()
		user.Character.RoomId = roomId
		*messages = nil
		handled, err := usercommands.TryCommand("market", strings.Join(rest, " "), user.UserId, events.CmdSkipScripts)
		require.NoError(t, err)
		require.True(t, handled)
		events.ProcessEvents()
		return stripTags(strings.Join(*messages, "\n"))
	}

	hide := goodFor(t, "Dunmar", 28)
	out := run(2004)
	assert.Contains(t, out, "Market prices in Dunmar:")
	assert.Regexp(t, `wolf hide\s+27 gold\s+22 gold\s+scarce`, out, "shipped StartStock 4, 20% spread")
	assert.Equal(t, 27, hide.PriceForStock(4))
	assert.Contains(t, run(2001), "There's no market here. The market in Dunmar is at Dunmar Market Square.")
	assert.Contains(t, run(2005), "Market prices in Old Kings Road:")
	assert.Contains(t, run(2002), "The market in Old Kings Road is at Trappers' Post.")
	assert.Contains(t, run(1), "There's no market here.", "Frostfang is not a market")

	events.AddToQueue(events.NewRound{RoundNumber: 1})
	events.ProcessEvents()

	stock, ok := module.zones["Dunmar"].Stock(28)
	require.True(t, ok)
	assert.Equal(t, 5, stock, "one bounded step toward target 20")
	assert.Regexp(t, `wolf hide\s+26 gold\s+21 gold\s+scarce`, run(2004), "prices follow stock")
	okrHide, _ := module.zones["Old Kings Road"].Stock(28)
	assert.Equal(t, 29, okrHide, "a glutted market drifts down")

	reloaded := NewRegistry()
	require.NoError(t, pluginStore{plug: module.plug}.Load(reloaded))
	assert.Equal(t, Registry{Zones: module.zones, News: module.news, NewsIn: module.newsIn}, *reloaded, "the round's drift was persisted")

	// Trading through the real command in the shipped square.
	user.Character.Gold = 100
	assert.Contains(t, run(2001, "buy", "hide"), "There's no market here.", "no trading at the gate")
	assert.Equal(t, 100, user.Character.Gold)
	assert.Contains(t, run(2004, "buy", "wolf hide"), "You buy the wolf hide at the market for 26 gold.")
	assert.Equal(t, 74, user.Character.Gold)
	_, carried := user.Character.FindInBackpack("wolf hide")
	assert.True(t, carried)
	reloaded = NewRegistry()
	require.NoError(t, pluginStore{plug: module.plug}.Load(reloaded))
	boughtStock, _ := reloaded.Zones["Dunmar"].Stock(28)
	assert.Equal(t, 4, boughtStock, "the purchase persisted to the real store")

	assert.Contains(t, run(2004, "sell", "hide"), "You sell the wolf hide at the market for 22 gold.")
	assert.Equal(t, 96, user.Character.Gold)
	_, carried = user.Character.FindInBackpack("wolf hide")
	assert.False(t, carried)
	reloaded = NewRegistry()
	require.NoError(t, pluginStore{plug: module.plug}.Load(reloaded))
	soldStock, _ := reloaded.Zones["Dunmar"].Stock(28)
	assert.Equal(t, 5, soldStock, "the sale persisted to the real store")

	// The Trappers' Post trades against its own ledger.
	okrBefore, _ := module.zones["Old Kings Road"].Stock(28)
	dunmarBefore, _ := module.zones["Dunmar"].Stock(28)
	gold := user.Character.Gold
	assert.Contains(t, run(2005, "buy", "hide"), "You buy the wolf hide at the market")
	okrAfter, _ := module.zones["Old Kings Road"].Stock(28)
	dunmarAfter, _ := module.zones["Dunmar"].Stock(28)
	assert.Equal(t, okrBefore-1, okrAfter)
	assert.Equal(t, dunmarBefore, dunmarAfter, "Dunmar's ledger is untouched")
	assert.Less(t, user.Character.Gold, gold)
	assert.Contains(t, run(2005, "sell", "hide"), "You sell the wolf hide at the market")
	okrAfter, _ = module.zones["Old Kings Road"].Stock(28)
	assert.Equal(t, okrBefore, okrAfter)

	// Phase 23b: the square sells the shipped whetstone with its 10 uses,
	// and only buys one back unused.
	stoneSpec := items.GetItemSpec(30)
	require.NotNil(t, stoneSpec)
	assert.Equal(t, "whetstone", stoneSpec.Name)
	assert.Equal(t, 10, stoneSpec.Uses)
	goodFor(t, "Old Kings Road", 30)
	user.Character.Gold = 200
	assert.Contains(t, run(2004, "buy", "whetstone"), "You buy the whetstone at the market")
	stone, carried := user.Character.FindInBackpack("whetstone")
	require.True(t, carried)
	assert.Equal(t, 10, stone.Uses)
	for i := range user.Character.Items {
		if user.Character.Items[i].ItemId == 30 {
			user.Character.Items[i].Uses = 9
		}
	}
	gold = user.Character.Gold
	assert.Contains(t, run(2004, "sell", "whetstone"), "not the ordinary article")
	assert.Equal(t, gold, user.Character.Gold)
	assert.Contains(t, run(2004, "buy", "whetstone"), "You buy the whetstone at the market")
	assert.Contains(t, run(2004, "sell", "whetstone"), "You sell the whetstone at the market", "the unused one sells")
	left, carried := user.Character.FindInBackpack("whetstone")
	require.True(t, carried)
	assert.Equal(t, 9, left.Uses, "the used one is kept")

	// A downed player can't trade.
	user.Character.Health = 0
	gold = user.Character.Gold
	run(2004, "buy", "hide")
	assert.Equal(t, gold, user.Character.Gold)
	downedStock, _ := module.zones["Dunmar"].Stock(28)
	assert.Equal(t, dunmarBefore, downedStock)
	user.Character.Health = 10

	// A restart restores the persisted stock instead of re-seeding.
	restarted := &MarketModule{
		plug:       module.plug,
		store:      pluginStore{plug: module.plug},
		roll:       func() uint64 { return 0 },
		itemExists: module.itemExists,
		itemNames:  module.itemNames,
		zoneExists: module.zoneExists,
		markets:    map[string][]market.Good{},
		zones:      map[string]ZoneMarket{},
	}
	restarted.load()
	restartedStock, _ := restarted.zones["Dunmar"].Stock(28)
	assert.Equal(t, 5, restartedStock)

	// The OnSave callback persists through plugins.Save().
	module.mu.Lock()
	module.zones["Dunmar"].Goods[0].Stock = 11
	module.mu.Unlock()
	plugins.Save()
	saved := NewRegistry()
	require.NoError(t, pluginStore{plug: module.plug}.Load(saved))
	savedStock, _ := saved.Zones["Dunmar"].Stock(28)
	assert.Equal(t, 11, savedStock)

	// Phase 20 trade rumours through the real commands in the shipped inn.
	// The news was taken from the shipped start stocks at load.
	assert.True(t, shippedRooms[2003].HasTag("inn"))
	require.Equal(t, 150, module.rumorRefresh, "shipped RumorRefreshRounds")
	require.Equal(t, 3, module.rumorsPerAsk, "shipped RumorsPerAsk")
	rumors := func(command string, roomId int) string {
		t.Helper()
		user.Character.RoomId = roomId
		*messages = nil
		handled, err := usercommands.TryCommand(command, "", user.UserId, events.CmdSkipScripts)
		require.NoError(t, err)
		require.True(t, handled)
		events.ProcessEvents()
		return stripTags(strings.Join(*messages, "\n"))
	}
	for _, command := range []string{"rumors", "rumours", "rumor", "rumour"} {
		out := rumors(command, 2003)
		assert.Contains(t, out, "You listen to the talk of the markets", command)
		assert.Len(t, strings.Split(strings.TrimSpace(out), "\n"), 4, "%s: a header and three rumours", command)
		assert.Regexp(t, `Dunmar Market Square|Trappers' Post`, out, "rumours name the shipped market rooms")
		assert.NotContains(t, out, "gold")
	}
	assert.Contains(t, rumors("rumors", 2001), "Try an inn.", "no rumours at the gate")
	assert.Contains(t, rumors("rumors", 2004), "Try an inn.", "no rumours in the market square")

	// A real NewRound on the last round of the interval refreshes the news
	// from live stock and persists it to the real store.
	module.mu.Lock()
	staleHide, _ := module.news["Dunmar"].Stock(28)
	module.newsIn = 1
	module.mu.Unlock()
	assert.Equal(t, 4, staleHide, "news still holds the shipped start stock")
	events.AddToQueue(events.NewRound{RoundNumber: 3})
	events.ProcessEvents()
	liveHide, _ := module.zones["Dunmar"].Stock(28)
	module.mu.Lock()
	newsHide, _ := module.news["Dunmar"].Stock(28)
	newsIn := module.newsIn
	module.mu.Unlock()
	assert.NotEqual(t, staleHide, liveHide, "trading and drift moved live stock")
	assert.Equal(t, liveHide, newsHide, "news refreshed from live stock")
	assert.Equal(t, 150, newsIn, "countdown restarted")
	reloaded = NewRegistry()
	require.NoError(t, pluginStore{plug: module.plug}.Load(reloaded))
	storedNews, _ := reloaded.News["Dunmar"].Stock(28)
	assert.Equal(t, liveHide, storedNews, "refreshed news persisted to the real store")

	// A corrupt or truncated real store file disables markets and is left
	// untouched for repair rather than re-seeded over.
	for _, corrupt := range []string{
		"",
		"zones: [not, a, map",
		"zones:\n  Dunmar:\n    goods:\n    - itemid: 28\n",
		"zones:\n  Dunmar:\n    goods:\n    - itemid: 28\n      stock: 9\nnews:\n  Dunmar:\n    goods:\n    - itemid: 28\n",
	} {
		require.NoError(t, module.plug.WriteBytes("market", []byte(corrupt)))
		broken := &MarketModule{
			plug:       module.plug,
			store:      pluginStore{plug: module.plug},
			roll:       func() uint64 { return 0 },
			itemExists: module.itemExists,
			itemNames:  module.itemNames,
			zoneExists: module.zoneExists,
			markets:    map[string][]market.Good{},
			zones:      map[string]ZoneMarket{},
		}
		broken.load()
		assert.Error(t, broken.loadErr, "corrupt store %q", corrupt)
		broken.onNewRound(events.NewRound{RoundNumber: 2})
		require.Error(t, broken.save())
		after, err := module.plug.ReadBytes("market")
		require.NoError(t, err)
		assert.Equal(t, corrupt, string(after), "the corrupt file is left for repair")
	}
}

// shippedZoneNames reads every room's zone field from the shipped world:
// the same key rooms.GetAllZoneNames reports at boot.
func shippedZoneNames(t *testing.T, dataDir string) map[string]bool {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dataDir, "rooms", "*", "*.yaml"))
	require.NoError(t, err)
	names := map[string]bool{}
	for _, path := range paths {
		if filepath.Base(path) == "zone-config.yaml" {
			continue
		}
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		var room struct {
			Zone string `yaml:"zone"`
		}
		require.NoError(t, yaml.Unmarshal(data, &room))
		if room.Zone != "" {
			names[room.Zone] = true
		}
	}
	return names
}

func goodFor(t *testing.T, zone string, itemID int) market.Good {
	t.Helper()
	for _, g := range module.markets[zone] {
		if g.ItemID == itemID {
			return g
		}
	}
	t.Fatalf("no good %d in %s", itemID, zone)
	return market.Good{}
}
