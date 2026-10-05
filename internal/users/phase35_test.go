package users

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestStatPointMigrationReloadAndCopyover(t *testing.T) {
	replayDataDir(t)
	g := configs.GetGamePlayConfig()
	g.Progression.StatPointsEveryNLevels = 2
	g.Progression.StatPointsPerLevel = 1
	t.Cleanup(configs.SetTestGamePlayConfig(g))
	u := veteran()
	u.Character.Level = 11
	u.Character.PeakLevel = 12
	u.Character.StatPointRhythm = 0
	u.Character.StatPoints = 3
	u.Character.Stats.Strength.Training = 8
	u.Character.Health = 3
	u.Character.Mana = 2
	require.NoError(t, SaveUser(*u))
	for _, deferVitals := range []bool{false, true, false} {
		got, err := loadUserById(7, deferVitals)
		require.NoError(t, err)
		assert.Equal(t, 7, got.Character.StatPoints, "6 - 2 owed exactly once")
		assert.Equal(t, 2, got.Character.StatPointRhythm)
		assert.Equal(t, 8, got.Character.Stats.Strength.Training)
		assert.Equal(t, 3, got.Character.Health)
		assert.Equal(t, 2, got.Character.Mana)
	}
	replay, err := NewReplayUser(u)
	require.NoError(t, err)
	got, err := LoadUserFile(replay.UserId)
	require.NoError(t, err)
	assert.Zero(t, got.Character.StatPoints)
	assert.Equal(t, 2, got.Character.StatPointRhythm)
}
func TestNewCharacterStatRhythmDoesNotMigrateEarnedPoints(t *testing.T) {
	replayDataDir(t)
	u := veteran()
	u.Character.Level = 12
	u.Character.PeakLevel = 12
	require.NoError(t, SaveUser(*u))
	got, err := LoadUserFile(7)
	require.NoError(t, err)
	assert.Equal(t, u.Character.StatPoints, got.Character.StatPoints)
}

func TestStatPointMigrationNormalLoadAndFailedSave(t *testing.T) {
	dir := replayDataDir(t)
	g := configs.GetGamePlayConfig()
	g.Progression.StatPointsEveryNLevels = 2
	g.Progression.StatPointsPerLevel = 1
	t.Cleanup(configs.SetTestGamePlayConfig(g))
	u := veteran()
	u.Character.Level = 12
	u.Character.PeakLevel = 0
	u.Character.StatPointRhythm = 0
	u.Character.StatPoints = 2
	require.NoError(t, SaveUser(*u))
	priorIndex := userIndex
	t.Cleanup(func() { userIndex = priorIndex })
	idx := InitUserIndex()
	require.NoError(t, idx.Create())
	require.NoError(t, idx.AddUser(u.UserId, u.Username))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "users", "7.yaml.new"), 0700))
	_, err := LoadUser(u.Username)
	require.Error(t, err, "a failed migration cannot enter play")
	assert.Error(t, migrateStatPointRhythm(u))
	assert.Equal(t, 2, u.Character.StatPoints)
	assert.Zero(t, u.Character.StatPointRhythm, "failed atomic save rolls back both")
	require.NoError(t, os.Remove(filepath.Join(dir, "users", "7.yaml.new")))
	got, err := LoadUser(u.Username)
	require.NoError(t, err)
	assert.Equal(t, 6, got.Character.StatPoints, "legacy missing peak uses current level")
	got, err = LoadUser(u.Username)
	require.NoError(t, err)
	assert.Equal(t, 6, got.Character.StatPoints)
}
