package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ValidPartyFollow is checked at actual dequeue, never only when a move is
// queued. Following cannot answer an interactive prompt or replay old authority.
func ValidPartyFollow(input events.Input) bool {
	if attack := input.PartyAttack; attack != nil {
		u := users.GetByUserId(input.UserId)
		p := parties.Get(input.UserId)
		target := mobs.GetInstance(attack.TargetMobInstanceId)
		return u != nil && u.Character != nil && u.GetPrompt() == nil && p != nil &&
			u.Character.RoomId == attack.OriginRoomId && p.LeaderUserId == attack.LeaderUserId &&
			attack.ConsentToken != 0 && p.AutoAttackToken(input.UserId) == attack.ConsentToken &&
			target != nil && target.Character.Health > 0 && !target.Character.CombatWithdrawn &&
			target.Character.RoomId == attack.OriginRoomId && battle.Allows(input.UserId, target.InstanceId)
	}
	order := input.PartyFollow
	if order == nil {
		return true
	}
	u := users.GetByUserId(input.UserId)
	p := parties.Get(input.UserId)
	return u != nil && u.Character != nil && u.GetPrompt() == nil &&
		u.Character.RoomId == order.OriginRoomId && p != nil &&
		p.LeaderUserId == order.LeaderUserId && order.ConsentToken != 0 &&
		p.FollowToken(input.UserId) == order.ConsentToken
}
