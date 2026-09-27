package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/engagement"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// resolveAttackTarget decides, given an enemy party's Formation and who is
// currently alive, which member an attack aimed at originalTarget should
// actually land on this round: front-row interception (11c) is applied
// first, then the result is checked for column-occupancy/lateral-range
// legality (11c). ok=false means the attack should be skipped entirely
// this round — the caller must leave Aggro untouched so the same target
// becomes legal again automatically once whatever blocks it dies (11c's
// self-healing model; see internal/formationcombat's own package doc).
// A target not placed in f fails open (ok=true, no redirect).
func resolveAttackTarget(attackerCol int, f company.Formation, originalTarget company.MemberKey, alive map[company.MemberKey]bool, reach formationcombat.Reach) (finalTarget company.MemberKey, ok bool) {
	if _, _, placed := f.Find(originalTarget); !placed {
		// A member not placed in the formation has no position to shield
		// or block it: the attack proceeds directly (Phase 29a; before, an
		// unplaced company member could never be struck).
		return originalTarget, true
	}
	target := originalTarget
	if interceptor, intercepted := formationcombat.InterceptFrontRow(f, originalTarget, alive); intercepted {
		target = interceptor
	}
	if !formationcombat.Legal(attackerCol, f, target, alive, reach) {
		return "", false
	}
	return target, true
}

// gateFormationAttack is the engine-facing adapter around
// resolveAttackTarget for a player attacking a mob. See resolveEnemyAttack
// for the fail-open contract shared with gateCompanionAttacksEnemy.
func gateFormationAttack(user *users.UserRecord, defMob *mobs.Mob, room *rooms.Room) (*mobs.Mob, bool) {
	attackerCol, ok := resolvePlayerColumn(user.UserId)
	if !ok {
		return defMob, true
	}
	reach := combat.ResolveReach(user.Character, false)
	return resolveEnemyAttack(attackerCol, defMob.InstanceId, room, reach)
}

// resolveEnemyAttack applies 11c legality/interception when a combatant in
// attackerCol attacks defenderInstanceId's assembled enemy party within
// room. It fails open (returns the original defender, true) whenever the
// defender can't be resolved into an assembled enemy party at all: the
// feature is "company vs. party" tactics, and a target with no formation
// concept combats exactly as it did before this change.
func resolveEnemyAttack(attackerCol int, defenderInstanceId int, room *rooms.Room, reach formationcombat.Reach) (*mobs.Mob, bool) {
	original := mobs.GetInstance(defenderInstanceId)

	party, ok := enemyparty.PartyOf(room, defenderInstanceId)
	if !ok {
		return original, true
	}

	alive := enemyparty.Alive(party)
	targetKey := mobparty.MemberKeyFor(defenderInstanceId)

	finalKey, ok := resolveAttackTarget(attackerCol, party.Formation, targetKey, alive, reach)
	if !ok {
		return nil, false
	}
	if finalKey == targetKey {
		return original, true
	}

	finalInstanceId, ok := mobparty.InstanceIdFromMemberKey(finalKey)
	if !ok {
		return original, true
	}
	finalMob := mobs.GetInstance(finalInstanceId)
	if finalMob == nil {
		return original, true
	}
	return finalMob, true
}

// gateMobVsMobAttack classifies both sides of a mob-vs-mob attack as a
// company member or not, and dispatches to the matching gate. Two hostile
// mobs, or two companions, fighting each other is left untouched — no
// formation concept applies to either. handled=true (only ever produced by
// gateEnemyAttacksCompanion, when interception redirects to the leader)
// means the caller already resolved the attack itself via
// resolveInterceptedAttackOnLeader (AttackMobVsPlayer, not AttackMobVsMob)
// and the caller should skip its own attack call and continue the round.
func gateMobVsMobAttack(mob, defMob *mobs.Mob, mobRoom *rooms.Room) (target *mobs.Mob, handled bool, ok bool) {
	attackerLeaderId, attackerKey, attackerIsCompanion := company.LeaderAndKeyForInstance(mob.InstanceId)
	defenderLeaderId, defenderKey, defenderIsCompanion := company.LeaderAndKeyForInstance(defMob.InstanceId)

	switch {
	case attackerIsCompanion && !defenderIsCompanion:
		target, ok := gateCompanionAttacksEnemy(mob, attackerLeaderId, attackerKey, defMob, mobRoom)
		return target, false, ok
	case !attackerIsCompanion && defenderIsCompanion:
		return gateEnemyAttacksCompanion(mob, defMob, mobRoom, defenderLeaderId, defenderKey)
	default:
		return defMob, false, true
	}
}

// gateCompanionAttacksEnemy gates "my companion attacks the enemy party" —
// the mob-vs-mob analog of gateFormationAttack, sharing its core via
// resolveEnemyAttack.
func gateCompanionAttacksEnemy(mob *mobs.Mob, leaderUserID int, attackerKey company.MemberKey, defMob *mobs.Mob, mobRoom *rooms.Room) (*mobs.Mob, bool) {
	f, ok := company.FormationFor(leaderUserID)
	if !ok {
		return defMob, true
	}
	_, col, found := f.Find(attackerKey)
	if !found {
		return defMob, true
	}
	reach := combat.ResolveReach(&mob.Character, mob.Reach)
	return resolveEnemyAttack(col, defMob.InstanceId, mobRoom, reach)
}

// gateEnemyAttacksCompanion gates "the enemy attacks my companion" — the
// mob-vs-mob analog of gateMobVsPlayerAttack's legality/interception half,
// generalized to any company member. When interception redirects to the
// leader, this crosses combat.Attack* functions (AttackMobVsMob to
// AttackMobVsPlayer) the same way gateMobVsPlayerAttack's redirect does in
// the opposite direction: it resolves the attack itself via
// resolveInterceptedAttackOnLeader and reports handled=true.
func gateEnemyAttacksCompanion(mob, defMob *mobs.Mob, mobRoom *rooms.Room, leaderUserID int, defenderKey company.MemberKey) (target *mobs.Mob, handled bool, ok bool) {
	f, formationOk := company.FormationFor(leaderUserID)
	if !formationOk {
		return defMob, false, true
	}
	attackerCol, colOk := resolveHostileAttackerColumn(mobRoom, mob.InstanceId)
	if !colOk {
		return defMob, false, true
	}
	leader := users.GetByUserId(leaderUserID)
	if leader == nil {
		return defMob, false, true
	}
	alive := aliveMapForCompany(leader, f)
	reach := combat.ResolveReach(&mob.Character, mob.Reach)

	finalKey, legalOk := resolveAttackTarget(attackerCol, f, defenderKey, alive, reach)
	if !legalOk {
		return nil, false, false
	}
	if finalKey == defenderKey {
		return defMob, false, true
	}
	if finalKey == company.LeaderMemberKey {
		defRoom := rooms.LoadRoom(leader.Character.RoomId)
		resolveInterceptedAttackOnLeader(mob, leader, mobRoom, defRoom)
		return nil, true, true
	}

	companionID, companionOk := company.CompanionIDFromMemberKey(finalKey)
	if !companionOk {
		return defMob, false, true
	}
	instanceId, instanceOk := company.InstanceFor(leaderUserID, companionID)
	if !instanceOk {
		return defMob, false, true
	}
	finalMob := mobs.GetInstance(instanceId)
	if finalMob == nil {
		return defMob, false, true
	}
	return finalMob, false, true
}

// resolvePlayerColumn returns the column of leaderUserID's own
// LeaderMemberKey within their company formation. ok is false when they
// have no company record, or (defensively) aren't placed in it.
func resolvePlayerColumn(leaderUserID int) (int, bool) {
	f, ok := company.FormationFor(leaderUserID)
	if !ok {
		return 0, false
	}
	_, col, found := f.Find(company.LeaderMemberKey)
	if !found {
		return 0, false
	}
	return col, true
}

// gateMobVsPlayerAttack decides whether mob's attack on defUser should be
// redirected to an intercepting companion (11c front-row interception) or
// skipped this round (11c legality), before the caller resolves the
// attack. Unlike gateFormationAttack (player-vs-mob), a redirect here
// means an entirely different combat.Attack* function — AttackMobVsMob
// against the interceptor, not AttackMobVsPlayer against defUser — so this
// function resolves the intercepted attack itself. handled=true tells the
// caller to skip its own attack resolution and continue the round.
// handled=false, ok=true means proceed exactly as before (attack defUser
// directly — no company, or no interception applies). ok=false means skip
// the round entirely; mob.Character.Aggro is left untouched in every case,
// so the engagement resumes automatically once whatever blocks it changes.
func gateMobVsPlayerAttack(mob *mobs.Mob, defUser *users.UserRecord, mobRoom, defRoom *rooms.Room) (handled bool, ok bool) {
	f, formationOk := company.FormationFor(defUser.UserId)
	if !formationOk {
		return false, true
	}

	attackerCol, attackerOk := resolveHostileAttackerColumn(mobRoom, mob.InstanceId)
	if !attackerOk {
		return false, true
	}

	alive := aliveMapForCompany(defUser, f)
	reach := combat.ResolveReach(&mob.Character, mob.Reach)

	finalKey, legalOk := resolveAttackTarget(attackerCol, f, company.LeaderMemberKey, alive, reach)
	if !legalOk {
		return false, false
	}
	if finalKey == company.LeaderMemberKey {
		return false, true
	}

	companionID, companionOk := company.CompanionIDFromMemberKey(finalKey)
	if !companionOk {
		return false, true
	}
	instanceId, instanceOk := company.InstanceFor(defUser.UserId, companionID)
	if !instanceOk {
		return false, true
	}
	interceptor := mobs.GetInstance(instanceId)
	if interceptor == nil {
		return false, true
	}

	resolveInterceptedMobAttack(mob, interceptor, mobRoom, defRoom, defUser.UserId)
	return true, true
}

// resolveHostileAttackerColumn returns a hostile mob's own column within
// its assembled enemy party (mirrors enemyparty.PartyOf, applied to the
// attacker instead of a target).
func resolveHostileAttackerColumn(room *rooms.Room, attackerInstanceId int) (int, bool) {
	party, ok := enemyparty.PartyOf(room, attackerInstanceId)
	if !ok {
		return 0, false
	}
	_, col, found := party.Formation.Find(mobparty.MemberKeyFor(attackerInstanceId))
	if !found {
		return 0, false
	}
	return col, true
}

// aliveMapForCompany reports, for the leader and every formation-placed
// companion, whether they're currently alive (the leader) or currently
// spawned, attached, and alive (a companion).
func aliveMapForCompany(leader *users.UserRecord, f company.Formation) map[company.MemberKey]bool {
	alive := map[company.MemberKey]bool{
		company.LeaderMemberKey: leader.Character.Health > 0,
	}
	for row := 0; row < company.FormationRows; row++ {
		for col := 0; col < company.FormationCols; col++ {
			key := f.At(row, col)
			if key == "" || key == company.LeaderMemberKey {
				continue
			}
			companionID, ok := company.CompanionIDFromMemberKey(key)
			if !ok {
				continue
			}
			instanceId, ok := company.InstanceFor(leader.UserId, companionID)
			if !ok {
				alive[key] = false
				continue
			}
			mob := mobs.GetInstance(instanceId)
			alive[key] = mob != nil && mob.Character.Health > 0
		}
	}
	return alive
}

// resolveInterceptedMobAttack resolves one round of an attack 11c's
// front-row interception redirected from the leader to interceptor. It
// deliberately never touches mob.Character.Aggro: interception is
// recomputed fresh every round, not a persistent retarget, so the same
// engagement resumes automatically against the leader once no living
// front-row companion remains to intercept. It mirrors
// NewRound_DoCombat.go's existing mob-vs-mob attack resolution
// (AttackMobVsMob, room-broadcast messages, onHurt scripting, offhand
// equipment-break) and its existing "leader is attacked" idle-companion
// retaliation loop, so an intercepted round behaves identically to a
// direct hit in every way except who takes the damage.
func resolveInterceptedMobAttack(mob, interceptor *mobs.Mob, mobRoom, defRoom *rooms.Room, defenderUserId int) {
	roundResult := combat.AttackMobVsMob(mob, interceptor)
	emitAttack(mobRef(mob), mobRef(interceptor), mob.Character.RoomId, &mob.Character, roundResult)
	roundExtraMobs = append(roundExtraMobs, interceptor.InstanceId)

	for _, instanceId := range mobRoom.GetMobs(rooms.FindCharmed) {
		if charmedMob := mobs.GetInstance(instanceId); charmedMob != nil {
			if charmedMob.Character.IsCharmed(defenderUserId) && charmedMob.Character.Aggro == nil {
				charmedMob.Character.Aggro = &characters.Aggro{Type: characters.DefaultAttack}
				charmedMob.Command(fmt.Sprintf("attack #%d", mob.InstanceId))
			}
		}
	}

	for _, buffId := range roundResult.BuffSource {
		mob.AddBuff(buffId, `combat`)
	}
	for _, buffId := range roundResult.BuffTarget {
		interceptor.AddBuff(buffId, `combat`)
	}
	for _, msg := range roundResult.MessagesToSourceRoom {
		mobRoom.SendText(msg)
	}
	for _, msg := range roundResult.MessagesToTargetRoom {
		defRoom.SendText(msg)
	}

	if !roundResult.Hit {
		return
	}

	scripting.TryMobScriptEvent(`onHurt`, interceptor.InstanceId, mob.InstanceId, `mob`, map[string]any{`damage`: roundResult.DamageToTarget, `crit`: roundResult.Crit})

	if interceptor.Character.Equipment.Offhand.ItemId == 0 {
		return
	}

	modifier := 0
	if roundResult.Crit {
		modifier = int(interceptor.Character.Equipment.Offhand.GetSpec().BreakChance)
	}
	if !interceptor.Character.Equipment.Offhand.BreakTest(modifier) {
		return
	}

	defRoom.SendText(fmt.Sprintf(`<ansi fg="214"><ansi fg="202">***</ansi> The <ansi fg="item">%s</ansi> <ansi fg="mobname">%s</ansi> was carrying breaks! <ansi fg="202">***</ansi></ansi>`, interceptor.Character.Equipment.Offhand.NameSimple(), interceptor.Character.Name))
	events.AddToQueue(events.ItemOwnership{MobInstanceId: interceptor.InstanceId, Item: interceptor.Character.Equipment.Offhand, Gained: false})
	interceptor.Character.RemoveFromBody(interceptor.Character.Equipment.Offhand)
	itm := items.New(20)
	if !interceptor.Character.StoreItem(itm) {
		defRoom.AddItem(itm, false)
		events.AddToQueue(events.ItemOwnership{MobInstanceId: interceptor.InstanceId, Item: itm, Gained: true})
	}
}

// resolveInterceptedAttackOnLeader resolves one round of an attack 11c's
// front-row interception redirected from a companion to their leader — the
// leader-as-interceptor case, symmetric with resolveInterceptedMobAttack
// but crossing combat.Attack* functions in the opposite direction
// (AttackMobVsMob to AttackMobVsPlayer). It deliberately never touches
// mob.Character.Aggro (left pointed at the companion's instance id):
// interception is recomputed fresh every round, so the same engagement
// resumes automatically against the companion once the leader no longer
// blocks. Mirrors NewRound_DoCombat.go's existing mob-vs-player attack
// resolution (AttackMobVsPlayer, the charmed-mob-assist loop, buffs and
// messages, offhand equipment-break) plus a CharacterVitalsChanged event so
// the leader's own client health bar updates — resolveInterceptedMobAttack
// has no equivalent event because a mob defender's health isn't pushed to
// any client the same way a player's is.
func resolveInterceptedAttackOnLeader(mob *mobs.Mob, leader *users.UserRecord, mobRoom, defRoom *rooms.Room) {
	roundResult := combat.AttackMobVsPlayer(mob, leader)
	emitAttack(mobRef(mob), userRef(leader), mob.Character.RoomId, &mob.Character, roundResult)
	roundExtraPlayers = append(roundExtraPlayers, leader.UserId)

	for _, instanceId := range mobRoom.GetMobs(rooms.FindCharmed) {
		if charmedMob := mobs.GetInstance(instanceId); charmedMob != nil {
			if charmedMob.Character.IsCharmed(leader.UserId) && charmedMob.Character.Aggro == nil {
				charmedMob.Character.Aggro = &characters.Aggro{Type: characters.DefaultAttack}
				charmedMob.Command(fmt.Sprintf("attack #%d", mob.InstanceId))
			}
		}
	}

	for _, buffId := range roundResult.BuffSource {
		mob.AddBuff(buffId, `combat`)
	}
	for _, buffId := range roundResult.BuffTarget {
		leader.AddBuff(buffId, `combat`)
	}
	for _, msg := range roundResult.MessagesToTarget {
		leader.SendText(msg)
	}
	for _, msg := range roundResult.MessagesToSourceRoom {
		mobRoom.SendText(msg, leader.UserId)
	}
	for _, msg := range roundResult.MessagesToTargetRoom {
		defRoom.SendText(msg, leader.UserId)
	}

	if roundResult.DamageToTarget != 0 {
		events.AddToQueue(events.CharacterVitalsChanged{UserId: leader.UserId})
	}

	if !roundResult.Hit || leader.Character.Equipment.Offhand.ItemId == 0 {
		return
	}

	modifier := 0
	if roundResult.Crit {
		modifier = int(leader.Character.Equipment.Offhand.GetSpec().BreakChance)
	}
	if !leader.Character.Equipment.Offhand.BreakTest(modifier) {
		return
	}

	leader.SendText(`<ansi fg="202">***</ansi>`)
	leader.SendText(fmt.Sprintf(`<ansi fg="214"><ansi fg="202">***</ansi> Your <ansi fg="item">%s</ansi> breaks! <ansi fg="202">***</ansi></ansi>`, leader.Character.Equipment.Offhand.NameSimple()))
	leader.SendText(`<ansi fg="202">***</ansi>`)
	defRoom.SendText(fmt.Sprintf(`<ansi fg="214"><ansi fg="202">***</ansi> The <ansi fg="item">%s</ansi> <ansi fg="username">%s</ansi> was carrying breaks! <ansi fg="202">***</ansi></ansi>`, leader.Character.Equipment.Offhand.NameSimple(), leader.Character.Name), leader.UserId)

	events.AddToQueue(events.ItemOwnership{UserId: leader.UserId, Item: leader.Character.Equipment.Offhand, Gained: false})
	leader.Character.RemoveFromBody(leader.Character.Equipment.Offhand)
	itm := items.New(20)
	if !leader.Character.StoreItem(itm) {
		defRoom.AddItem(itm, false)
		events.AddToQueue(events.ItemOwnership{UserId: leader.UserId, Item: itm, Gained: true})
	}
}

// partyCombatants adapts an assembled party's members into
// engagement.Combatant values (live HP, formation row/col) for
// engagement.AssignTarget. A member with no live mob instance reports
// HP 0, which AssignTarget already treats as ineligible.
func partyCombatants(party mobparty.Party, alive map[company.MemberKey]bool) []engagement.Combatant {
	combatants := make([]engagement.Combatant, 0, len(party.Members))
	for _, id := range party.Members {
		hp := 0
		if mob := mobs.GetInstance(id); mob != nil {
			hp = mob.Character.Health
		}
		row, col, _ := party.Formation.Find(mobparty.MemberKeyFor(id))
		combatants = append(combatants, engagement.Combatant{ID: id, HP: hp, Row: row, Col: col})
	}
	return combatants
}

// reassignWithinLostParty picks a new target (11b's weakest-HP preference)
// for a company attacker whose target lostId was just found dead or gone
// on its own turn. It chooses from lostId's own party while that party can
// still be assembled in room (a dead member stays in the room until its
// death is processed); failing that (the lost member is gone, or nothing in
// its party is in reach), from another party in room already hostile to
// leaderId. It never turns on a bystander party. An
// attacker not placed in the formation fails open, as at the gates.
// ok=false leaves the caller's "target lost" behavior unchanged; the
// engagement upkeep at the start of the next round (combat_engagement.go)
// catches anything this misses.
func reassignWithinLostParty(leaderId, lostId int, col int, placed bool, reach formationcombat.Reach, room *rooms.Room) (int, bool) {
	if room == nil || lostId <= 0 {
		return 0, false
	}
	lostParty, found := enemyparty.PartyOf(room, lostId)
	if found {
		if id, ok := chooseFromParty(col, placed, lostParty, enemyparty.Alive(lostParty), reach); ok {
			return id, true
		}
	}
	for _, party := range enemyparty.Parties(room) {
		if (found && party.ID == lostParty.ID) || !hostileTo(party, leaderId) {
			continue
		}
		if id, ok := chooseFromParty(col, placed, party, enemyparty.Alive(party), reach); ok {
			return id, true
		}
	}
	return 0, false
}

// hostileTo reports whether any living member of party attacks leaderId on
// sight: a hostile mob, or one whose group the leader has made hostile.
func hostileTo(party mobparty.Party, leaderId int) bool {
	for _, instanceId := range party.Members {
		mob := mobs.GetInstance(instanceId)
		if mob == nil || mob.Character.Health < 1 {
			continue
		}
		if mob.Hostile {
			return true
		}
		for _, group := range mob.Groups {
			if mobs.IsHostile(group, leaderId) {
				return true
			}
		}
	}
	return false
}

// reassignPlayerTarget attempts 11b's reassignment-on-target-loss for a
// player whose current mob target just became invalid. On success it sets
// a new Aggro target and returns true — the caller skips its own "target
// lost" message/clear, and combat resumes normally next round against the
// new target. false means unchanged pre-existing behavior: no company, or
// no living legal replacement in the lost target's party.
func reassignPlayerTarget(user *users.UserRecord, room *rooms.Room) bool {
	f, ok := company.FormationFor(user.UserId)
	if !ok || user.Character.Aggro == nil {
		return false
	}
	_, col, placed := f.Find(company.LeaderMemberKey)
	reach := combat.ResolveReach(user.Character, false)
	lostId := user.Character.Aggro.MobInstanceId
	newTargetId, ok := reassignWithinLostParty(user.UserId, lostId, col, placed, reach, room)
	if !ok {
		return false
	}
	emitTargetChange(userRef(user), mobRefById(lostId), mobRefById(newTargetId), room.RoomId)
	user.Character.SetAggro(0, newTargetId, attackType(user.Character.Aggro))
	events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})
	user.SendText(fmt.Sprintf(`You turn on <ansi fg="mobname">%s</ansi>.`, mobName(newTargetId)))
	return true
}

// reassignCompanionTarget is reassignPlayerTarget's companion-mob
// counterpart. It only applies when mob is a currently-attached company
// member; hostile mobs are kept engaged by the round-start upkeep instead.
func reassignCompanionTarget(mob *mobs.Mob, room *rooms.Room) bool {
	leaderUserID, key, isCompanion := company.LeaderAndKeyForInstance(mob.InstanceId)
	if !isCompanion || mob.Character.Aggro == nil {
		return false
	}
	f, ok := company.FormationFor(leaderUserID)
	if !ok {
		return false
	}
	_, col, placed := f.Find(key)
	reach := combat.ResolveReach(&mob.Character, mob.Reach)
	lostId := mob.Character.Aggro.MobInstanceId
	newTargetId, ok := reassignWithinLostParty(leaderUserID, lostId, col, placed, reach, room)
	if !ok {
		return false
	}
	emitTargetChange(mobRef(mob), mobRefById(lostId), mobRefById(newTargetId), room.RoomId)
	mob.Character.SetAggro(0, newTargetId, attackType(mob.Character.Aggro))
	events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
	room.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> turns on <ansi fg="mobname">%s</ansi>.`, mob.Character.Name, mobName(newTargetId)))
	return true
}
