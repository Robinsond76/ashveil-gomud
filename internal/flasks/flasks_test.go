package flasks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

func alchemist(level int) *characters.Character {
	return &characters.Character{Name: "Mira", Level: level, HPArchetype: Lineage}
}

func withReagents(c *characters.Character, n int) *characters.Character {
	for i := 0; i < n; i++ {
		c.Items = append(c.Items, items.Item{ItemId: ReagentItemID})
	}
	return c
}

func TestTheSatchelGrowsWithLevelAndStartsFull(t *testing.T) {
	for level, want := range map[int]int{1: 6, 2: 6, 3: 7, 9: 9, 30: 16} {
		c := alchemist(level)
		assert.Equal(t, want, Capacity(c), "level %d", level)
		assert.Equal(t, want, CapacityAt(level))
		assert.Equal(t, want, Remaining(c), "a new Alchemist is full")
		assert.Zero(t, Missing(c))
	}
}

func TestNobodyButAnAlchemistHasASatchel(t *testing.T) {
	c := &characters.Character{Name: "Garrick", Level: 9, HPArchetype: "warrior"}
	assert.False(t, IsAlchemist(c))
	assert.False(t, IsAlchemist(nil))
	assert.Zero(t, Capacity(c))
	assert.Zero(t, Remaining(c))
	assert.False(t, Spend(c))
	assert.False(t, NeedsBrewing(c))
}

func TestSpendingFlasksEmptiesTheSatchelAndNoFurther(t *testing.T) {
	c := alchemist(1)
	for i := 0; i < 6; i++ {
		assert.True(t, Spend(c))
	}
	assert.Zero(t, Remaining(c))
	assert.Equal(t, 6, Missing(c))
	assert.False(t, Spend(c), "nothing left to throw")
	assert.Equal(t, 6, c.FlasksSpent)
}

func TestBrewRefillsFromReagentsOneAFlaskFirstComeFirstServed(t *testing.T) {
	leader := withReagents(alchemist(1), 5)
	leader.FlasksSpent = 4
	friend := alchemist(1)
	friend.FlasksSpent = 3
	taken, out := Brew(leader, []*characters.Character{leader, friend})
	assert.Len(t, taken, 5, "one reagent a flask")
	assert.Equal(t, 4, out[0].Brewed)
	assert.Equal(t, 1, out[1].Brewed, "what is left of the reagents")
	assert.Zero(t, leader.FlasksSpent)
	assert.Equal(t, 2, friend.FlasksSpent)
	assert.Zero(t, Reagents(leader))
	assert.True(t, NeedsBrewing(friend))
}

func TestBrewWithNoReagentsOrNoNeedChangesNothing(t *testing.T) {
	c := alchemist(1)
	c.FlasksSpent = 2
	taken, out := Brew(c, []*characters.Character{c})
	assert.Empty(t, taken)
	assert.Zero(t, out[0].Brewed)
	assert.Equal(t, 2, c.FlasksSpent)

	full := withReagents(alchemist(1), 3)
	taken, _ = Brew(full, []*characters.Character{full})
	assert.Empty(t, taken, "a full satchel takes no reagents")
	assert.Equal(t, 3, Reagents(full))
}

func TestTakeReagentsLeavesOtherItemsAlone(t *testing.T) {
	c := withReagents(alchemist(1), 2)
	c.Items = append(c.Items, items.Item{ItemId: 1})
	assert.Len(t, TakeReagents(c, 9), 2)
	assert.Len(t, c.Items, 1)
}
