package hooks

import (
	"bytes"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// handOffWorld is a data dir with a users folder and one room, the leave,
// hand-off, and purge listeners registered, and the spawns recorded.
type handOffWorld struct {
	dir    string
	spawns []events.PlayerSpawn
}

func newHandOffWorld(t *testing.T) *handOffWorld {
	t.Helper()
	w := &handOffWorld{dir: t.TempDir()}
	require.NoError(t, os.MkdirAll(filepath.Join(w.dir, "users"), 0755))
	previous := configs.Flatten(configs.GetOverrides())
	flat := configs.Flatten(configs.GetOverrides())
	flat["FilePaths.DataFiles"] = w.dir
	require.NoError(t, configs.RestoreOverrides(flat))
	t.Cleanup(func() { require.NoError(t, configs.RestoreOverrides(previous)) })

	room := &rooms.Room{RoomId: 932001, Title: "Hall", Zone: "handoff"}
	rooms.SetTestRoom(room)
	t.Cleanup(func() { rooms.RemoveTestRoom(room.RoomId) })

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)

	ids := []events.ListenerId{
		events.RegisterListener(events.PlayerDespawn{}, HandleLeave, events.Last),
		events.RegisterListener(events.UserHandOff{}, HandleUserHandOff),
		events.RegisterListener(events.UserPurged{}, HandlePurge, events.Last),
		events.RegisterListener(events.PlayerSpawn{}, func(e events.Event) events.ListenerReturn {
			w.spawns = append(w.spawns, e.(events.PlayerSpawn))
			return events.Continue
		}),
	}
	t.Cleanup(func() {
		events.UnregisterListener(events.PlayerDespawn{}, ids[0])
		events.UnregisterListener(events.UserHandOff{}, ids[1])
		events.UnregisterListener(events.UserPurged{}, ids[2])
		events.UnregisterListener(events.PlayerSpawn{}, ids[3])
	})
	return w
}

// wire is the client end of a real (piped) connection, drained so the
// server never blocks writing to it.
type wire struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *wire) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func connect(t *testing.T) (connections.ConnectionId, *wire) {
	t.Helper()
	server, client := net.Pipe()
	cd := connections.Add(server, nil)
	cd.SetState(connections.LoggedIn)
	out := &wire{}
	go func() {
		chunk := make([]byte, 1024)
		for {
			n, err := client.Read(chunk)
			out.mu.Lock()
			out.buf.Write(chunk[:n])
			out.mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	t.Cleanup(func() {
		connections.Remove(cd.ConnectionId())
		_, _ = io.WriteString(client, "")
		client.Close()
	})
	return cd.ConnectionId(), out
}

func saveUser(t *testing.T, id int, name string) *users.UserRecord {
	t.Helper()
	u := users.NewUserRecord(id, 0)
	u.Username = "acct" + name
	u.Password = "$2a$test"
	u.Character.Name = name
	u.Character.RoomId = 932001
	require.NoError(t, users.SaveUser(*u))
	return u
}

// TestHandOffMovesAConnectionToAnotherUser: the first user leaves with a
// hand-off, through the engine's own leave handling, and the second is
// logged in on the same, still open, connection and spawned in their room.
func TestHandOffMovesAConnectionToAnotherUser(t *testing.T) {
	w := newHandOffWorld(t)
	connID, _ := connect(t)

	first := saveUser(t, 7, "Aria")
	_, _, err := users.LoginUser(first, connID)
	require.NoError(t, err)
	require.NoError(t, rooms.MoveToRoom(first.UserId, 932001, true))
	saveUser(t, 900000001, "Aria")

	events.AddToQueue(events.PlayerDespawn{UserId: 7, RoomId: 932001, HandOff: true})
	events.AddToQueue(events.UserHandOff{ConnectionId: connID, FromUserId: 7, ToUserId: 900000001})
	events.ProcessEvents()

	assert.Nil(t, users.GetByUserId(7), "the first user left the world")
	assert.NotNil(t, connections.Get(connID), "the connection stays open")
	now := users.GetByConnectionId(connID)
	require.NotNil(t, now, "someone is on the connection")
	assert.Equal(t, 900000001, now.UserId)
	require.Len(t, w.spawns, 1)
	assert.Equal(t, 900000001, w.spawns[0].UserId)
	assert.Equal(t, connID, w.spawns[0].ConnectionId)
	assert.Equal(t, []int{900000001}, rooms.LoadRoom(932001).GetPlayers(), "placed in their room")

	// The first user's file was saved on leaving.
	saved, err := users.LoadUserFile(7)
	require.NoError(t, err)
	assert.Equal(t, 932001, saved.Character.RoomId)
}

// TestHandOffRefusedClosesTheConnection: when the next user is already
// online elsewhere, the connection is closed as a logout would, never left
// with no one on it.
func TestHandOffRefusedClosesTheConnection(t *testing.T) {
	w := newHandOffWorld(t)
	connID, _ := connect(t)
	otherID, _ := connect(t)

	first := saveUser(t, 7, "Aria")
	_, _, err := users.LoginUser(first, connID)
	require.NoError(t, err)
	second := saveUser(t, 8, "Bex")
	_, _, err = users.LoginUser(second, otherID)
	require.NoError(t, err)

	events.AddToQueue(events.PlayerDespawn{UserId: 7, RoomId: 932001, HandOff: true})
	events.AddToQueue(events.UserHandOff{ConnectionId: connID, FromUserId: 7, ToUserId: 8})
	events.ProcessEvents()

	assert.Nil(t, connections.Get(connID), "hung up")
	assert.Equal(t, otherID, users.GetByUserId(8).ConnectionId(), "the other login is untouched")
	assert.Empty(t, w.spawns)
}

// TestHandOffAfterTheConnectionIsGone: the player hung up in between; the
// next user simply stays offline.
func TestHandOffAfterTheConnectionIsGone(t *testing.T) {
	w := newHandOffWorld(t)
	connID, _ := connect(t)
	saveUser(t, 8, "Bex")
	connections.Remove(connID)

	events.AddToQueue(events.UserHandOff{ConnectionId: connID, FromUserId: 7, ToUserId: 8})
	events.ProcessEvents()

	assert.Nil(t, users.GetByUserId(8))
	assert.Empty(t, w.spawns)
}

// TestPurgeRemovesTheUserFile: the purge's final listener removes an
// offline user's file, twice without error, and never an online user's.
func TestPurgeRemovesTheUserFile(t *testing.T) {
	w := newHandOffWorld(t)
	saveUser(t, 900000001, "Aria")
	path := filepath.Join(w.dir, "users", "900000001.yaml")
	require.FileExists(t, path)

	events.AddToQueue(events.UserPurged{UserId: 900000001})
	events.AddToQueue(events.UserPurged{UserId: 900000001})
	events.ProcessEvents()
	assert.NoFileExists(t, path)

	connID, _ := connect(t)
	online := saveUser(t, 8, "Bex")
	_, _, err := users.LoginUser(online, connID)
	require.NoError(t, err)
	events.AddToQueue(events.UserPurged{UserId: 8})
	events.ProcessEvents()
	assert.FileExists(t, filepath.Join(w.dir, "users", "8.yaml"), "an online user is never removed")
}
