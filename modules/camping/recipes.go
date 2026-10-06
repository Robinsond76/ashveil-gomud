package camping

// Phase 56 recipe discovery: `cook [ingredient]...` at a hearth or the
// leader's lit campfire tries a combination, `camp cook` makes what the
// leader already knows, and `recipes` lists the book. The book itself is
// internal/cookbook; the recipes stay YAML (CampRecipes and a hearth
// container's recipes).

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/cookbook"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func (r campRecipe) book() cookbook.Recipe {
	return cookbook.Recipe{Output: r.Output, Inputs: r.Inputs, Skill: r.Skill, MinLevel: r.MinLevel}
}

// knownCampRecipes keeps, in order, the dishes the leader can cook without
// trying them out.
func knownCampRecipes(user *users.UserRecord, recipes []campRecipe) []campRecipe {
	if user == nil || user.Character == nil {
		return nil
	}
	var out []campRecipe
	for _, r := range recipes {
		if cookbook.Knows(user.Character, r.book()) {
			out = append(out, r)
		}
	}
	return out
}

// campStock is the ingredients to hand: pack items, then cargo stacks.
func campStock(user *users.UserRecord) []cookbook.Stack {
	have := map[int]int{}
	if !user.Character.CompanyCargo {
		for _, itm := range user.Character.Items {
			have[itm.ItemId]++
		}
	}
	for _, s := range encumbrance.CargoContents(user.UserId) {
		have[s.ItemId] += s.Count
	}
	out := make([]cookbook.Stack, 0, len(have))
	for id, n := range have {
		out = append(out, cookbook.Stack{ItemID: id, Count: n})
	}
	return out
}

// tryCombination is one attempt: the exact ingredients named make the dish
// that matches (learning it the first time), or a makeshift meal when none
// does. A dish above the cook's rank, or a failed save, consumes nothing
// and teaches nothing.
func (m *CampingModule) tryCombination(user *users.UserRecord, room *rooms.Room, cook campCook, words []string, recipes []campRecipe, where string) string {
	if room == nil {
		return "You can't cook here."
	}
	inputs, problem := cookbook.Resolve(words, campStock(user))
	if problem != "" {
		return problem
	}
	var all []cookbook.Recipe
	for _, r := range recipes {
		all = append(all, r.book())
	}
	matched, ok := cookbook.Match(inputs, all)
	if !ok {
		makeshift := campRecipe{Output: items.MakeshiftMealItemId, Inputs: inputs}
		text, _ := m.cookRecipe(user, room, cook, &makeshift, where, " Nothing you know comes of that mix, so it is only a makeshift meal.")
		return text
	}
	if matched.Skill != "" && cook.rank(matched.Skill) < matched.MinLevel {
		return fmt.Sprintf("That mix might make something, but it needs %s %d, and %s.", matched.Skill, matched.MinLevel, bestRankText(cook, matched.Skill))
	}
	chosen := campRecipe{Output: matched.Output, Inputs: matched.Inputs, Skill: matched.Skill, MinLevel: matched.MinLevel}
	fresh := !cookbook.Knows(user.Character, matched)
	note := ""
	if fresh {
		note = fmt.Sprintf(` You have worked out a new recipe: <ansi fg="itemname">%s</ansi> (%s). It is in your recipe book (<ansi fg="command">recipes</ansi>).`, itemName(matched.Output), cookbook.Describe(matched.Inputs))
	}
	text, done := m.cookRecipe(user, room, cook, &chosen, where, note)
	if done && fresh {
		cookbook.Learn(user.Character, matched.Output)
	}
	return text
}

// experiment is `camp cook` with ingredients named, at the leader's camp.
func (m *CampingModule) experiment(user *users.UserRecord, room *rooms.Room, cook campCook, words []string) string {
	return m.tryCombination(user, room, cook, words, m.campSettings().Recipes, "over the campfire")
}

// hearthRecipes lists a room's hearth dishes in the cookbook's shape; the
// first container with recipes is the hearth.
func hearthRecipes(room *rooms.Room) (string, []campRecipe) {
	if room == nil {
		return "", nil
	}
	names := make([]string, 0, len(room.Containers))
	for name := range room.Containers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		c := room.Containers[name]
		if len(c.Recipes) == 0 {
			continue
		}
		outs := make([]int, 0, len(c.Recipes))
		for out := range c.Recipes {
			outs = append(outs, out)
		}
		sort.Ints(outs)
		var recipes []campRecipe
		for _, out := range outs {
			r := campRecipe{Output: out, Inputs: c.Recipes[out]}
			if req, gated := c.RecipeRequirements[out]; gated {
				r.Skill, r.MinLevel = req.SkillId, req.MinLevel
			}
			recipes = append(recipes, r)
		}
		return name, recipes
	}
	return "", nil
}

// cookCommand is `cook [ingredient]...`: at the leader's lit camp it tries
// a combination with the company's best cook; at a hearth, with the
// player's own Cooking.
func (m *CampingModule) cookCommand(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	words := strings.Fields(strings.ToLower(strings.TrimSpace(rest)))
	if m.inBattle != nil && m.inBattle(user.UserId) {
		user.SendText("You can't cook in the middle of a fight.")
		return true, nil
	}
	m.mu.Lock()
	camp, hasCamp := m.camps[user.UserId]
	m.mu.Unlock()
	if hasCamp && room != nil && camp.RoomID == room.RoomId && camp.FireLit {
		user.SendText(m.cook(user, room, words))
		return true, nil
	}
	if name, recipes := hearthRecipes(room); name != "" {
		if len(words) == 0 {
			user.SendText(fmt.Sprintf("Name what to cook at the %s, such as: cook meat thyme. Use the %s to cook what you already know (help recipes).", name, name))
			return true, nil
		}
		cook := campCook{Name: user.Character.Name, IsLeader: true, Ranks: skillsOf(user.Character.GetSkillLevel, recipes)}
		user.SendText(m.tryCombination(user, room, cook, words, recipes, "at the "+name))
		return true, nil
	}
	user.SendText("You need a hearth or your own lit campfire (camp, then camp fire) to cook. See help recipes.")
	return true, nil
}

// recipesLines lists the dishes the leader knows, and the hint that more
// can be found.
func (m *CampingModule) recipesLines(user *users.UserRecord) []string {
	recipes := knownCampRecipes(user, m.campSettings().Recipes)
	var lines []string
	for _, r := range recipes {
		need := ""
		if r.Skill != "" && r.MinLevel > 0 {
			need = fmt.Sprintf(" (%s %d)", r.Skill, r.MinLevel)
		}
		lines = append(lines, fmt.Sprintf("%s: %s%s", itemName(r.Output), cookbook.Describe(r.Inputs), need))
	}
	// Remedies (Phase 55) are in the same book.
	for _, a := range survival.Ailments() {
		if remedyKnown(user, a) {
			lines = append(lines, fmt.Sprintf("%s (remedy for %s): %s", a.RemedyName, strings.ToLower(a.Name), cookbook.Describe(remedyRecipe(a))))
		}
	}
	return lines
}

// recipesCommand is `recipes`: the book.
func (m *CampingModule) recipesCommand(_ string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	lines := m.recipesLines(user)
	if len(lines) == 0 {
		user.SendText("Your recipe book is empty. Try a combination with cook (help recipes).")
		return true, nil
	}
	user.SendText(`<ansi fg="yellow-bold">Your recipe book</ansi>`)
	for _, l := range lines {
		user.SendText("  " + l)
	}
	user.SendText(`More dishes can be worked out: <ansi fg="command">cook</ansi> a new combination at a hearth or your lit campfire, or learn them from recipe pages (<ansi fg="command">help recipes</ansi>).`)
	return true, nil
}
