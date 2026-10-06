package survival

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 55: ailments ride on the needs, set the battle condition, and count
// down one battle at a time.

func TestAilmentsSetTheBattleCondition(t *testing.T) {
	fit := Needs{Hunger: 100, Thirst: 100, Fatigue: 100}
	for _, tc := range []struct {
		name          string
		needs         Needs
		damage, guard int
		summary       string
	}{
		{"healthy is neutral", fit, 0, 0, ""},
		{"chill", Needs{Hunger: 100, Thirst: 100, Fatigue: 100, Chill: 3}, -10, 0, "Chill: -10% damage (3 battles)"},
		{"gut-ache", Needs{Hunger: 100, Thirst: 100, Fatigue: 100, GutAche: 1}, 0, -10, "Gut-ache: +10% damage taken (1 battle)"},
		{"fever", Needs{Hunger: 100, Thirst: 100, Fatigue: 100, Fever: 5}, -15, 0, "Fever: -15% damage (5 battles)"},
		{"stack with hunger", Needs{Hunger: 40, Thirst: 100, Fatigue: 100, Chill: 2}, -15, 0, "Hungry: -5% damage; Chill: -10% damage (2 battles)"},
	} {
		c := ConditionFor(tc.needs)
		assert.Equal(t, tc.damage, c.DamagePct, tc.name)
		assert.Equal(t, tc.guard, c.GuardPct, tc.name)
		assert.Equal(t, tc.summary, c.Summary(0), tc.name)
		assert.Equal(t, tc.summary == "", c.Neutral(), tc.name)
	}
}

func TestEveryAilmentHasACauseAnEffectAndARemedy(t *testing.T) {
	for _, a := range Ailments() {
		assert.Positive(t, a.Battles, a.Kind)
		assert.NotEmpty(t, a.Effect(), a.Kind)
		assert.NotEmpty(t, a.Cause, a.Kind)
		assert.NotEmpty(t, a.RemedyName, a.Kind)
		require.NotEmpty(t, a.Remedy, a.Kind)
		for _, ing := range a.Remedy {
			assert.Positive(t, ing.Count, a.Kind)
		}
		got, ok := FindAilment(a.Name)
		assert.True(t, ok, a.Kind)
		assert.Equal(t, a.Kind, got.Kind)
	}
	got, ok := FindAilment("gut")
	assert.True(t, ok)
	assert.Equal(t, AilmentGutAche, got.Kind)
	_, ok = FindAilment("plague")
	assert.False(t, ok)
}

func TestCatchingAndCuringAnAilment(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Ensure(7, LeaderMemberKey))

	caught, err := r.CatchAilment(7, LeaderMemberKey, AilmentChill)
	require.NoError(t, err)
	assert.True(t, caught)
	assert.Equal(t, 4, r.MustNeedsFor(7, LeaderMemberKey).Chill)

	assert.True(t, r.SpendMealBattle(7, []MemberKey{LeaderMemberKey}))
	assert.Equal(t, 3, r.MustNeedsFor(7, LeaderMemberKey).Chill)
	caught, err = r.CatchAilment(7, LeaderMemberKey, AilmentChill)
	require.NoError(t, err)
	assert.False(t, caught, "catching it again only starts it over")
	assert.Equal(t, 4, r.MustNeedsFor(7, LeaderMemberKey).Chill)

	cured, err := r.CureAilment(7, LeaderMemberKey, AilmentChill)
	require.NoError(t, err)
	assert.True(t, cured)
	assert.False(t, HasAilment(r.MustNeedsFor(7, LeaderMemberKey)))
	cured, _ = r.CureAilment(7, LeaderMemberKey, AilmentChill)
	assert.False(t, cured, "nothing to cure")

	caught, err = r.CatchAilment(7, LeaderMemberKey, "plague")
	require.NoError(t, err)
	assert.False(t, caught, "an unknown ailment changes nothing")
	_, err = r.CatchAilment(7, CompanionMemberKey(9), AilmentFever)
	assert.ErrorIs(t, err, ErrUnknownMember)
}

func TestEachBattleCountsDownAnAilment(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Ensure(7, LeaderMemberKey))
	require.NoError(t, r.Ensure(7, CompanionMemberKey(1)))
	_, _ = r.CatchAilment(7, LeaderMemberKey, AilmentGutAche)
	_, _ = r.CatchAilment(7, LeaderMemberKey, AilmentFever)
	for i := 0; i < 3; i++ {
		assert.True(t, r.SpendMealBattle(7, []MemberKey{LeaderMemberKey, CompanionMemberKey(1)}))
	}
	n := r.MustNeedsFor(7, LeaderMemberKey)
	assert.Zero(t, n.GutAche, "gut-ache lasts three battles")
	assert.Equal(t, 2, n.Fever, "fever lasts five")
	assert.False(t, r.SpendMealBattle(7, []MemberKey{CompanionMemberKey(1)}), "a healthy member changes nothing")
}

func TestNormalizeCapsAnAilmentAtItsLength(t *testing.T) {
	n := Normalize(Needs{Hunger: 50, Thirst: 50, Fatigue: 50, Chill: 99, GutAche: -3, Fever: 2})
	assert.Equal(t, 4, n.Chill)
	assert.Zero(t, n.GutAche)
	assert.Equal(t, 2, n.Fever)
}

func TestFrozenMeansFrostbitten(t *testing.T) {
	assert.False(t, IsFrozen(0))
	assert.False(t, IsFrozen(-49))
	assert.True(t, IsFrozen(-50))
	assert.True(t, IsFrozen(-100))
	assert.False(t, IsFrozen(80), "heat is not a chill")
}

func TestAilmentLabelsNameTheBattlesLeft(t *testing.T) {
	assert.Equal(t, []string{"Chill (2 battles)", "Fever (1 battle)"}, AilmentLabels(Needs{Chill: 2, Fever: 1}))
	assert.Empty(t, AilmentLabels(Needs{}))
}
