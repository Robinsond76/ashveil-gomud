package mobcommands

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"strconv"
	"strings"
)

// AwardCompanyXP is Phase 32e: when a kill pays a company's leader, every
// companion of that company that is alive, attached (still charmed by this leader), and in the room where
// the mob died earns the same figure, in full (no split). It returns one
// notice per companion that gained levels, for the leader, and how many
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
		owner, memberKey, ok := company.LeaderAndKeyForInstance(instanceID)
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
		before := mob.Character
		health, mana := mob.Character.Health, mob.Character.Mana
		levelled := false
		for {
			gained, _ := mob.Character.LevelUp()
			if !gained {
				break
			}
			levelled = true
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
			lineage, who := "", ""
			if id, ok := company.CompanionIDFromMemberKey(memberKey); ok {
				lineage, _ = company.CompanionArchetype(leaderUserID, id)
				who = "#" + strconv.Itoa(id)
			}
			lines = append(lines, companionLevelLine(before, mob.Character, lineage, who))
		}
	}
	return paid, lines
}

// One report spans all gained levels and the final derived companion training.
func companionLevelLine(before, after characters.Character, lineage, who string) string {
	changes := []string{fmt.Sprintf("Health %d -> %d", before.HealthMax.Value, after.HealthMax.Value), fmt.Sprintf("Mana %d -> %d", before.ManaMax.Value, after.ManaMax.Value),
		// Phase 35a2: what the level made it better at.
		fmt.Sprintf("Attack %d -> %d", before.AttackSkill(), after.AttackSkill()), fmt.Sprintf("Evasion %d -> %d", before.Evasion(), after.Evasion())}
	old := []int{before.Stats.Strength.ValueAdj, before.Stats.Speed.ValueAdj, before.Stats.Smarts.ValueAdj, before.Stats.Vitality.ValueAdj, before.Stats.Mysticism.ValueAdj, before.Stats.Perception.ValueAdj}
	now := []int{after.Stats.Strength.ValueAdj, after.Stats.Speed.ValueAdj, after.Stats.Smarts.ValueAdj, after.Stats.Vitality.ValueAdj, after.Stats.Mysticism.ValueAdj, after.Stats.Perception.ValueAdj}
	for i, name := range []string{"Strength", "Speed", "Smarts", "Vitality", "Mysticism", "Perception"} {
		if old[i] != now[i] {
			changes = append(changes, fmt.Sprintf("%s %d -> %d", name, old[i], now[i]))
		}
	}
	line := fmt.Sprintf("%s reaches level %d (%s).", after.Name, after.Level, strings.Join(changes, ", "))
	// Phase 38b: what its class gains next.
	class, _ := after.ClassState()
	if next := classes.Milestone(class, after.Level); next != "" {
		line += " " + next
	}
	// Phase 38c1: each rank it earned, and an elite promotion ready or
	// waiting on its gate.
	for _, note := range classes.LevelNotes(lineage, class, before.Level, after.Level, int(after.Alignment), who, after.Name) {
		line += " " + note
	}
	return line
}
