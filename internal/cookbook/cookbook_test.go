package cookbook

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

func withSpecs(t *testing.T) {
	t.Helper()
	for _, spec := range []*items.ItemSpec{
		{ItemId: 29, Name: "raw game meat", NameSimple: "meat", Type: items.Commodity, Goods: "provision"},
		{ItemId: 30018, Name: "wild thyme", NameSimple: "thyme", Type: items.Botanical},
		{ItemId: 30021, Name: "seared game meat", Type: items.Food, Nutrition: 40, Meal: "seared"},
		{ItemId: 9, Name: "waterskin", Type: items.Drink, Hydration: 50},
		{ItemId: 10, Name: "iron sword", Type: items.Weapon},
	} {
		items.SetTestItemSpec(spec)
		id := spec.ItemId
		t.Cleanup(func() { items.RemoveTestItemSpec(id) })
	}
}

func TestBookLearnsOnceAndSurvivesAReload(t *testing.T) {
	c := &characters.Character{}
	assert.Empty(t, Learned(c))
	assert.True(t, Learn(c, 30020))
	assert.False(t, Learn(c, 30020), "a dish is learned once")
	assert.True(t, Learn(c, 30019))
	assert.Equal(t, []int{30019, 30020}, Learned(c))
	// MiscData is what the character file saves; a fresh character with the
	// same data reads the same book.
	reloaded := &characters.Character{MiscData: map[string]any{BookKey: c.GetMiscData(BookKey)}}
	assert.Equal(t, []int{30019, 30020}, Learned(reloaded))
	assert.False(t, Learn(nil, 1))
	assert.False(t, Learn(c, 0))
}

func TestKnowsCommonDishesFromTheStart(t *testing.T) {
	c := &characters.Character{}
	common := Recipe{Output: 30021, Inputs: []int{29}, Skill: "cooking", MinLevel: 1}
	ungated := Recipe{Output: 5, Inputs: []int{29}}
	hard := Recipe{Output: 30019, Inputs: []int{29, 29, 30018}, Skill: "cooking", MinLevel: 3}
	assert.True(t, Knows(c, common))
	assert.True(t, Knows(c, ungated))
	assert.False(t, Knows(c, hard))
	Learn(c, 30019)
	assert.True(t, Knows(c, hard))
	assert.Equal(t, []Recipe{common, hard}, Known(c, []Recipe{common, hard}))
}

func TestMatchIgnoresOrderButNeedsTheExactMix(t *testing.T) {
	roast := Recipe{Output: 30020, Inputs: []int{29, 30018}}
	stew := Recipe{Output: 30019, Inputs: []int{29, 29, 30018}}
	all := []Recipe{stew, roast}
	got, ok := Match([]int{30018, 29}, all)
	assert.True(t, ok)
	assert.Equal(t, 30020, got.Output)
	got, ok = Match([]int{29, 30018, 29}, all)
	assert.True(t, ok)
	assert.Equal(t, 30019, got.Output)
	_, ok = Match([]int{29, 29}, all)
	assert.False(t, ok, "a subset or a different mix is no match")
	_, ok = Match([]int{29, 29, 29, 30018}, all)
	assert.False(t, ok)
}

func TestResolveNamesIngredientsAndRefusesTheRest(t *testing.T) {
	withSpecs(t)
	stock := []Stack{{29, 2}, {30018, 1}, {9, 1}, {10, 1}, {30021, 1}}
	got, msg := Resolve([]string{"meat", "thyme"}, stock)
	assert.Empty(t, msg)
	assert.Equal(t, []int{29, 30018}, got)
	got, _ = Resolve([]string{"2", "meat", "wild"}, stock)
	assert.Equal(t, []int{29, 29, 30018}, got, "a count, and a word of the name")
	_, msg = Resolve([]string{"mea"}, stock)
	assert.Empty(t, msg, "the start of a name works")
	for _, bad := range []string{"sword", "waterskin", "seared", "nothing"} {
		_, msg = Resolve([]string{bad}, stock)
		assert.Contains(t, msg, "no ingredient called", bad+" is not an ingredient the cook holds")
	}
	_, msg = Resolve([]string{"3", "meat"}, stock)
	assert.Contains(t, msg, "don't have 3")
	_, msg = Resolve(nil, stock)
	assert.Contains(t, msg, "Name what to cook")
	_, msg = Resolve([]string{"2", "meat", "2", "thyme", "meat"}, []Stack{{29, 9}, {30018, 9}})
	assert.Contains(t, msg, "at most 4")
}

func TestDescribeListsCounts(t *testing.T) {
	withSpecs(t)
	assert.Equal(t, "2 raw game meat, 1 wild thyme", Describe([]int{30018, 29, 29}))
}
