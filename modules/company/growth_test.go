package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSummonAndRestoreSpawnWithGrowth: Phase 33h1's weights reach every
// spawn: a new recruit gets its configured archetype's weights, and a
// restored one its archetype's plus its saved focus.
func TestSummonAndRestoreSpawnWithGrowth(t *testing.T) {
	archetypes.SetProvider(growthArchetypes{})
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	runtime := &fakeRuntime{resolved: map[string]int{"training dummy": 58}, nextInstanceID: 101}
	module := newTestModule(*domain.NewRegistry(), runtime)
	_, err := module.summon(7, 12, "training dummy")
	require.NoError(t, err)
	require.Len(t, runtime.spawnedGrowth, 1)
	assert.Equal(t, warriorGrowth, runtime.spawnedGrowth[0], "the dummy is a configured warrior")

	restored := &fakeRuntime{nextInstanceID: 201}
	module = newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{
			{ID: 1, MobTemplateID: 58, Archetype: "cleric", GrowthFocus: "speed"},
			{ID: 2, MobTemplateID: 58},
		}},
	}}, restored)
	require.NoError(t, module.restoreForLeader(7, 12))
	require.Len(t, restored.spawnedGrowth, 2)
	assert.Equal(t, domain.GrowthWeights{2, 2, 2, 3, 3, 0}, restored.spawnedGrowth[0])
	assert.Equal(t, domain.EvenGrowth, restored.spawnedGrowth[1], "no archetype: the even spread")
}

// TestRetrainCompanionFindsTheTrackedCompanion: the growth provider retrains
// only an instance the module tracks.
func TestRetrainCompanionFindsTheTrackedCompanion(t *testing.T) {
	archetypes.SetProvider(growthArchetypes{})
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	runtime := &fakeRuntime{live: map[int]bool{101: true, 102: true}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58, Archetype: "warrior", GrowthFocus: "perception"}}},
	}}, runtime)
	module.setInstance(7, 1, 101)
	assert.True(t, module.RetrainCompanion(101))
	assert.Equal(t, domain.GrowthWeights{4, 2, 0, 3, 0, 3}, runtime.retrained[101])
	assert.False(t, module.RetrainCompanion(102), "an untracked mob is not the company's")
}
