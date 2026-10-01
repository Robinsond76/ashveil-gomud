package hooks

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
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
		u.Character.Aggro = nil // nobody to resume fighting: the upkeep aims anew
		if req == nil {
			// fall through: nothing to resume
		} else if m := mobs.GetInstance(req.ResumeMobID); m != nil && m.Character.Health > 0 && m.Character.RoomId == u.Character.RoomId {
			u.Character.SetAggro(0, m.InstanceId, characters.DefaultAttack)
		} else if other := users.GetByUserId(req.ResumeUserID); req.ResumeUserID > 0 && other != nil && other.Character.Health > 0 && other.Character.RoomId == u.Character.RoomId {
			u.Character.SetAggro(other.UserId, 0, characters.DefaultAttack)
		}
		events.AddToQueue(events.AggroChanged{UserId: u.UserId, RoomId: u.Character.RoomId})
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
		if len(members) == 0 {
			u.SendText("You edge toward the " + name + " exit. The escape attempt comes next round.")
		} else {
			u.SendText("Your company gathers at the " + name + " exit. The escape attempt comes next round.")
		}
		return
	}
	mobility := withdrawal.Mobility(u.Character)
	for _, m := range members {
		mobility = min(mobility, withdrawal.Mobility(&m.Character))
	}
	// The fastest pursuer sets the pressure: every active foe of the
	// leader's battle, whoever it is striking (Phase 33c owner review), and
	// with no battle, a foe here aiming at the leader. Players fighting the
	// leader pursue too.
	pressure, pursuer := float64(0), ""
	press := func(c *characters.Character, name string) {
		if p := withdrawal.Mobility(c); p > pressure {
			pressure, pursuer = p, name
		}
	}
	b, inBattle := battle.Current(u.UserId)
	for _, id := range r.GetMobs() {
		m := mobs.GetInstance(id)
		if m == nil || m.Character.Health <= 0 || m.Character.CombatWithdrawn {
			continue
		}
		if inBattle && b.Has(id) || !inBattle && m.Character.Aggro != nil && m.Character.Aggro.UserId == u.UserId {
			press(&m.Character, mobTag(mobName(id)))
		}
	}
	// PvP pursuers retain their own battle and cannot be pulled into this company.
	for _, uid := range r.GetPlayers() {
		if uid == u.UserId {
			continue
		}
		other := users.GetByUserId(uid)
		if other != nil && other.Character.Health > 0 && other.Character.Aggro != nil && other.Character.Aggro.UserId == u.UserId {
			press(other.Character, userTag(other.Character.Name))
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
		resume(util.CapitalizeFirst(named(pursuer)) + " cuts off your withdrawal. The retreat fails; the fight goes on.")
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
	events.AddToQueue(events.AggroChanged{UserId: u.UserId, RoomId: destination})
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
	if len(members) == 0 {
		u.SendText("You break away and withdraw " + name + ".")
		r.SendText(u.Character.Name+" breaks away and withdraws "+name+".", u.UserId)
	} else {
		u.SendText("Your company withdraws together " + name + ".")
		r.SendText(u.Character.Name+" leads the company out "+name+".", u.UserId)
	}
	scripting.TryRoomScriptEvent("onExit", u.UserId, origin)
	if show, err := scripting.TryRoomScriptEvent("onEnter", u.UserId, destination); err != nil || show {
		usercommands.Look("", u, rooms.LoadRoom(destination), 0)
	}
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
