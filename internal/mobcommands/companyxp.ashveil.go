package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// awardCompanyXP is Phase 32e: when a kill pays a company's leader, every
// companion of that company that is alive, attached (still charmed by this leader), and in the room where
// the mob died earns the same figure, in full (no split). It returns one
// notice per level a companion gained, for the leader. Companions are
// found through the leader's charmed instances and confirmed against the
// company provider, so a mob charmed some other way is never paid. The
// experience is durable through the company module's snapshot seams.
func awardCompanyXP(leaderUserID int, leader *characters.Character, amount int, roomID int) []string {
	if amount <= 0 {
		return nil
	}
	var lines []string
	for _, instanceID := range leader.GetCharmIds() {
		owner, _, ok := company.LeaderAndKeyForInstance(instanceID)
		if !ok || owner != leaderUserID {
			continue
		}
		mob := mobs.GetInstance(instanceID)
		if mob == nil || mob.Character.RoomId != roomID || mob.Character.Health <= 0 || !mob.Character.IsCharmed(leaderUserID) {
			continue
		}
		mob.Character.GrantXP(amount)
		for {
			gained, _ := mob.Character.LevelUp()
			if !gained {
				break
			}
			lines = append(lines, company.LevelLine(mob.Character.Name, mob.Character.Level))
		}
		if len(lines) > 0 {
			// A respawn at this level would spend the level's points, so a
			// companion levelled live must too, or it is stronger after a
			// logout than before.
			mob.Character.AutoTrain()
		}
	}
	return lines
}
