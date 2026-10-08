package hooks

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/util"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/coordination"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/engagement"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/strategy"
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
//
// Only a company with a companion in the leader's room is kept: a solo
// player fights exactly as in GoMud. A leader who used `break` stays out
// until they attack again (engagement.StandDown). An idle enemy joins only
// if it would attack the leader anyway (a hostile mob, or a group the
// fight has made hostile), and never a shopkeeper or a mob in conversation.

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
		room := rooms.LoadRoom(leader.Character.RoomId)
		if room == nil {
			continue
		}
		side, ok := loadCompanySide(leader, room)
		if !ok || len(side.companionIds) == 0 {
			continue
		}
		engaged := false
		if side.anyAggro(room) {
			for _, party := range enemyparty.Parties(room) {
				// Phase 29b2: only the group the leader is in battle with.
				if !inBattleWith(leader.UserId, party) || !side.engagedWith(party) {
					continue
				}
				engaged = true
				side.markHostility(party)
				side.keepCompanyEngaged(party, room)
				side.keepPartyEngaged(party, room)
			}
		}
		if !engaged {
			// The fight is over: a leader who broke off may be pulled
			// into the next one.
			engagement.Resume(leader.UserId)
		}
	}
}

// anyAggro reports whether anyone in the room (the leader, a companion, or
// any mob) has an Aggro at all: a cheap early exit for rooms at peace.
func (s companySide) anyAggro(room *rooms.Room) bool {
	if s.leader.Character.Aggro != nil {
		return true
	}
	for _, instanceId := range room.GetMobs() {
		if mob := mobs.GetInstance(instanceId); mob != nil && mob.Character.Aggro != nil {
			return true
		}
	}
	return false
}

func loadCompanySide(leader *users.UserRecord, room *rooms.Room) (companySide, bool) {
	f, ok := enemyparty.CompanyFormation(leader.UserId)
	if !ok {
		return companySide{}, false
	}
	side := companySide{leader: leader, formation: f, companions: map[int]company.MemberKey{}}
	for _, instanceId := range room.GetMobs(rooms.FindCharmed) {
		leaderId, key, isCompanion := company.LeaderAndKeyForInstance(instanceId)
		if isCompanion && leaderId == leader.UserId && !isDollInstance(instanceId) { // Phase 39d: a doll has no aim of its own
			side.companions[instanceId] = key
			side.companionIds = append(side.companionIds, instanceId)
		}
	}
	sort.Ints(side.companionIds)
	side.alive = aliveMapForCompany(leader, f)
	// aliveMapForCompany covers only formation-placed members; an unplaced
	// companion here is alive too (and fails open as a target).
	for _, instanceId := range side.companionIds {
		mob := mobs.GetInstance(instanceId)
		side.alive[side.companions[instanceId]] = mob != nil && mob.Character.Health > 0
	}
	return side, true
}

// engagedWith reports whether any living member of either side has a plain
// attack on a member of the other. An attack on a mob that has just fallen
// counts too (Phase 83): when the last target of everyone on both sides falls
// in the same round, each aim is still on a body at the next round's upkeep,
// and reading that as "no one is fighting" broke the battle off with foes
// standing and opened it again as a second fight.
func (s companySide) engagedWith(party mobparty.Party) bool {
	members := partyMemberSet(party)
	if s.leader.Character.Health > 0 && (aggroOnMobIn(s.leader.Character.Aggro, members) || aimedAtTheFallen(s.leader.Character.Aggro)) {
		return true
	}
	for _, instanceId := range s.companionIds {
		if mob := mobs.GetInstance(instanceId); mob != nil && mob.Character.Health > 0 && (aggroOnMobIn(mob.Character.Aggro, members) || aimedAtTheFallen(mob.Character.Aggro)) {
			return true
		}
	}
	for _, instanceId := range party.Members {
		mob := mobs.GetInstance(instanceId)
		if mob == nil || mob.Character.Health < 1 {
			continue
		}
		if aggroOnCompany(mob.Character.Aggro, s.leader.UserId, s.companions) || aimedAtTheFallen(mob.Character.Aggro) {
			return true
		}
	}
	return false
}

// aimedAtTheFallen reports whether a is a plain attack on a mob that has
// fallen or been removed: its holder's fight is not over, only its target.
func aimedAtTheFallen(a *characters.Aggro) bool {
	if !plainAttack(a) || a.MobInstanceId <= 0 {
		return false
	}
	m := mobs.GetInstance(a.MobInstanceId)
	return m == nil || m.Character.Health < 1
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

// retargetKeepingStrike aims c at a new foe after its old one fell mid-round
// (Phase 87). A readied Opening Strike or Aimed Shot goes with it: the
// shot is not lost with its target, and keeps its bonus.
func retargetKeepingStrike(c *characters.Character, userId, mobInstanceId int) {
	old := c.Aggro
	if old != nil && old.Type == characters.BackStab && old.ExitName == `` {
		bonus := old.StrikeBonus
		c.SetAggro(userId, mobInstanceId, characters.DefaultAttack)
		if c.Aggro != nil {
			c.Aggro.Type, c.Aggro.StrikeBonus = characters.BackStab, bonus
		}
		return
	}
	c.SetAggro(userId, mobInstanceId, attackType(old))
}

// retargetable reports whether the upkeep may give a member a target: it
// has none, or its current one is a plain attack.
func retargetable(a *characters.Aggro) bool {
	return a == nil || plainAttack(a)
}

// markHostility keeps an engaged party's hostility to the leader from
// wearing off mid-fight, with the duration the leader's own blow sets
// (NewRound_DoCombat.go). A party member a company member is attacking has
// its groups made hostile, as the leader's blow would do (a companion
// fights on the leader's behalf); any other member's group is only
// refreshed if the fight already made it hostile, so hostility never
// spreads to a group nobody touched.
func (s companySide) markHostility(party mobparty.Party) {
	leaderId := s.leader.UserId
	rounds := configs.GetTimingConfig().MinutesToRounds(2) - s.leader.Character.Stats.Perception.ValueAdj
	attacked := map[int]bool{}
	if a := s.leader.Character.Aggro; plainAttack(a) && a.MobInstanceId > 0 {
		attacked[a.MobInstanceId] = true
	}
	for _, instanceId := range s.companionIds {
		if mob := mobs.GetInstance(instanceId); mob != nil && plainAttack(mob.Character.Aggro) && mob.Character.Aggro.MobInstanceId > 0 {
			attacked[mob.Character.Aggro.MobInstanceId] = true
		}
	}
	for _, instanceId := range party.Members {
		mob := mobs.GetInstance(instanceId)
		if mob == nil {
			continue
		}
		for _, group := range mob.Groups {
			if attacked[instanceId] || mobs.IsHostile(group, leaderId) {
				mobs.MakeHostile(group, leaderId, rounds)
			}
		}
	}
}

// joinsTheFight reports whether an idle party member is drawn into its
// party's fight with the company: only if it would attack the leader
// anyway (a hostile mob, or one whose group is hostile to the leader), and
// never a shopkeeper or a mob in conversation.
func joinsTheFight(mob *mobs.Mob, leaderId int) bool {
	if mob.HasShop() || mob.InConversation() {
		return false
	}
	if mob.Hostile {
		return true
	}
	for _, group := range mob.Groups {
		if mobs.IsHostile(group, leaderId) {
			return true
		}
	}
	return false
}

// canFight reports whether c can act in combat this round at all.
func canFight(c *characters.Character) bool {
	return c.Health > 0 && !c.CombatWithdrawn && !c.HasBuffFlag("no-combat")
}

// keepCompanyEngaged gives each living company member in the room a legal
// party target when it has none, or its target is dead, gone, or out of
// reach.
func (s companySide) keepCompanyEngaged(party mobparty.Party, room *rooms.Room) {
	alive := enemyparty.Alive(party)
	members := partyMemberSet(party)

	leader := s.leader
	if leader.Character.Aggro != nil {
		engagement.Resume(leader.UserId) // they are fighting again
	}
	stoodDown := leader.Character.Aggro == nil && engagement.StoodDown(leader.UserId)
	if canFight(leader.Character) && !stoodDown && retargetable(leader.Character.Aggro) {
		col, placed := s.column(company.LeaderMemberKey)
		reach := combat.ResolveReach(leader.Character, false)
		att := enemyparty.PlayerAttacker(leader)
		if previous, newId, byRule, ok := s.retarget(leader.Character.Aggro, att, col, placed, reach, party, members, alive, room, refocusing[leader.UserId]); ok {
			emitTargetChange(userRef(leader), mobRefById(previous), mobRefById(newId), room.RoomId)
			leader.Character.SetAggro(0, newId, attackType(leader.Character.Aggro))
			events.AddToQueue(events.AggroChanged{UserId: leader.UserId, RoomId: leader.Character.RoomId})
			if enemyparty.HealersDefault(leader.UserId) && enemyHealer(newId) {
				leader.SendText(healerMarked(mobTag(mobName(newId))))
			} else if byRule {
				leader.SendText(turnsToward(`You`, mobTag(mobName(newId))))
			} else {
				leader.SendText(leaderTurnText(previous, newId, alive))
			}
		}
	}

	for _, instanceId := range s.companionIds {
		mob := mobs.GetInstance(instanceId)
		if mob == nil || !canFight(&mob.Character) || !retargetable(mob.Character.Aggro) {
			continue
		}
		col, placed := s.column(s.companions[instanceId])
		reach := combat.ResolveReach(&mob.Character, mob.Reach)
		att := enemyparty.CompanionAttacker(leader.UserId, s.companions[instanceId], mob, s.leaderAim())
		previous, newId, _, ok := s.retarget(mob.Character.Aggro, att, col, placed, reach, party, members, alive, room, refocusing[leader.UserId])
		if !ok {
			continue
		}
		emitTargetChange(mobRef(mob), mobRefById(previous), mobRefById(newId), room.RoomId)
		mob.Character.SetAggro(0, newId, attackType(mob.Character.Aggro))
		events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
		room.SendText(turnsToward(mobTag(mobName(mob.InstanceId)), mobTag(mobName(newId))))
	}
}

// enemyHealer reports whether the mob instance is a healer (Phase 35e).
func enemyHealer(instanceId int) bool {
	m := mobs.GetInstance(instanceId)
	return m != nil && strategy.Role(m.EnemyRole()) == strategy.Healer
}

// retarget decides a company member's new party target. ok=false means
// leave its Aggro as it is: its target is still legal, it is fighting
// someone outside this party, or no legal alternative exists. previous is
// the party member it was aimed at, if any.
//
// Phase 32d: the new target comes from the member's strategy (att.Rule).
// A kept target is sticky, except under a rule that follows something
// (assist, defend): that member turns when the rule's own choice, in
// reach, is someone else.
// byRule is true when a kept, reachable target was left because the rule
// now points elsewhere (not "can't reach").
//
// Phase 30c: refocus is a round in which the company's focus changed; the
// member then turns to the focus's choice even from a legal target.
func (s companySide) retarget(a *characters.Aggro, att enemyparty.Attacker, col int, placed bool, reach formationcombat.Reach, party mobparty.Party, members map[int]bool, alive map[company.MemberKey]bool, room *rooms.Room, refocus bool) (previous int, newId int, byRule bool, ok bool) {
	state, current := classifyPartyTarget(a, members, alive, room.RoomId)
	g := enemyparty.Group{Party: party}
	switch state {
	case targetElsewhere:
		return 0, 0, false, false
	case targetInParty:
		// A hidden target can't be fought ("You can't seem to find your
		// target"), so it is moved off like an unreachable one.
		if !mobHidden(current) && gateLetsThrough(col, placed, party.Formation, mobparty.MemberKeyFor(current), alive, reach, groundContext{room: room}) {
			if att.Rule.ReaimsEachRound() || refocus {
				if choice, ok := enemyparty.RuleChoice(g, att, current); ok && choice != current {
					return current, choice, true, true
				}
			}
			return 0, 0, false, false
		}
	}
	// With no one in reach, the member keeps whatever it has, so 11c's
	// gates skip it until something changes (the self-healing model).
	if !anyLegal(col, placed, party, alive, reach) {
		return 0, 0, false, false
	}
	newId, ok = enemyparty.Aim(g, att)
	if !ok || newId == current {
		return 0, 0, false, false
	}
	return current, newId, false, true
}

// anyLegal reports whether a company attacker can reach any living,
// visible member of party.
func anyLegal(col int, placed bool, party mobparty.Party, alive map[company.MemberKey]bool, reach formationcombat.Reach) bool {
	for _, id := range party.Members {
		if legalAgainstParty(col, placed, party, id, alive, reach) {
			return true
		}
	}
	return false
}

// leaderAim is the party member the leader is striking, for companions on
// assist (0 when none).
func (s companySide) leaderAim() int {
	if a := s.leader.Character.Aggro; plainAttack(a) && a.MobInstanceId > 0 && s.leader.Character.Health > 0 {
		return a.MobInstanceId
	}
	return 0
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
func gateLetsThrough(col int, placed bool, f company.Formation, target company.MemberKey, alive map[company.MemberKey]bool, reach formationcombat.Reach, context ...groundContext) bool {
	if !placed {
		return alive[target]
	}
	_, ok := resolveAttackTarget(col, f, target, alive, reach, context...)
	return ok
}

// legalAgainstParty reports whether a company attacker (in col, if placed)
// may strike target in party. An unplaced attacker fails open, as it does
// at the attack gates.
func legalAgainstParty(col int, placed bool, party mobparty.Party, target int, alive map[company.MemberKey]bool, reach formationcombat.Reach) bool {
	if mobHidden(target) {
		return false // can't be seen to be chosen
	}
	if !placed {
		return alive[mobparty.MemberKeyFor(target)]
	}
	m := mobs.GetInstance(target)
	if m == nil {
		return formationcombat.Legal(col, party.Formation, mobparty.MemberKeyFor(target), alive, reach)
	}
	return enemyparty.Legal(rooms.LoadRoom(m.Character.RoomId), 0, col, party.Formation, mobparty.MemberKeyFor(target), alive, reach)
}

func leaderTurnText(previous, newId int, alive map[company.MemberKey]bool) string {
	name := mobName(newId)
	if previous > 0 && alive[mobparty.MemberKeyFor(previous)] {
		return fmt.Sprintf(`You can't reach %s from here. %s`, util.Article(mobTag(mobName(previous))), turnsToward(`You`, mobTag(name)))
	}
	return turnsToward(`You`, mobTag(name))
}

func mobHidden(instanceId int) bool {
	mob := mobs.GetInstance(instanceId)
	return mob != nil && mob.Character.HasBuffFlag("hidden")
}

func mobName(instanceId int) string {
	if mob := mobs.GetInstance(instanceId); mob != nil {
		return battle.EnemyDisplayName(instanceId, mob.Character.Name)
	}
	return battle.EnemyDisplayName(instanceId, `someone`)
}

// keepPartyEngaged gives each living party member a legal company target
// when it has none, or its target is dead, gone, or out of reach. A party
// member fighting someone else (another player, a mob outside this
// company) is left alone.
//
// Phase 33i2: the group fights by its coordination tier (fixed when the
// battle began). Its leader (the highest level) aims first, and its aim
// is the group's focus: a share of the group's fighters turn onto it
// whenever they can reach it, the leader of a drilled company picks it by
// the casters rule while the company has a caster, and a veteran
// company's leader turns onto a member chanting a heal. Everyone else
// re-aims as before, with at least the tier's targeting noise.
func (s companySide) keepPartyEngaged(party mobparty.Party, room *rooms.Room) {
	candidates, keys := s.combatants()
	if len(candidates) == 0 {
		return
	}
	tier, _ := enemyparty.BattleTier(s.leader.UserId)
	spec := coordination.SpecOf(tier)
	leaderId := groupLeader(party)
	order := make([]int, 0, len(party.Members))
	if leaderId > 0 {
		order = append(order, leaderId)
	}
	for _, id := range party.Members {
		if id != leaderId {
			order = append(order, id)
		}
	}
	// While the leader is busy (chanting, winding up) the group keeps the
	// focus it last had, if that member still stands (33i2 review finding
	// 4); the leader's own aim replaces it when it has one.
	var followers map[int]bool
	focus := company.MemberKey("")
	if b, ok := battle.Current(s.leader.UserId); ok && b.EnemyFocus != "" && s.alive[company.MemberKey(b.EnemyFocus)] {
		focus = company.MemberKey(b.EnemyFocus)
		followers = focusFollowers(party, leaderId, tier)
	}
	for _, instanceId := range order {
		mob := mobs.GetInstance(instanceId)
		if mob == nil || !canFight(&mob.Character) || !retargetable(mob.Character.Aggro) {
			continue
		}
		if mob.Character.Aggro == nil && !joinsTheFight(mob, s.leader.UserId) {
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
		standing := keep && !s.memberHidden(current) && gateLetsThrough(attackerCol, true, s.formation, current, s.alive, reach, groundForMob(mob, s.leader.UserId))
		foes := s.memberFoes(candidates, keys, attackerCol, reach)
		if leapReady(mob) {
			cover := enemyparty.Standing(s.formation, s.alive, s.leader.UserId)
			for i := range foes {
				key := keys[foes[i].ID]
				r, c, ok := s.formation.Find(key)
				if ok && r > 0 && openLeapColumn(s.formation, cover, c) && formationcombat.InLateralRange(attackerCol, c) && !(enemyparty.Narrow(room) && (attackerCol == 2 || c == 2)) {
					foes[i].Reachable = true
				}
			}
		}

		if instanceId == leaderId {
			// The leader breaks a heal (veteran), else keeps a standing aim,
			// else re-aims, by casters first when drilled.
			if spec.BreakHeals {
				if key, ok := s.healChanter(attackerCol, reach); ok && key != current {
					s.aimPartyMember(mob, key, room)
					current, standing = key, true
				}
			}
			if !standing {
				rule, noise := enemyRule(mob, tier)
				if spec.CastersFirst && anyCaster(foes) {
					rule = strategy.Casters
				}
				if idx, ok := s.enemyPick(mob, rule, noise, foes, candidates, attackerCol, reach); ok && keys[idx] != current {
					s.aimPartyMember(mob, keys[idx], room)
					current, standing = keys[idx], true
				} else if ok {
					standing = true
				}
			}
			if standing {
				focus = current
				followers = focusFollowers(party, leaderId, tier)
				s.announceFocus(party, room, leaderId, focus, tier)
			}
			continue
		}

		// A follower turns onto the focus whenever it can reach it.
		if followers[instanceId] && focus != "" && focus != current && !s.memberHidden(focus) &&
			gateLetsThrough(attackerCol, true, s.formation, focus, s.alive, reach, groundForMob(mob, s.leader.UserId)) {
			s.aimPartyMember(mob, focus, room)
			continue
		}
		if standing {
			continue
		}
		rule, noise := enemyRule(mob, tier)
		idx, ok := s.enemyPick(mob, rule, noise, foes, candidates, attackerCol, reach)
		if !ok || keys[idx] == current {
			continue
		}
		s.aimPartyMember(mob, keys[idx], room)
	}
}

// aimRoll is the roll enemy personalities draw their noise from (one per
// re-aim). Tests replace it with UseAimRollForTest. Game loop only.
var aimRoll strategy.Roll = strategy.RandomRoll

// UseAimRollForTest replaces the enemy aim roll, returning the restore.
func UseAimRollForTest(r strategy.Roll) func() {
	previous := aimRoll
	aimRoll = r
	return func() { aimRoll = previous }
}

// memberFoes are the company's living members (combatants' candidates and
// keys) as an enemy in attackerCol with reach sees them, for a personality:
// their health, cells, whether it can strike them, the leader (the
// player), and who is chanting or heals and casts (the casters rule).
func (s companySide) memberFoes(candidates []engagement.Combatant, keys []company.MemberKey, attackerCol int, reach formationcombat.Reach) []strategy.Foe {
	out := make([]strategy.Foe, 0, len(candidates))
	for _, c := range candidates {
		key := keys[c.ID]
		var char *characters.Character
		if key == company.LeaderMemberKey {
			char = s.leader.Character
		} else {
			for _, id := range s.companionIds {
				if s.companions[id] == key {
					if m := mobs.GetInstance(id); m != nil {
						char = &m.Character
					}
					break
				}
			}
		}
		if char == nil {
			continue
		}
		role := enemyparty.MemberStrategy(s.leader.UserId, key).Role
		out = append(out, strategy.Foe{
			ID:         c.ID,
			HP:         char.Health,
			MaxHP:      char.HealthMax.Value,
			Row:        c.Row,
			Col:        c.Col,
			Reachable:  s.legalAgainstCompany(attackerCol, key, reach),
			Leader:     key == company.LeaderMemberKey,
			StrikesPct: -1,
			Chanting:   char.Aggro != nil && char.Aggro.Type == characters.SpellCast,
			Caster:     role == strategy.Healer || role == strategy.Caster,
			Healer:     role == strategy.Healer,
		})
	}
	return out
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
	if key == "" || s.memberHidden(key) {
		return false
	}
	if _, _, placed := s.formation.Find(key); !placed {
		return s.alive[key]
	}
	return enemyparty.Legal(rooms.LoadRoom(s.leader.Character.RoomId), s.leader.UserId, attackerCol, s.formation, key, s.alive, reach)
}

// memberHidden reports whether a company member is hidden (sneaking), which
// mobs looking for trouble ignore.
func (s companySide) memberHidden(key company.MemberKey) bool {
	if key == company.LeaderMemberKey {
		return s.leader.Character.HasBuffFlag("hidden")
	}
	for _, instanceId := range s.companionIds {
		if s.companions[instanceId] == key {
			return mobHidden(instanceId)
		}
	}
	return false
}

func (s companySide) aimPartyMember(mob *mobs.Mob, key company.MemberKey, room *rooms.Room) {
	previous := s.aimRef(mob.Character.Aggro)
	targetName := fmt.Sprintf(`<ansi fg="username">%s</ansi>`, s.leader.Character.Name)
	next := userRef(s.leader)
	if key == company.LeaderMemberKey {
		mob.Character.SetAggro(s.leader.UserId, 0, attackType(mob.Character.Aggro))
		mob.PlayerAttacked(s.leader.UserId) // as the attack command records it
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
		targetName = mobTag(mobName(target.InstanceId))
		next = mobRef(target)
		mob.Character.SetAggro(0, instanceId, attackType(mob.Character.Aggro))
	}
	emitTargetChange(mobRef(mob), previous, next, room.RoomId)
	mob.PreventIdle = true
	events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
	room.SendText(turnsToward(mobTag(mobName(mob.InstanceId)), targetName))
}

// aimRef names what an enemy's Aggro was aimed at, for a target change.
func (s companySide) aimRef(a *characters.Aggro) combatstream.Ref {
	switch {
	case a == nil:
		return combatstream.Ref{}
	case a.UserId > 0:
		return userRef(users.GetByUserId(a.UserId))
	}
	return mobRefById(a.MobInstanceId)
}

// reassignEnemyTarget is the enemy side of an earned turn surviving an
// earlier kill (30g6a): a group member fighting a company whose company
// target has just fallen re-aims by its own rule and tier, as upkeep would
// next round, without the group's focus. It returns false when the mob is
// in no company battle in the room or no member can be reached.
func reassignEnemyTarget(mob *mobs.Mob, room *rooms.Room) bool {
	if mob == nil || room == nil || mob.Character.Aggro == nil || mob.Character.Health <= 0 {
		return false
	}
	if _, _, isCompanion := company.LeaderAndKeyForInstance(mob.InstanceId); isCompanion {
		return false
	}
	for _, userId := range room.GetPlayers() {
		leader := users.GetByUserId(userId)
		if leader == nil || leader.Character == nil {
			continue
		}
		side, ok := loadCompanySide(leader, room)
		if !ok {
			continue
		}
		for _, party := range enemyparty.Parties(room) {
			if partyMemberSet(party)[mob.InstanceId] && inBattleWith(leader.UserId, party) {
				return side.reaimEnemy(party, mob, room)
			}
		}
	}
	return false
}

// reaimEnemy aims one enemy group member at a living, reachable company
// member by its rule and tier noise.
func (s companySide) reaimEnemy(party mobparty.Party, mob *mobs.Mob, room *rooms.Room) bool {
	candidates, keys := s.combatants()
	if len(candidates) == 0 {
		return false
	}
	_, attackerCol, found := party.Formation.Find(mobparty.MemberKeyFor(mob.InstanceId))
	if !found {
		return false
	}
	reach := combat.ResolveReach(&mob.Character, mob.Reach)
	foes := s.memberFoes(candidates, keys, attackerCol, reach)
	tier, _ := enemyparty.BattleTier(s.leader.UserId)
	rule, noise := enemyRule(mob, tier)
	idx, ok := s.enemyPick(mob, rule, noise, foes, candidates, attackerCol, reach)
	if !ok || !s.alive[keys[idx]] {
		return false
	}
	s.aimPartyMember(mob, keys[idx], room)
	return true
}
