package hooks

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/withdrawal"
)

var retreatRoll = util.Rand
var retreatCover = map[int]bool{}

func UseRetreatRollForTest(roll func(int) int) func() {
	previous := retreatRoll
	retreatRoll = roll
	return func() { retreatRoll = previous }
}

func handleRetreat(u *users.UserRecord, r *rooms.Room) {
	a := u.Character.Aggro
	req := a.RetreatInfo
	delete(castAims, caster{userId: u.UserId})
	resume := func(reason string) {
		u.SendText(reason)
		u.Character.SetAggro(a.UserId, a.MobInstanceId, characters.DefaultAttack)
	}
	members, err := withdrawal.Present(u, req)
	if err != nil {
		resume(err.Error())
		return
	}
	name, destination, err := withdrawal.Route(u, r, req.ExitName)
	if err != nil {
		resume(err.Error())
		return
	}
	if a.RoundsWaiting > 0 {
		a.RoundsWaiting--
		f, _ := company.FormationFor(u.UserId)
		selected := map[int]bool{}
		for _, m := range members {
			selected[m.InstanceId] = true
		}
		for _, g := range guardMembers(u, f) {
			if g.mob != nil && !selected[g.mob.InstanceId] {
				continue
			}
			if g.char.Aggro != nil && g.char.Aggro.Type == characters.SpellCast {
				continue
			}
			if enemyparty.MemberStrategy(u.UserId, g.key).Role != strategy.Guardian || !ableToGuard(g) || !withdrawal.Eligible(g.char) {
				continue
			}
			if _, ok := battle.SpendGuard(u.UserId, string(g.key)); !ok {
				continue
			}
			req.Cover = 15
			req.CoverKey = string(g.key)
			if g.mob != nil {
				retreatCover[g.mob.InstanceId] = true
			}
			u.SendText(fmt.Sprintf("%s covers the withdrawal (one guard spent).", g.char.Name))
			break
		}
		u.SendText("Your company gathers at the " + name + " exit. The escape attempt comes next round.")
		return
	}
	mobility := withdrawal.Mobility(u.Character)
	for _, m := range members {
		mobility = min(mobility, withdrawal.Mobility(&m.Character))
	}
	pressure := float64(0)
	if b, ok := battle.Current(u.UserId); ok {
		for id := range b.Enemies {
			m := mobs.GetInstance(id)
			if m != nil && m.Character.RoomId == r.RoomId && m.Character.Health > 0 && !m.Character.CombatWithdrawn {
				pressure = max(pressure, withdrawal.Mobility(&m.Character))
			}
		}
	}
	// PvP pursuers retain their own battle and cannot be pulled into this company.
	for _, uid := range r.GetPlayers() {
		if uid == u.UserId {
			continue
		}
		other := users.GetByUserId(uid)
		if other != nil && other.Character.Health > 0 && other.Character.Aggro != nil && other.Character.Aggro.UserId == u.UserId {
			pressure = max(pressure, withdrawal.Mobility(other.Character))
		}
	}
	cover := 0
	if req.CoverKey == string(company.LeaderMemberKey) && ableToGuard(guardMember{char: u.Character}) {
		cover = req.Cover
	}
	for _, m := range members {
		_, key, _ := company.LeaderAndKeyForInstance(m.InstanceId)
		if string(key) == req.CoverKey && ableToGuard(guardMember{char: &m.Character}) {
			cover = req.Cover
		}
	}
	chance := withdrawal.Chance(mobility, pressure, cover)
	if chance < 100 && retreatRoll(100) >= chance {
		resume("The enemy holds your company here. The ordered retreat fails; your company resumes fighting.")
		return
	}
	origin := u.Character.RoomId
	if err := rooms.MoveToRoom(u.UserId, destination); err != nil {
		resume("Your company cannot reach that exit.")
		return
	}
	// MoveToRoom can map a special destination into the player's private room.
	destination = u.Character.RoomId
	ids := []int{}
	for _, m := range members {
		ids = append(ids, m.InstanceId)
	}
	if err := company.RelocateWithdrawal(u.UserId, origin, destination, ids); err != nil {
		_ = rooms.MoveToRoom(u.UserId, origin)
		resume("Your company could not withdraw together; the order is cancelled.")
		return
	}
	u.Character.Aggro = nil
	delete(castAims, caster{userId: u.UserId})
	for _, m := range members {
		m.Character.Aggro = nil
		delete(castAims, caster{mobId: m.InstanceId})
		delete(windUps, m.InstanceId)
		delete(windUpCooldown, m.InstanceId)
		clearAimsAt(m.InstanceId)
	}
	// Remove departed leaders from weapon and captured spell targets without
	// ending any other company's battle.
	for _, id := range mobs.GetAllMobInstanceIds() {
		if m := mobs.GetInstance(id); m != nil {
			clearDepartedLeader(&m.Character, u.UserId)
		}
	}
	for _, uid := range users.GetOnlineUserIds() {
		if other := users.GetByUserId(uid); other != nil {
			clearDepartedLeader(other.Character, u.UserId)
		}
	}
	emitCombat(combatstream.Event{Kind: combatstream.Flee, RoomId: origin, Source: userRef(u), Outcome: "ordered-retreat"})
	endBattle(u.UserId, combatstream.OutcomeBrokenOff)
	u.SendText("Your company withdraws together " + name + ".")
	r.SendText(u.Character.Name+" leads the company out "+name+".", u.UserId)
	scripting.TryRoomScriptEvent("onExit", u.UserId, origin)
	if show, err := scripting.TryRoomScriptEvent("onEnter", u.UserId, destination); err != nil || show {
		usercommands.Look("", u, rooms.LoadRoom(destination), 0)
	}
}

// Emergency escape may separate blocked living companions. Save 30e's return
// obligation before removing their live instances; a save failure holds the
// leader here. Already saved flights remain pending until this battle ends.
func prepareEmergencySeparation(u *users.UserRecord, r *rooms.Room) error {
	req := withdrawal.Capture(u, r, "")
	for _, member := range req.Members {
		m := mobs.GetInstance(member.InstanceID)
		if m == nil || m.Character.Health <= 0 || withdrawal.Eligible(&m.Character) {
			continue
		}
		cid, ok := company.CompanionIDFromMemberKey(company.MemberKey(member.Key))
		if !ok {
			return company.ErrUnknownMember
		}
		if err := company.BeginFlight(u.UserId, cid); err != nil {
			return err
		}
		cancelNerveCast(m)
		clearAimsAt(m.InstanceId)
		u.SendText(m.Character.Name + " is separated in the escape and will rejoin after the battle (5 loyalty).")
	}
	return nil
}

func clearDepartedLeader(c *characters.Character, uid int) {
	if c == nil || c.Aggro == nil {
		return
	}
	a := c.Aggro
	if a.Type != characters.SpellCast && a.UserId == uid {
		c.Aggro = nil
		return
	}
	if a.Type == characters.SpellCast {
		ids := a.SpellInfo.TargetUserIds[:0]
		for _, target := range a.SpellInfo.TargetUserIds {
			if target != uid {
				ids = append(ids, target)
			}
		}
		a.SpellInfo.TargetUserIds = ids
	}
}
