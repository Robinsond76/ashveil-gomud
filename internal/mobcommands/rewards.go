package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"math"
	"slices"
	"sort"
)

// Accepted alliance membership is neither contribution nor a reward claim.
// Each company needs actual positive damage to this enemy and a current battle
// against it. Mercy uses its surviving contribution snapshot after battle end.
func eligibleContributors(m *mobs.Mob, roomID int, mercy bool) []int {
	var result []int
	for uid, damage := range m.Character.PlayerDamage {
		u := users.GetByUserId(uid)
		if damage <= 0 || u == nil || u.Character == nil || u.Character.RoomId != roomID || u.Character.Health <= 0 || u.Character.CombatWithdrawn || users.IsLinkDeadConnection(u.ConnectionId()) {
			continue
		}
		if m.RewardContributors != nil && !mercy {
			if slices.Contains(m.RewardContributors, uid) {
				result = append(result, uid)
			}
			continue
		}
		b, active := battle.Current(uid)
		if (!active && !mercy) || (active && (b.RoomId != roomID || !b.Has(m.InstanceId))) {
			continue
		}
		result = append(result, uid)
	}
	sort.Ints(result)
	return result
}

func rewardShares(m *mobs.Mob, xp int, ids []int) map[int]int {
	shares := map[int]int{}
	if len(ids) == 0 {
		return shares
	}
	levels := 0
	for _, id := range ids {
		levels += users.GetByUserId(id).Character.Level
	}
	scaler := 1.0
	if levels > 0 && math.Abs(float64(m.Character.Level-levels)) > 5 {
		scaler = max(0.25, min(1.5, float64(m.Character.Level)/float64(levels)))
	}
	base := xp / 90
	base = max(0, base+(util.Rand(3)-1)*max(1, base/100))
	pool := int(math.Ceil(float64(base) * scaler))
	if m.IsElite {
		bonus := int(configs.GetGamePlayConfig().EliteXPBonus)
		if bonus <= 0 {
			bonus = 10
		}
		pool += int(math.Ceil(float64(pool) * float64(bonus) / 100))
	}
	for i, id := range ids {
		shares[id] = pool / len(ids)
		if i < pool%len(ids) {
			shares[id]++
		}
	}
	return shares
}

// CaptureRewardContributors freezes battle membership before round settlement
// ends a victorious battle or starts the next group. Presence/life are still
// checked at payout. A non-nil empty slice also records an ineligible death.
func CaptureRewardContributors(m *mobs.Mob) {
	if m.RewardContributors == nil {
		m.RewardContributors = append([]int{}, eligibleContributors(m, m.Character.RoomId, false)...)
	}
}
