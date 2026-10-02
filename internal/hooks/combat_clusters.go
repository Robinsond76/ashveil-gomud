package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// clusterSpell recomputes sparks' actual occupants at completion. The
// anchor was captured at cast start; a dead anchor does not silently retarget.
func clusterSpell(uid, mid int, info *characters.SpellAggroInfo) bool {
	if info == nil || info.SpellId != "sparks" {
		return true
	}
	anchorUser, anchorMob := info.ClusterUserID, info.ClusterMobID
	if anchorUser == 0 && anchorMob == 0 {
		if len(info.TargetUserIds) > 0 {
			anchorUser = info.TargetUserIds[0]
		} else if len(info.TargetMobInstanceIds) > 0 {
			anchorMob = info.TargetMobInstanceIds[0]
		}
	}
	var room *rooms.Room
	if uid > 0 {
		if u := users.GetByUserId(uid); u != nil {
			room = rooms.LoadRoom(u.Character.RoomId)
		}
	} else if m := mobs.GetInstance(mid); m != nil {
		room = rooms.LoadRoom(m.Character.RoomId)
	}
	info.TargetUserIds, info.TargetMobInstanceIds = nil, nil
	if room == nil {
		return false
	}
	owner, key := anchorUser, company.LeaderMemberKey
	if anchorMob > 0 {
		m := mobs.GetInstance(anchorMob)
		if m == nil || m.Character.Health < 1 || m.Character.RoomId != room.RoomId || m.Character.HasBuffFlag("hidden") || m.Character.CombatWithdrawn {
			return false
		}
		if o, k, ok := company.LeaderAndKeyForInstance(anchorMob); ok {
			owner, key = o, k
		} else {
			p, ok := enemyparty.PartyOf(room, anchorMob)
			if !ok {
				info.TargetMobInstanceIds = []int{anchorMob}
				return true
			}
			casterOwner := uid
			if mid > 0 {
				casterOwner, _, _ = company.LeaderAndKeyForInstance(mid)
			}
			b, inBattle := battle.Current(casterOwner)
			if !inBattle || !b.Has(anchorMob) {
				return false
			}
			alive := enemyparty.Alive(p)
			for _, k := range formationcombat.Cluster(p.Formation, mobparty.MemberKeyFor(anchorMob), alive, enemyparty.Narrow(room)) {
				id, ok := mobparty.InstanceIdFromMemberKey(k)
				if !ok || !b.Has(id) {
					continue
				}
				if target := mobs.GetInstance(id); target != nil && target.Character.RoomId == room.RoomId && !target.Character.HasBuffFlag("hidden") {
					info.TargetMobInstanceIds = append(info.TargetMobInstanceIds, id)
				}
			}
			return len(info.TargetMobInstanceIds) > 0
		}
	}
	u := users.GetByUserId(owner)
	if u == nil || u.Character == nil || u.Character.RoomId != room.RoomId || (anchorUser > 0 && (u.Character.Health < 1 || u.Character.HasBuffFlag("hidden"))) {
		return false
	}
	if mid > 0 {
		if b, ok := battle.Current(owner); !ok || !b.Has(mid) {
			return false
		}
	}
	f, formed := enemyparty.CompanyFormation(owner)
	if !formed {
		if anchorUser > 0 {
			info.TargetUserIds = []int{anchorUser}
		} else {
			info.TargetMobInstanceIds = []int{anchorMob}
		}
		return true
	}
	alive := enemyparty.CompanyAlive(owner, f)
	keys := formationcombat.Cluster(f, key, alive, enemyparty.Narrow(room))
	if _, _, placed := f.Find(key); !placed {
		keys = []company.MemberKey{key}
	}
	for _, k := range keys {
		if k == company.LeaderMemberKey {
			if alive[k] && !u.Character.HasBuffFlag("hidden") {
				info.TargetUserIds = append(info.TargetUserIds, owner)
			}
			continue
		}
		cid, ok := company.CompanionIDFromMemberKey(k)
		if !ok {
			continue
		}
		id, ok := company.InstanceFor(owner, cid)
		if !ok {
			continue
		}
		if m := mobs.GetInstance(id); m != nil && m.Character.Health > 0 && m.Character.RoomId == room.RoomId && !m.Character.HasBuffFlag("hidden") {
			info.TargetMobInstanceIds = append(info.TargetMobInstanceIds, id)
		}
	}
	return len(info.TargetUserIds)+len(info.TargetMobInstanceIds) > 0
}
