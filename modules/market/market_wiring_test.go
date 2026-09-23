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
// usercommands.TryCommand in shipped rooms; a real NewRound event is
// dispatched through events.ProcessEvents; and a fresh store reload proves
// the drifted stock persisted.
func TestMarketEndToEndThroughPluginsLoad(t *testing.T) {
	// Point at the shipped world for item specs and keyword aliases. Rooms
	// are registered in memory (loading the world's rooms would persist a
	// NextRoomId config override into the data dir); market zone names are
	// checked against the shipped zone-config files instead.
	_, thisFile, _, _ := runtime.Caller(0)
	dataDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
	require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": dataDir}))
	keywords.LoadAliases()
	items.LoadDataFiles()
	shippedZones := shippedZoneNames(t, dataDir)
	require.Contains(t, shippedZones, "Dunmar")
	for id, zone := range map[int]string{2001: "Dunmar", 2002: "Old Kings Road", 1: "Frostfang"} {
		rooms.SetTestRoom(&rooms.Room{RoomId: id, Zone: zone, Title: zone})
		t.Cleanup(func() { rooms.RemoveTestRoom(id) })
	}

	require.NotNil(t, module, "init registered the module")
	rolls := uint64(0)
	module.roll = func() uint64 { return rolls }
	module.zoneExists = func(zone string) bool { return shippedZones[zone] }
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

	run := func(roomId int) string {
		t.Helper()
		user.Character.RoomId = roomId
		*messages = nil
		handled, err := usercommands.TryCommand("market", "", user.UserId, events.CmdSkipScripts)
		require.NoError(t, err)
		require.True(t, handled)
		events.ProcessEvents()
		return stripTags(strings.Join(*messages, "\n"))
	}

	hide := goodFor(t, "Dunmar", 28)
	out := run(2001)
	assert.Contains(t, out, "Market prices in Dunmar:")
	assert.Regexp(t, `wolf hide\s+27 gold\s+\(scarce\)`, out, "shipped StartStock 4")
	assert.Equal(t, 27, hide.PriceForStock(4))
	assert.Contains(t, run(2002), "Market prices in Old Kings Road:")
	assert.Contains(t, run(1), "There's no market here.", "Frostfang is not a market")

	events.AddToQueue(events.NewRound{RoundNumber: 1})
	events.ProcessEvents()

	stock, ok := module.zones["Dunmar"].Stock(28)
	require.True(t, ok)
	assert.Equal(t, 5, stock, "one bounded step toward target 20")
	assert.Regexp(t, `wolf hide\s+26 gold\s+\(scarce\)`, run(2001), "price follows stock")
	assert.Equal(t, 26, hide.PriceForStock(5))
	okrHide, _ := module.zones["Old Kings Road"].Stock(28)
	assert.Equal(t, 27, okrHide, "a glutted market drifts down")

	reloaded := NewRegistry()
	require.NoError(t, pluginStore{plug: module.plug}.Load(reloaded))
	assert.Equal(t, Registry{Zones: module.zones}, *reloaded, "the round's drift was persisted")

	// A restart restores the persisted stock instead of re-seeding.
	restarted := &MarketModule{
		plug:       module.plug,
		store:      pluginStore{plug: module.plug},
		roll:       func() uint64 { return 0 },
		itemExists: module.itemExists,
		itemName:   module.itemName,
		zoneExists: module.zoneExists,
		markets:    map[string][]market.Good{},
		zones:      map[string]ZoneMarket{},
	}
	restarted.load()
	restartedStock, _ := restarted.zones["Dunmar"].Stock(28)
	assert.Equal(t, 5, restartedStock)
}

// shippedZoneNames reads every zone name from the shipped zone-config files.
func shippedZoneNames(t *testing.T, dataDir string) map[string]bool {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dataDir, "rooms", "*", "zone-config.yaml"))
	require.NoError(t, err)
	names := map[string]bool{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		var cfg rooms.ZoneConfig
		require.NoError(t, yaml.Unmarshal(data, &cfg))
		names[cfg.Name] = true
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
