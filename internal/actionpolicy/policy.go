// Package actionpolicy holds the shared player/follower command restrictions.
// It reads authoritative runtime state on the game loop and owns no saves.
package actionpolicy

import (
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

const BattleUnderWay = "The battle is under way: it plays out as you set it up."

func InBattle(u *users.UserRecord) bool {
	if u == nil || u.Character == nil {
		return false
	}
	if _, ok := battle.Current(u.UserId); ok {
		return true
	}
	return AimedAtMob(u)
}

// AimedAtMob reports whether the player is aimed at a mob in the room: a
// battle about to begin. An aim at a foe already slain is not one: the last
// blow of a fight kills its target and ends the battle that round, but the
// player's aim is only cleared by the next round's combat, and until then a
// finished fight kept answering "The battle is under way" (phase 44 follow-up).
func AimedAtMob(u *users.UserRecord) bool {
	if u == nil || u.Character == nil {
		return false
	}
	a := u.Character.Aggro
	if a == nil || a.MobInstanceId <= 0 || a.ExitName != "" {
		return false
	}
	if m := mobs.GetInstance(a.MobInstanceId); m != nil && m.Character.Health < 1 {
		return false
	}
	return true
}

func Management(command string) bool {
	switch command {
	case "equip", "remove", "gearup", "eat", "drink", "use", "give", "get", "drop", "put", "alchemy", "loot", "imbue":
		return true
	}
	return false
}

func Hostile(command string) bool {
	switch command {
	case "attack", "backstab", "shoot", "throw", "cast", "tackle", "disarm":
		return true
	}
	return false
}

// Member validates an order both before queueing and when it executes. A
// temporary follower can be managed, but charm alone cannot authorize a
// tracked companion belonging to another company. All requested hostility
// is refused; automatic battle upkeep remains the only combat controller.
func Member(order events.MemberOrder, m *mobs.Mob, command string) string {
	if order.Scripted {
		return scripted(order, m, command)
	}
	u := users.GetByUserId(order.UserID)
	if u == nil || u.Character == nil || u.Character.Health < 1 || m == nil || m.Character.Health < 1 || m.Character.CombatWithdrawn {
		return "That member is no longer available for orders."
	}
	if u.Character.RoomId != order.RoomID || m.Character.RoomId != order.RoomID {
		return "You and the member must still be in the same room."
	}
	leader, key, member := company.LeaderAndKeyForInstance(m.InstanceId)
	if !m.Character.IsCharmed(order.UserID) || m.Character.Charmed.RoundsRemaining == 0 || order.CharmToken != m.Character.Charmed || (member && leader != order.UserID) || string(key) != order.MemberKey {
		return "You no longer command that member."
	}
	if Hostile(command) {
		return "Members fight automatically. Use attack [group] to begin a battle."
	}
	if Management(command) && (InBattle(u) || m.Character.Aggro != nil) {
		return BattleUnderWay
	}
	if member && u.Character.CompanyCargo && (command == "equip" || command == "remove" || command == "eat" || command == "drink" || command == "give" || command == "drop") {
		return "Use company equip/remove for gear and company meal for supplies from shared cargo."
	}
	switch command {
	case "say", "look", "emote", "give", "get", "drop", "equip", "remove", "eat", "drink":
		return ""
	}
	return "That member order is not available. Use company to manage your band."
}

// scripted checks a follower's own script (see MemberOrder.Scripted). Its
// owner need not be present or the one whose command ran the script, and
// the script may move the member, but it never fights or changes gear in
// battle and stops once the charm or membership behind it changes.
func scripted(order events.MemberOrder, m *mobs.Mob, command string) string {
	if m == nil || m.Character.Health < 1 || m.Character.CombatWithdrawn {
		return "That member is no longer available for orders."
	}
	leader, key, member := company.LeaderAndKeyForInstance(m.InstanceId)
	if !m.Character.IsCharmed(order.UserID) || m.Character.Charmed.RoundsRemaining == 0 || order.CharmToken != m.Character.Charmed || (member && leader != order.UserID) || string(key) != order.MemberKey {
		return "You no longer command that member."
	}
	if Hostile(command) {
		return "Members fight automatically."
	}
	if Management(command) && (InBattle(users.GetByUserId(order.UserID)) || m.Character.Aggro != nil) {
		return BattleUnderWay
	}
	if u := users.GetByUserId(order.UserID); member && u != nil && u.Character.CompanyCargo && (command == "equip" || command == "remove" || command == "eat" || command == "drink" || command == "give" || command == "drop") {
		return "Use company management for shared cargo and equipment."
	}
	return ""
}
