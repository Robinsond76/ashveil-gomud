package exposure

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type deadRosterStub struct{ refs []survival.MemberRef }

func (r deadRosterStub) Roster(int) []survival.MemberRef { return r.refs }

// TestCompanionRosterSkipsDead: a dead companion feels no cold and is never
// drained (Phase 25b).
func TestCompanionRosterSkipsDead(t *testing.T) {
	survival.SetRosterProvider(deadRosterStub{refs: []survival.MemberRef{
		{Key: survival.LeaderMemberKey, Name: "Hero"},
		{Key: survival.CompanionMemberKey(1), Name: "Bear"},
		{Key: survival.CompanionMemberKey(2), Name: "Wolf", Dead: true},
	}})
	t.Cleanup(func() { survival.SetRosterProvider(nil) })
	members, ok := companionRoster(7)
	require.True(t, ok)
	require.Len(t, members, 1)
	assert.Equal(t, "Bear", members[0].Name)
}

// TestBandBuffsAreSurvivalConditions (Phase 26a).
func TestBandBuffsAreSurvivalConditions(t *testing.T) {
	for _, id := range []int{1010, 1013, 1020, 1023} {
		g, ok := companyview.GroupOf(id)
		assert.True(t, ok, id)
		assert.Equal(t, companyview.GroupSurvival, g)
	}
}
