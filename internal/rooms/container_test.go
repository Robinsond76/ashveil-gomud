package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func containerWith(recipes map[int][]int, reqs map[int]RecipeRequirement, itemIds ...int) Container {
	c := Container{Recipes: recipes, RecipeRequirements: reqs}
	for _, id := range itemIds {
		c.Items = append(c.Items, items.Item{ItemId: id})
	}
	return c
}

func levels(l map[string]int) func(string) int {
	return func(skillId string) int { return l[skillId] }
}

func TestReadyRecipesSortedAscending(t *testing.T) {
	c := containerWith(map[int][]int{
		300: {1},
		100: {1, 2},
		200: {9},
		50:  {1, 1},
	}, nil, 1, 2)

	for i := 0; i < 20; i++ {
		assert.Equal(t, []int{100, 300}, c.ReadyRecipes())
	}
	assert.Equal(t, 100, c.RecipeReady())
}

func TestReadyRecipesCountsDuplicateInputs(t *testing.T) {
	c := containerWith(map[int][]int{50: {1, 1}}, nil, 1)
	assert.Empty(t, c.ReadyRecipes())

	c.Items = append(c.Items, items.Item{ItemId: 1})
	assert.Equal(t, []int{50}, c.ReadyRecipes())
}

func TestSelectRecipe(t *testing.T) {
	recipes := map[int][]int{10: {1, 1, 2}, 20: {1, 2}, 30: {1}}
	reqs := map[int]RecipeRequirement{
		10: {SkillId: "cooking", MinLevel: 3},
		20: {SkillId: "cooking", MinLevel: 2},
	}

	tests := []struct {
		name        string
		contents    []int
		levels      map[string]int
		wantId      int
		wantBlocked RecipeRequirement
	}{
		{name: "nothing ready", contents: []int{2}, wantId: 0},
		{name: "lowest eligible wins", contents: []int{1, 1, 2}, levels: map[string]int{"cooking": 3}, wantId: 10},
		{name: "falls through gated to eligible", contents: []int{1, 1, 2}, levels: map[string]int{"cooking": 2}, wantId: 20},
		{name: "ungated recipe needs no skill", contents: []int{1, 2}, wantId: 30},
		{name: "low skill falls through to ungated", contents: []int{1, 1, 2}, levels: map[string]int{"cooking": 1}, wantId: 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := containerWith(recipes, reqs, tt.contents...)
			gotId, blocked := c.SelectRecipe(levels(tt.levels))
			assert.Equal(t, tt.wantId, gotId)
			assert.Equal(t, tt.wantBlocked, blocked)
		})
	}

	t.Run("all ready recipes gated", func(t *testing.T) {
		c := containerWith(map[int][]int{10: {1, 2}, 20: {1}}, map[int]RecipeRequirement{
			10: {SkillId: "cooking", MinLevel: 3},
			20: {SkillId: "cooking", MinLevel: 2},
		}, 1, 2)
		gotId, blocked := c.SelectRecipe(levels(nil))
		assert.Equal(t, 0, gotId)
		assert.Equal(t, RecipeRequirement{SkillId: "cooking", MinLevel: 2}, blocked, "reports the easiest requirement")
	})
}

func TestContainerValidateRecipes(t *testing.T) {
	tests := []struct {
		name    string
		c       Container
		wantErr bool
	}{
		{name: "no recipes", c: Container{}},
		{name: "ungated recipes", c: Container{Recipes: map[int][]int{5: {1}}}},
		{
			name: "valid requirement",
			c: Container{Recipes: map[int][]int{5: {1}},
				RecipeRequirements: map[int]RecipeRequirement{5: {SkillId: "cooking", MinLevel: 1}}},
		},
		{
			name: "requirement for unknown recipe",
			c: Container{Recipes: map[int][]int{5: {1}},
				RecipeRequirements: map[int]RecipeRequirement{6: {SkillId: "cooking", MinLevel: 1}}},
			wantErr: true,
		},
		{
			name: "empty skill id",
			c: Container{Recipes: map[int][]int{5: {1}},
				RecipeRequirements: map[int]RecipeRequirement{5: {SkillId: "  ", MinLevel: 1}}},
			wantErr: true,
		},
		{
			name: "nonpositive level",
			c: Container{Recipes: map[int][]int{5: {1}},
				RecipeRequirements: map[int]RecipeRequirement{5: {SkillId: "cooking", MinLevel: 0}}},
			wantErr: true,
		},
		{name: "recipe with no inputs", c: Container{Recipes: map[int][]int{5: {}}}, wantErr: true},
		{name: "nonpositive output id", c: Container{Recipes: map[int][]int{0: {1}}}, wantErr: true},
		{name: "nonpositive input id", c: Container{Recipes: map[int][]int{5: {0}}}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.c.ValidateRecipes()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestContainerValidateRecipesNormalizesSkillId(t *testing.T) {
	c := Container{Recipes: map[int][]int{5: {1}},
		RecipeRequirements: map[int]RecipeRequirement{5: {SkillId: " Cooking ", MinLevel: 2}}}
	require.NoError(t, c.ValidateRecipes())
	assert.Equal(t, "cooking", c.RecipeRequirements[5].SkillId)
}

func TestRoomValidateRejectsMalformedRecipeRequirement(t *testing.T) {
	r := NewRoom("test")
	r.Containers = map[string]Container{
		"hearth": {Recipes: map[int][]int{5: {1}},
			RecipeRequirements: map[int]RecipeRequirement{5: {SkillId: "cooking", MinLevel: -1}}},
	}
	err := r.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hearth")
}
