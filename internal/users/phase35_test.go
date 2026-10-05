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
	u.Character.StatPoints = 1
	u.Character.Stats.Strength.Training = 1
	u.Character.Health = 3
	u.Character.Mana = 2
	require.NoError(t, SaveUser(*u))
	for _, deferVitals := range []bool{false, true, false} {
		got, err := loadUserById(7, deferVitals)
		require.NoError(t, err)
		assert.Equal(t, 5, got.Character.StatPoints, "1 + (6 - 2) owed exactly once")
		assert.Equal(t, 2, got.Character.StatPointRhythm)
		assert.Equal(t, 1, got.Character.Stats.Strength.Training)
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

// A load while the server still runs the legacy rhythm (for example through
// a saved override) owes nothing yet and must not mark the character, or a
// later switch to the new rhythm would never pay it.
func TestStatPointMigrationWaitsForRhythmChange(t *testing.T) {
	replayDataDir(t)
	g := configs.GetGamePlayConfig()
	g.Progression.StatPointsEveryNLevels = 5
	g.Progression.StatPointsPerLevel = 1
	restore := configs.SetTestGamePlayConfig(g)
	t.Cleanup(func() { restore() })
	u := veteran()
	u.Character.Level = 12
	u.Character.PeakLevel = 12
	u.Character.StatPointRhythm = 0
	u.Character.StatPoints = 2
	require.NoError(t, SaveUser(*u))
	got, err := loadUserById(7, false)
	require.NoError(t, err)
	assert.Equal(t, 2, got.Character.StatPoints)
	assert.Zero(t, got.Character.StatPointRhythm, "legacy rhythm leaves the character unmarked")
	onDisk, err := LoadUserFile(7)
	require.NoError(t, err)
	assert.Zero(t, onDisk.Character.StatPointRhythm)

	g.Progression.StatPointsEveryNLevels = 2
	t.Cleanup(configs.SetTestGamePlayConfig(g))
	for range 2 {
		got, err = loadUserById(7, false)
		require.NoError(t, err)
		assert.Equal(t, 6, got.Character.StatPoints, "2 + (6 - 2) owed once the rhythm changes")
		assert.Equal(t, 2, got.Character.StatPointRhythm)
	}
}

// Points already held count toward the new total: a character that earned a
// point every level before 30g4, or was given points, is not paid again,
// and nothing is taken away.
func TestStatPointMigrationCountsHeldPoints(t *testing.T) {
	replayDataDir(t)
	g := configs.GetGamePlayConfig()
	g.Progression.StatPointsEveryNLevels = 2
	g.Progression.StatPointsPerLevel = 1
	t.Cleanup(configs.SetTestGamePlayConfig(g))
	for _, tc := range []struct {
		name             string
		level, peak      int
		points, training int
		want             int
	}{
		{"every-level veteran keeps all", 20, 20, 4, 15, 4},
		{"partly spent tops up", 12, 12, 1, 3, 3},
		{"lost level uses peak", 10, 12, 0, 2, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := veteran()
			u.Character.Level = tc.level
			u.Character.PeakLevel = tc.peak
			u.Character.StatPointRhythm = 0
			u.Character.StatPoints = tc.points
			u.Character.Stats.Speed.Training = tc.training
			require.NoError(t, SaveUser(*u))
			got, err := loadUserById(7, false)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.Character.StatPoints)
			assert.Equal(t, tc.training, got.Character.Stats.Speed.Training)
			assert.Equal(t, 2, got.Character.StatPointRhythm)
		})
	}
}

// A character created while the legacy rhythm still runs is caught up like
// any older character once the rhythm changes.
func TestNewCharacterUnderLegacyRhythmIsCaughtUp(t *testing.T) {
	replayDataDir(t)
	g := configs.GetGamePlayConfig()
	g.Progression.StatPointsEveryNLevels = 5
	g.Progression.StatPointsPerLevel = 1
	restore := configs.SetTestGamePlayConfig(g)
	t.Cleanup(func() { restore() })
	u := NewUserRecord(7, 70)
	u.Username = "newcomer"
	assert.Zero(t, u.Character.StatPointRhythm, "new characters start unmarked")
	u.Character.Level = 10
	u.Character.StatPoints = 2
	require.NoError(t, SaveUser(*u))
	got, err := loadUserById(7, false)
	require.NoError(t, err)
	assert.Equal(t, 2, got.Character.StatPoints)
	g.Progression.StatPointsEveryNLevels = 2
	t.Cleanup(configs.SetTestGamePlayConfig(g))
	got, err = loadUserById(7, false)
	require.NoError(t, err)
	assert.Equal(t, 5, got.Character.StatPoints, "2 + (5 - 2)")
}
