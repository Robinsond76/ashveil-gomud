// Package livecompanions lists a leader's company companions together with
// their spawned mobs. Several modules (exposure, walking, camping rest tiers,
// gathering, archetype utilities) used to each walk the survival roster,
// parse the member key, look up the mob instance and apply their own
// liveness rule; the walk lives here once and each caller passes the rule.
package livecompanions

import (
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/survival"
)

// Companion is one roster companion. Mob is nil when it is not spawned.
type Companion struct {
	Ref         survival.MemberRef
	CompanionID int
	Mob         *mobs.Mob
}

// Filter decides whether a roster entry is wanted. A nil Filter wants all.
type Filter func(survival.MemberRef) bool

// SkipDead drops companions awaiting resurrection (Phase 25b).
func SkipDead(ref survival.MemberRef) bool { return !ref.Dead }

// SkipInactive drops companions that spend and recover nothing: the dead,
// the separated (33h3) and constructs (38e).
func SkipInactive(ref survival.MemberRef) bool {
	return !ref.Dead && !ref.Away && !ref.Needless
}

// Of lists the leader's companions in roster order. The second result is
// false when no roster provider is registered. Entries whose key is not a
// companion's (the leader's own) are skipped, as are those the filter drops.
func Of(leaderUserID int, want Filter) ([]Companion, bool) {
	roster := survival.CurrentRoster(leaderUserID)
	if roster == nil {
		return nil, false
	}
	var out []Companion
	for _, ref := range roster {
		companionID, ok := company.CompanionIDFromMemberKey(ref.Key)
		if !ok || (want != nil && !want(ref)) {
			continue
		}
		c := Companion{Ref: ref, CompanionID: companionID}
		if instanceID, ok := company.InstanceFor(leaderUserID, companionID); ok {
			c.Mob = mobs.GetInstance(instanceID)
		}
		out = append(out, c)
	}
	return out, true
}

// Able lists the companions that are spawned, standing in a wanted room and
// not downed. inRoom may be nil to accept any room.
func Able(leaderUserID int, inRoom func(roomID int) bool) []Companion {
	all, _ := Of(leaderUserID, nil)
	out := all[:0]
	for _, c := range all {
		if c.Mob == nil || (inRoom != nil && !inRoom(c.Mob.Character.RoomId)) || c.Mob.Character.IsDisabled() {
			continue
		}
		out = append(out, c)
	}
	return out
}
