package survival

import (
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 33h3: a separated companion (#2) spends, recovers, and eats
// nothing until it rejoins, and the status says where it is.
func TestSeparatedCompanionIsFrozen(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	for _, key := range []domain.MemberKey{domain.LeaderMemberKey, domain.CompanionMemberKey(1), domain.CompanionMemberKey(2)} {
		require.NoError(t, m.registry.PutNeeds(7, key, halfNeeds))
	}
	useRoster(t, fakeRoster{members: map[int][]domain.MemberRef{7: {
		{Key: domain.LeaderMemberKey, Name: "Hero"},
		{Key: domain.CompanionMemberKey(1), Name: "Bear"},
		{Key: domain.CompanionMemberKey(2), Name: "Wolf", Away: true},
	}}})

	_, err := m.ApplyCompanyExertion(7, "op", domain.Exertion{Hunger: 10})
	require.NoError(t, err)
	_, err = m.ApplyCompanyRestRecovery(7, "rest", 20)
	require.NoError(t, err)
	assert.Equal(t, halfNeeds, m.registry.MustNeedsFor(7, domain.CompanionMemberKey(2)))
	assert.NotEqual(t, halfNeeds, m.registry.MustNeedsFor(7, domain.CompanionMemberKey(1)))

	// Review finding (33h3): it can't be fed while away.
	_, err = m.Provision(7, "wolf", domain.Benefit{Nutrition: 10})
	assert.ErrorIs(t, err, domain.ErrAwayMember)
	assert.Equal(t, halfNeeds, m.registry.MustNeedsFor(7, domain.CompanionMemberKey(2)))
	assert.Contains(t, m.status(7), "Wolf: separated")
}

// Phase 38e: a construct (a stone golem) spends, recovers and eats nothing,
// and the status says why; the living members around it still do.
func TestNeedlessCompanionTakesNoUpkeep(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	for _, key := range []domain.MemberKey{domain.LeaderMemberKey, domain.CompanionMemberKey(1), domain.CompanionMemberKey(2)} {
		require.NoError(t, m.registry.PutNeeds(7, key, halfNeeds))
	}
	useRoster(t, fakeRoster{members: map[int][]domain.MemberRef{7: {
		{Key: domain.LeaderMemberKey, Name: "Hero"},
		{Key: domain.CompanionMemberKey(1), Name: "Hound"},
		{Key: domain.CompanionMemberKey(2), Name: "Cairn", Needless: true},
	}}})

	_, err := m.ApplyCompanyExertion(7, "op", domain.Exertion{Hunger: 10})
	require.NoError(t, err)
	_, err = m.ApplyCompanyRestRecovery(7, "rest", 20)
	require.NoError(t, err)
	assert.Equal(t, halfNeeds, m.registry.MustNeedsFor(7, domain.CompanionMemberKey(2)))
	assert.NotEqual(t, halfNeeds, m.registry.MustNeedsFor(7, domain.CompanionMemberKey(1)), "the hound tires and eats like anyone")

	_, err = m.Provision(7, "cairn", domain.Benefit{Nutrition: 10})
	assert.ErrorIs(t, err, domain.ErrNeedlessMember)
	assert.Equal(t, halfNeeds, m.registry.MustNeedsFor(7, domain.CompanionMemberKey(2)))
	assert.Contains(t, m.status(7), "Cairn: needs no food, drink or rest")
}
