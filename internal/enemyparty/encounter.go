package enemyparty

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/encounters"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Encounter is one spawned random room encounter: its group, the foes and
// who they are reserved for.
type Encounter struct {
	ID     string // the spawn group, "encounter:<room>:<first foe>"
	RoomID int
	Owner  int
	Foes   []int // mob instance ids
	Boss   bool
}

// SpawnEncounter spawns a random room encounter (Phase 37) in roomID, set
// upon leaderUserID's company: the foes of foes (levels already planned from
// the zone's band), their detection rolled as for any ambush. All or none
// spawn: a template that will not build rolls back every foe made so far.
// A boss's group runs no strategy (coordination tier 1, Rabble) and the boss
// has 1.75x the HP of an ordinary foe of its level (35d). Must run on the
// game loop.
func SpawnEncounter(roomID, leaderUserID int, foes []encounters.Foe) (Encounter, error) {
	room := rooms.LoadRoom(roomID)
	if room == nil {
		return Encounter{}, fmt.Errorf("enemyparty: room %d is unavailable", roomID)
	}
	if len(foes) < 2 {
		return Encounter{}, fmt.Errorf("enemyparty: an encounter needs at least two foes, got %d", len(foes))
	}
	var built []*mobs.Mob
	rollback := func() {
		for _, m := range built {
			mobs.DestroyInstance(m.InstanceId)
		}
	}
	boss := false
	for _, f := range foes {
		mob := mobs.NewMobById(mobs.MobId(f.MobID), roomID, f.Level)
		if mob == nil {
			rollback()
			return Encounter{}, fmt.Errorf("enemyparty: encounter mob template %d is unavailable", f.MobID)
		}
		built = append(built, mob)
		if f.Boss {
			boss = true
			mob.Boss = true
			mob.Character.HealthMax.Training += int(float64(mob.Character.HealthMax.Value) * encounters.BossHPBonus)
			mob.Character.RecalculateStats()
			mob.Character.Health = mob.Character.HealthMax.Value
		}
	}
	id := EncounterGroup(roomID, built[0].InstanceId)
	summaries := make([]mobparty.MobSummary, len(built))
	for i, m := range built {
		m.EncounterOwner, m.EncounterID, m.EncounterBoss = leaderUserID, id, boss
		m.Solitary = false // the composition made them a group
		if boss {
			m.Coordination = 1 // the boss and its escorts run no strategy
		}
		summaries[i] = rooms.GroupSummary(m)
	}
	engage(roomID, room, built, leaderUserID, id, mobparty.Generate(summaries).Name)
	enc := Encounter{ID: id, RoomID: roomID, Owner: leaderUserID, Boss: boss}
	for _, m := range built {
		enc.Foes = append(enc.Foes, m.InstanceId)
	}
	return enc, nil
}

// Standing counts the encounter's foes still alive in its room.
func (e Encounter) Standing() int {
	n := 0
	for _, id := range e.Foes {
		if m := mobs.GetInstance(id); m != nil && m.Character.Health > 0 && m.Character.RoomId == e.RoomID {
			n++
		}
	}
	return n
}

// Remove takes every surviving foe of the encounter out of the world, once:
// the cleanup after a flee, retreat, abandonment or the leader's death. It
// gives no reward.
func (e Encounter) Remove() {
	room := rooms.LoadRoom(e.RoomID)
	for _, id := range e.Foes {
		if room != nil {
			room.RemoveMob(id)
		}
		mobs.DestroyInstance(id)
	}
}
