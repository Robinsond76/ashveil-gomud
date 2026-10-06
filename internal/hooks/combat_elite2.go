package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Phase 39i2: helpers the Beast Tamer, Gryphon Rider and Arbalist elites
// share for finding the foe beside, or behind, another in the enemy
// formation.

// standingFoe reports whether a mob can still be struck: alive, in the
// fight and not hidden.
func standingFoe(m *mobs.Mob) bool {
	return m != nil && m.Character.Health > 0 && !m.Character.CombatWithdrawn && !m.Character.HasBuffFlag("hidden")
}

// foesBesideRow are the standing foes in the same formation row as foe,
// nearest column first. When foes is not nil only those instances count.
func foesBesideRow(foe *mobs.Mob, room *rooms.Room, foes map[int]bool) []*mobs.Mob {
	party, ok := enemyparty.PartyOf(room, foe.InstanceId)
	if !ok {
		return nil
	}
	row, col, found := party.Formation.Find(mobparty.MemberKeyFor(foe.InstanceId))
	if !found {
		return nil
	}
	alive := enemyparty.Alive(party)
	var out []*mobs.Mob
	var dist []int
	for c := 0; c < company.FormationCols; c++ {
		key := party.Formation.At(row, c)
		if c == col || key == "" || !alive[key] {
			continue
		}
		id, ok := mobparty.InstanceIdFromMemberKey(key)
		if !ok || (foes != nil && !foes[id]) {
			continue
		}
		if m := mobs.GetInstance(id); standingFoe(m) {
			out = append(out, m)
			dist = append(dist, abs(c-col))
		}
	}
	for i := 1; i < len(out); i++ { // insertion sort: at most two foes
		for j := i; j > 0 && dist[j] < dist[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
			dist[j], dist[j-1] = dist[j-1], dist[j]
		}
	}
	return out
}

// foeBeside is the standing foe nearest foe in its formation row, nil when
// none stands beside it.
func foeBeside(foe *mobs.Mob, room *rooms.Room, foes map[int]bool) *mobs.Mob {
	if out := foesBesideRow(foe, room, foes); len(out) > 0 {
		return out[0]
	}
	return nil
}

// foeBehindInColumn is the nearest standing foe behind foe in its own
// formation column, nil when none stands there.
func foeBehindInColumn(foe *mobs.Mob, room *rooms.Room, foes map[int]bool) *mobs.Mob {
	party, ok := enemyparty.PartyOf(room, foe.InstanceId)
	if !ok {
		return nil
	}
	row, col, found := party.Formation.Find(mobparty.MemberKeyFor(foe.InstanceId))
	if !found {
		return nil
	}
	alive := enemyparty.Alive(party)
	for r := row + 1; r < company.FormationRows; r++ {
		key := party.Formation.At(r, col)
		if key == "" || !alive[key] {
			continue
		}
		id, ok := mobparty.InstanceIdFromMemberKey(key)
		if !ok || (foes != nil && !foes[id]) {
			continue
		}
		if m := mobs.GetInstance(id); standingFoe(m) {
			return m
		}
	}
	return nil
}
