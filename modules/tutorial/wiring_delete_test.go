package tutorial

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/modules/gmcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// TestDeleteCharacterThroughPluginsLoad drives Phase 32h through the real
// entry points: plugins.Load with the shipped modules, a real (piped)
// connection, the engine's own leave, hand-off, join, and purge handlers,
// and the real "delete" command answered as the world answers a prompt. A
// character with a company out, a camp, cargo, gear, gold, and experience
// deletes themselves: they come back on the same connection as a new
// character in the Void, where creation runs, with nothing of the old one
// in any module; the login and the old name survive and are free. A wrong
// password or name deletes nothing, and a restart between the flag and the
// purge is finished by the boot sweep. The clock never moves.
func TestDeleteCharacterThroughPluginsLoad(t *testing.T) {
	dataDir := t.TempDir()
	setOverrides(t, map[string]any{
		"FilePaths.DataFiles":        dataDir,
		"SpecialRooms.StartRoom":     1,
		"SpecialRooms.TutorialRooms": []any{"900", "901", "902", "903", "904", "905", "906", "907"},
	})
	writeTutorialWorld(t, dataDir)
	races.LoadDataFiles()
	items.LoadDataFiles()
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	mobs.LoadDataFiles()
	keywords.LoadAliases()
	require.NotNil(t, module)
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dataDir)
	buffs.RegisterFS(plugins.GetPluginRegistry())
	buffs.LoadDataFiles()
	templates.RegisterFS(plugins.GetPluginRegistry())

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
	ids := []events.ListenerId{
		events.RegisterListener(events.PlayerSpawn{}, hooks.HandleJoin),
		events.RegisterListener(events.PlayerDespawn{}, hooks.HandleLeave, events.Last),
		events.RegisterListener(events.UserHandOff{}, hooks.HandleUserHandOff),
		events.RegisterListener(events.UserPurged{}, hooks.HandlePurge, events.Last),
		events.RegisterListener(gmcp.GMCPOut{}, func(events.Event) events.ListenerReturn { return events.Cancel }, events.First),
	}
	t.Cleanup(func() {
		events.UnregisterListener(events.PlayerSpawn{}, ids[0])
		events.UnregisterListener(events.PlayerDespawn{}, ids[1])
		events.UnregisterListener(events.UserHandOff{}, ids[2])
		events.UnregisterListener(events.UserPurged{}, ids[3])
		events.UnregisterListener(gmcp.GMCPOut{}, ids[4])
	})
	messages := map[int][]string{}
	lid := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		m := e.(events.Message)
		messages[m.UserId] = append(messages[m.UserId], m.Text)
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, lid) })
	text := func(userID int) string {
		events.ProcessEvents()
		out := tagPattern.ReplaceAllString(strings.Join(messages[userID], "\n"), "")
		messages[userID] = nil
		return out
	}
	run := func(u *users.UserRecord, cmd, rest string) string {
		t.Helper()
		events.ProcessEvents()
		handled, err := usercommands.TryCommand(cmd, rest, u.UserId, events.CmdSkipScripts)
		require.NoError(t, err, cmd+" "+rest)
		require.True(t, handled, cmd+" "+rest)
		return text(u.UserId)
	}
	// answer replies to the open question as world.processInput does: the
	// answer, then the prompt's command again, then the connection's mask.
	answer := func(u *users.UserRecord, reply string) string {
		t.Helper()
		p := u.GetPrompt()
		require.NotNil(t, p, "a question is open")
		p.GetNextQuestion().Answer(reply)
		got := run(u, p.Command, p.Rest)
		u.SyncInputMask()
		return got
	}
	turn, round := util.GetTurnCount(), util.GetRoundCount()
	start := rooms.LoadRoom(1)
	require.NotNil(t, start)
	start.Tags = append(start.Tags, "camping")

	// A character who skipped the course with Tamsin, on a real connection.
	conn, wire := replayConnect(t)
	cs := connections.GetClientSettings(conn)
	cs.IsMudlet = true // Mudlet hides a field on WILL ECHO
	connections.OverwriteClientSettings(conn, cs)
	hash, err := bcrypt.GenerateFromPassword([]byte("hunter22"), bcrypt.MinCost)
	require.NoError(t, err)
	aria := users.NewUserRecord(7, conn)
	aria.Username = "acctaria"
	aria.Password = string(hash)
	aria.Macros = map[string]string{"=1": "look"}
	aria.Character.Name = "Aria"
	aria.Character.RaceId = 1
	aria.Character.Validate()
	aria.Character.ActionPoints = 1000
	aria.Character.RoomId = -1
	require.NoError(t, users.SaveUser(*aria))
	_, _, err = users.LoginUser(aria, conn)
	require.NoError(t, err)
	users.GetCharacterIndex().Add("Aria", 7)
	t.Cleanup(func() { users.GetCharacterIndex().Remove("Aria") })
	void := &rooms.Room{RoomId: -1, Title: "The Void"}
	_, err = usercommands.Start("", aria, void, 0)
	require.NoError(t, err)
	aria.GetPrompt().GetNextQuestion().Answer("no")
	_, err = usercommands.Start("", aria, void, 0)
	require.NoError(t, err)
	p := progressOf(aria.Character)
	p.Stage = StageCompany
	p.save(aria.Character)
	require.True(t, module.Begin(aria.UserId))
	text(aria.UserId)
	run(aria, "company", "recruit tamsin")
	run(aria, "tutorial", "skip yes")
	require.Equal(t, 1, aria.Character.RoomId)
	aria.Character.Gold = 777
	aria.Character.Experience = 4321
	require.True(t, aria.Character.StoreItem(items.New(30004)))
	require.True(t, aria.Character.StoreItem(items.New(30004)))
	stowed := items.New(30004)
	cargoItem := stowed.GetSpec().Name
	run(aria, "cargo", "put "+cargoItem)
	require.Contains(t, run(aria, "cargo", ""), cargoItem, "cargo stowed")
	assert.Contains(t, run(aria, "camp", ""), "You make camp here.")
	members, _ := company.CompanyMembers(7)
	require.Len(t, members, 1)
	_, camped := camping.LeaderRest(7)
	require.True(t, camped)

	// Refused mid-fight.
	aria.Character.Aggro = &characters.Aggro{MobInstanceId: 1}
	assert.Contains(t, run(aria, "delete", "character"), "too busy fighting")
	aria.Character.Aggro = nil

	// A wrong password deletes nothing, and isn't echoed: Mudlet is told
	// WILL ECHO while the question is open, WONT once it's answered.
	got := run(aria, "delete", "character")
	aria.SyncInputMask()
	assert.Contains(t, got, "This deletes Aria for good")
	assert.True(t, connections.InputMasked(conn), "the password isn't echoed")
	assert.Contains(t, answer(aria, "wrong"), "Nothing was deleted.")
	assert.False(t, connections.InputMasked(conn))
	assert.Contains(t, wire.String(), "\xff\xfb\x01", "WILL ECHO")
	assert.Contains(t, wire.String(), "\xff\xfc\x01", "WONT ECHO")

	// The right password and the wrong name delete nothing either.
	run(aria, "delete", "character")
	answer(aria, "hunter22")
	assert.Contains(t, answer(aria, "Brom"), "Nothing was deleted.")
	require.Same(t, aria, users.GetByConnectionId(conn))
	assert.Equal(t, 777, aria.Character.Gold)

	// Both right: Aria is gone, and the same login is back on the same
	// connection, in the Void, making a new character.
	run(aria, "delete", "character")
	answer(aria, "hunter22")
	assert.Contains(t, answer(aria, "aria"), "Aria is gone.")
	events.ProcessEvents()
	fresh := users.GetByConnectionId(conn)
	require.NotNil(t, fresh, "the connection has a user")
	assert.NotSame(t, aria, fresh)
	assert.Equal(t, 7, fresh.UserId)
	assert.Equal(t, "acctaria", fresh.Username, "the login stays")
	assert.Equal(t, "look", fresh.Macros["=1"], "and the player's macros")
	assert.False(t, fresh.Deleting)
	assert.Equal(t, -1, fresh.Character.RoomId, "the Void")
	assert.NotEqual(t, "Aria", fresh.Character.Name)
	assert.NotEqual(t, 777, fresh.Character.Gold)
	assert.Less(t, fresh.Character.Experience, 4321)
	assert.Empty(t, fresh.Character.GetAllBackpackItems())
	assert.NotContains(t, wire.String(), "Goodbye", "no goodbye on a hand-off")
	assertDeleted(t, 7)
	assert.NotContains(t, run(fresh, "cargo", ""), cargoItem, "no cargo")
	_, found := users.GetCharacterIndex().Find("Aria")
	assert.False(t, found, "the old name is free")

	// Creation runs.
	_, err = usercommands.Start("", fresh, void, 0)
	require.NoError(t, err)
	require.NotNil(t, fresh.GetPrompt(), "character creation asks its first question")

	// The same login works afterwards.
	users.ResetActiveUsers()
	connections.Remove(conn)
	conn2, _ := replayConnect(t)
	loaded, err := users.LoadUserFile(7)
	require.NoError(t, err)
	assert.True(t, loaded.PasswordMatches("hunter22"))
	_, _, err = users.LoginUser(loaded, conn2)
	require.NoError(t, err)

	// A restart between the flag and the purge: the flagged record can't
	// log in, and the boot sweep finishes the deletion.
	loaded.Character.Name = "Dain"
	loaded.Character.RoomId = 1
	loaded.Character.Gold = 55
	loaded.Deleting = true
	require.NoError(t, users.SaveUser(*loaded))
	users.ResetActiveUsers()
	connections.Remove(conn2)
	flagged, err := users.LoadUserFile(7)
	require.NoError(t, err)
	conn3, _ := replayConnect(t)
	_, msg, err := users.LoginUser(flagged, conn3)
	assert.Error(t, err)
	assert.Equal(t, users.DeletingLoginRefusal, msg)
	hooks.SweepDeletions()
	events.ProcessEvents()
	swept, err := users.LoadUserFile(7)
	require.NoError(t, err)
	assert.False(t, swept.Deleting)
	assert.NotEqual(t, "Dain", swept.Character.Name)
	assert.Equal(t, -1, swept.Character.RoomId)
	_, err = os.Stat(filepath.Join(dataDir, "users", "7.yaml"))
	assert.NoError(t, err, "the file stays")

	assert.Equal(t, turn, util.GetTurnCount(), "the clock never moves")
	assert.Equal(t, round, util.GetRoundCount())
}

// assertDeleted: no module holds the old character's state for the user.
func assertDeleted(t *testing.T, userID int) {
	t.Helper()
	members, _ := company.CompanyMembers(userID)
	assert.Empty(t, members, "no company")
	assert.False(t, company.HasClaimed(userID, 61), "no claims")
	_, ok := camping.LeaderRest(userID)
	assert.False(t, ok, "no camp")
	assert.Nil(t, module.copies[userID], "no course copies")
	assert.Nil(t, module.fights[userID], "no squad")
}
