package gathering

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
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

var wiringTags = regexp.MustCompile(`<[^>]*>`)

// TestGatheringThroughPluginsLoad drives the shipped module config, items and
// a shipped room through plugins.Load and usercommands.TryCommand.
func TestGatheringThroughPluginsLoad(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	_, thisFile, _, _ := runtime.Caller(0)
	dataDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
	require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": dataDir}))
	keywords.LoadAliases()
	items.LoadDataFiles()

	data, err := os.ReadFile(filepath.Join(dataDir, "rooms", "old_kings_road", "2002.yaml"))
	require.NoError(t, err)
	room := &rooms.Room{}
	require.NoError(t, yaml.Unmarshal(data, room))
	rooms.SetTestZoneRoom(room)
	t.Cleanup(func() { rooms.RemoveTestZoneRoom(room) })
	assert.Contains(t, room.Resources, "firewood", "the lightning-split oak is a firewood room")

	require.NotNil(t, module, "init registered the module")
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(t.TempDir())
	cfg := module.cfg()
	for _, id := range []int{cfg.Items.Firewood, cfg.Items.DampFirewood, cfg.Items.FishingLine, cfg.Items.RawFish, cfg.Items.BitterWeed, cfg.Items.RawMeat} {
		assert.NotNil(t, items.GetItemSpec(id), "shipped item %d exists", id)
	}

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 2002)
	user.Character.RoomId = 2002
	user.Character.Health = 10
	user.Username = "forager"
	user.Password = "$2a$test"
	users.SetTestUser(user)

	messages := []string{}
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		messages = append(messages, e.(events.Message).Text)
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })

	run := func(command, rest string) string {
		t.Helper()
		messages = nil
		handled, err := usercommands.TryCommand(command, rest, user.UserId, events.CmdSkipScripts)
		require.NoError(t, err)
		require.True(t, handled, command)
		events.ProcessEvents()
		return wiringTags.ReplaceAllString(strings.Join(messages, "\n"), "")
	}

	out := run("gather", "")
	assert.NotEmpty(t, out, "gather lists what the room offers")
	assert.Contains(t, strings.ToLower(out), "firewood")

	out = run("gather", "firewood")
	assert.Contains(t, strings.ToLower(out), "firewood", "the work starts")

	_, busy := module.Active(user.UserId)
	assert.True(t, busy, "the work is under way")
	// Looking keeps it going; any other typed command cancels it through the
	// real Input listener registered by plugins.Load.
	events.AddToQueue(events.Input{UserId: user.UserId, InputText: "look"})
	events.ProcessEvents()
	_, busy = module.Active(user.UserId)
	assert.True(t, busy)
	events.AddToQueue(events.Input{UserId: user.UserId, InputText: "say hello"})
	events.ProcessEvents()
	_, busy = module.Active(user.UserId)
	assert.False(t, busy)
}
