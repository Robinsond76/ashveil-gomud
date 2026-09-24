package mobs

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withTemplate installs a mob template for one test.
func withTemplate(t *testing.T, m *Mob) {
	t.Helper()
	prev, had := mobs[int(m.MobId)]
	mobs[int(m.MobId)] = m
	t.Cleanup(func() {
		if had {
			mobs[int(m.MobId)] = prev
		} else {
			delete(mobs, int(m.MobId))
		}
	})
}

// Regression (Ashveil Phase 22b review): a spawned mob shared the
// template's Items backing array, so removing an item from a live mob
// rewrote the template, and later spawns minted the wrong items.
func TestNewMobByIdCopiesTemplateItems(t *testing.T) {
	template := &Mob{MobId: 990001, Character: *characters.New()}
	template.Character.Level = 1
	template.Character.Items = []items.Item{{ItemId: 1}, {ItemId: 2}}
	withTemplate(t, template)

	mob := NewMobById(990001, 1)
	require.NotNil(t, mob)
	t.Cleanup(func() { DestroyInstance(mob.InstanceId) })
	require.True(t, mob.Character.RemoveItem(mob.Character.Items[0]))

	assert.Equal(t, []int{1, 2}, []int{template.Character.Items[0].ItemId, template.Character.Items[1].ItemId}, "the template is untouched")
	next := NewMobById(990001, 1)
	require.NotNil(t, next)
	t.Cleanup(func() { DestroyInstance(next.InstanceId) })
	require.Len(t, next.Character.Items, 2)
	assert.Equal(t, 1, next.Character.Items[0].ItemId)
}

func TestNewMobByIdNoEliteNeverRollsElite(t *testing.T) {
	template := &Mob{MobId: 990002, EliteChance: 100, Character: *characters.New()}
	template.Character.Level = 5
	withTemplate(t, template)

	elite := NewMobById(990002, 1)
	require.NotNil(t, elite)
	t.Cleanup(func() { DestroyInstance(elite.InstanceId) })
	assert.True(t, elite.IsElite, "the ordinary spawn rolls elite at 100%")

	for _, level := range []int{0, 3} {
		mob := NewMobByIdNoElite(990002, 1, level)
		require.NotNil(t, mob)
		t.Cleanup(func() { DestroyInstance(mob.InstanceId) })
		assert.False(t, mob.IsElite)
		want := level
		if want == 0 {
			want = 5
		}
		assert.Equal(t, want, mob.Character.Level)
	}
}
