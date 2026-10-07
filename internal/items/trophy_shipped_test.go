package items

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yamlv2 "gopkg.in/yaml.v2"
)

// shippedTrophies are the world's trophy items, ascending by id.
func shippedTrophies(t *testing.T) []*ItemSpec {
	t.Helper()
	var out []*ItemSpec
	for _, spec := range shippedSpecs(t) {
		if spec.Trophy != nil {
			out = append(out, spec)
		}
	}
	require.GreaterOrEqual(t, len(out), 5, "the test world ships trophies")
	return out
}

// Phase 71: every shipped trophy names races that exist and that ordinary
// foes spawn in the world (a trophy nobody can hunt is no trophy), and is a
// cheap light commodity: it is hunted, not bought, and its worth is the power.
func TestShippedTrophiesAreHuntableAndCheap(t *testing.T) {
	raceNames := map[int]string{}
	raceFiles, err := filepath.Glob(filepath.Join(dataRoot(), "races", "*.yaml"))
	require.NoError(t, err)
	for _, f := range raceFiles {
		data, err := os.ReadFile(f)
		require.NoError(t, err)
		var r struct {
			ID   int    `yaml:"raceid"`
			Name string `yaml:"name"`
		}
		require.NoError(t, yamlv2.Unmarshal(data, &r))
		raceNames[r.ID] = strings.ToLower(r.Name)
	}
	known := map[string]bool{}
	for _, n := range raceNames {
		known[n] = true
	}
	mobRace := map[int]string{}
	require.NoError(t, filepath.Walk(filepath.Join(dataRoot(), "mobs"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return err
		}
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		var m struct {
			MobId     int  `yaml:"mobid"`
			Boss      bool `yaml:"boss"`
			Character struct {
				RaceId int `yaml:"raceid"`
			} `yaml:"character"`
		}
		require.NoError(t, yamlv2.Unmarshal(data, &m))
		if !m.Boss {
			mobRace[m.MobId] = raceNames[m.Character.RaceId]
		}
		return nil
	}))
	spawning := map[string]int{}
	roomFiles, err := filepath.Glob(filepath.Join(dataRoot(), "rooms", "*", "*.yaml"))
	require.NoError(t, err)
	for _, f := range roomFiles {
		data, err := os.ReadFile(f)
		require.NoError(t, err)
		var r struct {
			Zone      string `yaml:"zone"`
			SpawnInfo []struct {
				MobId int `yaml:"mobid"`
			} `yaml:"spawninfo"`
		}
		require.NoError(t, yamlv2.Unmarshal(data, &r))
		if r.Zone == "Training" {
			continue
		}
		for _, s := range r.SpawnInfo {
			if race, ok := mobRace[s.MobId]; ok {
				spawning[race]++
			}
		}
	}

	parts := map[string]bool{}
	for _, spec := range shippedTrophies(t) {
		parts[spec.Trophy.Part] = true
		assert.LessOrEqual(t, spec.Value, 15, "%s is hunted, not bought: it is worth a few coins at most", spec.Name)
		assert.LessOrEqual(t, spec.Weight, 300, "%s is light", spec.Name)
		assert.Empty(t, spec.Goods, "%s is an enchanting trophy, not a trade good", spec.Name)
		found := 0
		for _, race := range spec.Trophy.Races {
			assert.True(t, known[race], "%s names race %q, which does not exist", spec.Name, race)
			found += spawning[race]
		}
		assert.Positive(t, found, "%s: no ordinary foe of %v spawns outside the Training zone", spec.Name, spec.Trophy.Races)
	}
	assert.True(t, parts[TrophyHeart] && parts[TrophyHide] && parts[TrophyAsh], "hearts, hides and ash all exist")
}

// Even three of every shipped trophy on one wearer (more than a body holds)
// stay inside each effect's aggregate cap, which is below the effect's cap,
// so enchants alone never reach what a relic may.
func TestEnchantsAloneNeverReachAnEffectsCap(t *testing.T) {
	total := map[string]int{}
	for _, spec := range shippedTrophies(t) {
		for k, v := range spec.Trophy.Effects {
			total[k] += 3 * v
		}
	}
	for k, v := range total {
		e, ok := classes.GearEffectFor(k)
		require.True(t, ok, k)
		held := min(v, TrophyAggregateCap(e))
		assert.Less(t, held, e.Max, "enchants alone never reach the cap of %s", k)
	}
}

// The test world ships an enchanter where a player will find one: beside the
// armorer in Frostfang and in the test area's armory.
func TestShippedEnchantersStandInTownAndTheTestArea(t *testing.T) {
	mobs := map[int]string{} // mob id -> zone
	require.NoError(t, filepath.Walk(filepath.Join(dataRoot(), "mobs"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return err
		}
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		var m struct {
			MobId     int    `yaml:"mobid"`
			Zone      string `yaml:"zone"`
			Character struct {
				Adjectives []string `yaml:"adjectives"`
			} `yaml:"character"`
		}
		require.NoError(t, yamlv2.Unmarshal(data, &m))
		for _, a := range m.Character.Adjectives {
			if a == "enchanter" {
				mobs[m.MobId] = m.Zone
			}
		}
		return nil
	}))
	require.NotEmpty(t, mobs)
	zones := map[string]bool{}
	roomFiles, err := filepath.Glob(filepath.Join(dataRoot(), "rooms", "*", "*.yaml"))
	require.NoError(t, err)
	for _, f := range roomFiles {
		data, err := os.ReadFile(f)
		require.NoError(t, err)
		var r struct {
			Zone      string `yaml:"zone"`
			SpawnInfo []struct {
				MobId int `yaml:"mobid"`
			} `yaml:"spawninfo"`
		}
		require.NoError(t, yamlv2.Unmarshal(data, &r))
		for _, s := range r.SpawnInfo {
			if zone, ok := mobs[s.MobId]; ok {
				assert.Equal(t, zone, r.Zone, "an enchanter spawns in its own zone")
				zones[r.Zone] = true
			}
		}
	}
	assert.True(t, zones["Frostfang"], "an enchanter in Frostfang")
	assert.True(t, zones["Test Area"], "an enchanter in the test area")
}
