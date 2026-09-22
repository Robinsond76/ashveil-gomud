package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/company"
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
func resolveAttackTarget(attackerCol int, f company.Formation, originalTarget company.MemberKey, alive map[company.MemberKey]bool, reach formationcombat.Reach) (finalTarget company.MemberKey, ok bool) {
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

	party, ok := resolveEnemyParty(room, defenderInstanceId)
	if !ok {
		return original, true
	}

	alive := aliveMapForParty(party)
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

// resolveEnemyParty finds the assembled mobparty.Party (fresh, never
// cached — matching 11a's own design) that currently contains
// targetInstanceId among the room's hostile (non-charmed) mobs.
func resolveEnemyParty(room *rooms.Room, targetInstanceId int) (mobparty.Party, bool) {
	summaries := hostileMobSummaries(room)
	for _, p := range mobparty.Assemble(summaries) {
		for _, id := range p.Members {
			if id == targetInstanceId {
				return p, true
			}
		}
	}
	return mobparty.Party{}, false
}

func hostileMobSummaries(room *rooms.Room) []mobparty.MobSummary {
	var summaries []mobparty.MobSummary
	for _, instanceId := range room.GetMobs() {
		mob := mobs.GetInstance(instanceId)
		if mob == nil || mob.Character.IsCharmed() {
			continue
		}
		summaries = append(summaries, mobparty.MobSummary{
			InstanceId: instanceId,
			Groups:     mob.Groups,
			EHP:        effectiveHP(mob.Character.HealthMax.Value, mob.Character.GetDefense()),
		})
	}
	return summaries
}

// aliveMapForParty reports, for every member of an assembled party,
// whether its live mob instance still exists and has positive HP.
func aliveMapForParty(p mobparty.Party) map[company.MemberKey]bool {
	alive := make(map[company.MemberKey]bool, len(p.Members))
	for _, id := range p.Members {
		mob := mobs.GetInstance(id)
		alive[mobparty.MemberKeyFor(id)] = mob != nil && mob.Character.Health > 0
	}
	return alive
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
// its assembled enemy party (mirrors resolveEnemyParty, applied to the
// attacker instead of a target).
func resolveHostileAttackerColumn(room *rooms.Room, attackerInstanceId int) (int, bool) {
	party, ok := resolveEnemyParty(room, attackerInstanceId)
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

// effectiveHP mirrors combat.RankMobs' own EHP formula
// (internal/combat/mob_rank.go:164-171), applied directly to a live
// combatant's current HealthMax/Defense instead of a simulated spec —
// cheap enough to call once per hostile mob per round, unlike RankMobs
// itself which ranks every mob spec in the game.
func effectiveHP(hp, defense int) float64 {
	defFrac := float64(defense) / 200.0
	if defFrac > 0.95 {
		defFrac = 0.95
	}
	if defFrac < 0 {
		defFrac = 0
	}
	return float64(hp) / (1.0 - defFrac)
}

// firstHostilePartyInRoom returns the first assembled hostile party
// currently in room, if any. mobparty.Assemble is never cached (11a's own
// design), so this is always a fresh snapshot.
func firstHostilePartyInRoom(room *rooms.Room) (mobparty.Party, bool) {
	parties := mobparty.Assemble(hostileMobSummaries(room))
	if len(parties) == 0 {
		return mobparty.Party{}, false
	}
	return parties[0], true
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

// reassignEnemyTarget picks a new legal target (11b's weakest-HP
// preference) for an attacker in attackerCol, from whichever hostile
// party is currently in room. ok=false means no hostile party is present,
// or none of its members are both alive and legal — the caller must fall
// back to its existing "target lost, give up" behavior unchanged.
func reassignEnemyTarget(attackerCol int, reach formationcombat.Reach, room *rooms.Room) (int, bool) {
	party, ok := firstHostilePartyInRoom(room)
	if !ok {
		return 0, false
	}

	alive := aliveMapForParty(party)
	candidates := partyCombatants(party, alive)

	legal := func(attacker, defender engagement.Combatant) bool {
		return formationcombat.Legal(attackerCol, party.Formation, mobparty.MemberKeyFor(defender.ID), alive, reach)
	}

	attacker := engagement.Combatant{Col: attackerCol}
	return engagement.AssignTarget(attacker, candidates, engagement.Weakest, legal)
}

// reassignPlayerTarget attempts 11b's reassignment-on-target-loss for a
// player whose current mob target just became invalid. On success it sets
// a new Aggro target and returns true — the caller skips its own "target
// lost" message/clear, and combat resumes normally next round against the
// new target. false means unchanged pre-existing behavior: no company
// formation, or no living legal replacement in any hostile party
// currently in the room.
func reassignPlayerTarget(user *users.UserRecord, room *rooms.Room) bool {
	col, ok := resolvePlayerColumn(user.UserId)
	if !ok {
		return false
	}
	reach := combat.ResolveReach(user.Character, false)
	newTargetId, ok := reassignEnemyTarget(col, reach, room)
	if !ok {
		return false
	}
	user.Character.SetAggro(0, newTargetId, characters.DefaultAttack)
	events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})
	return true
}

// reassignCompanionTarget is reassignPlayerTarget's companion-mob
// counterpart. It only applies when mob is a currently-attached company
// member (hostile mobs whose own target died are not reassigned — that's
// enemy AI, out of scope; see this plan's Design Decision 1).
func reassignCompanionTarget(mob *mobs.Mob, room *rooms.Room) bool {
	leaderUserID, key, isCompanion := company.LeaderAndKeyForInstance(mob.InstanceId)
	if !isCompanion {
		return false
	}
	f, ok := company.FormationFor(leaderUserID)
	if !ok {
		return false
	}
	_, col, found := f.Find(key)
	if !found {
		return false
	}
	reach := combat.ResolveReach(&mob.Character, mob.Reach)
	newTargetId, ok := reassignEnemyTarget(col, reach, room)
	if !ok {
		return false
	}
	mob.Character.SetAggro(0, newTargetId, characters.DefaultAttack)
	events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
	return true
}
