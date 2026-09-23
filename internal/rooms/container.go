package rooms

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/gamelock"
	"github.com/GoMudEngine/GoMud/internal/items"
)

type Container struct {
	Lock         gamelock.Lock `yaml:"lock,omitempty"`         // 0 - no lock. greater than zero = difficulty to unlock.
	Items        []items.Item  `yaml:"items,omitempty"`        // Save contents now, since players can put new items in there
	Gold         int           `yaml:"gold,omitempty"`         // Save contents now, since players can put new items in there
	DespawnRound uint64        `yaml:"despawnround,omitempty"` // If this is set, it's a chest that will disappear with time.
	Recipes      map[int][]int `yaml:"recipes,omitempty,flow"` // Item Id's (key) that are created when the recipe is present in the container (values) and it is "used"
	// Optional skill gates, keyed by recipe output item id. Recipes without an entry are ungated.
	RecipeRequirements map[int]RecipeRequirement `yaml:"reciperequirements,omitempty"`
}

// RecipeRequirement is the minimum skill level needed to make one recipe.
type RecipeRequirement struct {
	SkillId  string `yaml:"skillid"`
	MinLevel int    `yaml:"minlevel"`
}

func (c Container) HasLock() bool {
	return c.Lock.Difficulty > 0
}

func (c *Container) AddItem(i items.Item) {
	c.Items = append(c.Items, i)
}

func (c *Container) RemoveItem(i items.Item) {
	for j := len(c.Items) - 1; j >= 0; j-- {
		if c.Items[j].Equals(i) {
			c.Items = append(c.Items[:j], c.Items[j+1:]...)
			break
		}
	}
}

func (c *Container) FindItem(itemName string) (items.Item, bool) {

	// Search floor
	closeMatchItem, matchItem := items.FindMatchIn(itemName, c.Items...)

	if matchItem.ItemId != 0 {
		return matchItem, true
	}

	if closeMatchItem.ItemId != 0 {
		return closeMatchItem, true
	}

	return items.Item{}, false
}

func (c *Container) FindItemById(itemId int) (items.Item, bool) {

	// Search floor
	for _, matchItem := range c.Items {
		if matchItem.ItemId == itemId {
			return matchItem, true
		}
	}

	return items.Item{}, false
}

// Returns the lowest output itemId whose recipe is satisfied by the contents, or 0.
// Ignores skill requirements; use SelectRecipe to honor them.
func (c *Container) RecipeReady() int {
	if ready := c.ReadyRecipes(); len(ready) > 0 {
		return ready[0]
	}
	return 0
}

// ReadyRecipes returns every output itemId whose inputs are all present, ascending.
func (c *Container) ReadyRecipes() []int {

	ready := []int{}

	for finalItemId, recipeList := range c.Recipes {

		neededItems := map[int]int{}
		for _, inputItemId := range recipeList {
			neededItems[inputItemId] += 1
		}

		satisfied := true
		for inputItemId, qty := range neededItems {
			if c.Count(inputItemId) < qty {
				satisfied = false
				break
			}
		}

		if satisfied {
			ready = append(ready, finalItemId)
		}
	}

	sort.Ints(ready)
	return ready
}

// SelectRecipe picks the lowest ready output itemId whose requirement the actor meets,
// given a skill-level lookup. When recipes are ready but all are gated, it returns 0 and
// the easiest unmet requirement (lowest level, then lowest output id).
func (c *Container) SelectRecipe(skillLevel func(skillId string) int) (int, RecipeRequirement) {

	var blocked RecipeRequirement

	for _, finalItemId := range c.ReadyRecipes() {
		req, gated := c.RecipeRequirements[finalItemId]
		if !gated || skillLevel(req.SkillId) >= req.MinLevel {
			return finalItemId, RecipeRequirement{}
		}
		if blocked.MinLevel == 0 || req.MinLevel < blocked.MinLevel {
			blocked = req
		}
	}

	return 0, blocked
}

// ValidateRecipes rejects malformed recipes and requirements, normalizing skill ids.
func (c *Container) ValidateRecipes() error {

	for finalItemId, recipeList := range c.Recipes {
		if finalItemId < 1 {
			return fmt.Errorf("recipe output item id %d must be positive", finalItemId)
		}
		if len(recipeList) == 0 {
			return fmt.Errorf("recipe for item %d has no inputs", finalItemId)
		}
		for _, inputItemId := range recipeList {
			if inputItemId < 1 {
				return fmt.Errorf("recipe for item %d has invalid input item id %d", finalItemId, inputItemId)
			}
		}
	}

	for finalItemId, req := range c.RecipeRequirements {
		if _, ok := c.Recipes[finalItemId]; !ok {
			return fmt.Errorf("recipe requirement for item %d has no matching recipe", finalItemId)
		}
		req.SkillId = strings.ToLower(strings.TrimSpace(req.SkillId))
		if req.SkillId == `` {
			return fmt.Errorf("recipe requirement for item %d has no skillid", finalItemId)
		}
		if req.MinLevel < 1 {
			return fmt.Errorf("recipe requirement for item %d needs minlevel of at least 1", finalItemId)
		}
		c.RecipeRequirements[finalItemId] = req
	}

	return nil
}

func (c *Container) Count(itemId int) int {
	total := 0
	for _, containsItem := range c.Items {
		if containsItem.ItemId == itemId {
			total++
		}
	}
	return total
}
