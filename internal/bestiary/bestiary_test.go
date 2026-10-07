package bestiary

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// Phase 71: the habits tier names the trophies a kind may yield, by race,
// with the chance for an ordinary foe and "always" for a boss; a kind whose
// race carries none says nothing, and nothing shows before the habits tier.
func TestHabitsNameTheTrophiesAKindMayYield(t *testing.T) {
	items.SetTestItemSpec(&items.ItemSpec{ItemId: 99751, Name: "grave ash", Type: items.Commodity, Value: 12,
		Trophy: &items.TrophySpec{Part: items.TrophyAsh, Races: []string{"ghostly spirit"}, Chance: 25, Effects: map[string]int{classes.SpellPct: 6}}})
	t.Cleanup(func() { items.RemoveTestItemSpec(99751) })

	foe := &mobs.Mob{MobId: 99752, Character: *characters.New()}
	foe.Character.Name = "wisp"
	entry, ok := Build(foe, 6)
	require.True(t, ok)
	assert.Contains(t, strings.Join(entry.Habits, "\n"), "Hunted for trophies: grave ash (about 25 in 100 kills) (help enchanting).")
	early, _ := Build(foe, 3)
	assert.NotContains(t, strings.Join(early.Habits, "\n"), "Hunted for")
	assert.NotContains(t, strings.Join(early.Defences, "\n"), "Hunted for")

	foe.Boss = true
	entry, _ = Build(foe, 3)
	assert.Contains(t, strings.Join(entry.Habits, "\n"), "Hunted for a trophy it always yields, one of: grave ash (help enchanting).")

	foe.Character.Zone = "Training"
	entry, _ = Build(foe, 3)
	assert.NotContains(t, strings.Join(entry.Habits, "\n"), "Hunted for", "the training yard yields none")

	items.RemoveTestItemSpec(99751)
	foe.Character.Zone = ""
	entry, _ = Build(foe, 3)
	assert.NotContains(t, strings.Join(entry.Habits, "\n"), "Hunted for")
}
