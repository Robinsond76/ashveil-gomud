package mobs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/coordination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// TestShippedCoordinatedGroups (Phase 33i2): each of the first three
// coordination tiers ships a group with a healer and a guardian, spawned
// together in one room, at levels that put the group in that tier; and the
// unwoundable templates say so.
func TestShippedCoordinatedGroups(t *testing.T) {
	loadShippedPronounData(t)
	cases := []struct {
		tier  coordination.Tier
		room  string
		group string
		ids   []int // every spawn in the room, by template
	}{
		{coordination.Rabble, "frost_lake/310", "lake-poachers", []int{87, 87, 88, 89}},
		{coordination.Band, "frostfang_slums/489", "slum-ruffians", []int{90, 91, 29}},
		{coordination.Drilled, "catacombs/113", "catacomb-denizen", []int{92, 93, 18}},
	}
	for _, c := range cases {
		data, err := os.ReadFile(filepath.Join("_datafiles", "world", "default", "rooms", c.room+".yaml"))
		require.NoError(t, err)
		var r struct {
			SpawnInfo []struct {
				MobId int `yaml:"mobid"`
			} `yaml:"spawninfo"`
		}
		require.NoError(t, yaml.Unmarshal(data, &r))
		var got, levels []int
		roles := map[string]bool{}
		for _, s := range r.SpawnInfo {
			got = append(got, s.MobId)
			spec := GetMobSpec(MobId(s.MobId))
			require.NotNil(t, spec, "template %d", s.MobId)
			require.NoError(t, spec.Validate())
			levels = append(levels, spec.Character.Level)
			roles[spec.EnemyRole()] = true
			if spec.EnemyRole() == "healer" {
				assert.True(t, spec.Character.HasSpell("heal"), "%s knows Minor Heal", spec.Character.Name)
			}
		}
		assert.ElementsMatch(t, c.ids, got, c.room)
		assert.True(t, roles["healer"] && roles["guardian"], "%s has a healer and a guardian", c.room)
		assert.Equal(t, c.tier, coordination.Of(levels, nil), "%s fights as %s", c.room, coordination.SpecOf(c.tier).Word)
	}
	for _, id := range []MobId{15, 32, 92} {
		spec := GetMobSpec(id)
		require.NotNil(t, spec)
		assert.False(t, spec.TakesWounds(), "%s takes no wounds", spec.Character.Name)
	}
}
