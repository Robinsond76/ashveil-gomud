package mobs

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/windup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// TestShippedForestOgre (Phase 30d2): the Dark Forest's ogre loads as a
// solitary hostile ogre with a great club that winds up Crushing Blow,
// and road room 530 spawns one.
func TestShippedForestOgre(t *testing.T) {
	loadShippedPronounData(t)
	spec := GetMobSpec(85)
	require.NotNil(t, spec)
	assert.Equal(t, "forest ogre", spec.Character.Name)
	assert.Equal(t, 22, spec.Character.RaceId)
	race := races.GetRace(22)
	require.NotNil(t, race)
	assert.Equal(t, "ogre", race.Name)
	assert.True(t, spec.Hostile)
	assert.True(t, spec.Solitary, "it stands alone")
	assert.Equal(t, "he", spec.Character.Pronouns)
	assert.Equal(t, 10022, spec.Character.Equipment.Weapon.ItemId, "the ogre's great club")
	assert.Equal(t, map[string]int{"crushing-blow": 35}, spec.WindUps)

	data, err := os.ReadFile(filepath.Join("_datafiles", "world", "default", "rooms", "dark_forest", "530.yaml"))
	require.NoError(t, err)
	var r struct {
		SpawnInfo []struct {
			MobId int `yaml:"mobid"`
		} `yaml:"spawninfo"`
	}
	require.NoError(t, yaml.Unmarshal(data, &r))
	var ids []int
	for _, s := range r.SpawnInfo {
		ids = append(ids, s.MobId)
	}
	assert.True(t, slices.Contains(ids, 85), "room 530 spawns the ogre: %v", ids)
}

// TestShippedWindUpsAreRegistered: every shipped mob's windups name a
// registered ability with a chance in 1-100.
func TestShippedWindUpsAreRegistered(t *testing.T) {
	loadShippedPronounData(t)
	items.LoadDataFiles()
	found := 0
	for _, m := range GetAllMobInfo() {
		for id, chance := range m.WindUps {
			found++
			_, ok := windup.Get(id)
			assert.True(t, ok, "mob %d: unknown wind-up %q", m.MobId, id)
			assert.True(t, chance >= 1 && chance <= 100, "mob %d: %s chance %d", m.MobId, id, chance)
			// Review fix: a foe that shoots never swings a plain attack, so
			// it could not wind up (only DefaultAttack does).
			if w := m.Character.Equipment.Weapon; w.ItemId > 0 {
				assert.NotEqual(t, items.Shooting, w.GetSpec().Subtype, "mob %d: a shooter can't wind up", m.MobId)
			}
		}
	}
	assert.Positive(t, found, "at least one shipped wind-up")
}
