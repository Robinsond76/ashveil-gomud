package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/stance"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// companyStance is a member's weapon stance (Phase 69). Name is the stance
// as a player reads it; Ready is whether the member holds what it needs now
// (omitted when the member isn't here to check); Needs says what that is.
type companyStance struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Gain  string `json:"gain"`
	Cost  string `json:"cost"`
	Needs string `json:"needs"`
	Ready *bool  `json:"ready,omitempty"`
}

// stanceOf is a member's chosen stance, nil when it has none.
func stanceOf(leaderUserID int, key company.MemberKey) *companyStance {
	st := stance.For(leaderUserID, string(key))
	d, ok := stance.Lookup(st)
	if !ok {
		return nil
	}
	out := &companyStance{Key: string(d.Key), Name: d.Name, Gain: d.Gain, Cost: d.Cost, Needs: d.Needs}
	if gear, known := memberGear(leaderUserID, key); known {
		ready := stance.Fits(st, gear)
		out.Ready = &ready
	}
	return out
}

// stancesFit is the stance keys a member's gear can use, nil when the gear
// can't be read.
func stancesFit(leaderUserID int, key company.MemberKey) *[]string {
	gear, known := memberGear(leaderUserID, key)
	if !known {
		return nil
	}
	fit := []string{}
	for _, st := range stance.Available(gear) {
		fit = append(fit, string(st))
	}
	return &fit
}

// memberGear is what a member holds, known for the player and a companion
// that is out.
func memberGear(leaderUserID int, key company.MemberKey) (stance.Gear, bool) {
	if key == company.LeaderMemberKey {
		if u := users.GetByUserId(leaderUserID); u != nil && u.Character != nil {
			return u.Character.StanceGear(), true
		}
		return stance.Gear{}, false
	}
	id, ok := company.CompanionIDFromMemberKey(key)
	if !ok {
		return stance.Gear{}, false
	}
	instanceID, live := company.InstanceFor(leaderUserID, id)
	if !live {
		return stance.Gear{}, false
	}
	if m := mobs.GetInstance(instanceID); m != nil {
		return m.Character.StanceGear(), true
	}
	return stance.Gear{}, false
}

// memberTempo reads a member's combat tempo (Phase 82d); tests replace it.
var memberTempo = hooks.MemberTempo
