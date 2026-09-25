package death

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
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
	"github.com/GoMudEngine/GoMud/modules/gmcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type gmcpSent struct {
	userID int
	module string
	body   map[string]any
}

// TestCompanyGMCPThroughPluginsLoad drives Phase 26b through the real
// entry points: plugins.Load with the GMCP, company, survival, travel,
// camping, death, and encumbrance modules; the Company payloads are
// captured from the real GMCPOut events that the GMCP module dispatches.
// Login sends the snapshot, a recruit resends it, damage sends only vitals,
// a death shows the member dead with its time, a resurrection shows it
// present again, and a web request resends the snapshot.
func TestCompanyGMCPThroughPluginsLoad(t *testing.T) {
	dataDir := t.TempDir()
	useDataDir(t, dataDir)
	writeWiringWorld(t, dataDir)
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "plugin-data"), 0755))
	races.LoadDataFiles()
	items.LoadDataFiles()
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	mobs.LoadDataFiles()
	keywords.LoadAliases()
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dataDir)

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	t.Cleanup(func() {
		for _, instance := range mobs.GetAllMobInstanceIds() {
			if mob := mobs.GetInstance(instance); mob != nil {
				if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
					room.RemoveMob(instance)
				}
			}
			mobs.DestroyInstance(instance)
		}
	})
	user := users.NewUserRecord(27, 2727)
	user.Username = "ysolde"
	user.Password = "$2a$test"
	user.Character.Name = "Ysolde"
	user.Character.RaceId = 1
	user.Character.Level = 5
	user.Character.ActionPoints = 100
	user.Character.Validate()
	user.Character.Health = user.Character.HealthMax.Value
	users.SetTestUser(user)
	other := users.NewUserRecord(28, 2828)
	other.Character.Name = "Watcher"
	users.SetTestUser(other)
	require.NoError(t, rooms.MoveToRoom(user.UserId, 2004))
	require.NoError(t, rooms.MoveToRoom(other.UserId, 2004))
	t.Cleanup(func() {
		for _, u := range []*users.UserRecord{user, other} {
			if room := rooms.LoadRoom(u.Character.RoomId); room != nil {
				room.RemovePlayer(u.UserId)
			}
		}
	})

	var sent []gmcpSent
	id := events.RegisterListener(gmcp.GMCPOut{}, func(e events.Event) events.ListenerReturn {
		out := e.(gmcp.GMCPOut)
		if out.Module != "Company" && out.Module != "Company.Vitals" {
			return events.Continue
		}
		var body map[string]any
		if raw, ok := out.Payload.([]byte); ok {
			require.NoError(t, json.Unmarshal(raw, &body))
		}
		sent = append(sent, gmcpSent{out.UserId, out.Module, body})
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(gmcp.GMCPOut{}, id) })
	take := func() []gmcpSent {
		events.ProcessEvents()
		out := sent
		sent = nil
		return out
	}
	mine := func(list []gmcpSent) []gmcpSent {
		var out []gmcpSent
		for _, s := range list {
			if s.userID == user.UserId {
				out = append(out, s)
			}
		}
		return out
	}
	run := func(command, rest string) {
		t.Helper()
		events.ProcessEvents()
		handled, err := usercommands.TryCommand(command, rest, user.UserId, events.CmdSkipScripts)
		require.NoError(t, err)
		require.True(t, handled)
	}
	newRound := func() {
		events.AddToQueue(events.NewRound{RoundNumber: util.GetRoundCount()})
	}
	member := func(s gmcpSent, key string) map[string]any {
		t.Helper()
		for _, m := range s.body["members"].([]any) {
			if mm := m.(map[string]any); mm["key"] == key {
				return mm
			}
		}
		t.Fatalf("no member %s in %v", key, s.body)
		return nil
	}
	turn, round := util.GetTurnCount(), util.GetRoundCount()
	take()

	// Login: the first refresh sends the full snapshot; the next, nothing.
	events.AddToQueue(events.PlayerSpawn{UserId: user.UserId, RoomId: 2004})
	events.ProcessEvents()
	newRound()
	got := mine(take())
	require.Len(t, got, 1)
	assert.Equal(t, "Company", got[0].module)
	assert.Equal(t, "Ysolde", got[0].body["leader"].(map[string]any)["name"])
	assert.Empty(t, got[0].body["members"])
	newRound()
	assert.Empty(t, mine(take()), "unchanged: nothing sent")

	// A recruit: the command's own refresh resends the snapshot.
	run("company", "summon training dummy")
	got = mine(take())
	require.Len(t, got, 1)
	assert.Equal(t, "Company", got[0].module)
	assert.Equal(t, "present", member(got[0], "companion:1")["status"])

	// Damage: only vitals.
	instanceID, ok := company.InstanceFor(user.UserId, 1)
	require.True(t, ok)
	mob := mobs.GetInstance(instanceID)
	mob.Character.Health = 3
	newRound()
	got = mine(take())
	require.Len(t, got, 1)
	assert.Equal(t, "Company.Vitals", got[0].module)
	assert.Equal(t, 3.0, got[0].body["vitals"].(map[string]any)["companion:1"].(map[string]any)["hp"])

	// Death: dead, with its time to raise.
	_, err := mobcommands.Suicide("", mob, rooms.LoadRoom(mob.Character.RoomId))
	require.NoError(t, err)
	newRound()
	got = mine(take())
	require.NotEmpty(t, got)
	last := got[len(got)-1]
	assert.Equal(t, "Company", last.module)
	dead := member(last, "companion:1")
	assert.Equal(t, "dead", dead["status"])
	assert.Equal(t, 10800.0, dead["rescue_seconds"])
	assert.Equal(t, 1.0, last.body["dead"])

	// The chapel raises it: present again.
	run("east", "")
	require.Equal(t, 2007, user.Character.RoomId)
	spawnKeeper(t, 2007)
	take()
	run("resurrect", "#1")
	got = mine(take())
	require.NotEmpty(t, got)
	last = got[len(got)-1]
	assert.Equal(t, "present", member(last, "companion:1")["status"])
	assert.Equal(t, 0.0, last.body["dead"])

	// A web request resends the snapshot at once.
	events.AddToQueue(gmcp.GMCPCompanyRequest{UserId: user.UserId})
	got = mine(take())
	require.Len(t, got, 1)
	assert.Equal(t, "Company", got[0].module)

	// Privacy: the other player gets only their own (empty) company.
	events.AddToQueue(gmcp.GMCPCompanyRequest{UserId: other.UserId})
	theirs := 0
	for _, s := range take() {
		if s.userID == other.UserId {
			theirs++
			assert.Equal(t, "Watcher", s.body["leader"].(map[string]any)["name"])
			assert.Empty(t, s.body["members"])
		}
	}
	assert.Equal(t, 1, theirs)

	assert.Equal(t, turn, util.GetTurnCount(), "never advances the clock")
	assert.Equal(t, round, util.GetRoundCount())
}
