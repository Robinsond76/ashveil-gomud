package summons

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

func TestInfoGrowsWithTheRoute(t *testing.T) {
	hierarch := classes.EffectsFor("hierarch", 60, nil)
	info, ok := Info(Angel, hierarch, false)
	assert.True(t, ok)
	assert.Equal(t, 100, info.HPPct)
	assert.Equal(t, 10, info.Sides, "Sword of the Host")
	assert.Equal(t, 50, info.Smite)
	assert.Equal(t, 3, info.Guards)
	assert.Equal(t, 2, info.MercyEvery)
	assert.True(t, info.MercyFull)
	assert.True(t, info.MercyTwo)
	assert.True(t, info.Cleanse)

	early, _ := Info(Angel, classes.EffectsFor("hierarch", 30, nil), true)
	assert.Equal(t, 125, early.HPPct, "a holy symbol adds a quarter")
	assert.Equal(t, 8, early.Sides)
	assert.Equal(t, 1, early.Guards)
	assert.Equal(t, 3, early.MercyEvery)

	demon, _ := Info(Demon, classes.EffectsFor("demonologist", 30, nil), false)
	assert.Equal(t, 80, demon.HPPct, "a Demon has 80% of a warrior's health")
	assert.Equal(t, 6, demon.Sides)
	assert.Equal(t, 1, demon.Dread)
	assert.False(t, demon.Mastered)

	_, ok = Info("imp", hierarch, false)
	assert.False(t, ok)
}

func TestFallenFoesAreRememberedPerLeaderStrongestFirst(t *testing.T) {
	ResetFallenForTest()
	t.Cleanup(ResetFallenForTest)
	assert.False(t, HasFallen(7))
	NoteFallen(7, Fallen{MobID: 1, Level: 3})
	NoteFallen(7, Fallen{MobID: 2, Level: 9})
	NoteFallen(7, Fallen{MobID: 3, Level: 5})
	NoteFallen(8, Fallen{MobID: 4, Level: 1})
	assert.True(t, HasFallen(7))
	f, ok := TakeFallen(7)
	assert.True(t, ok)
	assert.Equal(t, mobs.MobId(2), f.MobID, "the strongest")
	f, _ = TakeFallen(7)
	assert.Equal(t, mobs.MobId(3), f.MobID)
	ClearFallen(7)
	assert.False(t, HasFallen(7), "a battle's end forgets its fallen")
	assert.True(t, HasFallen(8), "another leader's are kept")
	_, ok = TakeFallen(7)
	assert.False(t, ok)
}

func TestAFightRemembersOnlyItsLastFewFallen(t *testing.T) {
	ResetFallenForTest()
	t.Cleanup(ResetFallenForTest)
	for i := 1; i <= 6; i++ {
		NoteFallen(7, Fallen{MobID: mobs.MobId(i), Level: i})
	}
	f, _ := TakeFallen(7)
	assert.Equal(t, mobs.MobId(6), f.MobID)
	for i := 0; i < 3; i++ {
		TakeFallen(7)
	}
	_, ok := TakeFallen(7)
	assert.False(t, ok, "four are kept")
}

func TestRaiseRefusesWithoutAFoeToRaise(t *testing.T) {
	_, err := Raise(Caller{}, Fallen{})
	assert.Error(t, err)
}
