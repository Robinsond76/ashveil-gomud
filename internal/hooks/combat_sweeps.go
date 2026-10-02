package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

var battlefieldRound uint64
var leapCooldown = map[int]uint64{}
var sweepCooldown = map[int]uint64{}
var battlefieldPowers = map[int]combat.Power{}

func beginBattlefieldRound() {
	battlefieldRound++
	clear(battlefieldPowers)
	for _, cooldown := range []map[int]uint64{leapCooldown, sweepCooldown} {
		for id := range cooldown {
			if mobs.GetInstance(id) == nil {
				delete(cooldown, id)
			}
		}
	}
}

func enemySpecial(m *mobs.Mob) bool {
	if m == nil || m.Character.IsCharmed() || !canFight(&m.Character) || WindingUp(m.InstanceId) {
		return false
	}
	_, _, companion := company.LeaderAndKeyForInstance(m.InstanceId)
	return !companion && combat.ResolveReach(&m.Character, m.Reach) != formationcombat.ReachAny
}

func leapReady(m *mobs.Mob) bool {
	return enemySpecial(m) && m.Leap && leapCooldown[m.InstanceId] <= battlefieldRound
}

func openLeapColumn(f company.Formation, standing map[company.MemberKey]bool, col int) bool {
	key := f.At(0, col)
	return key == "" || !standing[key]
}

// noteLeap spends the leap only when the strike needed it: a target an
// ordinary strike (or an open flank) already reaches is a plain attack.
func noteLeap(m *mobs.Mob, attackerCol int, f company.Formation, intended, final company.MemberKey, alive map[company.MemberKey]bool, reach formationcombat.Reach, owner int) {
	if !leapReady(m) || intended != final {
		return
	}
	plain := groundForMob(m, owner)
	plain.leap = false
	if target, ok := resolveAttackTarget(attackerCol, f, intended, alive, reach, plain); ok && target == final {
		return
	}
	r, c, ok := f.Find(final)
	if !ok || r == 0 {
		return
	}
	standing := enemyparty.Standing(f, alive, owner)
	if !openLeapColumn(f, standing, c) {
		return
	}
	leapCooldown[m.InstanceId] = battlefieldRound + 3
	battlefieldPowers[m.InstanceId] = combat.Power{Name: "leap", Multiplier: 1}
	mobHolder(m).say("", "%s leaps past the broken front line.", " (leap)")
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: m.Character.RoomId, Source: mobRef(m), Status: "leap"})
}

// trySweep is one turn, at most one ordinary weapon strike per living
// front-row opponent. Existing attack helpers preserve scripts, counters, wounds,
// attribution, and affected-member resolution. It cannot recurse.
func trySweep(m *mobs.Mob, owner int, room *rooms.Room) bool {
	if !enemySpecial(m) || !m.Sweep || surprised(0, m.InstanceId) || sweepCooldown[m.InstanceId] > battlefieldRound || physicalReserve(0, m) {
		return false
	}
	u := users.GetByUserId(owner)
	if u == nil || u.Character == nil || u.Character.RoomId != m.Character.RoomId {
		return false
	}
	f, ok := enemyparty.CompanyFormation(owner)
	if !ok {
		return false
	}
	cols := 3
	if enemyparty.Narrow(room) {
		cols = 2
	}
	var keys []company.MemberKey
	alive := enemyparty.CompanyAlive(owner, f)
	for c := 0; c < cols; c++ {
		if key := f.At(0, c); key != "" && alive[key] && !sweepHidden(u, key) {
			keys = append(keys, key)
		}
	}
	if len(keys) < 2 {
		return false
	}
	sweepCooldown[m.InstanceId] = battlefieldRound + 3
	battlefieldPowers[m.InstanceId] = combat.Power{Name: "sweep", Multiplier: 1}
	defer delete(battlefieldPowers, m.InstanceId)
	mobHolder(m).say("", "%s sweeps a heavy blow across the front line.", " (sweep)")
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: room.RoomId, Source: mobRef(m), Status: "sweep"})
	// Each member is struck at most once: a guardian who took a ward's blow
	// isn't struck again, and one already struck doesn't step in again.
	struck := map[company.MemberKey]bool{}
	for _, key := range keys {
		if !canFight(&m.Character) {
			break
		}
		if struck[key] || !enemyparty.CompanyAlive(owner, f)[key] || sweepHidden(u, key) {
			continue
		}
		struck[key] = true
		if guard, ok := guardianFor(u, f, key, struck); ok {
			struck[guard.key] = true
			if guard.user != nil {
				resolveInterceptedAttackOnLeader(m, guard.user, room, room)
			} else if guard.mob != nil {
				resolveInterceptedMobAttack(m, guard.mob, room, room, owner)
			}
			continue
		}
		if key == company.LeaderMemberKey {
			if u.Character.HasBuffFlag("hidden") {
				continue
			}
			resolveInterceptedAttackOnLeader(m, u, room, room)
			continue
		}
		cid, ok := company.CompanionIDFromMemberKey(key)
		if !ok {
			continue
		}
		id, ok := company.InstanceFor(owner, cid)
		if !ok {
			continue
		}
		if target := mobs.GetInstance(id); target != nil && !target.Character.HasBuffFlag("hidden") {
			resolveInterceptedMobAttack(m, target, room, room, owner)
		}
	}
	return true
}

func sweepHidden(u *users.UserRecord, key company.MemberKey) bool {
	if key == company.LeaderMemberKey {
		return u.Character.HasBuffFlag("hidden")
	}
	cid, ok := company.CompanionIDFromMemberKey(key)
	if !ok {
		return true
	}
	id, ok := company.InstanceFor(u.UserId, cid)
	if !ok {
		return true
	}
	m := mobs.GetInstance(id)
	return m == nil || m.Character.HasBuffFlag("hidden")
}
