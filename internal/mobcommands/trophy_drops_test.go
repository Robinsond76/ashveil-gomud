package mobcommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/encounters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/loot"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	trophyDropID   = 98891
	trophyOtherID  = 98892
	trophyFoeRace  = "ghostly spirit" // what a test foe is: no race is loaded
	trophyBossMobA = 98893
)

func trophyDropSpecs(t *testing.T, chance int) {
	t.Helper()
	items.SetTestItemSpec(&items.ItemSpec{ItemId: trophyDropID, Name: "test ash", Type: items.Commodity, Value: 12,
		Trophy: &items.TrophySpec{Part: items.TrophyAsh, Races: []string{trophyFoeRace}, Chance: chance, Effects: map[string]int{classes.SpellPct: 6}}})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: trophyOtherID, Name: "test hide", Type: items.Commodity, Value: 12,
		Trophy: &items.TrophySpec{Part: items.TrophyHide, Races: []string{"ogre"}, Chance: 100, Effects: map[string]int{classes.Armor: 2}}})
	t.Cleanup(func() { items.RemoveTestItemSpec(trophyDropID); items.RemoveTestItemSpec(trophyOtherID) })
}

func trophiesIn(w *dropWorld, id int) int {
	n := 0
	for _, c := range w.room.Corpses {
		for _, it := range c.Items {
			if it.ItemId == id {
				n++
			}
		}
	}
	return n
}

// Through the real kill path: a foe whose race carries a trophy (here at
// 100%) drops it, the spoils line names it, a foe of another race drops
// none, and the Training yard drops none.
func TestAFoeDropsTheTrophyOfItsRaceThroughTheDeathPath(t *testing.T) {
	trophyDropSpecs(t, 100)
	w := newDropWorld(t, 1, encounters.Band{})
	foe := w.foe(98894, nil)
	t.Cleanup(func() { mobs.RemoveTestInstance(foe.InstanceId) })
	require.Equal(t, trophyFoeRace, strings.ToLower(foe.Character.Race()))
	_, err := Suicide("", foe, w.room)
	require.NoError(t, err)
	assert.Equal(t, 1, trophiesIn(w, trophyDropID), "no drop profile is needed")
	assert.Zero(t, trophiesIn(w, trophyOtherID), "another race's trophy does not drop")
	assert.Contains(t, strings.Join(loot.TakeSpoils(w.users[0].UserId), " "), "test ash")

	w2 := newDropWorld(t, 1, encounters.Band{})
	w2.room.Zone = "Training"
	practice := w2.foe(98895, nil)
	t.Cleanup(func() { mobs.RemoveTestInstance(practice.InstanceId) })
	_, err = Suicide("", practice, w2.room)
	require.NoError(t, err)
	assert.Zero(t, trophiesIn(w2, trophyDropID), "the training yard drops no trophies")
}

// A boss always yields one, and each allied company that fought rolls its own.
func TestABossAlwaysDropsATrophyToEachCompany(t *testing.T) {
	trophyDropSpecs(t, 1) // 1 in 100 for an ordinary foe
	w := newDropWorld(t, 2, encounters.Band{})
	boss := w.foe(trophyBossMobA, func(m *mobs.Mob) { m.Boss, m.EncounterID = true, "" })
	t.Cleanup(func() { mobs.RemoveTestInstance(boss.InstanceId) })
	_, err := Suicide("", boss, w.room)
	require.NoError(t, err)
	assert.Equal(t, 2, trophiesIn(w, trophyDropID), "one for each company")
	for _, u := range w.users {
		assert.Contains(t, strings.Join(loot.TakeSpoils(u.UserId), " "), "test ash")
	}
}
