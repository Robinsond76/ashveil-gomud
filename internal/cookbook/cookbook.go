// Package cookbook is Phase 56's recipe discovery: the recipes a character
// has learned, and the rules for trying an ingredient combination.
//
// A recipe is YAML data (a hearth container's recipes, the camp's
// CampRecipes). Dishes that need no more than Cooking 1 are common
// knowledge; every other dish must be learned, by working out its
// ingredients (`cook`) or by reading a recipe page (an item with `recipe:`).
// The book lives in the character's MiscData, so it follows the character
// through restarts and is the same at a hearth and at camp.
package cookbook

import (
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
)

const (
	// BookKey is the MiscData key holding the learned dish item IDs.
	BookKey = "recipebook"
	// MaxIngredients is the most ingredients one attempt takes.
	MaxIngredients = 4
	// BasicLevel is the highest Cooking level a dish may need and still be
	// common knowledge.
	BasicLevel = 1
	// MakeshiftMealItemId is what a failed attempt makes: food, no buff.
	MakeshiftMealItemId = items.MakeshiftMealItemId
)

// Recipe is one dish: its output item, its ingredient items (a repeat means
// two of that ingredient) and the skill level it takes.
type Recipe struct {
	Output   int
	Inputs   []int
	Skill    string
	MinLevel int
}

// Basic reports whether everyone knows the dish from the start.
func (r Recipe) Basic() bool { return r.MinLevel <= BasicLevel }

// Learned lists the dishes a character has learned, ascending.
func Learned(c *characters.Character) []int {
	if c == nil {
		return nil
	}
	raw, _ := c.GetMiscData(BookKey).(string)
	var out []int
	for _, part := range strings.Split(raw, ",") {
		if id, err := strconv.Atoi(strings.TrimSpace(part)); err == nil && id > 0 {
			out = append(out, id)
		}
	}
	sort.Ints(out)
	return out
}

// Knows reports whether the character can cook the recipe without
// experimenting: it is basic or in the book.
func Knows(c *characters.Character, r Recipe) bool {
	if r.Basic() {
		return true
	}
	for _, id := range Learned(c) {
		if id == r.Output {
			return true
		}
	}
	return false
}

// Learn writes a dish into the character's book; false when it was already
// there.
func Learn(c *characters.Character, output int) bool {
	if c == nil || output < 1 {
		return false
	}
	have := Learned(c)
	for _, id := range have {
		if id == output {
			return false
		}
	}
	have = append(have, output)
	sort.Ints(have)
	parts := make([]string, len(have))
	for i, id := range have {
		parts[i] = strconv.Itoa(id)
	}
	c.SetMiscData(BookKey, strings.Join(parts, ","))
	return true
}

// Known filters recipes to those the character can cook, keeping order.
func Known(c *characters.Character, recipes []Recipe) []Recipe {
	var out []Recipe
	for _, r := range recipes {
		if Knows(c, r) {
			out = append(out, r)
		}
	}
	return out
}

// Match finds the recipe whose ingredients are exactly the given items,
// whatever the order.
func Match(inputs []int, recipes []Recipe) (Recipe, bool) {
	want := append([]int(nil), inputs...)
	sort.Ints(want)
	for _, r := range recipes {
		have := append([]int(nil), r.Inputs...)
		sort.Ints(have)
		if len(have) != len(want) {
			continue
		}
		same := true
		for i := range have {
			if have[i] != want[i] {
				same = false
				break
			}
		}
		if same {
			return r, true
		}
	}
	return Recipe{}, false
}

// IsIngredient reports whether an item can go into a pot: raw game, herbs
// and other plain food. A cooked meal, a tool or a weapon never can.
func IsIngredient(spec *items.ItemSpec) bool {
	if spec == nil || spec.Meal != "" || spec.Recipe > 0 || spec.Hydration > 0 || spec.ItemId == items.MakeshiftMealItemId {
		return false
	}
	switch {
	case spec.Type == items.Botanical:
		return true
	case spec.Goods == "provision":
		return true
	case spec.Type == items.Food && spec.Nutrition > 0:
		return true
	}
	return false
}

// Stack is a count of one ingredient item the cook has to hand.
type Stack struct {
	ItemID int
	Count  int
}

// Resolve turns the words after `cook` into ingredient item IDs, one entry
// per portion: "meat meat thyme" and "2 meat thyme" both give two meat and
// a thyme. A word names an ingredient by its short name, any word of its
// name, or the start of one. It fails with a sentence for the player when a
// word names nothing the cook holds, more are named than are held, or the
// attempt is empty or too large.
func Resolve(words []string, stock []Stack) ([]int, string) {
	have := map[int]int{}
	var order []int
	for _, s := range stock {
		spec := items.GetItemSpec(s.ItemID)
		if s.Count < 1 || !IsIngredient(spec) {
			continue
		}
		if _, seen := have[s.ItemID]; !seen {
			order = append(order, s.ItemID)
		}
		have[s.ItemID] += s.Count
	}
	sort.Ints(order)
	var picked []int
	qty := 1
	for _, word := range words {
		word = strings.ToLower(strings.TrimSpace(word))
		if word == "" {
			continue
		}
		if n, err := strconv.Atoi(word); err == nil {
			if n < 1 || n > MaxIngredients {
				return nil, "Use between 1 and 4 of an ingredient."
			}
			qty = n
			continue
		}
		id := nameMatch(word, order)
		if id == 0 {
			return nil, `You have no ingredient called "` + word + `" to cook with. Ingredients are raw game, fish and herbs you carry or keep in company cargo.`
		}
		for i := 0; i < qty; i++ {
			picked = append(picked, id)
		}
		qty = 1
	}
	if len(picked) == 0 {
		return nil, "Name what to cook, such as: cook meat thyme"
	}
	if len(picked) > MaxIngredients {
		return nil, "A pot takes at most 4 ingredients at a time."
	}
	used := map[int]int{}
	for _, id := range picked {
		used[id]++
		if used[id] > have[id] {
			spec := items.GetItemSpec(id)
			return nil, "You don't have " + strconv.Itoa(used[id]) + " " + spec.Name + "."
		}
	}
	return picked, ""
}

// nameMatch picks the ingredient a word names: an exact short name first,
// then a whole word of a name, then the start of one.
func nameMatch(word string, candidates []int) int {
	for pass := 0; pass < 3; pass++ {
		for _, id := range candidates {
			spec := items.GetItemSpec(id)
			if spec == nil {
				continue
			}
			switch pass {
			case 0:
				if strings.EqualFold(spec.NameSimple, word) {
					return id
				}
			case 1:
				for _, w := range strings.Fields(strings.ToLower(spec.Name)) {
					if w == word {
						return id
					}
				}
			case 2:
				if strings.HasPrefix(strings.ToLower(spec.Name), word) || strings.HasPrefix(strings.ToLower(spec.NameSimple), word) {
					return id
				}
			}
		}
	}
	return 0
}

// Describe lists a recipe's ingredients as "2 raw game meat, 1 wild thyme".
func Describe(inputs []int) string {
	counts := map[int]int{}
	var ids []int
	for _, id := range inputs {
		if counts[id] == 0 {
			ids = append(ids, id)
		}
		counts[id]++
	}
	sort.Ints(ids)
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		name := strconv.Itoa(id)
		if spec := items.GetItemSpec(id); spec != nil {
			name = spec.Name
		}
		parts = append(parts, strconv.Itoa(counts[id])+" "+name)
	}
	return strings.Join(parts, ", ")
}
