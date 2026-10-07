package bestiary

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

func TestTiersComeFromKills(t *testing.T) {
	for _, tc := range []struct {
		kills int
		boss  bool
		want  Tier
	}{
		{0, false, Unknown}, {1, false, Lore}, {2, false, Lore}, {3, false, Defences},
		{5, false, Defences}, {6, false, Habits}, {40, false, Habits},
		{0, true, Unknown}, {1, true, Lore}, {2, true, Defences}, {3, true, Habits},
	} {
		assert.Equal(t, tc.want, TierFor(tc.kills, tc.boss), "%d kills, boss %v", tc.kills, tc.boss)
	}
}

func TestEntryProgressAndFind(t *testing.T) {
	e := Entry{Name: "grey wolf", Kills: 1, Tier: Lore, NextAt: 3}
	assert.Equal(t, "2 more kills to learn its defences", e.Progress())
	e.Kills, e.Tier, e.NextAt = 2, Lore, 3
	assert.Equal(t, "1 more kill to learn its defences", e.Progress())
	e.Kills, e.Tier, e.NextAt = 3, Defences, 6
	assert.Equal(t, "3 more kills to learn its habits", e.Progress())
	e.Kills, e.Tier, e.NextAt = 6, Habits, 0
	assert.Equal(t, "everything is known", e.Progress())

	list := []Entry{
		{MobID: 1, Name: "grey wolf", Kills: 2},
		{MobID: 2, Name: "grey wolf", Kills: 7},
		{MobID: 3, Name: "wolf cub", Kills: 1},
		{MobID: 4, Name: "dire wolf", Kills: 1},
	}
	got, ok := Find(list, "grey wolf")
	assert.True(t, ok)
	assert.Equal(t, 2, got.MobID, "of equal names, the better known")
	got, _ = Find(list, "wolf")
	assert.Equal(t, 3, got.MobID, "a prefix beats a contained word")
	got, _ = Find(list, "dire")
	assert.Equal(t, 4, got.MobID)
	_, ok = Find(list, "ogre")
	assert.False(t, ok, "only known kinds are searched")
	_, ok = Find(list, "  ")
	assert.False(t, ok)
}

func TestKillsOfReadsTheCharactersOwnTally(t *testing.T) {
	assert.Nil(t, KillsOf(nil))
	c := &characters.Character{}
	c.KD.AddMobKill(85)
	c.KD.AddMobKill(85)
	assert.Equal(t, map[int]int{85: 2}, KillsOf(c))
}
