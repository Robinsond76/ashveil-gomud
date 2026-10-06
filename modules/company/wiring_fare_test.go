package company

import (
	"sync"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 50 wiring: the needs survival reports set each member's battle
// condition as the battle begins through the real beginBattle, a meal buff
// adds its own and counts a battle off, and the fight opens naming both.

// fareNeeds is a survival company service with fixed needs that records the
// meal battles spent.
type fareNeeds struct {
	fakeNeeds
	mu    sync.Mutex
	spent [][]survival.MemberKey
}

func (f *fareNeeds) SpendMealBattle(_ int, keys []survival.MemberKey) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.spent = append(f.spent, keys)
	return nil
}

func useFareNeeds(t *testing.T, needs map[survival.MemberKey]survival.Needs) *fareNeeds {
	t.Helper()
	f := &fareNeeds{}
	for key, n := range needs {
		f.needs = append(f.needs, survival.MemberNeeds{Key: key, Name: string(key), Needs: n})
	}
	survival.SetCompanyService(f)
	survival.SetMealService(f)
	t.Cleanup(func() {
		survival.SetCompanyService(nil)
		survival.SetMealService(nil)
	})
	return f
}

func TestHungryAndParchedMembersGoInWorse(t *testing.T) {
	b := sigilBrawl(t)
	starving := survival.Needs{Hunger: 10, Thirst: 10, Fatigue: 100}
	hungry := survival.Needs{Hunger: 40, Thirst: 100, Fatigue: 100}
	useFareNeeds(t, map[survival.MemberKey]survival.Needs{
		survival.LeaderMemberKey:          starving,
		survival.CompanionMemberKey(1):    hungry,
		survival.CompanionMemberKey(2):    survival.FullNeeds(),
		survival.CompanionMemberKey(3):    survival.FullNeeds(),
		survival.CompanionMemberKey(4):    survival.FullNeeds(),
		survival.MemberKey("companion:9"): starving, // not in the fight
	})
	out := b.beginFight()

	rt := b.aria.Character.RTState()
	assert.Equal(t, -survival.ConditionClearPct, rt.FareDamage, "starving: a clear damage cut")
	assert.Equal(t, -survival.ConditionClearPct, rt.FareGuard, "parched: takes clearly more damage")
	one := b.companion(1).Character.RTState()
	assert.Equal(t, -survival.ConditionSmallPct, one.FareDamage, "hungry: a small damage cut")
	assert.Zero(t, one.FareGuard)
	two := b.companion(2).Character.RTState()
	assert.Zero(t, two.FareDamage+two.FareGuard, "a well-kept member fights as it always has")

	assert.Contains(t, out, "Going in:")
	assert.Contains(t, out, "Starving, Parched: -10% damage, +10% damage taken")
	fare := battle.FareOf(7)
	assert.Contains(t, fare, "leader")
	assert.Contains(t, fare, "companion:1")
	assert.NotContains(t, fare, "companion:2", "a neutral member is not marked")
}

func TestAMealBuffAppliesAndCountsABattleOff(t *testing.T) {
	b := sigilBrawl(t)
	stew := survival.FullNeeds()
	stew.Meal, stew.MealBattles = "stew", 2
	fish := survival.FullNeeds()
	fish.Meal, fish.MealBattles = "fish", 3
	f := useFareNeeds(t, map[survival.MemberKey]survival.Needs{
		survival.LeaderMemberKey:       fish,
		survival.CompanionMemberKey(1): stew,
	})
	b.aria.Character.ManaMax.Value, b.aria.Character.Mana = 300, 100
	out := b.beginFight()

	assert.Equal(t, 10, b.companion(1).Character.RTState().FareGuard, "hearty: 10% less damage taken")
	assert.Greater(t, b.aria.Character.Mana, 100, "clear-headed: 20% of max mana back as the battle begins")
	assert.Contains(t, out, "Hearty: 10% less damage taken (2 battles)")
	assert.Contains(t, out, "Clear-headed")
	require.Len(t, f.spent, 1, "one battle counted off, once")
	assert.ElementsMatch(t, []survival.MemberKey{survival.LeaderMemberKey, survival.CompanionMemberKey(1)}, f.spent[0])
}

func TestNoSurvivalServiceChangesNothing(t *testing.T) {
	b := sigilBrawl(t)
	out := b.beginFight()
	rt := b.aria.Character.RTState()
	assert.Zero(t, rt.FareDamage+rt.FareGuard)
	assert.NotContains(t, out, "Going in:")
	assert.Nil(t, battle.FareOf(7))
}
