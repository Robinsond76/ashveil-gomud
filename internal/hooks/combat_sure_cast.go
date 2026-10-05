package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 35b: a spell a character owns never fizzles in battle. The cast
// roll still decides casts outside a battle (and the spellbook's chance).

// playerCastsSure reports whether the player's cast skips the roll: they
// are in a battle and the spell is in their spellbook.
func playerCastsSure(user *users.UserRecord, spellId string) bool {
	if _, ok := battle.Current(user.UserId); !ok {
		return false
	}
	return user.Character.HasSpell(spellId)
}

// mobCastsSure reports whether a mob's cast skips the roll: it is in a
// battle (an enemy in some player's battle, or a companion or charmed ally
// whose player is in one) and it knows the spell (its spellbook, or, for a
// companion, its archetype's spells at its level).
func mobCastsSure(mob *mobs.Mob, spellId string) bool {
	if !mobInBattle(mob.InstanceId) {
		return false
	}
	if mob.Character.HasSpell(spellId) {
		return true
	}
	leaderId, key, ok := company.LeaderAndKeyForInstance(mob.InstanceId)
	if !ok {
		return false
	}
	id, ok := company.CompanionIDFromMemberKey(key)
	if !ok {
		return false
	}
	arch, _ := company.CompanionArchetype(leaderId, id)
	for _, known := range archetypes.CompanionSpells(arch, mob.Character.Level) {
		if known == spellId {
			return true
		}
	}
	return false
}

// mobInBattle reports whether a mob is in a battle: an enemy in a player's
// battle, or an ally of a player who is in one.
func mobInBattle(instanceId int) bool {
	if battle.Engaged(instanceId) {
		return true
	}
	if owner, ok := allyOwner(instanceId); ok {
		_, inBattle := battle.Current(owner)
		return inBattle
	}
	return false
}
