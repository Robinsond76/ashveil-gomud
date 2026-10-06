package company

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 32f: `company eat`, `company drink`, and `company meal` feed and
// water every living member present (the leader and each companion
// walking with them) from the cargo, then the member's own pack, then the
// leader's (owner, 2026-09-28).

type mealKind int

const (
	mealEat mealKind = iota
	mealDrink
	mealBoth
)

// larderSource is where a meal comes from, in the order it is drawn on.
type larderSource int

const (
	fromCargo larderSource = iota
	fromOwnPack
	fromLeaderPack
)

// mealBuffs are the only buffs a meal may carry: Well Fed (17), Very Well
// Fed (18), and Hydrated (34). Anything else (a potion, ale) is a choice
// made by hand.
var mealBuffs = map[int]bool{17: true, 18: true, 34: true}

// larderItem is one thing the company could eat or drink, with the uses
// the planner may still draw from it.
type larderItem struct {
	Source larderSource
	// Owner is the companion who carries it, for fromOwnPack.
	Owner     survival.MemberKey
	ItemId    int
	Item      items.Item // the carried instance; zero for cargo
	Name      string
	Edible    bool
	Drinkable bool
	Nutrition int
	Hydration int
	BuffIds   []int
	Uses      int
}

// mealStep is one member eating or drinking one use of one larder item.
type mealStep struct {
	Member survival.MemberNeeds
	Drink  bool
	Food   int // index into the larder
}

// mealPlan is the steps in order, and who still goes without.
type mealPlan struct {
	Steps   []mealStep
	Hungry  []string
	Thirsty []string
}

// larderEntry builds an eligible larder item from a spec, or false: it
// must feed or water, and carry no buff but a meal's.
func larderEntry(spec items.ItemSpec) (larderItem, bool) {
	edible := spec.Subtype == items.Edible && spec.Nutrition > 0
	drinkable := spec.Subtype == items.Drinkable && spec.Hydration > 0
	if !edible && !drinkable {
		return larderItem{}, false
	}
	for _, id := range spec.BuffIds {
		if !mealBuffs[id] {
			return larderItem{}, false
		}
	}
	return larderItem{
		ItemId: spec.ItemId, Name: spec.Name, Edible: edible, Drinkable: drinkable,
		Nutrition: spec.Nutrition, Hydration: spec.Hydration, BuffIds: spec.BuffIds,
	}, true
}

// planMeal decides who eats and drinks what, without doing it. Members
// most in need go first; one already at the top band is skipped. Each
// member draws one use from the first source (cargo, their own pack, the
// leader's pack) that has something suitable: the smallest item that
// covers their need, else the largest. Food that also waters counts
// toward thirst before anyone drinks.
func planMeal(needs []survival.MemberNeeds, larder []larderItem, kind mealKind) mealPlan {
	larder = append([]larderItem(nil), larder...)
	current := append([]survival.MemberNeeds(nil), needs...)
	plan := mealPlan{}
	pass := func(drink bool) {
		value := func(n survival.Needs) int {
			if drink {
				return n.Thirst
			}
			return n.Hunger
		}
		order := make([]int, len(current))
		for i := range order {
			order[i] = i
		}
		sort.SliceStable(order, func(a, b int) bool {
			return value(current[order[a]].Needs) < value(current[order[b]].Needs)
		})
		for _, i := range order {
			member := &current[i]
			if survival.BandFor(value(member.Needs)) == survival.BandFull {
				continue
			}
			pick := pickFood(larder, member.Key, 100-value(member.Needs), drink)
			if pick < 0 {
				if drink {
					plan.Thirsty = append(plan.Thirsty, member.Name)
				} else {
					plan.Hungry = append(plan.Hungry, member.Name)
				}
				continue
			}
			plan.Steps = append(plan.Steps, mealStep{Member: *member, Drink: drink, Food: pick})
			larder[pick].Uses--
			benefit := mealBenefit(larder[pick], drink)
			member.Needs.Hunger = min(100, member.Needs.Hunger+benefit.Nutrition)
			member.Needs.Thirst = min(100, member.Needs.Thirst+benefit.Hydration)
		}
	}
	if kind != mealDrink {
		pass(false)
	}
	if kind != mealEat {
		pass(true)
	}
	return plan
}

// pickFood is the larder index a member eats (or drinks) from, or -1.
func pickFood(larder []larderItem, member survival.MemberKey, deficit int, drink bool) int {
	for _, source := range []larderSource{fromCargo, fromOwnPack, fromLeaderPack} {
		cover, largest := -1, -1
		for i, food := range larder {
			if food.Uses <= 0 || food.Source != source || (source == fromOwnPack && food.Owner != member) {
				continue
			}
			amount := food.Nutrition
			if drink {
				if !food.Drinkable {
					continue
				}
				amount = food.Hydration
			} else if !food.Edible {
				continue
			}
			if amount >= deficit && (cover < 0 || amount < mealAmount(larder[cover], drink)) {
				cover = i
			}
			if largest < 0 || amount > mealAmount(larder[largest], drink) {
				largest = i
			}
		}
		if cover >= 0 {
			return cover
		}
		if largest >= 0 {
			return largest
		}
	}
	return -1
}

func mealAmount(food larderItem, drink bool) int {
	if drink {
		return food.Hydration
	}
	return food.Nutrition
}

// mealBenefit is what one use gives: food gives both its nutrition and
// water, as `eat` does; a drink gives its water, as `drink` does.
func mealBenefit(food larderItem, drink bool) survival.Benefit {
	if drink {
		return survival.Benefit{Hydration: food.Hydration}
	}
	return survival.Benefit{Nutrition: food.Nutrition, Hydration: food.Hydration}
}

// companionIDOf is a companion member key's id; false for the leader.
func companionIDOf(key survival.MemberKey) (int, bool) {
	id, err := strconv.Atoi(strings.TrimPrefix(string(key), "companion:"))
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// larderFor gathers what the company could eat: the cargo, each living
// companion's own pack (live, else recorded), and the leader's pack.
func (m *CompanyModule) larderFor(user *users.UserRecord, needs []survival.MemberNeeds) []larderItem {
	if user.Character.CompanyCargo {
		return packLarder(user.Character.Items, fromLeaderPack, survival.LeaderMemberKey)
	}
	larder := []larderItem{}
	for _, stack := range encumbrance.CargoContents(user.UserId) {
		spec := items.GetItemSpec(stack.ItemId)
		if spec == nil {
			continue
		}
		food, ok := larderEntry(*spec)
		if !ok {
			continue
		}
		per := stack.Uses
		if per == 0 {
			per = max(spec.Uses, 1)
		}
		food.Source, food.Uses = fromCargo, per*stack.Count
		larder = append(larder, food)
	}
	for _, member := range needs {
		id, ok := companionIDOf(member.Key)
		if !ok {
			continue
		}
		carried, ok := m.carriedBy(user.UserId, id)
		if !ok {
			continue
		}
		larder = append(larder, packLarder(carried, fromOwnPack, member.Key)...)
	}
	return append(larder, packLarder(user.Character.Items, fromLeaderPack, survival.LeaderMemberKey)...)
}

func packLarder(carried []items.Item, source larderSource, owner survival.MemberKey) []larderItem {
	out := []larderItem{}
	for _, itm := range carried {
		if itm.ItemId <= 0 {
			continue
		}
		food, ok := larderEntry(itm.GetSpec())
		if !ok {
			continue
		}
		food.Source, food.Owner, food.Item = source, owner, itm
		food.Name = itm.DisplayName()
		food.Uses = max(itm.Uses, 1)
		out = append(out, food)
	}
	return out
}

// carriedBy is a companion's carried items: the live mob's when it is out
// and still the company's, else its record's. False when it has none to
// offer (charmed away, fallen, unknown).
func (m *CompanyModule) carriedBy(leaderUserID, companionID int) ([]items.Item, bool) {
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return nil, false
	}
	for _, c := range record.Companions {
		if c.ID != companionID || c.Dead() {
			continue
		}
		if instanceID, tracked := m.instance(leaderUserID, c.ID); tracked && m.runtime.IsLive(instanceID) {
			if m.runtime.CharmedByOther(leaderUserID, instanceID) {
				return nil, false
			}
			if state, ok := m.runtime.Snapshot(instanceID); ok {
				return state.Items, true
			}
		}
		if c.State != nil {
			return c.State.Items, true
		}
	}
	return nil, false
}

// useCompanionItem takes one use of a companion's own item: from the live
// mob when it is out, else from its record, saved.
func (m *CompanyModule) useCompanionItem(leaderUserID, companionID int, itm items.Item) bool {
	if instanceID, tracked := m.instance(leaderUserID, companionID); tracked && m.runtime.IsLive(instanceID) {
		if !m.runtime.UseItem(instanceID, itm) {
			return false
		}
		// Record the gear now, as a gear change does (32f review
		// finding 7): a live mob's UseItem fires no ItemOwnership.
		m.refreshSnapshot(leaderUserID, companionID)
		return true
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return false
	}
	for _, c := range record.Companions {
		if c.ID != companionID || c.State == nil {
			continue
		}
		before := c.State.Clone()
		state := c.State.Clone()
		for i := range state.Items {
			if !state.Items[i].Equals(itm) {
				continue
			}
			if state.Items[i].Uses > 1 {
				state.Items[i].Uses--
			} else {
				state.Items = append(state.Items[:i], state.Items[i+1:]...)
			}
			if m.registry.SetState(leaderUserID, companionID, state) != nil {
				return false
			}
			if err := m.save(); err != nil {
				_ = m.registry.SetState(leaderUserID, companionID, before)
				mudlog.Error("company: meal", "error", err)
				return false
			}
			return true
		}
	}
	return false
}

// presentNeeds keeps the leader and the companions walking with them
// (32f review finding 6): one elsewhere, or not out, eats on their own.
func (m *CompanyModule) presentNeeds(leaderUserID int, needs []survival.MemberNeeds) []survival.MemberNeeds {
	present := map[int]bool{}
	for _, id := range m.CompanionsWithLeader(leaderUserID) {
		present[id] = true
	}
	out := make([]survival.MemberNeeds, 0, len(needs))
	for _, n := range needs {
		if id, isCompanion := companionIDOf(n.Key); isCompanion && !present[id] {
			continue
		}
		out = append(out, n)
	}
	return out
}

// mealView runs a meal: plan, then for each step use the item and only
// then provision the member, so no one is fed from food that wasn't spent
// (32f review finding 7). It never holds another module's lock across the
// steps: survival and encumbrance each lock and save per call.
func (m *CompanyModule) mealView(user *users.UserRecord, room *rooms.Room, kind mealKind) string {
	// Phase 32d: a battle plays out as it was set up; no meals in one.
	if usercommands.InBattle(user) {
		return usercommands.BattleUnderWay
	}
	needs := survival.CompanyNeeds(user.UserId)
	if len(needs) == 0 {
		return "Your company's needs can't be read right now."
	}
	needs = m.presentNeeds(user.UserId, needs)
	larder := m.larderFor(user, needs)
	// Phase 40a: at a water source everyone drinks from it first, and no
	// drinking item is spent; the plan only feeds.
	atSource := kind != mealEat && usercommands.HasWater(room)
	planKind := kind
	if atSource {
		planKind = mealEat
	}
	plan := mealPlan{}
	if !atSource || kind == mealBoth {
		plan = planMeal(needs, larder, planKind)
	}

	lines := []string{}
	ate, drank := false, false
	for _, step := range plan.Steps {
		food := larder[step.Food]
		companionID, isCompanion := companionIDOf(step.Member.Key)
		if !m.stillThere(user, food) {
			continue
		}
		selector := ""
		if isCompanion {
			selector = "#" + strconv.Itoa(companionID)
		}
		if !m.useFood(user, food) {
			mudlog.Warn("company: meal item not used", "leader", user.UserId, "item", food.ItemId)
			lines = append(lines, fmt.Sprintf("%s couldn't get at the %s.", step.Member.Name, food.Name))
			continue
		}
		result, err := survival.Provision(user.UserId, selector, mealBenefit(food, step.Drink))
		if err != nil {
			lines = append(lines, fmt.Sprintf("%s couldn't be provisioned: %s", step.Member.Name, err))
			continue
		}
		if step.Drink {
			drank = true
		} else {
			ate = true
		}
		if !isCompanion {
			user.Character.CancelBuffsWithFlag("hidden")
			flag := "food"
			if step.Drink {
				flag = "drink"
			}
			for _, buffId := range food.BuffIds {
				user.AddBuff(buffId, flag)
			}
		}
		lines = append(lines, mealLine(step, food, result, isCompanion))
	}
	if atSource {
		if sourceLines := m.waterFromSource(user); len(sourceLines) > 0 {
			drank = true
			lines = append(lines, sourceLines...)
		}
	}
	for _, name := range plan.Hungry {
		lines = append(lines, fmt.Sprintf("%s is still hungry; there's nothing left to eat.", name))
	}
	for _, name := range plan.Thirsty {
		lines = append(lines, fmt.Sprintf("%s is still thirsty; there's nothing left to drink.", name))
	}
	if len(lines) == 0 {
		switch kind {
		case mealEat:
			return "No one in your company is hungry."
		case mealDrink:
			return "No one in your company is thirsty."
		}
		return "No one in your company is hungry or thirsty."
	}
	if room != nil && (ate || drank) {
		verb := "eats"
		switch {
		case ate && drank:
			verb = "eats and drinks"
		case drank:
			verb = "drinks"
		}
		room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi>'s company %s.`, user.Character.Name, verb), user.UserId)
	}
	return strings.Join(lines, "\n")
}

// stillThere checks a planned item is still where the plan found it, so
// no one is provisioned from food that has gone.
func (m *CompanyModule) stillThere(user *users.UserRecord, food larderItem) bool {
	switch food.Source {
	case fromCargo:
		for _, s := range encumbrance.CargoContents(user.UserId) {
			if s.ItemId == food.ItemId {
				return true
			}
		}
		return false
	case fromOwnPack:
		ownerID, _ := companionIDOf(food.Owner)
		carried, ok := m.carriedBy(user.UserId, ownerID)
		return ok && hasItem(carried, food.Item)
	}
	return hasItem(user.Character.Items, food.Item)
}

func hasItem(carried []items.Item, itm items.Item) bool {
	for i := range carried {
		if carried[i].Equals(itm) {
			return true
		}
	}
	return false
}

// useFood takes the use a provisioned member ate or drank.
func (m *CompanyModule) useFood(user *users.UserRecord, food larderItem) bool {
	switch food.Source {
	case fromCargo:
		if err := encumbrance.ConsumeCargoUse(user.UserId, food.ItemId); err != nil {
			mudlog.Error("company: meal", "error", err)
			return false
		}
		return true
	case fromOwnPack:
		ownerID, _ := companionIDOf(food.Owner)
		return m.useCompanionItem(user.UserId, ownerID, food.Item)
	}
	if user.Character.CompanyCargo {
		return encumbrance.ConsumeCargoItemUse(user.UserId, food.Item) == nil
	}
	// The leader's pack, the way `eat` uses it.
	for i := range user.Character.Items {
		if user.Character.Items[i].Equals(food.Item) {
			current := user.Character.Items[i]
			if user.Character.UseItem(current) < 1 {
				events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: current, Gained: false})
			}
			return true
		}
	}
	return false
}

func mealLine(step mealStep, food larderItem, result survival.ProvisionResult, isCompanion bool) string {
	where := "cargo"
	switch food.Source {
	case fromOwnPack:
		where = "own pack"
	case fromLeaderPack:
		where = "company cargo"
	}
	subject, verb := fmt.Sprintf(`<ansi fg="username">%s</ansi>`, result.Name), "eats a"
	if step.Drink {
		verb = "drinks from a"
	}
	if !isCompanion {
		subject, verb = "You", "eat some of the"
		if step.Drink {
			verb = "drink from the"
		}
	}
	status := "Hunger: " + survival.HungerLabel(result.Needs.Hunger)
	if step.Drink {
		status = "Thirst: " + survival.ThirstLabel(result.Needs.Thirst)
	}
	return fmt.Sprintf(`%s %s <ansi fg="itemname">%s</ansi> (%s). %s.`, subject, verb, food.Name, where, status)
}

// waterFromSource waters every present member below the top thirst band
// from the room's water, costing nothing (Phase 40a). It reads thirst
// fresh, after any meal's own water.
func (m *CompanyModule) waterFromSource(user *users.UserRecord) []string {
	needs := survival.CompanyNeeds(user.UserId)
	if len(needs) == 0 {
		return nil
	}
	needs = m.presentNeeds(user.UserId, needs)
	lines := []string{}
	for _, member := range needs {
		if survival.BandFor(member.Needs.Thirst) == survival.BandFull {
			continue
		}
		selector := ""
		companionID, isCompanion := companionIDOf(member.Key)
		if isCompanion {
			selector = "#" + strconv.Itoa(companionID)
		}
		result, err := survival.Provision(user.UserId, selector, survival.Benefit{Hydration: usercommands.WaterSourceHydration})
		if err != nil {
			lines = append(lines, fmt.Sprintf("%s couldn't be watered: %s", member.Name, err))
			continue
		}
		thirst := "Thirst: " + survival.ThirstLabel(result.Needs.Thirst)
		if isCompanion {
			lines = append(lines, fmt.Sprintf(`<ansi fg="username">%s</ansi> drinks from the water here. %s.`, result.Name, thirst))
			continue
		}
		user.Character.CancelBuffsWithFlag("hidden")
		user.AddBuff(usercommands.WaterSourceBuff, "drink")
		lines = append(lines, "You drink from the water here. "+thirst+".")
	}
	return lines
}
