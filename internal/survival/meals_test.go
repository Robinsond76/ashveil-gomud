package survival

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConditionForBands(t *testing.T) {
	for _, tc := range []struct {
		name               string
		needs              Needs
		damage, guard, hit int
		words              []string
		summary            string
	}{
		{"well kept", Needs{Hunger: 60, Thirst: 60, Fatigue: 60}, 0, 0, 0, nil, ""},
		{"hungry", Needs{Hunger: 50, Thirst: 100, Fatigue: 100}, -5, 0, 0, []string{"Hungry"}, "Hungry: -5% damage"},
		{"starving", Needs{Hunger: 0, Thirst: 100, Fatigue: 100}, -10, 0, 0, []string{"Starving"}, "Starving: -10% damage"},
		{"thirsty", Needs{Hunger: 100, Thirst: 26, Fatigue: 100}, 0, -5, 0, []string{"Thirsty"}, "Thirsty: +5% damage taken"},
		{"parched", Needs{Hunger: 100, Thirst: 25, Fatigue: 100}, 0, -10, 0, []string{"Parched"}, "Parched: +10% damage taken"},
		{"exhausted", Needs{Hunger: 100, Thirst: 100, Fatigue: 10}, 0, 0, 10, []string{"Exhausted"}, "Exhausted: -10 hit"},
		{"all", Needs{Hunger: 40, Thirst: 0, Fatigue: 0}, -5, -10, 20, []string{"Hungry", "Dehydrated", "Collapsed"}, "Hungry, Dehydrated, Collapsed: -5% damage, +10% damage taken, -20 hit"},
	} {
		c := ConditionFor(tc.needs)
		assert.Equal(t, tc.damage, c.DamagePct, tc.name)
		assert.Equal(t, tc.guard, c.GuardPct, tc.name)
		assert.Equal(t, tc.hit, c.HitCut, tc.name)
		assert.Equal(t, tc.words, c.Words, tc.name)
		assert.Equal(t, tc.summary, c.Summary(0), tc.name)
		assert.Equal(t, tc.summary == "", c.Neutral(), tc.name)
	}
}

func TestAMealAddsToTheCondition(t *testing.T) {
	c := ConditionFor(Needs{Hunger: 40, Thirst: 100, Fatigue: 100, Meal: "seared", MealBattles: 1})
	assert.Equal(t, 0, c.DamagePct, "hungry -5 and a strong meal +5")
	assert.Equal(t, "Hungry: -5% damage; Strong: +5% damage (1 battle)", c.Summary(1))
	c = ConditionFor(Needs{Hunger: 100, Thirst: 100, Fatigue: 100, Meal: "fish", MealBattles: 3})
	assert.Equal(t, 20, c.ManaPct)
	assert.False(t, c.Neutral())
	c = ConditionFor(Needs{Hunger: 100, Thirst: 100, Fatigue: 100, Meal: "fish"})
	assert.True(t, c.Neutral(), "a spent meal does nothing")
}

func TestEveryMealKindHasASpec(t *testing.T) {
	for _, k := range MealKinds() {
		m, ok := MealFor(k)
		assert.True(t, ok, k)
		assert.Positive(t, m.Battles, k)
		assert.NotEmpty(t, m.Effect(), k)
	}
	assert.Len(t, meals, len(MealKinds()))
}

func TestFatigueHitPenaltyMatchesTheBands(t *testing.T) {
	assert.Equal(t, []int{20, 10, 10, 5, 5, 0, 0}, []int{FatigueHitPenalty(0), FatigueHitPenalty(1), FatigueHitPenalty(25), FatigueHitPenalty(26), FatigueHitPenalty(50), FatigueHitPenalty(51), FatigueHitPenalty(100)})
}
