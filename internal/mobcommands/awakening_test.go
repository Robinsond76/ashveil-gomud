package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/encounters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	awakenRelicID = 98871
	awakenBossMob = 98872
)

func awakenRelic(t *testing.T) {
	t.Helper()
	items.SetTestItemSpec(&items.ItemSpec{ItemId: awakenRelicID, Name: "Test Waker", Type: items.Weapon, Subtype: items.Slashing, Hands: 1, Tier: 5,
		Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 6},
		Relic: &items.RelicSpec{Signature: "Waking", Effects: map[string]int{classes.Wounded: 10}, ILvl: 20, Mob: awakenBossMob, Chance: 5,
			Awakenings: []items.AwakeningSpec{
				{Name: "Spirit-Taker", Kind: items.AwakenSlay, Races: []string{"ghostly spirit"}, Count: 2, Target: "spirits", Effects: map[string]int{classes.Damage: 1}},
				{Name: "Master's End", Kind: items.AwakenLair, Mob: awakenBossMob, Target: "the master", Effects: map[string]int{classes.Armor: 3}},
			}}})
	t.Cleanup(func() { items.RemoveTestItemSpec(awakenRelicID) })
}

// Phase 67: the real death path credits a slain foe's race to the relics
// the leader's company wears, and not in the Training zone.
func TestSlainFoesAdvanceWornRelicAwakeningsThroughTheDeathPath(t *testing.T) {
	chronicle.SetProvider(chronicle.NewMemory())
	t.Cleanup(func() { chronicle.SetProvider(nil) })
	awakenRelic(t)
	w := newDropWorld(t, 1, encounters.Band{Low: 5, High: 7})
	leader := w.users[0]
	leader.Character.Equipment.Weapon = items.New(awakenRelicID)
	blade := &leader.Character.Equipment.Weapon

	w.room.Zone = "Training" // a room puts its zone on the mobs it takes in
	training := w.foe(98873, nil)
	t.Cleanup(func() { mobs.RemoveTestInstance(training.InstanceId) })
	_, err := Suicide("", training, w.room)
	require.NoError(t, err)
	assert.Zero(t, blade.AwakeningProgress(0), "training kills count for nothing")
	w.room.Zone = dropZone

	for _, id := range []int{98874, 98875} {
		foe := w.foe(id, nil)
		t.Cleanup(func() { mobs.RemoveTestInstance(foe.InstanceId) })
		_, err = Suicide("", foe, w.room)
		require.NoError(t, err)
	}
	assert.True(t, blade.Awakened(0), "two spirits slain while it was worn")
	assert.Equal(t, 1, leader.Character.ClassEffects().Int(classes.Damage))
	assert.Equal(t, 1, chronicle.Total(leader.UserId, chronicle.Awakened))
}

// A lair master's death writes the boss deed in the chronicle, and that
// deed wakes the lair awakening of every company that fought it.
func TestABossKillWakesTheLairAwakening(t *testing.T) {
	chronicle.SetProvider(chronicle.NewMemory())
	t.Cleanup(func() { chronicle.SetProvider(nil) })
	awakenRelic(t)
	w := newDropWorld(t, 2, encounters.Band{Low: 5, High: 7})
	w.users[0].Character.Equipment.Weapon = items.New(awakenRelicID)
	// The second company carries the relic but does not wear it.
	w.users[1].Character.Items = append(w.users[1].Character.Items, items.New(awakenRelicID))

	boss := w.foe(awakenBossMob, func(m *mobs.Mob) { m.Boss, m.EncounterBoss, m.Character.Level = true, true, 7 })
	t.Cleanup(func() { mobs.RemoveTestInstance(boss.InstanceId) })
	_, err := Suicide("", boss, w.room)
	require.NoError(t, err)

	worn := &w.users[0].Character.Equipment.Weapon
	assert.True(t, worn.Awakened(1))
	assert.Equal(t, 3, w.users[0].Character.ClassEffects().Int(classes.Armor))
	assert.Zero(t, w.users[1].Character.Items[0].AwakeningProgress(1), "an unworn relic sleeps on")
}
