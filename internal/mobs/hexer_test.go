package mobs

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// TestShippedGoblinHexer (Phase 30d1): the Dark Forest's enemy caster
// loads as a hostile goblin that knows Withering Hex and casts it in a
// fight, and the forest imps' rooms 402 and 531 spawn one beside the imp.
func TestShippedGoblinHexer(t *testing.T) {
	loadShippedPronounData(t)
	spec := GetMobSpec(70)
	require.NotNil(t, spec)
	assert.Equal(t, "goblin hexer", spec.Character.Name)
	assert.Equal(t, 5, spec.Character.RaceId, "a goblin: it aims by casters")
	assert.True(t, spec.Hostile)
	assert.Contains(t, spec.Character.SpellBook, "hex")
	assert.Equal(t, []string{"cast hex"}, spec.CombatCommands)
	assert.Greater(t, spec.ActivityLevel, 0)
	assert.Equal(t, []string{"forest-creature"}, spec.Groups, "grouped with the imps")

	for _, room := range []string{"402", "531"} {
		data, err := os.ReadFile(filepath.Join("_datafiles", "world", "default", "rooms", "dark_forest", room+".yaml"))
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
		assert.True(t, slices.Contains(ids, 33) && slices.Contains(ids, 70), "room %s spawns an imp and a hexer: %v", room, ids)
	}
}
