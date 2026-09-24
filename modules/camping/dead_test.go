package camping

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
)

// TestDeadCompanionsNeitherRestNorPay: a dead companion is granted no rest
// tier (not even durably, for when it returns) and isn't charged at an inn
// (Phase 25b).
func TestDeadCompanionsNeitherRestNorPay(t *testing.T) {
	survival.SetRosterProvider(rosterStub{refs: []survival.MemberRef{
		{Key: survival.LeaderMemberKey, Name: "Hero"},
		{Key: survival.CompanionMemberKey(1), Name: "Bear"},
		{Key: survival.CompanionMemberKey(2), Name: "Wolf", Dead: true},
	}})
	t.Cleanup(func() { survival.SetRosterProvider(nil) })
	m := &CampingModule{}
	_, roster := m.companions(7)
	assert.Equal(t, []int{1}, roster)
	assert.Equal(t, 2, m.companyMembers(7), "the leader and Bear")
}
