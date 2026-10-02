package hooks

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/interrupt"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/windup"
)

// Phase 30d2: physical wind-ups (internal/windup holds the abilities). An
// enemy whose template lists `windups` may spend a turn, in plain view,
// preparing a heavy blow (the ogre's Crushing Blow). At its next turn the
// blow lands: its ordinary swing, through the gates (reach, interception,
// guards), resolved by internal/combat as one strike with the ability's
// power. Only heavy force breaks a wind-up (a crit that landed, a stagger,
// a knockdown, a stun); an ordinary blow does nothing and is not told. A
// blow that lands, is wasted, or is broken is followed by a cooldown.
//
// All of it is runtime only, on the game loop. Aggro is never saved, so a
// restart or copyover mid-fight drops a wind-up with the fight.

// windUpState is a foe winding up.
type windUpState struct {
	ability   windup.Ability
	turnsLeft int // its turns still to spend before the blow lands
	userId    int // the target it named at the start (a player), or
	mobId     int // a mob
}

// landingState is the foe whose swing this turn is a wind-up's blow.
type landingState struct {
	mobId   int
	ability windup.Ability
	struck  bool // combat was handed the power: the swing happened
	told    bool // its WindUpLand was emitted
}

var (
	// windUps are the foes winding up, by mob instance id.
	windUps = map[int]*windUpState{}
	// windUpCooldown counts the turns each foe must still swing normally
	// before it may wind up again.
	windUpCooldown = map[int]int{}
	// landing is the wind-up landing in the current mob's turn, if any.
	landing *landingState
	// windUpRoll rolls whether a foe starts a wind-up; tests replace it.
	windUpRoll = util.Rand
)

func init() {
	combat.SetPowerProvider(windUpPower)
}

// UseWindUpRollForTest replaces the wind-up start dice until the returned
// restore is called.
func UseWindUpRollForTest(roll func(int) int) (restore func()) {
	prev := windUpRoll
	windUpRoll = roll
	return func() { windUpRoll = prev }
}

// WindingUp reports whether a mob is winding up a blow now (for tests and
// the look of a fight).
func WindingUp(mobInstanceId int) bool {
	_, ok := windUps[mobInstanceId]
	return ok
}

// windUpPower hands internal/combat the power of a landing foe's swing,
// once: the swing (or the guard's, 30c2) is the Crushing Blow.
func windUpPower(mobInstanceId int) (combat.Power, bool) {
	if p, ok := battlefieldPowers[mobInstanceId]; ok {
		return p, true
	}
	if landing == nil || landing.mobId != mobInstanceId || landing.struck {
		return combat.Power{}, false
	}
	landing.struck = true
	p := combat.Power{Name: landing.ability.Name, Multiplier: landing.ability.Multiplier}
	if landing.ability.KnockDown {
		p.Status = status.KnockedDown
	}
	return p, true
}

// windUpRound starts a combat round: a landing left over is told, and a
// wind-up of a foe gone, fallen, or no longer on a plain attack (casting,
// fleeing, out of the fight) is dropped. A cooldown ends with its foe's
// fight.
func windUpRound() {
	finishLanding()
	members := fightMembers()
	for id := range windUps {
		m := mobs.GetInstance(id)
		if m == nil || m.Character.Health < 1 || m.Character.Aggro == nil || m.Character.Aggro.Type != characters.DefaultAttack {
			delete(windUps, id)
		}
	}
	for id := range windUpCooldown {
		if m := mobs.GetInstance(id); m == nil || m.Character.Health < 1 || !mobHolder(m).inFight(members) {
			delete(windUpCooldown, id)
		}
	}
}

// isLanding reports whether the mob's swing this turn is a wind-up's.
func isLanding(mobInstanceId int) bool {
	return landing != nil && landing.mobId == mobInstanceId
}

// windUpTurn runs a mob's wind-up at its physical turn, before its combat
// commands. It reports whether the turn is spent: winding up, starting
// one, or a blow wasted. When a blow lands it returns false, and the
// mob's swing that follows is the blow.
func windUpTurn(mob *mobs.Mob) bool {
	id := mob.InstanceId
	if st := windUps[id]; st != nil {
		st.turnsLeft--
		if st.turnsLeft > 0 {
			return true
		}
		delete(windUps, id)
		windUpCooldown[id] = windup.Cooldown
		if !aimWindUp(mob, st) {
			wasteWindUp(mob, st.ability)
			return true
		}
		windUpLine(mob, st.ability.Release, ``, ``)
		landing = &landingState{mobId: id, ability: st.ability}
		return false
	}
	if windUpCooldown[id] > 0 {
		windUpCooldown[id]--
		return false
	}
	if a, target, ok := startsWindUp(mob); ok {
		startWindUp(mob, a, target)
		return true
	}
	return false
}

// finishLanding closes the landing of the turn just over: a blow whose
// swing never happened (its target out of reach, a hold) is told as
// wasted.
func finishLanding() {
	l := landing
	landing = nil
	if l == nil || l.struck {
		return
	}
	if m := mobs.GetInstance(l.mobId); m != nil && m.Character.Health >= 1 {
		wasteWindUp(m, l.ability)
	}
}

// aimWindUp aims a landing foe at the target it named, or at its aim
// when that one has gone; false when neither stands in its room.
func aimWindUp(mob *mobs.Mob, st *windUpState) bool {
	if standsBy(mob, st.userId, st.mobId) {
		mob.Character.SetAggro(st.userId, st.mobId, characters.DefaultAttack, 0)
		return true
	}
	if agg := mob.Character.Aggro; agg != nil && standsBy(mob, agg.UserId, agg.MobInstanceId) {
		mob.Character.SetAggro(agg.UserId, agg.MobInstanceId, characters.DefaultAttack, 0)
		return true
	}
	return false
}

// standsBy reports whether the player userId, or else the mob mobId,
// stands alive in the mob's room.
func standsBy(mob *mobs.Mob, userId, mobId int) bool {
	roomId := mob.Character.RoomId
	if userId > 0 {
		u := users.GetByUserId(userId)
		return u != nil && u.Character.RoomId == roomId && u.Character.Health >= 1
	}
	if mobId > 0 {
		m := mobs.GetInstance(mobId)
		return m != nil && m.Character.RoomId == roomId && m.Character.Health >= 1
	}
	return false
}

// startsWindUp decides whether an enemy starts a wind-up this turn: it has
// one, is no company companion or charmed pet, has no wait, and its aim
// stands where it can strike it now. Each ability rolls its chance, in id
// order.
func startsWindUp(mob *mobs.Mob) (windup.Ability, statusHolder, bool) {
	if len(mob.WindUps) == 0 || mob.Character.GetCharmedUserId() > 0 {
		return windup.Ability{}, statusHolder{}, false
	}
	if _, _, companion := company.LeaderAndKeyForInstance(mob.InstanceId); companion {
		return windup.Ability{}, statusHolder{}, false
	}
	agg := mob.Character.Aggro
	if agg == nil || agg.Type != characters.DefaultAttack || agg.RoundsWaiting > 0 {
		return windup.Ability{}, statusHolder{}, false
	}
	target, ok := windUpAim(mob)
	if !ok {
		return windup.Ability{}, statusHolder{}, false
	}
	ids := make([]string, 0, len(mob.WindUps))
	for id := range mob.WindUps {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		a, known := windup.Get(id)
		if known && windup.RollStart(mob.WindUps[id], windUpRoll) {
			return a, target, true
		}
	}
	return windup.Ability{}, statusHolder{}, false
}

// windUpAim is the mob's aim when it could strike it now: in its room,
// standing, seen, not held back by a battle (29b2), and in reach through
// the company's formation (11c's legality; a guard or an interceptor may
// still take the blow).
func windUpAim(mob *mobs.Mob) (statusHolder, bool) {
	agg := mob.Character.Aggro
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return statusHolder{}, false
	}
	reach := combat.ResolveReach(&mob.Character, mob.Reach)
	if agg.UserId > 0 {
		u := users.GetByUserId(agg.UserId)
		if u == nil || u.Character.RoomId != mob.Character.RoomId || u.Character.Health < 1 || u.Character.HasBuffFlag("hidden") || holdsAgainstPlayer(mob, u.UserId) {
			return statusHolder{}, false
		}
		return userHolder(u), reachesMember(mob, room, u, company.LeaderMemberKey, reach)
	}
	d := mobs.GetInstance(agg.MobInstanceId)
	if d == nil || d.Character.RoomId != mob.Character.RoomId || d.Character.Health < 1 || d.Character.HasBuffFlag("hidden") || mobHolds(mob, d) {
		return statusHolder{}, false
	}
	leaderId, key, companion := company.LeaderAndKeyForInstance(d.InstanceId)
	if !companion {
		return mobHolder(d), true
	}
	leader := users.GetByUserId(leaderId)
	if leader == nil {
		return mobHolder(d), true
	}
	return mobHolder(d), reachesMember(mob, room, leader, key, reach)
}

// reachesMember reports whether the mob may strike a company member now
// (11c's legality, failing open as the gates do).
func reachesMember(mob *mobs.Mob, room *rooms.Room, leader *users.UserRecord, key company.MemberKey, reach formationcombat.Reach) bool {
	f, ok := enemyparty.CompanyFormation(leader.UserId)
	if !ok {
		return true
	}
	col, ok := resolveHostileAttackerColumn(room, mob.InstanceId)
	if !ok {
		return true
	}
	_, legal := resolveAttackTarget(col, f, key, aliveMapForCompany(leader, f), reach, groundForMob(mob, leader.UserId))
	return legal
}

// startWindUp begins a wind-up: the telegraph (the target told "you"), the
// event; the turn is spent.
func startWindUp(mob *mobs.Mob, a windup.Ability, target statusHolder) {
	st := &windUpState{ability: a, turnsLeft: a.Rounds}
	if target.user != nil {
		st.userId = target.user.UserId
	} else if target.mob != nil {
		st.mobId = target.mob.InstanceId
	}
	windUps[mob.InstanceId] = st

	plural := `rounds`
	if a.Rounds == 1 {
		plural = `round`
	}
	suffix := fmt.Sprintf(` (winding up: %s, %d %s)`, a.Name, a.Rounds, plural)
	windUpLine(mob, a.Telegraph, suffix, target.tag(), target.user)
	emitCombat(combatstream.Event{Kind: combatstream.WindUpStart, RoomId: mob.Character.RoomId, Source: mobRef(mob), Target: target.ref, SpellId: a.Id, Status: a.Name})
}

// wasteWindUp tells a blow with nothing left to strike.
func wasteWindUp(mob *mobs.Mob, a windup.Ability) {
	windUpLine(mob, a.Wasted, fmt.Sprintf(` (%s wasted)`, a.Name), ``)
	emitCombat(combatstream.Event{Kind: combatstream.WindUpLand, RoomId: mob.Character.RoomId, Source: mobRef(mob), SpellId: a.Id, Status: a.Name, Outcome: combatstream.OutcomeWasted})
}

// breakWindUp breaks a foe's wind-up: the line, an Interrupt credited to
// the striker (none, and the Lost line, when a status alone cost it the
// turn), the cooldown.
func breakWindUp(by statusHolder, mob *mobs.Mob) {
	st := windUps[mob.InstanceId]
	if st == nil {
		return
	}
	delete(windUps, mob.InstanceId)
	windUpCooldown[mob.InstanceId] = windup.Cooldown
	a := st.ability
	line := a.Broken
	if by.ref.Zero() {
		line = a.Lost
	}
	windUpLine(mob, line, fmt.Sprintf(` (%s interrupted)`, a.Name), ``)
	if !by.ref.Zero() {
		emitCombat(combatstream.Event{Kind: combatstream.Interrupt, RoomId: mob.Character.RoomId, Source: by.ref, Target: mobRef(mob), SpellId: a.Id, Status: a.Name, Outcome: combatstream.OutcomeSucceeded})
	}
}

// afterWindUpBlow applies what a resolved blow does to wind-ups: a landing
// foe's swing is reported, and heavy force on a foe winding up breaks it.
// Called from afterBlow at every blow site.
func afterWindUpBlow(attacker, defender statusHolder, r combat.AttackResult) {
	if l := landing; l != nil && l.struck && !l.told && attacker.mob != nil && attacker.mob.InstanceId == l.mobId {
		l.told = true
		outcome := combatstream.OutcomeMiss
		if r.Hit && r.DamageToTarget > 0 { // a blow the armor took reads as a miss (29c)
			outcome = combatstream.OutcomeHit
		}
		emitCombat(combatstream.Event{Kind: combatstream.WindUpLand, RoomId: attacker.char.RoomId, Source: attacker.ref, Target: defender.ref,
			SpellId: l.ability.Id, Status: l.ability.Name, Outcome: outcome, Damage: r.DamageToTarget, Crit: r.CritLanded})
	}
	if defender.mob == nil || defender.char.Health < 1 {
		return
	}
	if _, winding := windUps[defender.mob.InstanceId]; winding && interrupt.BreaksWindUp(r.Hit, r.DamageToTarget, heavyBlow(r)) {
		breakWindUp(attacker, defender.mob)
	}
}

// windUpLostTurn drops the wind-up of a foe a status cost its turn (a
// stun or knockdown not from a blow, which would already have broken it).
func windUpLostTurn(mob *mobs.Mob) {
	breakWindUp(statusHolder{}, mob)
}

// windUpLine tells the room a wind-up line in the 29c voice. A player
// named as the target (you) is told it with "you" and left out of the
// room's.
func windUpLine(mob *mobs.Mob, line, suffix, target string, you ...*users.UserRecord) {
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return
	}
	actor := named(mobTag(mobName(mob.InstanceId)))
	his := mob.Character.CombatPronouns().Possessive
	weapon := windUpWeapon(&mob.Character)
	var exclude []int
	if len(you) > 0 && you[0] != nil {
		you[0].SendText(util.CapitalizeFirst(windup.Render(line, actor, his, weapon, `you`)) + suffix)
		exclude = append(exclude, you[0].UserId)
	}
	room.SendText(util.CapitalizeFirst(windup.Render(line, actor, his, weapon, target))+suffix, exclude...)
}

// windUpWeapon is the weapon a line names: the held weapon's simple name,
// else the race's unarmed one.
func windUpWeapon(c *characters.Character) string {
	if c.Equipment.Weapon.ItemId > 0 {
		return c.Equipment.Weapon.NameSimple()
	}
	if r := races.GetRace(c.GetRaceId()); r != nil && r.UnarmedName != `` {
		return r.UnarmedName
	}
	return `fists`
}
