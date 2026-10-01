package enemyparty

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// SpawnAmbush spawns a hostile encounter of a mob template in roomID that
// attacks the leader: a named pair of its kind, or one if it is solitary
// (Phase 29b2: no lone enemies). It returns the first foe's instance ID.
// Travel ambushes (Phase 12c) and camp raids (Phase 33f3) both use it. It
// must run on the game loop.
func SpawnAmbush(roomID, mobTemplateID, leaderUserID int) (int, error) {
	room := rooms.LoadRoom(roomID)
	if room == nil {
		return 0, fmt.Errorf("enemyparty: room %d is unavailable", roomID)
	}
	mob := mobs.NewMobById(mobs.MobId(mobTemplateID), roomID)
	if mob == nil {
		return 0, fmt.Errorf("enemyparty: encounter mob template %d is unavailable", mobTemplateID)
	}
	foes := []*mobs.Mob{mob}
	if !mob.Solitary {
		if second := mobs.NewMobById(mobs.MobId(mobTemplateID), roomID); second != nil {
			foes = append(foes, second)
		}
	}
	group, groupName := "", ""
	if len(foes) > 1 {
		group = EncounterGroup(roomID, mob.InstanceId)
		// Phase 32c: the pair is named as it forms ("a band of ruffians")
		// and keeps the name while it stands.
		summaries := make([]mobparty.MobSummary, len(foes))
		for i, foe := range foes {
			summaries[i] = rooms.GroupSummary(foe)
		}
		groupName = mobparty.Generate(summaries).Name
	}
	for _, foe := range foes {
		foe.Hostile = true
		foe.MaxWander = 0
		foe.SpawnGroup = group
		foe.GroupName = groupName
		room.AddMob(foe.InstanceId)
		foe.Command(fmt.Sprintf("attack @%d", leaderUserID))
	}
	return mob.InstanceId, nil
}

// EncounterGroup names an encounter's spawn group after its first foe.
func EncounterGroup(roomID, firstInstanceID int) string {
	return fmt.Sprintf("encounter:%d:%d", roomID, firstInstanceID)
}
