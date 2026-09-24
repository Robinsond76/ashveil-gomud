package company_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisplayAlignmentMapsEngineScale(t *testing.T) {
	for _, tc := range []struct{ engine, display int }{
		{-100, 1}, {-101, 1}, {-1000, 1},
		{0, 50}, {-1, 50}, {1, 50}, {2, 51}, {-2, 49},
		{100, 100}, {101, 100}, {60, 80},
	} {
		assert.Equal(t, tc.display, company.DisplayAlignment(tc.engine), "engine %d", tc.engine)
	}
	for a := -100; a < 100; a++ {
		assert.LessOrEqual(t, company.DisplayAlignment(a), company.DisplayAlignment(a+1), "monotonic at %d", a)
	}
}

func TestAlignmentBandNames(t *testing.T) {
	assert.Equal(t, "neutral", company.AlignmentBand(0))
	assert.Equal(t, "holy", company.AlignmentBand(100))
	assert.Equal(t, "holy", company.AlignmentBand(500), "clamped")
	assert.Equal(t, "unholy", company.AlignmentBand(-500), "clamped")
	assert.Equal(t, "good", company.AlignmentBand(60))
	assert.Equal(t, "corrupt", company.AlignmentBand(-40))
}

func TestAverageAlignmentRounds(t *testing.T) {
	assert.Equal(t, 0, company.AverageAlignment(nil))
	assert.Equal(t, 10, company.AverageAlignment([]int{10}))
	assert.Equal(t, 2, company.AverageAlignment([]int{1, 2}), "1.5 rounds away from zero")
	assert.Equal(t, -2, company.AverageAlignment([]int{-1, -2}), "-1.5 rounds away from zero")
	assert.Equal(t, 1, company.AverageAlignment([]int{1, 1, 2}), "1.33 rounds to 1")
	assert.Equal(t, 34, company.AverageAlignment([]int{100, 0, 1}))
}

func TestDriftTowardNeverOvershoots(t *testing.T) {
	assert.Equal(t, 12, company.DriftToward(10, 50, 2))
	assert.Equal(t, 8, company.DriftToward(10, -50, 2))
	assert.Equal(t, 11, company.DriftToward(10, 11, 2), "stops at the target")
	assert.Equal(t, 9, company.DriftToward(10, 9, 5))
	assert.Equal(t, 10, company.DriftToward(10, 10, 2))
	assert.Equal(t, 10, company.DriftToward(10, 50, 0), "no step, no drift")
	assert.Equal(t, 100, company.DriftToward(99, 500, 5), "clamped to the scale")
}

func TestTickAlignmentUsesRestOfCompany(t *testing.T) {
	rules := company.DefaultAlignmentRules()
	rules.DriftStep = 5
	members := []company.MemberAlignment{
		{ID: 1, Alignment: 40, Loyalty: 50},
		{ID: 2, Alignment: -20, Loyalty: 50},
	}
	result := company.TickAlignment(60, members, rules)
	require.Len(t, result.Members, 2)
	// #1's target is avg(60, -20) = 20: moves 40 -> 35.
	// #2's target is avg(60, 40) = 50, from the pre-tick 40: -20 -> -15.
	assert.Equal(t, company.MemberAlignment{ID: 1, Alignment: 35, Loyalty: 52}, result.Members[0])
	assert.Equal(t, company.MemberAlignment{ID: 2, Alignment: -15, Loyalty: 45}, result.Members[1], "gap 70 is past the 60 tolerance")
	assert.Equal(t, []company.MemberAlignment{{ID: 1, Alignment: 40, Loyalty: 50}, {ID: 2, Alignment: -20, Loyalty: 50}}, members, "input not modified")
}

func TestTickAlignmentLoyalty(t *testing.T) {
	rules := company.DefaultAlignmentRules()
	// Leader 60; lone companion at -10 is 70 away (past 60): loses 5.
	result := company.TickAlignment(60, []company.MemberAlignment{{ID: 1, Alignment: -10, Loyalty: 50}}, rules)
	assert.Equal(t, 45, result.Members[0].Loyalty)
	assert.Equal(t, -8, result.Members[0].Alignment)
	// Exactly at tolerance is content: gains 2.
	result = company.TickAlignment(60, []company.MemberAlignment{{ID: 1, Alignment: 0, Loyalty: 50}}, rules)
	assert.Equal(t, 52, result.Members[0].Loyalty)
	// Capped at 100.
	result = company.TickAlignment(60, []company.MemberAlignment{{ID: 1, Alignment: 60, Loyalty: 99}}, rules)
	assert.Equal(t, 100, result.Members[0].Loyalty)
	assert.Empty(t, result.Warned)
	assert.Empty(t, result.Deserters)
}

func TestTickAlignmentWarnsOnceAndDeserts(t *testing.T) {
	rules := company.DefaultAlignmentRules()
	far := func(loyalty int) company.TickResult {
		return company.TickAlignment(100, []company.MemberAlignment{{ID: 3, Alignment: -100, Loyalty: loyalty}}, rules)
	}
	assert.Equal(t, []int{3}, far(28).Warned, "28 -> 23 crosses below 25")
	assert.Empty(t, far(23).Warned, "already below: no repeat warning")
	assert.Empty(t, far(23).Deserters)
	result := far(5)
	assert.Equal(t, 0, result.Members[0].Loyalty)
	assert.Equal(t, []int{3}, result.Deserters)
	assert.Empty(t, result.Warned, "a deserter isn't also warned")
	assert.Equal(t, []int{3}, far(0).Deserters, "a postponed deserter still deserts")
	// A zero-loyalty member that is content again recovers.
	result = company.TickAlignment(0, []company.MemberAlignment{{ID: 3, Alignment: 0, Loyalty: 0}}, rules)
	assert.Equal(t, 2, result.Members[0].Loyalty)
	assert.Empty(t, result.Deserters)
}

func TestCanRecruitBoundary(t *testing.T) {
	rules := company.DefaultAlignmentRules()
	assert.True(t, company.CanRecruit(80, 0, rules))
	assert.True(t, company.CanRecruit(-80, 0, rules))
	assert.False(t, company.CanRecruit(81, 0, rules))
	assert.False(t, company.CanRecruit(90, -40, rules), "a holy recruit won't join a corrupt company")
}

func TestSetDispositionReplacesAndClamps(t *testing.T) {
	registry := company.NewRegistry()
	c, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	require.NoError(t, registry.SetDisposition(7, c.ID, company.Disposition{Alignment: 300, Loyalty: -4}))
	record, _ := registry.Get(7)
	require.NotNil(t, record.Companions[0].Disposition)
	assert.Equal(t, company.Disposition{Alignment: 100, Loyalty: 0}, *record.Companions[0].Disposition)
	assert.ErrorIs(t, registry.SetDisposition(7, 99, company.Disposition{}), company.ErrUnknownMember)
	assert.ErrorIs(t, registry.SetDisposition(8, 1, company.Disposition{}), company.ErrUnknownMember)
}

func TestGetCopiesDoNotShareDisposition(t *testing.T) {
	registry := company.NewRegistry()
	c, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	require.NoError(t, registry.SetDisposition(7, c.ID, company.Disposition{Alignment: 10, Loyalty: 50}))
	before, _ := registry.Get(7)
	require.NoError(t, registry.SetDisposition(7, c.ID, company.Disposition{Alignment: 20, Loyalty: 60}))
	assert.Equal(t, 10, before.Companions[0].Disposition.Alignment, "an earlier copy keeps its value")
	before.Companions[0].Disposition.Alignment = 99
	after, _ := registry.Get(7)
	assert.Equal(t, 20, after.Companions[0].Disposition.Alignment, "writing a copy doesn't reach the registry")
}

func TestRegistryCloneIsDeep(t *testing.T) {
	registry := company.NewRegistry()
	c, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	require.NoError(t, registry.SetDisposition(7, c.ID, company.Disposition{Alignment: 10, Loyalty: 50}))
	registry.DriftIn = 12
	clone := registry.Clone()
	require.NoError(t, registry.SetDisposition(7, c.ID, company.Disposition{Alignment: 20, Loyalty: 60}))
	_, err = registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	assert.Equal(t, 12, clone.DriftIn)
	require.Len(t, clone.Companies[7].Companions, 1)
	assert.Equal(t, 10, clone.Companies[7].Companions[0].Disposition.Alignment)
}
