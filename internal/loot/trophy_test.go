package loot

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scriptedSource returns the given draws in order, then zeros.
type scriptedSource struct{ draws []int }

func (s *scriptedSource) Intn(n int) int {
	if len(s.draws) == 0 {
		return 0
	}
	d := s.draws[0]
	s.draws = s.draws[1:]
	return d % n
}

const (
	trophyOgreHeart = 970301
	trophyOgreHide  = 970302
	trophyWolfHide  = 970303
)

func trophyWorld(t *testing.T) {
	t.Helper()
	specs := []*items.ItemSpec{
		{ItemId: trophyOgreHeart, Name: "ogre heart", Type: items.Commodity, Value: 12,
			Trophy: &items.TrophySpec{Part: items.TrophyHeart, Races: []string{"ogre", "troll"}, Chance: 20, Effects: map[string]int{classes.Damage: 1}}},
		{ItemId: trophyOgreHide, Name: "ogre hide", Type: items.Commodity, Value: 12,
			Trophy: &items.TrophySpec{Part: items.TrophyHide, Races: []string{"ogre"}, Chance: 60, Effects: map[string]int{classes.Armor: 2}}},
		{ItemId: trophyWolfHide, Name: "wolf hide trophy", Type: items.Commodity, Value: 12,
			Trophy: &items.TrophySpec{Part: items.TrophyHide, Races: []string{"canine"}, Chance: 25, Effects: map[string]int{classes.Evasion: 3}}},
	}
	for _, s := range specs {
		items.SetTestItemSpec(s)
	}
	t.Cleanup(func() {
		for _, s := range specs {
			items.RemoveTestItemSpec(s.ItemId)
		}
	})
}

func TestTrophiesAreChosenByRace(t *testing.T) {
	trophyWorld(t)
	ids := func(race string) []int {
		var out []int
		for _, s := range Trophies(race) {
			out = append(out, s.ItemId)
		}
		return out
	}
	assert.Equal(t, []int{trophyOgreHeart, trophyOgreHide}, ids("ogre"))
	assert.Equal(t, []int{trophyOgreHeart, trophyOgreHide}, ids("Ogre"), "the race name is read without case")
	assert.Equal(t, []int{trophyOgreHeart}, ids("troll"))
	assert.Equal(t, []int{trophyWolfHide}, ids("canine"))
	assert.Empty(t, ids("human"))
	assert.Nil(t, TrophyRoll(Ordinary, "human", &scriptedSource{}))
}

func TestTrophyChanceByKind(t *testing.T) {
	trophyWorld(t)
	heart := items.GetItemSpec(trophyOgreHeart)
	assert.Equal(t, 20, TrophyChance(*heart, Ordinary))
	assert.Equal(t, 40, TrophyChance(*heart, Elite))
	assert.Equal(t, 100, TrophyChance(*heart, BossRoll))
	hide := items.GetItemSpec(trophyOgreHide)
	assert.Equal(t, 100, TrophyChance(*hide, Elite), "an elite's doubled chance stops at 100")
}

func TestTrophyRollDropsOnTheChosenTrophysChance(t *testing.T) {
	trophyWorld(t)
	// First draw picks the trophy (0 = heart, 1 = hide); the second is the percent roll.
	drop := TrophyRoll(Ordinary, "ogre", &scriptedSource{draws: []int{0, 19}})
	require.NotNil(t, drop)
	assert.Equal(t, trophyOgreHeart, drop.ItemId)
	assert.Nil(t, TrophyRoll(Ordinary, "ogre", &scriptedSource{draws: []int{0, 20}}), "20 of 100 is the first miss")
	assert.Nil(t, TrophyRoll(Ordinary, "ogre", &scriptedSource{draws: []int{1, 60}}))
	drop = TrophyRoll(Ordinary, "ogre", &scriptedSource{draws: []int{1, 59}})
	require.NotNil(t, drop)
	assert.Equal(t, trophyOgreHide, drop.ItemId)

	assert.Nil(t, TrophyRoll(Ordinary, "ogre", &scriptedSource{draws: []int{0, 99}}))
	drop = TrophyRoll(BossRoll, "ogre", &scriptedSource{draws: []int{0, 99}})
	require.NotNil(t, drop, "a boss always drops one")
	assert.NotZero(t, drop.UUID, "a real item instance")
}
