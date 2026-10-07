package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/bonds"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 65: bonds in battle. The company module keeps how each pair of
// companions feels (internal/bonds); here a friend in reach steps in for a
// friend who is hurt, a guardian won't step in for a rival it was set to
// guard, and a guardian with no ward set passes over a rival. All of it is
// automatic, says so in the log, and is small: a friend has one step in a
// battle (two for kin), only for a friend at half health or less, and the
// pair's bond moves only with the real rescue or refusal.

// bondLine is the room's line for a friend's step ("Tamsin Reed steps in
// front of Aria for a friend. (bond guard, 1 left)"). Names are tagged or
// "You"/"you", as guardLine's.
func bondGuardLine(guardian, ward string, left int) string {
	count := "none left"
	if left > 0 {
		count = fmt.Sprintf("%d left", left)
	}
	if guardian == `You` {
		return fmt.Sprintf(`You step in front of %s for a friend. (bond guard, %s)`, ward, count)
	}
	return util.CapitalizeFirst(fmt.Sprintf(`%s steps in front of %s for a friend. (bond guard, %s)`, guardian, ward, count))
}

// refusalLine is the room's line for a rival who lets the blow fall:
// "Merek lets the blow fall on Ysolde. (no guard: they can't stand each other)".
func refusalLine(guardian, ward string) string {
	return util.CapitalizeFirst(fmt.Sprintf(`%s lets the blow fall on %s. (no guard: they can't stand each other)`, guardian, ward))
}

// tellBond sends one line to the leader and the room, and reports it on the
// combat stream as kind (with the status "bond").
func tellBond(leader *users.UserRecord, fightID uint64, kind combatstream.Kind, g, ward guardMember, line func(guardian, ward string) string) {
	roomID := leader.Character.RoomId
	switch {
	case g.user != nil:
		leader.SendText(line(`You`, ward.tag()))
	case ward.user != nil:
		leader.SendText(line(g.tag(), `you`))
	default:
		leader.SendText(line(g.tag(), ward.tag()))
	}
	if room := rooms.LoadRoom(roomID); room != nil {
		room.SendText(line(g.tag(), ward.tag()), leader.UserId)
	}
	emitCombat(combatstream.Event{Kind: kind, FightID: fightID, RoomId: roomID, Source: g.ref(), Target: ward.ref(), Status: "bond"})
}

// bondFriendGuard finds a friend of the struck member who steps in: in
// reach, standing, with a bond step left, when the struck member is at half
// health or less. It spends the step, says so, and reports the rescue to
// the bond. ok is false when no one does.
func bondFriendGuard(leader *users.UserRecord, f company.Formation, fightID uint64, members []guardMember, ward guardMember, struck company.MemberKey, struckAlready map[company.MemberKey]bool) (guardMember, bool) {
	if max := ward.char.HealthMax.Value; max < 1 || ward.char.Health*100 > max*bonds.GuardBelowPct {
		return guardMember{}, false
	}
	narrow := enemyparty.Narrow(rooms.LoadRoom(leader.Character.RoomId))
	for _, g := range members {
		if g.key == struck || struckAlready[g.key] || !ableToGuard(g) || !formationcombat.GuardGround(f, g.key, struck, narrow) {
			continue
		}
		allowed := bonds.Guards(company.BondValue(leader.UserId, g.key, struck))
		rt := g.char.RTState()
		if allowed < 1 || rt.BondGuards >= allowed {
			continue
		}
		rt.BondGuards++
		left := allowed - rt.BondGuards
		tellBond(leader, fightID, combatstream.GuardUsed, g, ward, func(gd, wd string) string { return bondGuardLine(gd, wd, left) })
		company.BondEvent(leader.UserId, g.key, struck, bonds.Rescue)
		return g, true
	}
	return guardMember{}, false
}

// bondRefuses reports whether guardian g, whose set ward is the struck
// member, refuses because the two are rivals. When it would otherwise have
// stepped in (the ward is in reach) it says so, once a battle, and the
// refusal sours the bond.
func bondRefuses(leader *users.UserRecord, f company.Formation, fightID uint64, g, ward guardMember, struck company.MemberKey) bool {
	if !bonds.IsRival(company.BondValue(leader.UserId, g.key, struck)) {
		return false
	}
	narrow := enemyparty.Narrow(rooms.LoadRoom(leader.Character.RoomId))
	if formationcombat.GuardGround(f, g.key, struck, narrow) {
		if rt := g.char.RTState(); !rt.BondRefused {
			rt.BondRefused = true
			tellBond(leader, fightID, combatstream.GuardRefused, g, ward, refusalLine)
			company.BondEvent(leader.UserId, g.key, struck, bonds.Refusal)
		}
	}
	return true
}
