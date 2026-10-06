package scripting

import (
	"sort"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// tremorRoll rolls a Tremor's chances; tests replace it.
var tremorRoll = util.Rand

// UseTremorRollForTest replaces Tremor's dice until the returned restore is
// called.
func UseTremorRollForTest(roll func(int) int) (restore func()) {
	prev := tremorRoll
	tremorRoll = roll
	return func() { tremorRoll = prev }
}

// TremorBossPenalty is how many points a boss's chance to be knocked down by
// a Tremor is below another foe's (the 38a boss rule, a Mountain Speaker's
// tremor never less than 5).
const TremorBossPenalty = 15

// Tremor shakes the ground under the front row of the actor's foes (Phase
// 39i, a Mountain Speaker's Tremor): each standing foe of the battle in the
// foe group's front-most occupied row has pct percent chance (a boss
// pct-15, at least 5) to be knocked down. It returns the names of the foes
// felled; an empty list when the earth did not move anyone.
func (a ScriptActor) Tremor(pct int) []string {
	c := a.characterRecord
	leader := a.battleLeader()
	if c == nil || leader == 0 || pct <= 0 {
		return nil
	}
	b, ok := battle.Current(leader)
	if !ok {
		return nil
	}
	room := rooms.LoadRoom(c.RoomId)
	if room == nil {
		return nil
	}
	type foe struct {
		mob *mobs.Mob
		row int
	}
	var foes []foe
	front := company.FormationRows
	for id := range b.Enemies {
		m := mobs.GetInstance(id)
		if m == nil || m.Character.Health < 1 || m.Character.CombatWithdrawn || m.Character.RoomId != c.RoomId || m.Character.HasBuffFlag("hidden") {
			continue
		}
		row := 0
		if party, ok := enemyparty.PartyOf(room, id); ok {
			if r, _, found := party.Formation.Find(mobparty.MemberKeyFor(id)); found {
				row = r
			}
		}
		foes = append(foes, foe{m, row})
		front = min(front, row)
	}
	sort.Slice(foes, func(i, j int) bool { return foes[i].mob.InstanceId < foes[j].mob.InstanceId })
	var felled []string
	for _, f := range foes {
		if f.row != front || status.Live(&f.mob.Character, status.KnockedDown) {
			continue
		}
		chance := pct
		if f.mob.Boss {
			chance = max(5, pct-TremorBossPenalty)
		}
		if tremorRoll(100) >= chance {
			continue
		}
		events.AddToQueue(events.Buff{MobInstanceId: f.mob.InstanceId, BuffId: status.KnockedDown, Source: `spell`})
		if m := GetMob(f.mob.InstanceId); m != nil {
			felled = append(felled, m.GetCombatName(false))
		}
	}
	return felled
}
