package formationcombat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
)

func TestGuardReach(t *testing.T) {
	var f company.Formation
	for _, p := range []struct {
		key      company.MemberKey
		row, col int
	}{{company.LeaderMemberKey, 0, 0}, {company.CompanionMemberKey(1), 1, 1}, {company.CompanionMemberKey(2), 2, 2}} {
		if err := f.Place(p.key, p.row, p.col); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		g, w company.MemberKey
		want bool
	}{
		{company.CompanionMemberKey(1), company.LeaderMemberKey, true},       // next column
		{company.LeaderMemberKey, company.CompanionMemberKey(1), true},       // either way
		{company.CompanionMemberKey(2), company.CompanionMemberKey(1), true}, // next column, other row
		{company.CompanionMemberKey(2), company.LeaderMemberKey, false},      // two away
		{company.CompanionMemberKey(3), company.LeaderMemberKey, true},       // unplaced guardian fails open
		{company.LeaderMemberKey, company.CompanionMemberKey(3), true},       // unplaced ward fails open
	}
	for _, c := range cases {
		if got := GuardReach(f, c.g, c.w); got != c.want {
			t.Errorf("GuardReach(%s, %s) = %v", c.g, c.w, got)
		}
	}
}
