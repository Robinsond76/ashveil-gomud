package company

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompanyMovesAsOneThroughGo drives Phase 32a's movement lines through
// the real entry points: the shipped candidates recruited with the real
// command, the leader walking with the real "go", and the companions
// following through the real mob "go" (the test runs queued mob commands
// as the world loop does). The leader sees no companion line; a watcher in
// each room sees exactly one company line; a companion walking alone keeps
// its own lines. The clock never moves.
func TestCompanyMovesAsOneThroughGo(t *testing.T) {
	dataDir := t.TempDir()
	useDataDir(t, dataDir)
	world := shippedWorld(t)
	shipped := func(rel string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(world, rel))
		require.NoError(t, err)
		return string(data)
	}
	fixtures := map[string]string{
		"biomes/default.yaml":                                "biomeid: default\nname: Test\nsymbol: '.'\n",
		"keywords.yaml":                                      "direction-aliases: {}\n",
		"races/1-human.yaml":                                 shipped("races/1-human.yaml"),
		"rooms/dunmar/zone-config.yaml":                      "name: Dunmar\nroomid: 2003\n",
		"rooms/dunmar/2003.yaml":                             "roomid: 2003\nzone: Dunmar\ntitle: The Waymark Inn\ndescription: A hiring slate hangs by the hearth.\nexits:\n  east:\n    roomid: 2001\n",
		"rooms/dunmar/2001.yaml":                             "roomid: 2001\nzone: Dunmar\ntitle: Dunmar West Gate\ndescription: A gate.\nexits:\n  west:\n    roomid: 2003\n",
		"mobs/dunmar/61-tamsin_reed.yaml":                    shipped("mobs/dunmar/61-tamsin_reed.yaml"),
		"mobs/dunmar/62-brother_oswin.yaml":                  shipped("mobs/dunmar/62-brother_oswin.yaml"),
		"mobs/dunmar/63-garrick_vane.yaml":                   shipped("mobs/dunmar/63-garrick_vane.yaml"),
		"items/weapons-10000/10015-crude_cudgel.yaml":        "itemid: 10015\nname: crude cudgel\nnamesimple: cudgel\ntype: weapon\nhands: 1\nsubtype: bludgeoning\ndamage:\n  diceroll: 1d4\n",
		"items/armor-20000/offhand/20004-wooden_shield.yaml": "itemid: 20004\nname: wooden shield\nnamesimple: shield\ntype: offhand\nsubtype: wearable\n",
	}
	for path, data := range fixtures {
		fullPath := filepath.Join(dataDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(data), 0600))
	}
	for _, dir := range []string{"users", "combat-messages"} {
		require.NoError(t, os.MkdirAll(filepath.Join(dataDir, dir), 0755))
	}
	races.LoadDataFiles()
	items.LoadDataFiles()
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	mobs.LoadDataFiles()
	keywords.LoadAliases()
	inn, gate := rooms.LoadRoom(2003), rooms.LoadRoom(2001)
	require.NotNil(t, inn)
	require.NotNil(t, gate)

	useFakeLifecycle(t, &fakeLifecycle{})
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dataDir)
	require.NoError(t, module.loadErr)

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	player := func(id int, name string, room *rooms.Room) *users.UserRecord {
		u := users.NewUserRecord(id, uint64(id))
		u.Username = "acct" + name
		u.Password = "$2a$test"
		u.Character.Name = name
		u.Character.RaceId = 1
		u.Character.Alignment = 30
		u.Character.Validate()
		u.Character.ActionPoints = 1000
		u.Character.RoomId = room.RoomId
		users.SetTestUser(u)
		room.AddPlayer(u.UserId)
		t.Cleanup(func() {
			if r := rooms.LoadRoom(u.Character.RoomId); r != nil {
				r.RemovePlayer(u.UserId)
			}
		})
		return u
	}
	dain := player(7, "Dain", inn)
	behind := player(8, "Mira", inn)   // stays in the inn
	ahead := player(9, "Corwen", gate) // waits at the gate
	t.Cleanup(func() {
		for _, instance := range mobs.GetAllMobInstanceIds() {
			nativeRuntime{}.Detach(7, instance)
		}
		module.registry = *domain.NewRegistry()
		module.instances = map[int]map[int]int{}
	})

	// The world loop's part: queued mob commands run as they arrive.
	mid := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
		if in, ok := e.(events.Input); ok && in.MobInstanceId > 0 {
			cmd, rest, _ := strings.Cut(in.InputText, " ")
			_, _ = mobcommands.TryCommand(strings.ToLower(cmd), rest, in.MobInstanceId, in.MemberOrder)
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Input{}, mid) })
	var sent []events.Message
	lid := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		sent = append(sent, e.(events.Message))
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, lid) })
	// seenBy is what a player who stayed put received.
	seenBy := func(u *users.UserRecord) []string {
		var out []string
		for _, m := range sent {
			if m.UserId == u.UserId || (m.UserId == 0 && m.RoomId == u.Character.RoomId && !contains(m.ExcludeUserIds, u.UserId)) {
				out = append(out, companyTagPattern.ReplaceAllString(m.Text, ""))
			}
		}
		return out
	}
	run := func(u *users.UserRecord, cmd, rest string) {
		t.Helper()
		handled, err := usercommands.TryCommand(cmd, rest, u.UserId, events.CmdSkipScripts)
		require.NoError(t, err)
		require.True(t, handled)
		events.ProcessEvents()
	}
	turn, round := util.GetTurnCount(), util.GetRoundCount()

	run(dain, "company", "recruit tamsin")
	run(dain, "company", "recruit oswin")
	companions := []int{}
	for _, key := range []int{1, 2} {
		instanceID, ok := domain.InstanceFor(dain.UserId, key)
		require.True(t, ok, "companion %d", key)
		companions = append(companions, instanceID)
	}

	sent = nil
	run(dain, "go", "east")
	require.Equal(t, gate.RoomId, dain.Character.RoomId)
	for _, id := range companions {
		require.Equal(t, gate.RoomId, mobs.GetInstance(id).Character.RoomId, "the company came along")
	}
	for _, m := range sent {
		assert.NotContains(t, m.Text, "Tamsin", "no companion line to anyone")
		assert.NotContains(t, m.Text, "Oswin", "no companion line to anyone")
	}
	left := []string{}
	for _, line := range seenBy(behind) {
		if strings.Contains(line, "Dain") {
			left = append(left, line)
		}
	}
	if assert.Len(t, left, 1, "one departure line") {
		assert.Contains(t, left[0], "Dain leads their company towards the east exit.")
	}
	arrived := []string{}
	for _, line := range seenBy(ahead) {
		if strings.Contains(line, "Dain") {
			arrived = append(arrived, line)
		}
	}
	if assert.Len(t, arrived, 1, "one arrival line") {
		assert.Contains(t, arrived[0], "Dain arrives from the west, their company behind.")
	}

	// A companion walking on its own still announces itself.
	sent = nil
	_, err := mobcommands.TryCommand("go", "west", companions[0])
	require.NoError(t, err)
	events.ProcessEvents()
	require.Equal(t, inn.RoomId, mobs.GetInstance(companions[0]).Character.RoomId)
	assert.Contains(t, strings.Join(seenBy(ahead), "\n"), "Tamsin Reed leaves towards the west exit.")
	assert.Contains(t, strings.Join(seenBy(behind), "\n"), "Tamsin Reed enters from the east.")

	// Review fix: walking back to rejoin the leader is its own move, not
	// the company line's, so it still shows.
	sent = nil
	_, err = mobcommands.TryCommand("go", "east", companions[0])
	require.NoError(t, err)
	events.ProcessEvents()
	require.Equal(t, gate.RoomId, mobs.GetInstance(companions[0]).Character.RoomId)
	assert.Contains(t, strings.Join(seenBy(behind), "\n"), "Tamsin Reed leaves towards the east exit.")

	// Review fix: a leader moved without a company line (a flight, travel's
	// arrival, a sneak) leaves companions who follow announcing themselves.
	sent = nil
	require.NoError(t, rooms.MoveToRoom(dain.UserId, inn.RoomId))
	_, err = mobcommands.TryCommand("go", "west", companions[1])
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, strings.Join(seenBy(behind), "\n"), "Brother Oswin enters from the east.")

	// A leader with no companion beside them walks as anyone does.
	sent = nil
	run(ahead, "go", "west")
	assert.Contains(t, strings.Join(seenBy(behind), "\n"), "Corwen enters from the east.")

	assert.Equal(t, turn, util.GetTurnCount(), "never advances the clock")
	assert.Equal(t, round, util.GetRoundCount())
}

func contains(ids []int, id int) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}
