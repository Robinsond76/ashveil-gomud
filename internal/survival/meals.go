package survival

import (
	"fmt"
	"strings"
	"sync"
)

// Phase 50: a member's needs set its battle condition when a battle begins,
// and a cooked meal gives a buff that lasts a number of battles (never game
// time). Both are fixed for the battle; nothing here ticks.

// MealSpec is one cooked meal's buff. Exactly the percentages it sets apply.
type MealSpec struct {
	Kind    string // as items' `meal:` field names it
	Name    string // the buff's name, as status and the battle line show it
	Battles int    // battles it lasts
	// DamagePct raises the damage the member's blows and spells deal.
	DamagePct int
	// GuardPct cuts the damage blows and spells deal the member.
	GuardPct int
	// ManaPct restores that share of the member's max mana as a battle begins.
	ManaPct int
}

// Effect is the buff's effect in a few words.
func (m MealSpec) Effect() string {
	var parts []string
	if m.DamagePct > 0 {
		parts = append(parts, fmt.Sprintf("+%d%% damage", m.DamagePct))
	}
	if m.GuardPct > 0 {
		parts = append(parts, fmt.Sprintf("%d%% less damage taken", m.GuardPct))
	}
	if m.ManaPct > 0 {
		parts = append(parts, fmt.Sprintf("%d%% of max mana back as each battle begins", m.ManaPct))
	}
	return strings.Join(parts, ", ")
}

// Meals are the cooked meals' buffs, by kind. Items name a kind with
// `meal:`; one buff at a time, and a new meal replaces it.
var meals = map[string]MealSpec{
	"seared": {Kind: "seared", Name: "Strong", Battles: 2, DamagePct: 5},
	"roast":  {Kind: "roast", Name: "Steady", Battles: 3, DamagePct: 5, GuardPct: 5},
	"stew":   {Kind: "stew", Name: "Hearty", Battles: 3, GuardPct: 10},
	"fish":   {Kind: "fish", Name: "Clear-headed", Battles: 3, ManaPct: 20},
}

// MealFor is the meal buff of a kind.
func MealFor(kind string) (MealSpec, bool) {
	m, ok := meals[kind]
	return m, ok
}

// MealKinds lists every meal kind (for data validation).
func MealKinds() []string {
	return []string{"seared", "roast", "stew", "fish"}
}

// Battle condition penalties, by band: Hungry and Thirsty cost the small
// one, Starving, Parched and worse the clear one. Hunger cuts the damage a
// member deals, thirst makes it take more; fatigue keeps its existing
// hit-chance penalty (FatigueHitPenalty). (A tempo cut for thirst was
// tried: a whole company losing the same early turn swung the even mirror
// from 50% to 12% wins, so thirst works on the damage taken instead.)
const (
	ConditionSmallPct = 5
	ConditionClearPct = 10
)

func needPenalty(value int) int {
	switch BandFor(value) {
	case BandDepleted, BandCritical:
		return ConditionClearPct
	case BandLow:
		return ConditionSmallPct
	}
	return 0
}

// FatigueHitPenalty is the percentage points off a member's physical hit
// chance at a Fatigue value: Tired 5, Exhausted 10, Collapsed 20.
func FatigueHitPenalty(rest int) int {
	switch {
	case rest <= 0:
		return 20
	case rest <= 25:
		return 10
	case rest <= 50:
		return 5
	}
	return 0
}

// Condition is a member's battle condition: what its needs and meal buff
// do in a battle that begins now.
type Condition struct {
	DamagePct int // net percent on the damage it deals (hunger, meal)
	GuardPct  int // net percent less damage it takes (meal; thirst makes it negative)
	ManaPct   int // share of max mana restored as the battle begins (meal)
	HitCut    int // points off its physical hit chance (fatigue)
	HungerCut int // percent off its damage from hunger
	ThirstCut int // percent more damage it takes from thirst
	Meal      MealSpec
	Ailments  []ActiveAilment // Phase 55: ailments the member has
	Words     []string        // what is wrong, as labels: "Hungry", "Parched", "Exhausted"
}

// ActiveAilment is an ailment a member has, with the battles it has left.
type ActiveAilment struct {
	AilmentSpec
	Battles int
}

// Neutral reports whether the condition changes nothing in battle.
func (c Condition) Neutral() bool {
	return c.DamagePct == 0 && c.GuardPct == 0 && c.ManaPct == 0 && c.HitCut == 0
}

// Summary is the condition in one line, with battles left on the meal:
// "Hungry, Thirsty: -5% damage, +5% damage taken; Hearty: 10% less damage
// taken (2 battles)". Empty when neutral and unfed.
func (c Condition) Summary(battles int) string {
	var parts []string
	if len(c.Words) > 0 {
		var fx []string
		if c.HungerCut > 0 {
			fx = append(fx, fmt.Sprintf("-%d%% damage", c.HungerCut))
		}
		if c.ThirstCut > 0 {
			fx = append(fx, fmt.Sprintf("+%d%% damage taken", c.ThirstCut))
		}
		if c.HitCut > 0 {
			fx = append(fx, fmt.Sprintf("-%d hit", c.HitCut))
		}
		parts = append(parts, strings.Join(c.Words, ", ")+": "+strings.Join(fx, ", "))
	}
	if c.Meal.Kind != "" {
		parts = append(parts, fmt.Sprintf("%s: %s (%s)", c.Meal.Name, c.Meal.Effect(), BattlesLeft(battles)))
	}
	for _, a := range c.Ailments {
		parts = append(parts, fmt.Sprintf("%s: %s (%s)", a.Name, a.Effect(), BattlesLeft(a.Battles)))
	}
	return strings.Join(parts, "; ")
}

// BattlesLeft is "1 battle" or "3 battles".
func BattlesLeft(n int) string {
	if n == 1 {
		return "1 battle"
	}
	return fmt.Sprintf("%d battles", n)
}

// ConditionFor is the battle condition of a member with these needs.
func ConditionFor(n Needs) Condition {
	n = Normalize(n)
	var c Condition
	if p := needPenalty(n.Hunger); p > 0 {
		c.DamagePct, c.HungerCut = -p, p
		c.Words = append(c.Words, HungerLabel(n.Hunger))
	}
	if p := needPenalty(n.Thirst); p > 0 {
		c.GuardPct, c.ThirstCut = -p, p
		c.Words = append(c.Words, ThirstLabel(n.Thirst))
	}
	if p := FatigueHitPenalty(n.Fatigue); p > 0 {
		c.HitCut = p
		c.Words = append(c.Words, FatigueLabel(n.Fatigue))
	}
	if m, ok := MealFor(n.Meal); ok && n.MealBattles > 0 {
		c.Meal = m
		c.DamagePct += m.DamagePct
		c.GuardPct += m.GuardPct
		c.ManaPct += m.ManaPct
	}
	for _, a := range ActiveAilments(n) {
		c.Ailments = append(c.Ailments, ActiveAilment{AilmentSpec: a, Battles: battlesOf(n, a.Kind)})
		c.DamagePct -= a.DamagePct
		c.GuardPct -= a.GuardPct
	}
	return c
}

// SetMeal gives a member a meal's buff for its full length, replacing any
// other. An unknown kind changes nothing.
func (r *Registry) SetMeal(leaderUserID int, key MemberKey, kind string) error {
	spec, ok := MealFor(kind)
	if !ok {
		return nil
	}
	needs, found := r.NeedsFor(leaderUserID, key)
	if !found {
		return ErrUnknownMember
	}
	needs.Meal, needs.MealBattles = spec.Kind, spec.Battles
	return r.PutNeeds(leaderUserID, key, needs)
}

// SpendMealBattle counts one battle off each named member's meal buff and
// ailments (Phase 55), and reports whether anything changed.
func (r *Registry) SpendMealBattle(leaderUserID int, keys []MemberKey) bool {
	changed := false
	for _, key := range keys {
		needs, ok := r.NeedsFor(leaderUserID, key)
		if !ok || (needs.MealBattles <= 0 && !HasAilment(needs)) {
			continue
		}
		if needs.MealBattles > 0 {
			needs.MealBattles--
			if needs.MealBattles == 0 {
				needs.Meal = ""
			}
		}
		spendAilmentBattle(&needs)
		if r.PutNeeds(leaderUserID, key, needs) == nil {
			changed = true
		}
	}
	return changed
}

// MealService is implemented by modules/survival: it counts a battle off
// the meal buffs of the members that fought it, in one durable write.
type MealService interface {
	SpendMealBattle(leaderUserID int, keys []MemberKey) error
}

var (
	mealServiceMu sync.RWMutex
	mealService   MealService
)

// SetMealService registers the active meal service. nil clears it.
func SetMealService(s MealService) {
	mealServiceMu.Lock()
	defer mealServiceMu.Unlock()
	mealService = s
}

// SpendMealBattle counts one battle off the members' meal buffs through the
// registered module; with none loaded it does nothing.
func SpendMealBattle(leaderUserID int, keys []MemberKey) error {
	mealServiceMu.RLock()
	s := mealService
	mealServiceMu.RUnlock()
	if s == nil || len(keys) == 0 {
		return nil
	}
	return s.SpendMealBattle(leaderUserID, keys)
}
