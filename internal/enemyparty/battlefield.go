package enemyparty

import (
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Narrow(room *rooms.Room) bool { return room != nil && room.HasTag("narrow") }

// CompanyFormation is the combat projection; the editor still reads the
// durable company.FormationFor. No saved placement is changed here.
func CompanyFormation(uid int) (company.Formation, bool) {
	f, ok := company.FormationFor(uid)
	u := users.GetByUserId(uid)
	if !ok || u == nil || u.Character == nil {
		return f, ok
	}
	if Narrow(rooms.LoadRoom(u.Character.RoomId)) {
		f = formationcombat.Fold(f, CompanyAlive(uid, f))
	}
	return f, ok
}

func CompanyAlive(uid int, f company.Formation) map[company.MemberKey]bool {
	out := map[company.MemberKey]bool{}
	u := users.GetByUserId(uid)
	if u == nil || u.Character == nil {
		return out
	}
	out[company.LeaderMemberKey] = u.Character.Health > 0 && !u.Character.CombatWithdrawn
	for _, row := range f {
		for _, key := range row {
			id, ok := company.CompanionIDFromMemberKey(key)
			if !ok {
				continue
			}
			instance, ok := company.InstanceFor(uid, id)
			if !ok {
				continue
			}
			m := mobs.GetInstance(instance)
			out[key] = m != nil && m.Character.Health > 0 && m.Character.RoomId == u.Character.RoomId && !m.Character.CombatWithdrawn
		}
	}
	return out
}

func Standing(f company.Formation, alive map[company.MemberKey]bool, uid int) map[company.MemberKey]bool {
	out := map[company.MemberKey]bool{}
	for _, row := range f {
		for _, key := range row {
			if key == "" || !alive[key] {
				continue
			}
			if key == company.LeaderMemberKey {
				if u := users.GetByUserId(uid); u != nil {
					out[key] = !status.Live(u.Character, status.KnockedDown) && !u.Character.HasBuffFlag("hidden") && !u.Character.HasBuffFlag("no-combat")
				}
				continue
			}
			id, ok := mobparty.InstanceIdFromMemberKey(key)
			if !ok {
				if cid, yes := company.CompanionIDFromMemberKey(key); yes {
					id, ok = company.InstanceFor(uid, cid)
				}
			}
			if ok {
				if m := mobs.GetInstance(id); m != nil {
					out[key] = !status.Live(&m.Character, status.KnockedDown) && !m.Character.HasBuffFlag("hidden") && !m.Character.HasBuffFlag("no-combat")
				}
			}
		}
	}
	return out
}

func Legal(room *rooms.Room, uid, col int, f company.Formation, key company.MemberKey, alive map[company.MemberKey]bool, reach formationcombat.Reach) bool {
	return formationcombat.LegalGround(col, f, key, alive, Standing(f, alive, uid), reach, Narrow(room))
}

func groundParty(room *rooms.Room, p mobparty.Party) mobparty.Party {
	if Narrow(room) {
		p.Formation = formationcombat.Fold(p.Formation, Alive(p))
	}
	return p
}
