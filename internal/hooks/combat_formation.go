package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/company"
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
// resolveAttackTarget for a player attacking a mob. It fails open (returns
// defMob, true — unchanged pre-existing behavior) whenever the player has
// no company formation to apply, or the target mob can't be resolved into
// an assembled enemy party at all: the feature is "company vs. party"
// tactics, and a player/target with no formation concept combats exactly
// as it did before this change.
func gateFormationAttack(user *users.UserRecord, defMob *mobs.Mob, room *rooms.Room) (*mobs.Mob, bool) {
	attackerCol, ok := resolvePlayerColumn(user.UserId)
	if !ok {
		return defMob, true
	}

	party, ok := resolveEnemyParty(room, defMob.InstanceId)
	if !ok {
		return defMob, true
	}

	alive := aliveMapForParty(party)
	reach := combat.ResolveReach(user.Character, false)
	targetKey := mobparty.MemberKeyFor(defMob.InstanceId)

	finalKey, ok := resolveAttackTarget(attackerCol, party.Formation, targetKey, alive, reach)
	if !ok {
		return nil, false
	}
	if finalKey == targetKey {
		return defMob, true
	}

	finalInstanceId, ok := mobparty.InstanceIdFromMemberKey(finalKey)
	if !ok {
		return defMob, true
	}
	finalMob := mobs.GetInstance(finalInstanceId)
	if finalMob == nil {
		return defMob, true
	}
	return finalMob, true
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
