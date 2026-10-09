package survival

import (
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Phase 50: a cooked meal's buff is given by eating, saved with the needs,
// survives a restart, and counts one battle off per battle fought.

func TestEatingAMealGivesItsBuffAndSavesIt(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.registry.Ensure(7, domain.LeaderMemberKey))
	_, err := m.Provision(7, "", domain.Benefit{Nutrition: 50, Meal: "stew"})
	require.NoError(t, err)
	n := m.registry.MustNeedsFor(7, domain.LeaderMemberKey)
	assert.Equal(t, "stew", n.Meal)
	assert.Equal(t, 3, n.MealBattles)
	assert.Equal(t, m.registry, m.store.(*fakeStore).saved, "written before success is reported")

	_, err = m.Provision(7, "", domain.Benefit{Nutrition: 40, Meal: "fish"})
	require.NoError(t, err)
	n = m.registry.MustNeedsFor(7, domain.LeaderMemberKey)
	assert.Equal(t, "fish", n.Meal, "a new meal replaces the old one")

	_, err = m.Provision(7, "", domain.Benefit{Nutrition: 10})
	require.NoError(t, err)
	assert.Equal(t, "fish", m.registry.MustNeedsFor(7, domain.LeaderMemberKey).Meal, "plain food keeps the buff")
}

func TestAMealBuffSurvivesARestart(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.registry.Ensure(7, domain.LeaderMemberKey))
	_, err := m.Provision(7, "", domain.Benefit{Nutrition: 50, Meal: "roast"})
	require.NoError(t, err)
	require.NoError(t, m.SpendMealBattle(7, []domain.MemberKey{domain.LeaderMemberKey}))
	require.NoError(t, m.flush())

	data, err := yaml.Marshal(m.store.(*fakeStore).saved)
	require.NoError(t, err)
	var loaded domain.Registry
	require.NoError(t, decodeRegistry(data, &loaded))
	n := loaded.MustNeedsFor(7, domain.LeaderMemberKey)
	assert.Equal(t, "roast", n.Meal)
	assert.Equal(t, 2, n.MealBattles, "one battle spent before the restart")
}

func TestDecodeDropsAnUnknownOrSpentMeal(t *testing.T) {
	data := []byte("leaders:\n" +
		"  7:\n" +
		"    leader: {hunger: 50, thirst: 50, fatigue: 50, meal: banquet, meal_battles: 3}\n" +
		"    companion:1: {hunger: 50, thirst: 50, fatigue: 50, meal: stew, meal_battles: 0}\n" +
		"    companion:2: {hunger: 50, thirst: 50, fatigue: 50, meal: stew, meal_battles: 99}\n")
	var registry domain.Registry
	require.NoError(t, decodeRegistry(data, &registry))
	assert.Equal(t, domain.Needs{Hunger: 50, Thirst: 50, Fatigue: 50}, registry.MustNeedsFor(7, domain.LeaderMemberKey))
	assert.Equal(t, domain.Needs{Hunger: 50, Thirst: 50, Fatigue: 50}, registry.MustNeedsFor(7, domain.CompanionMemberKey(1)))
	assert.Equal(t, 3, registry.MustNeedsFor(7, domain.CompanionMemberKey(2)).MealBattles, "capped at the meal's length")
}

func TestSpendingTheLastBattleEndsTheBuff(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.registry.Ensure(7, domain.LeaderMemberKey))
	require.NoError(t, m.registry.Ensure(7, domain.CompanionMemberKey(1)))
	_, err := m.Provision(7, "", domain.Benefit{Nutrition: 40, Meal: "seared"})
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		require.NoError(t, m.SpendMealBattle(7, []domain.MemberKey{domain.LeaderMemberKey, domain.CompanionMemberKey(1)}))
	}
	n := m.registry.MustNeedsFor(7, domain.LeaderMemberKey)
	assert.Empty(t, n.Meal)
	assert.Zero(t, n.MealBattles)
	assert.True(t, m.dirty)
}

func TestSurvivalStatusShowsTheBattleCondition(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.registry.PutNeeds(7, domain.LeaderMemberKey, domain.Needs{Hunger: 20, Thirst: 100, Fatigue: 100, Meal: "stew", MealBattles: 2}))
	out := m.status(7)
	assert.Contains(t, out, "In battle: Starving: -10% damage; Hearty: 10% less damage taken (2 battles)")
}
