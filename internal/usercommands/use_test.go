package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	useTestMeat   = 9101
	useTestHerb   = 9102
	useTestStew   = 9110 // meat, meat, herb; cooking 3
	useTestRoast  = 9111 // meat; cooking 1
	useTestUngate = 9120 // herb; no requirement
)

func setupUseTest(t *testing.T) {
	t.Helper()
	skills.SetTestData([]*skills.Skill{{SkillId: "cooking", Name: "Cooking", Description: "Cook.", MaxLevel: 4}}, nil)
	t.Cleanup(func() { skills.SetTestData(nil, nil) })

	for _, spec := range []items.ItemSpec{
		{ItemId: useTestMeat, Name: "raw meat", NameSimple: "meat", Type: items.Commodity},
		{ItemId: useTestHerb, Name: "herb", NameSimple: "herb", Type: items.Botanical},
		{ItemId: useTestStew, Name: "stew", NameSimple: "stew", Type: items.Food, Subtype: items.Edible},
		{ItemId: useTestRoast, Name: "roast", NameSimple: "roast", Type: items.Food, Subtype: items.Edible},
		{ItemId: useTestUngate, Name: "tea", NameSimple: "tea", Type: items.Food, Subtype: items.Edible},
	} {
		spec := spec
		items.SetTestItemSpec(&spec)
		t.Cleanup(func() { items.RemoveTestItemSpec(spec.ItemId) })
	}
}

func useTestRoom(recipes map[int][]int, reqs map[int]rooms.RecipeRequirement, contents ...int) *rooms.Room {
	room := testRoom()
	c := rooms.Container{Recipes: recipes, RecipeRequirements: reqs}
	for _, id := range contents {
		c.AddItem(items.New(id))
	}
	room.Containers = map[string]rooms.Container{"hearth": c}
	return room
}

func cook(cookingLevel int) *users.UserRecord {
	user := users.NewUserRecord(7, 1)
	user.Character.Name = "Cook"
	if cookingLevel > 0 {
		user.Character.Skills = map[string]int{"cooking": cookingLevel}
	}
	return user
}

func containerIds(room *rooms.Room) []int {
	ids := []int{}
	for _, itm := range room.Containers["hearth"].Items {
		ids = append(ids, itm.ItemId)
	}
	return ids
}

func TestUseContainerCookingRecipeGating(t *testing.T) {
	setupUseTest(t)

	recipes := map[int][]int{
		useTestStew:  {useTestMeat, useTestMeat, useTestHerb},
		useTestRoast: {useTestMeat},
	}
	reqs := map[int]rooms.RecipeRequirement{
		useTestStew:  {SkillId: "cooking", MinLevel: 3},
		useTestRoast: {SkillId: "cooking", MinLevel: 1},
	}

	t.Run("skilled cook makes the lowest eligible recipe and consumes its inputs", func(t *testing.T) {
		room := useTestRoom(recipes, reqs, useTestMeat, useTestMeat, useTestHerb)
		handled, err := Use("hearth", cook(3), room, 0)
		require.NoError(t, err)
		assert.True(t, handled)
		assert.Equal(t, []int{useTestStew}, containerIds(room))
	})

	t.Run("lower skill falls through to a recipe the cook can make", func(t *testing.T) {
		room := useTestRoom(recipes, reqs, useTestMeat, useTestMeat, useTestHerb)
		_, err := Use("hearth", cook(1), room, 0)
		require.NoError(t, err)
		assert.ElementsMatch(t, []int{useTestMeat, useTestHerb, useTestRoast}, containerIds(room))
	})

	t.Run("insufficient skill consumes nothing", func(t *testing.T) {
		room := useTestRoom(recipes, reqs, useTestMeat, useTestMeat, useTestHerb)
		handled, err := Use("hearth", cook(0), room, 0)
		require.NoError(t, err)
		assert.True(t, handled)
		assert.Equal(t, []int{useTestMeat, useTestMeat, useTestHerb}, containerIds(room))
	})

	t.Run("ungated recipe still works without the skill", func(t *testing.T) {
		room := useTestRoom(map[int][]int{useTestUngate: {useTestHerb}}, nil, useTestHerb)
		_, err := Use("hearth", cook(0), room, 0)
		require.NoError(t, err)
		assert.Equal(t, []int{useTestUngate}, containerIds(room))
	})

	t.Run("multiple ready recipes pick the lowest eligible output id every time", func(t *testing.T) {
		for i := 0; i < 20; i++ {
			room := useTestRoom(map[int][]int{
				useTestUngate: {useTestHerb},
				useTestRoast:  {useTestMeat},
			}, nil, useTestMeat, useTestHerb)
			_, err := Use("hearth", cook(0), room, 0)
			require.NoError(t, err)
			assert.ElementsMatch(t, []int{useTestHerb, useTestRoast}, containerIds(room))
		}
	})
}

func captureUserText(t *testing.T, fn func()) string {
	t.Helper()
	events.ProcessEvents()
	var messages []string
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		messages = append(messages, e.(events.Message).Text)
		return events.Continue
	})
	defer events.UnregisterListener(events.Message{}, id)
	fn()
	events.ProcessEvents()
	return strings.Join(messages, "")
}

func TestUseContainerRefusalNamesRequirement(t *testing.T) {
	setupUseTest(t)
	room := useTestRoom(
		map[int][]int{useTestStew: {useTestMeat}},
		map[int]rooms.RecipeRequirement{useTestStew: {SkillId: "cooking", MinLevel: 3}},
		useTestMeat,
	)
	out := captureUserText(t, func() {
		_, err := Use("hearth", cook(1), room, 0)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "level 3")
	assert.Contains(t, out, "Cooking")
}

func TestLookContainerListsRecipesInOrderWithRequirement(t *testing.T) {
	setupUseTest(t)
	room := useTestRoom(
		map[int][]int{useTestUngate: {useTestHerb}, useTestStew: {useTestMeat}},
		map[int]rooms.RecipeRequirement{useTestStew: {SkillId: "cooking", MinLevel: 3}},
	)
	room.Tags = []string{"lit"} // other tests in this package can leave the world dark
	out := captureUserText(t, func() {
		_, err := Look("hearth", cook(0), room, 0)
		require.NoError(t, err)
	})
	stewAt := strings.Index(out, "stew")
	teaAt := strings.Index(out, "tea")
	require.True(t, stewAt >= 0 && teaAt >= 0, out)
	assert.Less(t, stewAt, teaAt, "lower output id listed first")
	assert.Contains(t, out, "requires")
	assert.Equal(t, 1, strings.Count(out, "requires"), "only the gated recipe shows a requirement")
}
