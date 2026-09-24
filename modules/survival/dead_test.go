package survival

import (
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deadRoster is a leader, a living companion (#1), and a dead one (#2).
func deadRoster(t *testing.T, m *SurvivalModule) {
	t.Helper()
	for _, key := range []domain.MemberKey{domain.LeaderMemberKey, domain.CompanionMemberKey(1), domain.CompanionMemberKey(2)} {
		require.NoError(t, m.registry.PutNeeds(7, key, domain.Needs{Hunger: 50, Thirst: 50, Fatigue: 50}))
	}
	useRoster(t, fakeRoster{members: map[int][]domain.MemberRef{7: {
		{Key: domain.LeaderMemberKey, Name: "Hero"},
		{Key: domain.CompanionMemberKey(1), Name: "Bear"},
		{Key: domain.CompanionMemberKey(2), Name: "Wolf", Dead: true},
	}}})
}

var halfNeeds = domain.Needs{Hunger: 50, Thirst: 50, Fatigue: 50}

// TestCompanyExertionSkipsDead: travel exertion spends nothing of a dead
// companion (Phase 25b).
func TestCompanyExertionSkipsDead(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	deadRoster(t, m)
	results, err := m.ApplyCompanyExertion(7, "op", domain.Exertion{Hunger: 10})
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, 40, m.registry.MustNeedsFor(7, domain.CompanionMemberKey(1)).Hunger)
	assert.Equal(t, halfNeeds, m.registry.MustNeedsFor(7, domain.CompanionMemberKey(2)))
	replay, err := m.ApplyCompanyExertion(7, "op", domain.Exertion{Hunger: 10})
	require.NoError(t, err)
	assert.Len(t, replay, 2)
}

// TestCompanyRestSkipsDead: a rest recovers nothing for a dead companion.
func TestCompanyRestSkipsDead(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	deadRoster(t, m)
	results, err := m.ApplyCompanyRestRecovery(7, "rest", 20)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, 70, m.registry.MustNeedsFor(7, domain.CompanionMemberKey(1)).Fatigue)
	assert.Equal(t, halfNeeds, m.registry.MustNeedsFor(7, domain.CompanionMemberKey(2)))
	replay, err := m.ApplyCompanyRestRecovery(7, "rest", 20)
	require.NoError(t, err)
	assert.Len(t, replay, 2)
}

// TestProvisionRefusesDead: a dead companion can't be fed, by number or
// name, though it is still recognized as a member.
func TestProvisionRefusesDead(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	deadRoster(t, m)
	for _, selector := range []string{"#2", "2", "wolf"} {
		assert.True(t, m.IsMemberSelector(7, selector), selector)
		_, err := m.Provision(7, selector, domain.Benefit{Nutrition: 10})
		assert.ErrorIs(t, err, domain.ErrDeadMember, selector)
	}
	assert.Equal(t, halfNeeds, m.registry.MustNeedsFor(7, domain.CompanionMemberKey(2)))
	_, err := m.Provision(7, "bear", domain.Benefit{Nutrition: 10})
	assert.NoError(t, err)
}

// TestStatusMarksDead: the status shows the dead companion as fallen.
func TestStatusMarksDead(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	deadRoster(t, m)
	text := m.status(7)
	assert.Contains(t, text, "Wolf: fallen")
	assert.Contains(t, text, "Bear: Hunger 50")
}

// TestCompanyNeedsSkipsDead (review finding 1): a dead companion's frozen
// fatigue can't hold the company back from travel, which gates on
// CompanyNeeds, and isn't shown as if alive.
func TestCompanyNeedsSkipsDead(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	deadRoster(t, m)
	require.NoError(t, m.registry.PutNeeds(7, domain.CompanionMemberKey(2), domain.Needs{Fatigue: 0}))
	needs := m.CompanyNeeds(7)
	require.Len(t, needs, 2)
	for _, n := range needs {
		assert.NotEqual(t, domain.CompanionMemberKey(2), n.Key)
	}
}
