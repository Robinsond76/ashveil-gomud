package standing

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	domain "github.com/GoMudEngine/GoMud/internal/standing"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	// The real modules standing is wired into: the company alignment
	// provider, the market, and the inn (with survival, which the company
	// and camping modules need to load).
	_ "github.com/GoMudEngine/GoMud/modules/camping"
	_ "github.com/GoMudEngine/GoMud/modules/company"
	_ "github.com/GoMudEngine/GoMud/modules/market"
	_ "github.com/GoMudEngine/GoMud/modules/survival"
)

var tagPattern = regexp.MustCompile(`<[^>]*>`)

// TestStandingThroughPluginsLoad drives Phase 21b through the real modules:
// plugins.Load registers standing, company, market, camping, and survival
// with their shipped config overlays; the shipped Dunmar rooms are
// registered zone-indexed; commands run through usercommands.TryCommand.
// The company alignment comes from the real company module (a leader with
// no companions averages to their own alignment).
func TestStandingThroughPluginsLoad(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	dataDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
	require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": dataDir}))
	keywords.LoadAliases()
	items.LoadDataFiles()
	shipped := map[int]*rooms.Room{}
	for id, path := range map[int]string{
		2001: "rooms/dunmar/2001.yaml",
		2003: "rooms/dunmar/2003.yaml",
		2004: "rooms/dunmar/2004.yaml",
		2006: "rooms/dunmar/2006.yaml",
		2002: "rooms/old_kings_road/2002.yaml",
		2005: "rooms/old_kings_road/2005.yaml",
		1:    "rooms/frostfang/1.yaml",
	} {
		data, err := os.ReadFile(filepath.Join(dataDir, path))
		require.NoError(t, err)
		room := &rooms.Room{}
		require.NoError(t, yaml.Unmarshal(data, room))
		require.Equal(t, id, room.RoomId)
		shipped[id] = room
		rooms.SetTestZoneRoom(room)
		t.Cleanup(func() { rooms.RemoveTestZoneRoom(room) })
	}
	assert.True(t, shipped[2006].HasTag("blackmarket"), "Tanner's Back Alley is a black market")
	assert.Equal(t, 2006, shipped[2004].Exits["south"].RoomId)
	assert.Equal(t, 2004, shipped[2006].Exits["north"].RoomId)

	require.NotNil(t, module, "init registered the module")
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(t.TempDir())
	cfg := module.config()
	assert.Equal(t, map[string]int{"Dunmar": 40, "Old Kings Road": 0, "Frostfang": 30}, cfg.settlements, "shipped settlements")
	assert.Equal(t, domain.DefaultRules(), cfg.rules)
	assert.Equal(t, "blackmarket", domain.BlackMarketTag())

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 1)
	user.Username = "outlaw"
	user.Password = "$2a$test" // not a plaintext password, so commands aren't gated
	user.Character.Gold = 1000
	users.SetTestUser(user)
	messages := captureMessages(t)
	run := func(roomID int, command, rest string) string {
		t.Helper()
		user.Character.RoomId = roomID
		*messages = nil
		handled, err := usercommands.TryCommand(command, rest, user.UserId, events.CmdSkipScripts)
		require.NoError(t, err)
		require.True(t, handled)
		events.ProcessEvents()
		return tagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
	}

	// Distrusted: gap 100 from Dunmar.
	user.Character.Alignment = -60
	out := run(2004, "standing", "")
	assert.Contains(t, out, "Dunmar (alignment 40, virtuous) regards your company (alignment -60, evil) as distrusted.")
	assert.Contains(t, out, "Old Kings Road: tolerated")
	assert.Contains(t, out, "Frostfang: distrusted")
	out = run(2004, "market", "")
	assert.Regexp(t, `wolf hide\s+33 gold\s+17 gold\s+scarce`, out, "shipped stock 4: 27/22 base, marked 20%")
	assert.Contains(t, run(2004, "market", "buy hide"), "for 33 gold")
	assert.Equal(t, 967, user.Character.Gold)
	assert.Contains(t, run(2003, "inn", ""), "Your company is distrusted here, so the room costs 50% more.")
	assert.Contains(t, run(2006, "market", "buy hide"), "for 28 gold", "the black market serves the distrusted at stock 3's base price")
	assert.Equal(t, 939, user.Character.Gold)

	// Shunned: gap 140.
	user.Character.Alignment = -100
	assert.Contains(t, run(2004, "market", ""), "The traders of Dunmar won't deal with your company.")
	assert.Contains(t, run(2003, "inn", "rest"), "won't give your company a room")
	assert.Equal(t, 939, user.Character.Gold)
	out = run(2006, "market", "")
	assert.Contains(t, out, "Black market prices in Dunmar:")
	assert.Regexp(t, `wolf hide\s+29 gold`, out, "stock 2, no markup")

	// Welcome: turned away from the black market, normal prices elsewhere.
	user.Character.Alignment = 40
	assert.Contains(t, run(2006, "market", "buy hide"), "No one here will deal with your company.")
	assert.Equal(t, 939, user.Character.Gold)
	assert.Regexp(t, `wolf hide\s+29 gold`, run(2004, "market", ""))
	assert.Contains(t, run(2001, "market", ""), "The market in Dunmar is at Dunmar Market Square.", "the hint names the square, not the alley")
	assert.Contains(t, run(2001, "standing", ""), "as welcome")
}
