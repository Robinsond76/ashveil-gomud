package enemyparty

import (
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/coordination"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// Coordination is a group's coordination tier now (Phase 33i2): from its
// living members' levels and templates. A battle fixes it when it begins
// (BattleTier); the assessment reads it live.
func Coordination(p mobparty.Party) coordination.Tier {
	var levels, explicit []int
	for _, id := range p.Members {
		m := mobs.GetInstance(id)
		if m == nil || m.Character.Health < 1 {
			continue
		}
		levels = append(levels, m.Character.Level)
		explicit = append(explicit, m.Coordination)
	}
	return coordination.Of(levels, explicit)
}

// BattleTier is the coordination tier of the group in the player's
// battle, as fixed when it began. ok is false with no battle.
func BattleTier(userId int) (coordination.Tier, bool) {
	b, ok := battle.Current(userId)
	if !ok {
		return coordination.None, false
	}
	if b.Coordination < 1 {
		return coordination.Rabble, true
	}
	return coordination.Tier(b.Coordination), true
}
