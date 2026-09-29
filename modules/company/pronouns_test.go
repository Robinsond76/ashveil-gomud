package company

import (
	"fmt"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGeneratedRecruitPronounsRestore(t *testing.T) {
	for _, id := range []int{1, 2} {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			b := newBrawl(t)
			// The durable identity is the value saved by generated recruitment.
			require.NoError(t, module.registry.SetIdentity(7, id, domain.Identity{Name: "Hild Marrow", Description: "A generated recruit."}))
			nativeRuntime{}.Detach(7, b.companion(id).InstanceId)
			module.clearInstance(7, id)
			require.NoError(t, module.restoreForLeader(7, b.road.RoomId))
			assert.Equal(t, "they", b.companion(id).Character.Pronouns)
			require.NoError(t, module.save())
			// Reload through the real plugin store and native PlayerSpawn seam.
			nativeRuntime{}.Detach(7, b.companion(id).InstanceId)
			module.clearInstance(7, id)
			module.registry = *domain.NewRegistry()
			require.NoError(t, module.store.Load(&module.registry))
			module.onPlayerSpawn(events.PlayerSpawn{UserId: 7, RoomId: b.road.RoomId})
			restored := b.companion(id)
			assert.Equal(t, "Hild Marrow", restored.Character.Name)
			assert.Equal(t, "they", restored.Character.Pronouns)
			module.onMobDeath(events.MobDeath{InstanceId: restored.InstanceId, Level: restored.Character.Level})
			nativeRuntime{}.Detach(7, restored.InstanceId)
			result, err := module.ResurrectCompanion(7, fmt.Sprintf("#%d", id), b.road.RoomId)
			require.NoError(t, err)
			require.True(t, result.Spawned)
			assert.Equal(t, "they", b.companion(id).Character.Pronouns)
		})
	}
}

func TestAuthoredRecruitPronounsRestore(t *testing.T) {
	b := newBrawl(t)
	for id, want := range map[int]string{1: "she", 2: "he"} {
		original := b.companion(id)
		assert.Equal(t, want, original.Character.Pronouns)
		require.NoError(t, module.save())
		nativeRuntime{}.Detach(7, original.InstanceId)
		module.clearInstance(7, id)
		require.NoError(t, module.store.Load(&module.registry))
		module.onPlayerSpawn(events.PlayerSpawn{UserId: 7, RoomId: b.road.RoomId})
		restored := b.companion(id)
		assert.Equal(t, want, restored.Character.Pronouns)
		assert.Equal(t, original.Character.Name, restored.Character.Name)
		assert.NotEqual(t, original.InstanceId, restored.InstanceId)
		assert.NotNil(t, mobs.GetInstance(restored.InstanceId))
	}
}
