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

	"github.com/GoMudEngine/GoMud/internal/camping"
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
// does. A known dish above the cook's rank, or a failed save, consumes
// nothing and teaches nothing; an unknown dish above the cook's rank is a
// miss like any other (56 review).
func (m *CampingModule) tryCombination(user *users.UserRecord, room *rooms.Room, cook campCook, words []string, recipes []campRecipe, where string) string {
	if room == nil {
		return "You can't cook here."
	}
	inputs, problem := cookbook.Resolve(words, campStock(user))
	if problem != "" {
		return problem
	}
	// 56 review: herbs only season a dish. Every dish has game or fish in
	// it, so this refusal reveals nothing, and a herb is never 25 Hunger.
	if !cookbook.HasFood(inputs) {
		return "Herbs alone only season a pot. Add raw game or fish: cook meat thyme."
	}
	var all []cookbook.Recipe
	for _, r := range recipes {
		all = append(all, r.book())
	}
	matched, ok := cookbook.Match(inputs, all)
	fresh := ok && !cookbook.Knows(user.Character, matched)
	if ok && matched.Skill != "" && cook.rank(matched.Skill) < matched.MinLevel {
		if fresh {
			// 56 review: an unknown dish above the cook's rank is a plain
			// miss, so a refusal never confirms a mix for free.
			ok = false
		} else {
			return fmt.Sprintf("You know that dish, but it needs %s %d, and %s. Nothing was used.", matched.Skill, matched.MinLevel, bestRankText(cook, matched.Skill))
		}
	}
	if !ok {
		makeshift := campRecipe{Output: items.MakeshiftMealItemId, Inputs: inputs}
		text, _ := m.cookRecipe(user, room, cook, &makeshift, where, " Nothing you know comes of that mix, so it is only a makeshift meal.")
		return text
	}
	chosen := campRecipe{Output: matched.Output, Inputs: matched.Inputs, Skill: matched.Skill, MinLevel: matched.MinLevel}
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
// first hearth container (rooms.Container.IsHearth) is the hearth.
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
		if !c.IsHearth(name) { // 56 review: a loom is not a hearth
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
	rows := m.recipeRows(user)
	lines := make([]string, 0, len(rows))
	for _, r := range rows {
		parts := make([]string, 0, len(r.Needs))
		for _, n := range r.Needs {
			parts = append(parts, fmt.Sprintf("%d %s", n.Count, n.Name))
		}
		need := ""
		if r.Skill != "" && r.Level > 0 {
			need = fmt.Sprintf(" (%s %d)", r.Skill, r.Level)
		}
		switch r.Kind {
		case "hearth":
			lines = append(lines, r.Name+" (at a hearth)")
		case "remedy":
			lines = append(lines, fmt.Sprintf("%s (remedy for %s): %s", r.Name, r.For, strings.Join(parts, ", ")))
		default:
			lines = append(lines, fmt.Sprintf("%s: %s%s", r.Name, strings.Join(parts, ", "), need))
		}
	}
	return lines
}

// bookNeeds counts a recipe's ingredients against what is to hand.
func bookNeeds(inputs []int, have func(itemID int) int) ([]camping.RecipeNeed, bool) {
	counts := map[int]int{}
	var ids []int
	for _, id := range inputs {
		if counts[id] == 0 {
			ids = append(ids, id)
		}
		counts[id]++
	}
	sort.Ints(ids)
	needs := make([]camping.RecipeNeed, 0, len(ids))
	ready := true
	for _, id := range ids {
		n := have(id)
		needs = append(needs, camping.RecipeNeed{Name: itemName(id), Count: counts[id], Have: n})
		if n < counts[id] {
			ready = false
		}
	}
	return needs, ready
}

// recipeRows is the recipe book as structured rows for the Camp tab: dishes,
// remedies and dishes learned at a hearth, each with its ingredients, what
// is to hand, and whether it can be made now.
func (m *CampingModule) recipeRows(user *users.UserRecord) []camping.RecipeRow {
	all := m.campSettings().Recipes
	recipes := knownCampRecipes(user, all)
	stock := map[int]int{}
	for _, s := range campStock(user) {
		stock[s.ItemID] = s.Count
	}
	// Dishes cook from the pack and cargo (campStock); remedies draw herbs
	// from the whole company, companions' packs included (cureWith).
	dishHave := func(id int) int { return stock[id] }
	herbHave := func(id int) int { return m.gearCount(user.UserId, id) }
	cook := m.bestCook(user, all)
	var rows []camping.RecipeRow
	for _, r := range recipes {
		needs, ready := bookNeeds(r.Inputs, dishHave)
		if r.Skill != "" && cook.rank(r.Skill) < r.MinLevel {
			ready = false
		}
		row := camping.RecipeRow{Name: itemName(r.Output), Kind: "dish", Needs: needs, Ready: ready}
		if r.Skill != "" && r.MinLevel > 0 {
			row.Skill, row.Level = r.Skill, r.MinLevel
		}
		rows = append(rows, row)
	}
	// 56 review: a dish learned at a hearth or from a page that the camp
	// does not cook is still in the book.
	inCamp := map[int]bool{}
	for _, r := range all {
		inCamp[r.Output] = true
	}
	for _, id := range cookbook.Learned(user.Character) {
		if _, instrument := camping.InstrumentOf(id); instrument {
			continue // music craft lists the instruments a member can make
		}
		if !inCamp[id] {
			rows = append(rows, camping.RecipeRow{Name: itemName(id), Kind: "hearth"})
		}
	}
	// Remedies (Phase 55) are in the same book.
	for _, a := range survival.Ailments() {
		if remedyKnown(user, a) {
			needs, ready := bookNeeds(remedyRecipe(a), herbHave)
			rows = append(rows, camping.RecipeRow{Name: a.RemedyName, Kind: "remedy", For: strings.ToLower(a.Name), Needs: needs, Ready: ready})
		}
	}
	return rows
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
