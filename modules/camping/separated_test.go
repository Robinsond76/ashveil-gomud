package camping

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
)

// Review finding (33h3): a separated companion is granted no rest tier
// and isn't charged a bed at an inn; it wasn't there.
func TestSeparatedCompanionsNeitherRestNorPay(t *testing.T) {
	survival.SetRosterProvider(rosterStub{refs: []survival.MemberRef{
		{Key: survival.LeaderMemberKey, Name: "Hero"},
		{Key: survival.CompanionMemberKey(1), Name: "Bear"},
		{Key: survival.CompanionMemberKey(2), Name: "Wolf", Away: true},
	}})
	t.Cleanup(func() { survival.SetRosterProvider(nil) })
	m := &CampingModule{}
	_, roster := m.companions(7)
	assert.Equal(t, []int{1}, roster)
	assert.Equal(t, 2, m.companyMembers(7), "the leader and Bear")
}
