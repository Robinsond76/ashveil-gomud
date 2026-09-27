package hooks

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/engagement"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 29a: engagement upkeep. Once a company and an enemy party are
// fighting in a room, every living member of both sides keeps a legal
// target for as long as the other side has one to offer. It runs once at
// the start of each combat round, before any attack is resolved, and fixes
// what the 5v5 simulation found: a leader stuck on an unreachable target,
// a killer (or a leader whose target someone else killed) standing idle,
// and the rest of an enemy party watching its members die one at a time.
//
// A member keeps a target that is still legal. A member with no legal
// alternative keeps whatever it has, so 11c's gates skip it until
// something changes (the self-healing model). Spell casts, ranged attacks
// through an exit, and fights with anyone outside this company/party pair
// are left alone. Nothing here is persisted.

// companySide is one leader's company as it stands in the leader's room.
type companySide struct {
	leader       *users.UserRecord
	formation    company.Formation
	companions   map[int]company.MemberKey // live instance id -> member key, in the room
	companionIds []int                     // the same instance ids, ascending, for a stable order
	alive        map[company.MemberKey]bool
}

// upkeepEngagements keeps every engaged company/party pair in the room of
// each online leader fighting as a whole.
func upkeepEngagements() {
	for _, leader := range users.GetAllActiveUsers() {
		if leader == nil || leader.Character == nil {
			continue
		}
		side, ok := loadCompanySide(leader)
		if !ok {
			continue
		}
		room := rooms.LoadRoom(leader.Character.RoomId)
		if room == nil {
			continue
		}
		for _, party := range enemyparty.Parties(room) {
			if !side.engagedWith(party) {
				continue
			}
			refreshHostility(leader, party)
			side.keepCompanyEngaged(party, room)
			side.keepPartyEngaged(party, room)
		}
	}
}

func loadCompanySide(leader *users.UserRecord) (companySide, bool) {
	f, ok := company.FormationFor(leader.UserId)
	if !ok {
		return companySide{}, false
	}
	side := companySide{leader: leader, formation: f, companions: map[int]company.MemberKey{}}
	if room := rooms.LoadRoom(leader.Character.RoomId); room != nil {
		for _, instanceId := range room.GetMobs(rooms.FindCharmed) {
			leaderId, key, isCompanion := company.LeaderAndKeyForInstance(instanceId)
			if isCompanion && leaderId == leader.UserId {
				side.companions[instanceId] = key
				side.companionIds = append(side.companionIds, instanceId)
			}
		}
	}
	sort.Ints(side.companionIds)
	side.alive = aliveMapForCompany(leader, f)
	return side, true
}

// engagedWith reports whether any living member of either side has a plain
// attack on a member of the other.
func (s companySide) engagedWith(party mobparty.Party) bool {
	members := partyMemberSet(party)
	if s.leader.Character.Health > 0 && aggroOnMobIn(s.leader.Character.Aggro, members) {
		return true
	}
	for _, instanceId := range s.companionIds {
		if mob := mobs.GetInstance(instanceId); mob != nil && mob.Character.Health > 0 && aggroOnMobIn(mob.Character.Aggro, members) {
			return true
		}
	}
	for _, instanceId := range party.Members {
		mob := mobs.GetInstance(instanceId)
		if mob == nil || mob.Character.Health < 1 {
			continue
		}
		if aggroOnCompany(mob.Character.Aggro, s.leader.UserId, s.companions) {
			return true
		}
	}
	return false
}

func partyMemberSet(party mobparty.Party) map[int]bool {
	set := make(map[int]bool, len(party.Members))
	for _, id := range party.Members {
		set[id] = true
	}
	return set
}

func aggroOnMobIn(a *characters.Aggro, members map[int]bool) bool {
	return plainAttack(a) && a.MobInstanceId > 0 && members[a.MobInstanceId]
}

func aggroOnCompany(a *characters.Aggro, leaderId int, companions map[int]company.MemberKey) bool {
	if !plainAttack(a) {
		return false
	}
	if a.UserId > 0 {
		return a.UserId == leaderId
	}
	_, ok := companions[a.MobInstanceId]
	return ok && a.MobInstanceId > 0
}

// plainAttack reports whether a is an ordinary melee or ranged attack in
// the same room: the only kind of Aggro the upkeep counts or replaces.
func plainAttack(a *characters.Aggro) bool {
	return a != nil && (a.Type == characters.DefaultAttack || a.Type == characters.Shooting) && a.ExitName == ``
}

// attackType is the Aggro type a retargeted member keeps: its current one
// (a same-room `shoot` stays a shot), or DefaultAttack when it had none.
func attackType(a *characters.Aggro) characters.AggroType {
	if plainAttack(a) {
		return a.Type
	}
	return characters.DefaultAttack
}

// retargetable reports whether the upkeep may give a member a target: it
// has none, or its current one is a plain attack.
func retargetable(a *characters.Aggro) bool {
	return a == nil || plainAttack(a)
}

// refreshHostility keeps every group of an engaged party hostile to the
// leader for as long as the fight lasts, with the duration the leader's own
// blow sets (NewRound_DoCombat.go), so hostility can't wear off mid-fight.
func refreshHostility(leader *users.UserRecord, party mobparty.Party) {
	rounds := configs.GetTimingConfig().MinutesToRounds(2) - leader.Character.Stats.Perception.ValueAdj
	for _, instanceId := range party.Members {
		mob := mobs.GetInstance(instanceId)
		if mob == nil {
			continue
		}
		for _, group := range mob.Groups {
			mobs.MakeHostile(group, leader.UserId, rounds)
		}
	}
}

// keepCompanyEngaged gives each living company member in the room a legal
// party target when it has none, or its target is dead, gone, or out of
// reach.
func (s companySide) keepCompanyEngaged(party mobparty.Party, room *rooms.Room) {
	alive := enemyparty.Alive(party)
	members := partyMemberSet(party)

	leader := s.leader
	if leader.Character.Health > 0 && retargetable(leader.Character.Aggro) {
		col, placed := s.column(company.LeaderMemberKey)
		reach := combat.ResolveReach(leader.Character, false)
		if previous, newId, ok := s.retarget(leader.Character.Aggro, col, placed, reach, party, members, alive, room); ok {
			leader.Character.SetAggro(0, newId, attackType(leader.Character.Aggro))
			events.AddToQueue(events.AggroChanged{UserId: leader.UserId, RoomId: leader.Character.RoomId})
			leader.SendText(leaderTurnText(previous, newId, alive))
		}
	}

	for _, instanceId := range s.companionIds {
		mob := mobs.GetInstance(instanceId)
		if mob == nil || mob.Character.Health < 1 || !retargetable(mob.Character.Aggro) {
			continue
		}
		col, placed := s.column(s.companions[instanceId])
		reach := combat.ResolveReach(&mob.Character, mob.Reach)
		_, newId, ok := s.retarget(mob.Character.Aggro, col, placed, reach, party, members, alive, room)
		if !ok {
			continue
		}
		mob.Character.SetAggro(0, newId, attackType(mob.Character.Aggro))
		events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
		room.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> turns on <ansi fg="mobname">%s</ansi>.`, mob.Character.Name, mobName(newId)))
	}
}

// retarget decides a company member's new party target. ok=false means
// leave its Aggro as it is: its target is still legal, it is fighting
// someone outside this party, or no legal alternative exists. previous is
// the party member it was aimed at, if any.
func (s companySide) retarget(a *characters.Aggro, col int, placed bool, reach formationcombat.Reach, party mobparty.Party, members map[int]bool, alive map[company.MemberKey]bool, room *rooms.Room) (previous int, newId int, ok bool) {
	state, current := classifyPartyTarget(a, members, alive, room.RoomId)
	switch state {
	case targetElsewhere:
		return 0, 0, false
	case targetInParty:
		if gateLetsThrough(col, placed, party.Formation, mobparty.MemberKeyFor(current), alive, reach) {
			return 0, 0, false
		}
	}
	newId, ok = chooseFromParty(col, placed, party, alive, reach)
	if !ok || newId == current {
		return 0, 0, false
	}
	return current, newId, true
}

// targetState classifies an attacker's current Aggro for the upkeep.
type targetState int

const (
	targetNone      targetState = iota // no target, or a dead or gone one: give it one
	targetInParty                      // a living member of the party in question
	targetElsewhere                    // someone alive and here, outside the fight: leave it
)

// classifyPartyTarget classifies a company member's Aggro against party.
// current is the party member it names (alive or dead), else 0.
func classifyPartyTarget(a *characters.Aggro, members map[int]bool, alive map[company.MemberKey]bool, roomId int) (targetState, int) {
	if a == nil {
		return targetNone, 0
	}
	if a.UserId > 0 {
		if u := users.GetByUserId(a.UserId); u != nil && u.Character.Health > 0 && u.Character.RoomId == roomId {
			return targetElsewhere, 0
		}
		return targetNone, 0
	}
	if members[a.MobInstanceId] {
		if alive[mobparty.MemberKeyFor(a.MobInstanceId)] {
			return targetInParty, a.MobInstanceId
		}
		return targetNone, a.MobInstanceId
	}
	if mob := mobs.GetInstance(a.MobInstanceId); mob != nil && mob.Character.Health > 0 && mob.Character.RoomId == roomId {
		return targetElsewhere, 0
	}
	return targetNone, 0
}

func (s companySide) column(key company.MemberKey) (int, bool) {
	_, col, found := s.formation.Find(key)
	return col, found
}

// gateLetsThrough reports whether the attack gates would let an attack on
// target go ahead this round: it is legal, or 11c's front-row interception
// catches it on a legal member in front (resolveAttackTarget). The upkeep
// only moves an aim the gates would skip ("You can't reach that target"),
// so interception works exactly as before. An unplaced attacker fails open.
func gateLetsThrough(col int, placed bool, f company.Formation, target company.MemberKey, alive map[company.MemberKey]bool, reach formationcombat.Reach) bool {
	if !placed {
		return alive[target]
	}
	_, ok := resolveAttackTarget(col, f, target, alive, reach)
	return ok
}

// legalAgainstParty reports whether a company attacker (in col, if placed)
// may strike target in party. An unplaced attacker fails open, as it does
// at the attack gates.
func legalAgainstParty(col int, placed bool, party mobparty.Party, target int, alive map[company.MemberKey]bool, reach formationcombat.Reach) bool {
	if !placed {
		return alive[mobparty.MemberKeyFor(target)]
	}
	return formationcombat.Legal(col, party.Formation, mobparty.MemberKeyFor(target), alive, reach)
}

// chooseFromParty picks the weakest legal living member of party for a
// company attacker (11b's preference). An unplaced attacker may choose any
// living member.
func chooseFromParty(col int, placed bool, party mobparty.Party, alive map[company.MemberKey]bool, reach formationcombat.Reach) (int, bool) {
	legal := func(_, defender engagement.Combatant) bool {
		return legalAgainstParty(col, placed, party, defender.ID, alive, reach)
	}
	return engagement.AssignTarget(engagement.Combatant{Col: col}, partyCombatants(party, alive), engagement.Weakest, legal)
}

func leaderTurnText(previous, newId int, alive map[company.MemberKey]bool) string {
	name := mobName(newId)
	if previous > 0 && alive[mobparty.MemberKeyFor(previous)] {
		return fmt.Sprintf(`You can't reach <ansi fg="mobname">%s</ansi> from here. You turn on <ansi fg="mobname">%s</ansi>.`, mobName(previous), name)
	}
	return fmt.Sprintf(`You turn on <ansi fg="mobname">%s</ansi>.`, name)
}

func mobName(instanceId int) string {
	if mob := mobs.GetInstance(instanceId); mob != nil {
		return mob.Character.Name
	}
	return `someone`
}

// keepPartyEngaged gives each living party member a legal company target
// when it has none, or its target is dead, gone, or out of reach. A party
// member fighting someone else (another player, a mob outside this
// company) is left alone.
func (s companySide) keepPartyEngaged(party mobparty.Party, room *rooms.Room) {
	candidates, keys := s.combatants()
	if len(candidates) == 0 {
		return
	}
	for _, instanceId := range party.Members {
		mob := mobs.GetInstance(instanceId)
		if mob == nil || mob.Character.Health < 1 || !retargetable(mob.Character.Aggro) {
			continue
		}
		current, keep, other := s.currentCompanyTarget(mob.Character.Aggro, room)
		if other {
			continue
		}
		_, attackerCol, found := party.Formation.Find(mobparty.MemberKeyFor(instanceId))
		if !found {
			continue
		}
		reach := combat.ResolveReach(&mob.Character, mob.Reach)
		if keep && gateLetsThrough(attackerCol, true, s.formation, current, s.alive, reach) {
			continue
		}
		legal := func(_, defender engagement.Combatant) bool {
			return s.legalAgainstCompany(attackerCol, keys[defender.ID], reach)
		}
		idx, ok := engagement.AssignTarget(engagement.Combatant{Col: attackerCol}, candidates, engagement.Weakest, legal)
		if !ok || keys[idx] == current {
			continue
		}
		s.aimPartyMember(mob, keys[idx], room)
	}
}

// combatants lists the company's living members in the room as
// engagement.Combatants. Each ID is an index into keys.
func (s companySide) combatants() ([]engagement.Combatant, []company.MemberKey) {
	var candidates []engagement.Combatant
	var keys []company.MemberKey
	add := func(key company.MemberKey, hp int) {
		row, col, _ := s.formation.Find(key)
		candidates = append(candidates, engagement.Combatant{ID: len(keys), HP: hp, Row: row, Col: col})
		keys = append(keys, key)
	}
	if s.leader.Character.Health > 0 {
		add(company.LeaderMemberKey, s.leader.Character.Health)
	}
	for _, instanceId := range s.companionIds {
		if mob := mobs.GetInstance(instanceId); mob != nil && mob.Character.Health > 0 {
			add(s.companions[instanceId], mob.Character.Health)
		}
	}
	return candidates, keys
}

// currentCompanyTarget classifies a party member's Aggro against this
// company: keep=true with the member key it names when that member is
// alive here; other=true when it is fighting someone outside the company
// who is still alive and here (left alone).
func (s companySide) currentCompanyTarget(a *characters.Aggro, room *rooms.Room) (current company.MemberKey, keep bool, other bool) {
	if a == nil {
		return "", false, false
	}
	if a.UserId > 0 {
		if a.UserId != s.leader.UserId {
			if u := users.GetByUserId(a.UserId); u != nil && u.Character.Health > 0 && u.Character.RoomId == room.RoomId {
				return "", false, true
			}
			return "", false, false
		}
		return company.LeaderMemberKey, s.leader.Character.Health > 0, false
	}
	if key, ok := s.companions[a.MobInstanceId]; ok {
		return key, s.alive[key], false
	}
	if mob := mobs.GetInstance(a.MobInstanceId); mob != nil && mob.Character.Health > 0 && mob.Character.RoomId == room.RoomId {
		return "", false, true
	}
	return "", false, false
}

// legalAgainstCompany reports whether an enemy in attackerCol may strike
// key. A member not placed in the formation fails open (always legal), as
// it does at the attack gates.
func (s companySide) legalAgainstCompany(attackerCol int, key company.MemberKey, reach formationcombat.Reach) bool {
	if key == "" {
		return false
	}
	if _, _, placed := s.formation.Find(key); !placed {
		return s.alive[key]
	}
	return formationcombat.Legal(attackerCol, s.formation, key, s.alive, reach)
}

func (s companySide) aimPartyMember(mob *mobs.Mob, key company.MemberKey, room *rooms.Room) {
	targetName := s.leader.Character.Name
	if key == company.LeaderMemberKey {
		mob.Character.SetAggro(s.leader.UserId, 0, attackType(mob.Character.Aggro))
	} else {
		var instanceId int
		for _, id := range s.companionIds {
			if s.companions[id] == key {
				instanceId = id
				break
			}
		}
		target := mobs.GetInstance(instanceId)
		if target == nil {
			return
		}
		targetName = target.Character.Name
		mob.Character.SetAggro(0, instanceId, attackType(mob.Character.Aggro))
	}
	mob.PreventIdle = true
	events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
	room.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> turns on <ansi fg="username">%s</ansi>.`, mob.Character.Name, targetName))
}
