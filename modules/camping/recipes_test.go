package camping

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/cookbook"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// discoveryWorld is a camp with a lit fire, an unlettered cook (Cooking 3,
// no learned dishes) and the shipped recipe shapes.
func discoveryWorld(t *testing.T) (*raidWorld, *fakeCargo) {
	t.Helper()
	for _, spec := range []*items.ItemSpec{
		{ItemId: 29, Name: "raw game meat", NameSimple: "meat", Type: items.Commodity, Goods: "provision", Weight: 300},
		{ItemId: 30018, Name: "wild thyme", NameSimple: "thyme", Type: items.Botanical, Weight: 20},
		{ItemId: 30019, Name: "hunter's stew", Type: items.Food, Meal: "stew", Weight: 700},
		{ItemId: 30020, Name: "thyme-roasted game", Type: items.Food, Meal: "roast", Weight: 500},
		{ItemId: 30021, Name: "seared game meat", Type: items.Food, Meal: "seared", Weight: 400},
		{ItemId: items.MakeshiftMealItemId, Name: "makeshift meal", Type: items.Food, Nutrition: 25, Weight: 300},
		{ItemId: 99001, Name: "iron sword", Type: items.Weapon, Weight: 1000},
	} {
		items.SetTestItemSpec(spec)
		id := spec.ItemId
		t.Cleanup(func() { items.RemoveTestItemSpec(id) })
	}
	w := newRaidWorld(t, 0)
	for _, dish := range []int{30019, 30020, 30021, 30024} {
		cookbook.Learn(w.user.Character, 0*dish) // no-op: start from an empty book
	}
	w.user.Character.SetMiscData(cookbook.BookKey, nil)
	w.m.campCfg.Recipes = []campRecipe{
		{Output: 30019, Inputs: []int{29, 29, 30018}, Skill: "cooking", MinLevel: 3},
		{Output: 30020, Inputs: []int{29, 30018}, Skill: "cooking", MinLevel: 2},
		{Output: 30021, Inputs: []int{29}, Skill: "cooking", MinLevel: 1},
	}
	w.m.inBattle = func(int) bool { return false }
	cargo := newFakeCargo(100000)
	useCargo(t, cargo)
	skills.SetTestData([]*skills.Skill{{SkillId: "cooking", Name: "Cooking", MaxLevel: 4}}, nil)
	t.Cleanup(func() { skills.SetTestData(nil, nil) })
	w.user.Character.Skills = map[string]int{"cooking": 3}
	return w, cargo
}

func TestCookingANewCombinationLearnsItOnce(t *testing.T) {
	w, cargo := discoveryWorld(t)
	cargo.stacks[29] = 2
	cargo.stacks[30018] = 2

	assert.Contains(t, w.m.cook(w.user, w.room, nil), "seared game meat", "bare cook makes only known dishes; meat alone is a common dish")
	assert.Equal(t, 1, cargo.stacks[30021], "the common seared meat was made")
	cargo.stacks[29] = 2

	text := w.m.cook(w.user, w.room, []string{"meat", "thyme"})
	assert.Contains(t, text, "thyme-roasted game")
	assert.Contains(t, text, "worked out a new recipe")
	assert.Equal(t, 1, cargo.stacks[29])
	assert.Equal(t, 1, cargo.stacks[30018])
	assert.Equal(t, 1, cargo.stacks[30020])
	assert.Equal(t, []int{30020}, cookbook.Learned(w.user.Character))

	again := w.m.cook(w.user, w.room, []string{"thyme", "meat"})
	assert.NotContains(t, again, "worked out", "a dish is learned once, in any order")
	assert.Equal(t, 2, cargo.stacks[30020])
	assert.Equal(t, []int{30020}, cookbook.Learned(w.user.Character))
}

func TestKnownDishCooksWithoutNamingIngredients(t *testing.T) {
	w, cargo := discoveryWorld(t)
	cookbook.Learn(w.user.Character, 30020)
	cargo.stacks[29] = 1
	cargo.stacks[30018] = 1
	// Bare camp cook picks the first known dish that is ready.
	assert.Contains(t, w.m.cook(w.user, w.room, nil), "thyme-roasted game")
	assert.Equal(t, 1, cargo.stacks[30020])
	assert.Equal(t, 0, cargo.stacks[30018])
}

func TestAnUnlearnedDishIsNotMadeWithoutTrying(t *testing.T) {
	w, cargo := discoveryWorld(t)
	cargo.stacks[29] = 2
	cargo.stacks[30018] = 1
	// Stew is ready, but unknown: the common seared meat is all a bare cook makes.
	text := w.m.cook(w.user, w.room, nil)
	assert.Contains(t, text, "seared game meat")
	assert.NotContains(t, text, "stew")
	assert.Empty(t, cookbook.Learned(w.user.Character))
}

func TestAMissMakesAMakeshiftMealAndTeachesNothing(t *testing.T) {
	w, cargo := discoveryWorld(t)
	cargo.stacks[30018] = 2
	text := w.m.cook(w.user, w.room, []string{"thyme", "thyme"})
	assert.Contains(t, text, "makeshift meal")
	assert.Equal(t, 0, cargo.stacks[30018], "the herbs are gone")
	assert.Equal(t, 1, cargo.stacks[items.MakeshiftMealItemId])
	assert.Empty(t, cookbook.Learned(w.user.Character))
	meal := items.New(items.MakeshiftMealItemId)
	assert.True(t, meal.IsSpecialForSale(), "a merchant never buys it back")
}

func TestATooHardDishConsumesAndTeachesNothing(t *testing.T) {
	w, cargo := discoveryWorld(t)
	w.user.Character.Skills["cooking"] = 2
	cargo.stacks[29] = 2
	cargo.stacks[30018] = 1
	text := w.m.cook(w.user, w.room, []string{"meat", "meat", "thyme"})
	assert.Contains(t, text, "needs cooking 3")
	assert.Equal(t, 2, cargo.stacks[29])
	assert.Equal(t, 1, cargo.stacks[30018])
	assert.Empty(t, cookbook.Learned(w.user.Character))
}

func TestFailedSaveDoesNotTeachTheDish(t *testing.T) {
	w, cargo := discoveryWorld(t)
	cargo.stacks[29] = 1
	cargo.stacks[30018] = 1
	cargo.failWithdraw = map[int]bool{29: true}
	text := w.m.cook(w.user, w.room, []string{"meat", "thyme"})
	assert.Contains(t, text, "couldn't be gathered")
	assert.Empty(t, cookbook.Learned(w.user.Character), "nothing was cooked, so nothing was learned")
}

func TestCookRejectsNonIngredientsAndBadNames(t *testing.T) {
	w, cargo := discoveryWorld(t)
	cargo.stacks[29] = 1
	cargo.stacks[30018] = 1
	cargo.stacks[99001] = 1
	assert.Contains(t, w.m.cook(w.user, w.room, []string{"sword"}), "no ingredient called")
	assert.Contains(t, w.m.cook(w.user, w.room, []string{"meat", "meat"}), "don't have 2")
	assert.Contains(t, w.m.cook(w.user, w.room, []string{"2", "meat", "2", "thyme", "meat"}), "at most 4")
	assert.Equal(t, 1, cargo.stacks[29], "nothing consumed by refusals")
	assert.Equal(t, 1, cargo.stacks[99001])
}

func TestRecipesCommandListsTheBook(t *testing.T) {
	w, _ := discoveryWorld(t)
	lines := w.m.recipesLines(w.user)
	require.Len(t, lines, 1, "only the common dish at first")
	assert.Contains(t, lines[0], "seared game meat")
	cookbook.Learn(w.user.Character, 30019)
	lines = w.m.recipesLines(w.user)
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], "hunter's stew: 2 raw game meat, 1 wild thyme (cooking 3)")
}

func TestCookCommandNeedsAHearthOrLitFire(t *testing.T) {
	w, _ := discoveryWorld(t)
	camp := w.m.camps[7]
	camp.FireLit = false
	w.m.camps[7] = camp
	messages := captureMessages(t)
	_, err := w.m.cookCommand("meat", w.user, w.room, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*messages, ""), "hearth or your own lit campfire")
}

// A hearth (a container with recipes) takes the same experiments with the
// player's own Cooking, learning into the same book.
func TestCookAtAHearthLearnsToo(t *testing.T) {
	w, cargo := discoveryWorld(t)
	camp := w.m.camps[7]
	camp.FireLit = false
	w.m.camps[7] = camp
	w.room.Containers = map[string]rooms.Container{"hearth": {
		Recipes:            map[int][]int{30020: {29, 30018}},
		RecipeRequirements: map[int]rooms.RecipeRequirement{30020: {SkillId: "cooking", MinLevel: 2}},
	}}
	cargo.stacks[29] = 1
	cargo.stacks[30018] = 1
	messages := captureMessages(t)
	_, err := w.m.cookCommand("meat thyme", w.user, w.room, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	out := strings.Join(*messages, "")
	assert.Contains(t, out, "at the hearth")
	assert.Contains(t, out, "worked out a new recipe")
	assert.Equal(t, []int{30020}, cookbook.Learned(w.user.Character))
	assert.Equal(t, 1, cargo.stacks[30020])
}

// The Camp tab lists the book, and the manual cooking capability lists only
// the dishes the leader knows.
func TestCampStateAndCapabilityListOnlyKnownDishes(t *testing.T) {
	w, cargo := discoveryWorld(t)
	state, ok := w.m.CampStateOf(7, 100, []string{"camping"})
	require.True(t, ok)
	require.Len(t, state.Recipes, 1)
	assert.Contains(t, state.Recipes[0], "seared game meat: 1 raw game meat")
	cookbook.Learn(w.user.Character, 30020)
	state, _ = w.m.CampStateOf(7, 100, []string{"camping"})
	require.Len(t, state.Recipes, 2)
	cargo.stacks[29] = 1
	cargo.stacks[30018] = 1
	view, _ := w.m.CookingCapability(7)
	assert.Contains(t, view.Description, "thyme-roasted game requires")
	assert.NotContains(t, view.Description, "hunter's stew", "an unlearned dish is not listed")
}
