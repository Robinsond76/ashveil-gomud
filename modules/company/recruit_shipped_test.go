package company

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func shippedWorld(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
}

// shippedModuleConfig reads a key of the shipped overlay through the real
// config path, as PluginConfig.Get does.
func shippedModuleConfig(t *testing.T, key string) any {
	t.Helper()
	data, err := files.ReadFile("files/data-overlays/config.yaml")
	require.NoError(t, err)
	var dataMap map[string]any
	require.NoError(t, yaml.Unmarshal(data, &dataMap))
	overlay := map[string]any{}
	for k, v := range dataMap {
		overlay["Modules.company."+k] = v
	}
	require.NoError(t, configs.AddOverlayOverrides(overlay))
	return configs.Flatten(configs.GetModulesConfig())["company."+key]
}

func globOne(t *testing.T, pattern string) string {
	t.Helper()
	matches, err := filepath.Glob(pattern)
	require.NoError(t, err)
	require.Len(t, matches, 1, pattern)
	return matches[0]
}

func itemFileExists(t *testing.T, world string, itemID int) bool {
	t.Helper()
	prefix := strconv.Itoa(itemID) + "-"
	found := false
	require.NoError(t, filepath.WalkDir(filepath.Join(world, "items"), func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasPrefix(d.Name(), prefix) && strings.HasSuffix(d.Name(), ".yaml") {
			found = true
		}
		return err
	}))
	return found
}

// TestShippedRecruitersResolve pins the shipped recruiters: every room and
// candidate template exists, at least two distinct tutorial candidates
// ship, no candidate can be summoned for free, and no candidate can be
// farmed for gear or gold.
func TestShippedRecruitersResolve(t *testing.T) {
	world := shippedWorld(t)
	recs := parseRecruiters(shippedModuleConfig(t, "Recruiters"))
	require.NotEmpty(t, recs)
	allowed := allowedTemplateIDs(shippedModuleConfig(t, "AllowedCompanionMobIDs"))
	archetypesByTemplate := parseCompanionArchetypes(shippedModuleConfig(t, "CompanionArchetypes"))

	archetypeData, err := os.ReadFile(filepath.Join("..", "archetype", "files", "data-overlays", "config.yaml"))
	require.NoError(t, err)
	var archetypeConfig struct {
		Archetypes []struct {
			ArchetypeId string `yaml:"ArchetypeId"`
		} `yaml:"Archetypes"`
	}
	require.NoError(t, yaml.Unmarshal(archetypeData, &archetypeConfig))
	knownArchetypes := map[string]bool{}
	for _, a := range archetypeConfig.Archetypes {
		knownArchetypes[a.ArchetypeId] = true
	}
	require.NotEmpty(t, knownArchetypes)

	tutorial := map[int]bool{}
	for roomID, rec := range recs {
		globOne(t, filepath.Join(world, "rooms", "*", strconv.Itoa(roomID)+".yaml"))
		require.NotEmpty(t, rec.Candidates, "room %d", roomID)
		for _, c := range rec.Candidates {
			if c.Tutorial {
				tutorial[c.MobTemplateID] = true
			} else {
				assert.Positive(t, c.Price, "%s: a non-tutorial candidate has a price", c.ID)
			}
			_, summonable := allowed[c.MobTemplateID]
			assert.False(t, summonable, "%s: a candidate template must not be summonable for free", c.ID)
			assert.True(t, knownArchetypes[archetypesByTemplate[c.MobTemplateID]], "%s: has a configured archetype", c.ID)

			path := globOne(t, filepath.Join(world, "mobs", "*", strconv.Itoa(c.MobTemplateID)+"-*.yaml"))
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			var mob mobs.Mob
			require.NoError(t, yaml.Unmarshal(data, &mob))
			assert.Equal(t, c.MobTemplateID, int(mob.MobId))
			assert.Equal(t, mob.Filepath(), filepath.ToSlash(path[len(filepath.Join(world, "mobs"))+1:]), "the loader's path rule")
			assert.Zero(t, mob.ItemDropChance, "%s: a death drops nothing", c.ID)
			assert.Empty(t, mob.Character.Items, "%s: carries nothing", c.ID)
			assert.Zero(t, mob.Character.Gold, "%s: has no gold", c.ID)
			assert.False(t, mob.Hostile, c.ID)
			assert.Positive(t, mob.Character.Level, c.ID)
			for _, slot := range characters.AllSlots() {
				if itm := mob.Character.Equipment.Get(slot); itm != nil && itm.ItemId > 0 {
					assert.True(t, itemFileExists(t, world, itm.ItemId), "%s: item %d exists", c.ID, itm.ItemId)
				}
			}
		}
	}
	assert.GreaterOrEqual(t, len(tutorial), 2, "at least two distinct tutorial candidates")
}
