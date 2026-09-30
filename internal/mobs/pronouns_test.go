package mobs

import (
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadShippedPronounData(t *testing.T) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
	t.Chdir(root)
	races.LoadDataFiles()
	LoadDataFiles()
}

func TestShippedPronounDefaults(t *testing.T) {
	loadShippedPronounData(t)

	for _, raceID := range []int{0, 6, 7, 9, 10, 11, 12, 13, 14, 16, 17, 18, 19, 20, 21} {
		race := races.GetRace(raceID)
		require.NotNilf(t, race, "race %d", raceID)
		assert.Equalf(t, "it", race.DefaultPronouns, "race %d", raceID)
	}
	for _, raceID := range []int{1, 2, 4, 5, 8, 15, 22} {
		race := races.GetRace(raceID)
		require.NotNilf(t, race, "race %d", raceID)
		assert.Emptyf(t, race.DefaultPronouns, "race %d", raceID)
	}

	for _, tc := range []struct {
		id   MobId
		want string
	}{
		{61, "she"}, {62, "he"}, {63, "he"}, {64, "she"}, {65, "she"}, {66, "she"}, {69, "he"},
	} {
		t.Run(strconv.Itoa(int(tc.id)), func(t *testing.T) {
			template := GetMobSpec(tc.id)
			require.NotNil(t, template)
			assert.Equal(t, tc.want, template.Character.Pronouns)
			spawned := NewMobByIdNoElite(tc.id, 1, 0)
			require.NotNil(t, spawned)
			t.Cleanup(func() { DestroyInstance(spawned.InstanceId) })
			assert.Equal(t, tc.want, spawned.Character.Pronouns)
		})
	}

	assert.Equal(t, "it", GetMobSpec(60).Character.CombatPronouns().Subject, "reptile uses its race default")
}
