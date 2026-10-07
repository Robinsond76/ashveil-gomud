package tutorial

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
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
)

// replayWire is the client end of a real (piped) connection, drained so
// the server never blocks writing to it.
type replayWire struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	closed bool
}

func (w *replayWire) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func replayConnect(t *testing.T) (connections.ConnectionId, *replayWire) {
	t.Helper()
	server, client := net.Pipe()
	cd := connections.Add(server, nil)
	cd.SetState(connections.LoggedIn)
	out := &replayWire{}
	go func() {
		chunk := make([]byte, 4096)
		for {
			n, err := client.Read(chunk)
			out.mu.Lock()
			out.buf.Write(chunk[:n])
			if err != nil {
				out.closed = true
			}
			out.mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	t.Cleanup(func() {
		connections.Remove(cd.ConnectionId())
		client.Close()
	})
	return cd.ConnectionId(), out
}

// snapshot is what "exactly back" compares: where the real character is,
// what they carry, their purse and experience, their company and camp.
type snapshot struct {
	room, gold, xp, level int
	items                 []int
	members               []string
	claimed61             bool
	campRoom              int
	camped                bool
}

func snapshotOf(u *users.UserRecord) snapshot {
	s := snapshot{room: u.Character.RoomId, gold: u.Character.Gold, xp: u.Character.Experience, level: u.Character.Level}
	for _, itm := range u.Character.GetAllBackpackItems() {
		s.items = append(s.items, itm.ItemId)
	}
	sort.Ints(s.items)
	members, _ := company.CompanyMembers(u.UserId)
	for _, member := range members {
		s.members = append(s.members, member.Name)
	}
	sort.Strings(s.members)
	s.claimed61 = company.HasClaimed(u.UserId, 61)
	_, s.camped = camping.LeaderRest(u.UserId)
	return s
}

// TestTutorialReplayThroughPluginsLoad drives Phase 32b through the real
// entry points: plugins.Load with the shipped modules, a real (piped)
// connection, the engine's own leave, hand-off, join, and purge handlers,
// and the real "tutorial" command. A real character with a company, a
// camp, gear, gold, and experience replays the course as a level-1
// character with nothing, and comes back exactly as they were by each way
// out: skipping, graduating, dying, and quitting. Every exit leaves no
// module state and no file for the throwaway; a restart mid-replay is
// swept at boot. The clock never moves.
func TestTutorialReplayThroughPluginsLoad(t *testing.T) {
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

	// Player help: "help tutorial" (and "help replay") renders the replay.
	templates.RegisterFS(plugins.GetPluginRegistry())
	for _, topic := range []string{"tutorial", "replay"} {
		page, err := usercommands.GetHelpContents(topic)
		require.NoError(t, err, topic)
		page = tagPattern.ReplaceAllString(page, "")
		assert.Contains(t, page, "tutorial replay yes", topic)
		assert.Contains(t, page, "Nothing from a replay is kept", topic)
	}

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

	// The engine's own join, leave, hand-off, and purge handling.
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
	var messages = map[int][]string{}
	freshEvents(t)
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
	template := func(u *users.UserRecord) int { return rooms.GetOriginalRoom(u.Character.RoomId) }
	caps := func(u *users.UserRecord) int { return countItem(u, 20043) }
	turn, round := util.GetTurnCount(), util.GetRoundCount()
	start := rooms.LoadRoom(1)
	require.NotNil(t, start)
	start.Tags = append(start.Tags, "camping")
	start.Resources = append(start.Resources, "firewood") // deadfall: the fire is free (40a2) // so the real character can camp there

	// A real character, logged in on a real connection, who skipped the
	// course after recruiting Tamsin: a company of one in the start room.
	conn, wire := replayConnect(t)
	aria := users.NewUserRecord(7, conn)
	aria.Username = "acctaria"
	aria.Password = "$2a$test"
	aria.Character.Name = "Aria"
	aria.Character.RaceId = 1
	aria.Character.Validate()
	aria.Character.ActionPoints = 1000
	aria.Character.RoomId = -1
	require.NoError(t, users.SaveUser(*aria))
	_, _, err := users.LoginUser(aria, conn)
	require.NoError(t, err)
	void := &rooms.Room{RoomId: -1, Title: "The Void"}
	aria.Character.MarkIronOffered() // these tests are about the tutorial, not the Iron question (Phase 77)
	_, err = usercommands.Start("", aria, void, 0)
	require.NoError(t, err)
	q := aria.GetPrompt().GetNextQuestion()
	require.NotNil(t, q)
	q.Answer("no")
	_, err = usercommands.Start("", aria, void, 0)
	require.NoError(t, err)
	p := progressOf(aria.Character)
	p.Stage = StageCompany
	p.save(aria.Character)
	require.True(t, module.Begin(aria.UserId))
	text(aria.UserId)
	require.Equal(t, 901, template(aria))
	run(aria, "company", "recruit tamsin")
	run(aria, "tutorial", "skip yes")
	require.Equal(t, 1, aria.Character.RoomId)
	aria.Character.Gold = 777
	aria.Character.Experience = 4321
	require.True(t, aria.Character.StoreItem(items.New(30004)))
	assert.Contains(t, run(aria, "camp", ""), "You make camp here.")
	before := snapshotOf(aria)
	require.Equal(t, []string{"Tamsin Reed"}, before.members)
	require.True(t, before.camped)
	require.True(t, before.claimed61)

	// Refused mid-fight.
	aria.Character.Aggro = &characters.Aggro{MobInstanceId: 1}
	assert.Contains(t, run(aria, "tutorial", "replay yes"), "too busy")
	aria.Character.Aggro = nil
	require.Same(t, aria, users.GetByConnectionId(conn))

	// It asks first.
	assert.Contains(t, run(aria, "tutorial", "replay"), "tutorial replay yes")
	require.Same(t, aria, users.GetByConnectionId(conn))

	// replay starts a replay and returns the throwaway on the connection,
	// rested enough to walk.
	replay := func() *users.UserRecord {
		t.Helper()
		real := users.GetByConnectionId(conn)
		require.NotNil(t, real)
		require.False(t, real.IsReplay())
		run(real, "tutorial", "replay yes")
		events.ProcessEvents()
		throwaway := users.GetByConnectionId(conn)
		require.NotNil(t, throwaway, "someone is on the connection")
		require.True(t, throwaway.IsReplay())
		assert.Nil(t, users.GetByUserId(7), "the real character left the world")
		assert.NotNil(t, connections.Get(conn), "the connection stayed")
		throwaway.Character.ActionPoints = 1000
		return throwaway
	}
	// realBack checks the real character is back on the connection,
	// exactly as they were, and the throwaway is gone everywhere.
	realBack := func(throwawayID int) *users.UserRecord {
		t.Helper()
		events.ProcessEvents()
		real := users.GetByConnectionId(conn)
		require.NotNil(t, real)
		require.Equal(t, 7, real.UserId, "the real character is back")
		assert.Equal(t, before, snapshotOf(real), "exactly as they were")
		assert.Zero(t, caps(real), "no graduation cap")
		instanceID, ok := company.InstanceFor(7, 1)
		require.True(t, ok, "Tamsin came back")
		assert.Equal(t, 1, mobs.GetInstance(instanceID).Character.RoomId)
		assertPurged(t, dataDir, throwawayID)
		assert.Nil(t, users.GetByUserId(throwawayID))
		return real
	}

	// 1. Skip. The throwaway is a new, level-1 Aria with nothing, in the
	// course's first room.
	practice := replay()
	got := text(practice.UserId)
	assert.Equal(t, "Aria", practice.Character.Name)
	assert.Equal(t, 1, practice.Character.Level)
	assert.Equal(t, 900, template(practice), "the first lesson")
	assert.NotEqual(t, 900, practice.Character.RoomId, "a copy of its own")
	assert.Contains(t, got, "stage 1 of 8")
	assert.Empty(t, practice.Character.GetAllBackpackItems(), "nothing of the real one's")
	assert.NotEqual(t, 777, practice.Character.Gold)
	members, _ := company.CompanyMembers(practice.UserId)
	assert.Empty(t, members, "no company")
	_, camped := camping.LeaderRest(practice.UserId)
	assert.False(t, camped, "no camp")
	assert.Contains(t, run(practice, "tutorial", "replay yes"), "already replaying", "refused inside a replay")

	// The replay recruits Tamsin and Oswin and camps; the real company and
	// camp are untouched.
	p = progressOf(practice.Character)
	p.Stage = StageCompany
	p.save(practice.Character)
	require.True(t, module.Begin(practice.UserId))
	text(practice.UserId)
	require.Equal(t, 901, template(practice))
	run(practice, "company", "recruit tamsin")
	run(practice, "company", "recruit oswin")
	assert.True(t, company.HasClaimed(practice.UserId, 62))
	assert.False(t, company.HasClaimed(7, 62), "not the real company's claim")
	p = progressOf(practice.Character)
	p.Stage = StageCamp
	p.save(practice.Character)
	require.True(t, module.Begin(practice.UserId))
	text(practice.UserId)
	require.Equal(t, 905, template(practice))
	assert.Contains(t, run(practice, "camp", ""), "You make camp here.")
	realMembers, _ := company.CompanyMembers(7)
	assert.Len(t, realMembers, 1, "the real company is untouched")
	got = run(practice, "tutorial", "skip yes")
	assert.Contains(t, got+text(practice.UserId), "You set the practice character aside.")
	aria = realBack(practice.UserId)
	assert.NotContains(t, wire.String(), "Goodbye", "no goodbye on a hand-off")

	// 2. Graduate, twice in a row, with no leftover claims or progress.
	practice = replay()
	text(practice.UserId)
	assert.Equal(t, 900, template(practice), "from the top again")
	assert.False(t, company.HasClaimed(practice.UserId, 61), "no claims left over")
	p = progressOf(practice.Character)
	p.Stage = StageDeparture
	p.save(practice.Character)
	require.True(t, module.Begin(practice.UserId))
	text(practice.UserId)
	require.Equal(t, 903, template(practice))
	got = run(practice, "gate", "")
	assert.Contains(t, got+text(practice.UserId), "You have finished your training")
	aria = realBack(practice.UserId)

	// 3. Dying ends the course (27c): the death's move out of it, which is
	// all the tutorial sees of one, hands back too.
	practice = replay()
	text(practice.UserId)
	require.NoError(t, rooms.MoveToRoom(practice.UserId, 1))
	events.ProcessEvents()
	aria = realBack(practice.UserId)

	// 4. Quitting (the engine's logoff): the connection closes, the
	// throwaway is purged, and the next login is the real character,
	// unchanged.
	practice = replay()
	text(practice.UserId)
	events.AddToQueue(events.PlayerDespawn{UserId: practice.UserId, RoomId: practice.Character.RoomId, Username: practice.Username, CharacterName: practice.Character.Name})
	events.ProcessEvents()
	assert.Nil(t, connections.Get(conn), "the connection closed")
	assertPurged(t, dataDir, practice.UserId)
	assert.Nil(t, users.GetByUserId(7), "the real character stays out until they log in")
	conn, wire = replayConnect(t)
	relog := func() {
		t.Helper()
		loaded, err := users.LoadUserFile(7)
		require.NoError(t, err)
		loaded.Username = "acctaria"
		loggedIn, _, err := users.LoginUser(loaded, conn)
		require.NoError(t, err)
		events.AddToQueue(events.PlayerSpawn{UserId: 7, ConnectionId: conn, RoomId: loggedIn.Character.RoomId, Username: loggedIn.Username, CharacterName: loggedIn.Character.Name})
		require.NoError(t, rooms.MoveToRoom(7, loggedIn.Character.RoomId, true))
		events.ProcessEvents()
	}
	relog()
	aria = realBack(practice.UserId)

	// 5. A restart mid-replay: the online state is lost, the throwaway's
	// file and saved module state remain; the boot sweep purges them, and
	// the next login is the real character.
	practice = replay()
	text(practice.UserId)
	p = progressOf(practice.Character)
	p.Stage = StageCompany
	p.save(practice.Character)
	require.True(t, module.Begin(practice.UserId))
	run(practice, "company", "recruit tamsin")
	require.NoError(t, users.SaveUser(*practice))
	crashed := practice.UserId
	users.ResetActiveUsers()
	connections.Remove(conn)
	require.FileExists(t, filepath.Join(dataDir, "users", strconv.Itoa(crashed)+".yaml"))
	require.True(t, company.HasClaimed(crashed, 61), "saved company state left behind")
	module.sweepReplays()
	events.ProcessEvents()
	assertPurged(t, dataDir, crashed)
	conn, _ = replayConnect(t)
	relog()
	aria = realBack(crashed)

	assert.Equal(t, turn, util.GetTurnCount(), "the clock never moves")
	assert.Equal(t, round, util.GetRoundCount())
	_ = aria
}

// assertPurged: no module holds state for the user and their file is gone.
func assertPurged(t *testing.T, dataDir string, userID int) {
	t.Helper()
	_, err := os.Stat(filepath.Join(dataDir, "users", strconv.Itoa(userID)+".yaml"))
	assert.True(t, os.IsNotExist(err), "the throwaway's file is gone")
	members, _ := company.CompanyMembers(userID)
	assert.Empty(t, members, "no company")
	assert.False(t, company.HasClaimed(userID, 61), "no claims")
	_, ok := camping.LeaderRest(userID)
	assert.False(t, ok, "no camp")
	assert.Nil(t, module.copies[userID], "no course copies")
	assert.Nil(t, module.fights[userID], "no squad")
}
