package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/encounters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const masteryTemplate = 98881

// slay kills one more of a template through the real death path.
func slay(t *testing.T, w *dropWorld, instance int) {
	t.Helper()
	foe := w.foe(instance, func(m *mobs.Mob) { m.MobId = masteryTemplate })
	t.Cleanup(func() { mobs.RemoveTestInstance(foe.InstanceId) })
	_, err := Suicide("", foe, w.room)
	require.NoError(t, err)
}

func masterySpec(t *testing.T, boss bool) {
	t.Helper()
	spec := &mobs.Mob{MobId: masteryTemplate, Boss: boss, Zone: dropZone}
	spec.Character.Name = "big rat"
	mobs.SetTestSpec(spec)
	t.Cleanup(func() { mobs.RemoveTestSpec(masteryTemplate) })
}

// Phase 85: the kill that teaches a kind's habits (the 6th) writes one
// "mastered" deed through the real death path; the 5th and 7th write none.
func TestTheKillThatTeachesAKindsHabitsWritesOneMasteredDeed(t *testing.T) {
	chronicle.SetProvider(chronicle.NewMemory())
	t.Cleanup(func() { chronicle.SetProvider(nil) })
	masterySpec(t, false)
	w := newDropWorld(t, 1, encounters.Band{Low: 5, High: 7})
	leader := w.users[0]
	mastered := func() []chronicle.Entry {
		return chronicle.Query(leader.UserId, chronicle.Filter{Kinds: []chronicle.Kind{chronicle.Mastered}})
	}

	for i := 0; i < 5; i++ {
		slay(t, w, 98890+i)
	}
	assert.Empty(t, mastered(), "five kills teach lore and defences only")

	slay(t, w, 98896)
	deeds := mastered()
	require.Len(t, deeds, 1, "the sixth kill teaches its habits")
	assert.Equal(t, "mob:98881", deeds[0].Ref)
	assert.Equal(t, "big rat", deeds[0].Subject)
	assert.Equal(t, dropZone, deeds[0].Zone)
	assert.Equal(t, "The company learned the habits of the big rat at "+w.room.Title+".", chronicle.Prose(deeds[0]))

	slay(t, w, 98897)
	assert.Len(t, mastered(), 1, "the seventh kill adds nothing")
	assert.Equal(t, 1, chronicle.Total(leader.UserId, chronicle.Mastered))
}

// A boss teaches faster: its habits come at the third kill.
func TestABossKindIsMasteredAtItsThirdKill(t *testing.T) {
	chronicle.SetProvider(chronicle.NewMemory())
	t.Cleanup(func() { chronicle.SetProvider(nil) })
	masterySpec(t, true)
	w := newDropWorld(t, 1, encounters.Band{Low: 5, High: 7})
	leader := w.users[0]

	slay(t, w, 98890)
	slay(t, w, 98891)
	assert.Zero(t, chronicle.Total(leader.UserId, chronicle.Mastered))
	slay(t, w, 98892)
	assert.Equal(t, 1, chronicle.Total(leader.UserId, chronicle.Mastered))
}

// Training kills teach nothing, so they write nothing.
func TestTrainingKillsWriteNoMasteredDeed(t *testing.T) {
	chronicle.SetProvider(chronicle.NewMemory())
	t.Cleanup(func() { chronicle.SetProvider(nil) })
	masterySpec(t, true)
	w := newDropWorld(t, 1, encounters.Band{Low: 5, High: 7})
	w.room.Zone = "Training"
	t.Cleanup(func() { w.room.Zone = dropZone })
	for i := 0; i < 3; i++ {
		slay(t, w, 98890+i)
	}
	assert.Zero(t, chronicle.Total(w.users[0].UserId, chronicle.Mastered))
}
