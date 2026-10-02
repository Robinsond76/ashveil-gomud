package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// AwardCompanyXP is Phase 32e: when a kill pays a company's leader, every
// companion of that company that is alive, attached (still charmed by this leader), and in the room where
// the mob died earns the same figure, in full (no split). It returns one
// notice per level a companion gained, for the leader, and how many
// companions it paid. Companions are
// found through the leader's charmed instances and confirmed against the
// company provider, so a mob charmed some other way is never paid. The
// experience is durable through the company module's snapshot seams.
// Phase 33h1 contracts pay through the same rule from the quest hook.
func AwardCompanyXP(leaderUserID int, leader *characters.Character, amount int, roomID int) (paid int, lines []string) {
	if amount <= 0 {
		return 0, nil
	}
	for _, instanceID := range leader.GetCharmIds() {
		owner, _, ok := company.LeaderAndKeyForInstance(instanceID)
		if !ok || owner != leaderUserID {
			continue
		}
		mob := mobs.GetInstance(instanceID)
		if mob == nil || mob.Character.RoomId != roomID || mob.Character.Health <= 0 || mob.Character.CombatWithdrawn || !mob.Character.IsCharmed(leaderUserID) {
			continue
		}
		mob.Character.GrantXP(amount)
		paid++
		// Phase 33h2: a companion's level-up keeps its health and mana
		// (GoMud's level-up refills them), so a level is no free rest. It
		// never lowers them either: a level never lowers the maximum.
		health, mana := mob.Character.Health, mob.Character.Mana
		levelled := false
		for {
			gained, _ := mob.Character.LevelUp()
			if !gained {
				break
			}
			levelled = true
			lines = append(lines, company.LevelLine(mob.Character.Name, mob.Character.Level))
		}
		if levelled {
			// A respawn at this level deals the level's points by the
			// companion's growth (Phase 33h1), so a companion levelled live
			// must get the same training, or a logout would change it.
			if !company.RetrainCompanion(instanceID) {
				mob.Character.AutoTrain()
			}
			mob.Character.Health = min(health, mob.Character.HealthMax.Value)
			mob.Character.Mana = min(mana, mob.Character.ManaMax.Value)
		}
	}
	return paid, lines
}
