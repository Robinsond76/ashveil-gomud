package hooks

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func coldWait(h statusHolder) {
	a := h.char.Aggro
	if a == nil || !a.ColdDelayed || a.ColdNotice {
		return
	}
	a.ColdNotice = true
	if a.Type == characters.SpellCast {
		h.say("Your numb fingers slow the chant.", "%s's numb fingers slow the chant.", " (cold: one extra round)")
	} else {
		h.say("Your numb fingers fumble the sling stone.", "%s's numb fingers fumble the sling stone.", " (cold: one extra round)")
	}
}

func consumeAmbush(u *users.UserRecord, p mobparty.Party, room *rooms.Room) {
	advantage, observer, pending := 0, "", false
	for _, id := range p.Members {
		m := mobs.GetInstance(id)
		if m == nil || m.AmbushOwner != u.UserId {
			continue
		}
		advantage, observer, pending = m.AmbushAdvantage, m.AmbushObserver, true
		m.AmbushOwner = 0 // consumed, even if another company already fights it
	}
	if !pending || groupInOtherBattle(u.UserId, room.RoomId, p.ID) {
		return
	}
	battle.SetOpening(u.UserId, advantage)
	subject := observer
	verb := "spots"
	if observer == "you" {
		subject, verb = "You", "spot"
	}
	switch advantage {
	case -1:
		u.SendText("The ambush catches your company off guard. The enemy has the opening round.")
	case 1:
		u.SendText(fmt.Sprintf("%s %s the ambush early. Your company has the opening round.", subject, verb))
	default:
		u.SendText(fmt.Sprintf("%s %s the ambush. Neither side is caught off guard.", subject, verb))
	}
}

// surprised is battle authority, not a buff or a presentation event. A
// second player's fight never refreshes a group's opening advantage.
func surprised(uid, mobID int) bool {
	if mobID > 0 {
		if owner, _, ok := company.LeaderAndKeyForInstance(mobID); ok {
			uid = owner
			mobID = 0
		}
	}
	for _, owner := range battle.Players() {
		b, ok := battle.Current(owner)
		if !ok || b.StartRound != combatRound.Load() {
			continue
		}
		if uid == owner && b.Opening < 0 {
			return true
		}
		if mobID > 0 && b.Has(mobID) && b.Opening > 0 {
			return true
		}
	}
	return false
}

// Ground is shared by target selection and attack resolution.
type groundContext struct {
	room *rooms.Room
	uid  int
	leap bool
}

func groundForMob(m *mobs.Mob, uid int) groundContext {
	return groundContext{room: rooms.LoadRoom(m.Character.RoomId), uid: uid, leap: leapReady(m)}
}

func physicalReserve(uid int, mob *mobs.Mob) bool {
	var room *rooms.Room
	var f company.Formation
	var key company.MemberKey
	if uid > 0 {
		u := users.GetByUserId(uid)
		if u == nil {
			return false
		}
		room = rooms.LoadRoom(u.Character.RoomId)
		f, _ = enemyparty.CompanyFormation(uid)
		key = company.LeaderMemberKey
	} else if mob != nil {
		room = rooms.LoadRoom(mob.Character.RoomId)
		if owner, k, ok := company.LeaderAndKeyForInstance(mob.InstanceId); ok {
			f, _ = enemyparty.CompanyFormation(owner)
			key = k
		} else if p, ok := enemyparty.PartyOf(room, mob.InstanceId); ok {
			f, key = p.Formation, mobparty.MemberKeyFor(mob.InstanceId)
		}
	}
	_, col, ok := f.Find(key)
	return enemyparty.Narrow(room) && ok && col == 2
}
