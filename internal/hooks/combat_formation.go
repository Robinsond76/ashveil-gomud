package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
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
