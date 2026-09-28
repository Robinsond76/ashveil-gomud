package users

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func replayDataDir(t *testing.T) string {
	t.Helper()
	mudlog.SetupLogger(nil, "", "", false)
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "users"), 0755))
	previous := configs.Flatten(configs.GetOverrides())
	flat := configs.Flatten(configs.GetOverrides())
	flat["FilePaths.DataFiles"] = dir
	require.NoError(t, configs.RestoreOverrides(flat))
	t.Cleanup(func() { require.NoError(t, configs.RestoreOverrides(previous)) })
	resetUserManager()
	t.Cleanup(resetUserManager)
	return dir
}

func veteran() *UserRecord {
	u := NewUserRecord(7, 70)
	u.Username = "acctaria"
	u.Password = "$2a$hash"
	u.ScreenReader = true
	u.Aliases = map[string]string{"k": "kill"}
	u.Character.Name = "Aria"
	u.Character.RaceId = 1
	u.Character.Level = 9
	u.Character.Experience = 12345
	u.Character.Gold = 900
	u.Character.RoomId = 1
	u.Character.StoreItem(items.Item{ItemId: 10015})
	u.Character.SetMiscData("tutorial-state", "graduated")
	return u
}

// TestNewReplayUserIsAFreshCopy: the replay has the real character's name
// and race at level 1 with nothing of theirs, the player's own settings, a
// reserved id and username, and the replay flag, and it is saved but in
// neither index.
func TestNewReplayUserIsAFreshCopy(t *testing.T) {
	dir := replayDataDir(t)
	real := veteran()

	replay, err := NewReplayUser(real)
	require.NoError(t, err)
	assert.Equal(t, ReplayUserIdBase, replay.UserId)
	assert.Equal(t, "replay:900000000", replay.Username)
	assert.Equal(t, 7, replay.ReplayOf)
	assert.True(t, replay.IsReplay())
	assert.False(t, real.IsReplay())
	assert.Equal(t, "Aria", replay.Character.Name)
	assert.Equal(t, 1, replay.Character.RaceId)
	assert.Equal(t, 1, replay.Character.Level)
	assert.Empty(t, replay.Character.GetAllBackpackItems())
	assert.Nil(t, replay.Character.GetMiscData("tutorial-state"), "no tutorial progress")
	assert.Equal(t, characterNewGold(), replay.Character.Gold, "a new character's purse")
	assert.Equal(t, -1, replay.Character.RoomId, "the Void until placed")
	assert.False(t, replay.HasPlaintextPassword())
	assert.True(t, replay.ScreenReader)
	assert.Equal(t, "kill", replay.Aliases["k"])

	assert.FileExists(t, filepath.Join(dir, "users", "900000000.yaml"))
	_, found := GetUserIndex().FindByUsername(replay.Username)
	assert.False(t, found, "not in the user index")
	id, _ := GetCharacterIndex().Find("Aria")
	assert.NotEqual(t, replay.UserId, id, "not in the character index")

	loaded, err := LoadUserFile(replay.UserId)
	require.NoError(t, err)
	assert.Equal(t, 7, loaded.ReplayOf, "the flag is durable")

	// The next replay takes the next free id, and a replay can't replay.
	second, err := NewReplayUser(real)
	require.NoError(t, err)
	assert.Equal(t, ReplayUserIdBase+1, second.UserId)
	_, err = NewReplayUser(replay)
	assert.Error(t, err)
}

func characterNewGold() int { return NewUserRecord(1, 0).Character.Gold }

// TestOfflineReplayUserIds: replay files left by a restart are listed;
// an online replay and ordinary users aren't.
func TestOfflineReplayUserIds(t *testing.T) {
	replayDataDir(t)
	real := veteran()
	require.NoError(t, SaveUser(*real))
	left, err := NewReplayUser(real)
	require.NoError(t, err)
	online, err := NewReplayUser(real)
	require.NoError(t, err)
	SetTestUser(online)

	assert.Equal(t, []int{left.UserId}, OfflineReplayUserIds())
	assert.Same(t, online, OnlineReplayOf(7))
	assert.Nil(t, OnlineReplayOf(8))
	assert.Nil(t, OnlineReplayOf(0))
}

// TestRemoveUserFile refuses an online user and is idempotent.
func TestRemoveUserFile(t *testing.T) {
	dir := replayDataDir(t)
	real := veteran()
	replay, err := NewReplayUser(real)
	require.NoError(t, err)
	path := filepath.Join(dir, "users", "900000000.yaml")

	SetTestUser(replay)
	assert.Error(t, RemoveUserFile(replay.UserId))
	assert.FileExists(t, path)

	RemoveTestUser(replay.UserId)
	require.NoError(t, RemoveUserFile(replay.UserId))
	assert.NoFileExists(t, path)
	require.NoError(t, RemoveUserFile(replay.UserId), "twice is fine")
}
