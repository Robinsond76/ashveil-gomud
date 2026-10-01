// Package enemyparty adapts a live room's hostile mobs into Phase 11a's
// mobparty.Party values, and reports which party members are alive. It is
// the engine-facing counterpart of the GoMud-free mobparty package: the
// combat round (internal/hooks) and module commands (modules/company's
// `formation reach`) both read enemy formations through it.
//
// Parties are assembled fresh on every call and never cached or persisted,
// matching mobparty's own design.
package enemyparty

import (
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Parties assembles room's non-charmed mobs into parties. Charmed mobs
// (anyone's companions) are never part of an enemy party.
func Parties(room *rooms.Room) []mobparty.Party {
	if room == nil {
		return nil
	}
	return mobparty.Assemble(summaries(room))
}

// PartyOf returns the party within room that contains instanceId.
func PartyOf(room *rooms.Room, instanceId int) (mobparty.Party, bool) {
	for _, p := range Parties(room) {
		for _, id := range p.Members {
			if id == instanceId {
				return p, true
			}
		}
	}
	return mobparty.Party{}, false
}

// Alive reports, for every member of p, whether its live mob instance still
// exists and has positive HP.
func Alive(p mobparty.Party) map[company.MemberKey]bool {
	alive := make(map[company.MemberKey]bool, len(p.Members))
	for _, id := range p.Members {
		mob := mobs.GetInstance(id)
		alive[mobparty.MemberKeyFor(id)] = mob != nil && mob.Character.Health > 0 && !mob.Character.CombatWithdrawn
	}
	return alive
}

// EffectiveHP mirrors combat.RankMobs' own EHP formula
// (internal/combat/mob_rank.go), applied directly to a live combatant's
// current HealthMax/Defense instead of a simulated spec.
func EffectiveHP(hp, defense int) float64 {
	defFrac := float64(defense) / 200.0
	if defFrac > 0.95 {
		defFrac = 0.95
	}
	if defFrac < 0 {
		defFrac = 0
	}
	return float64(hp) / (1.0 - defFrac)
}

func summaries(room *rooms.Room) []mobparty.MobSummary {
	var out []mobparty.MobSummary
	for _, instanceId := range room.GetMobs() {
		mob := mobs.GetInstance(instanceId)
		if mob == nil || mob.Character.IsCharmed() || mob.Character.CombatWithdrawn {
			continue
		}
		s := rooms.GroupSummary(mob)
		s.EHP = EffectiveHP(mob.Character.HealthMax.Value, mob.Character.GetDefense())
		out = append(out, s)
	}
	return out
}
