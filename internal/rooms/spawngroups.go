package rooms

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// Phase 29b2: spawn groups. A room's hostile mobs fight as groups of at
// least two, built from its spawn list ("think Ogre Battle"): after the
// spawn pass, the hostile mobs it spawned with no group join one, and a new
// group of one is topped up from the list. A mob marked solitary stands alone, and
// shopkeepers, practice foes, companions, and mobs that aren't hostile are
// never grouped. Spawn groups are runtime only: after a restart the room
// respawns and regroups.

// MinSpawnGroup is the fewest members a spawn group forms with.
const MinSpawnGroup = 2

// groupable is a hostile mob in the room as the planner sees it.
type groupable struct {
	InstanceId int
	MobId      int
	Group      string // its spawn group, "" when it has none
	Fighting   bool
}

// spawnGroupPlan is what the planner decided: the group each ungrouped
// mob joins, and the groups to top up by one.
type spawnGroupPlan struct {
	Assign map[int]string
	TopUp  []string
}

// planSpawnGroups decides who joins which group. An ungrouped mob that
// isn't fighting joins the smallest group in the room that isn't fighting
// and has room, else the ungrouped mobs left form new groups as even as
// they can be. Only a new group of one is topped up: a group whittled down
// by a fight is never reinforced, and an ungrouped mob in a fight waits
// until it is over. newID names a new group.
func planSpawnGroups(ms []groupable, newID func() string) spawnGroupPlan {
	plan := spawnGroupPlan{Assign: map[int]string{}}
	size := map[string]int{}
	fighting := map[string]bool{}
	var order []string
	var loose []int
	for _, m := range ms {
		if m.Group == "" {
			if !m.Fighting {
				loose = append(loose, m.InstanceId)
			}
			continue
		}
		if _, ok := size[m.Group]; !ok {
			order = append(order, m.Group)
		}
		size[m.Group]++
		fighting[m.Group] = fighting[m.Group] || m.Fighting
	}

	var left []int
	for _, id := range loose {
		best := ""
		for _, g := range order {
			if fighting[g] || size[g] >= mobparty.MaxPartySize {
				continue
			}
			if best == "" || size[g] < size[best] {
				best = g
			}
		}
		if best == "" {
			left = append(left, id)
			continue
		}
		plan.Assign[id] = best
		size[best]++
	}

	for _, n := range mobparty.EvenSizes(len(left), mobparty.MaxPartySize) {
		g := newID()
		for _, id := range left[:n] {
			plan.Assign[id] = g
		}
		if n < MinSpawnGroup {
			plan.TopUp = append(plan.TopUp, g)
		}
		left = left[n:]
	}
	return plan
}

// topUpEntry picks the spawn entry a group of one is topped up from: the
// first hostile entry on the list of another kind than the lone member,
// else the first hostile entry. ok is false when the list has none.
func topUpEntry(pool []SpawnInfo, loneMobId int) (SpawnInfo, bool) {
	if len(pool) == 0 {
		return SpawnInfo{}, false
	}
	for _, e := range pool {
		if e.MobId != loneMobId {
			return e, true
		}
	}
	return pool[0], true
}

// groupsHere reports whether a mob takes part in spawn groups.
func groupsHere(mob *mobs.Mob) bool {
	return mob != nil && mob.Hostile && !mob.Solitary && !mob.Practice &&
		!mob.HasShop() && !mob.Character.IsCharmed() && mob.Character.Health > 0
}

// hostilePool is the room's spawn entries that spawn hostile mobs that
// may be grouped: the list groups are built and topped up from.
func (r *Room) hostilePool() []SpawnInfo {
	var pool []SpawnInfo
	for _, e := range r.SpawnInfo {
		if e.MobId <= 0 {
			continue
		}
		spec := mobs.GetMobSpec(mobs.MobId(e.MobId))
		if spec == nil || spec.Solitary || spec.Practice || spec.HasShop() {
			continue
		}
		if spec.Hostile || e.ForceHostile {
			pool = append(pool, e)
		}
	}
	return pool
}

// FormSpawnGroups puts the hostile mobs the room's spawn list made into
// spawn groups of at least two, topping up a new group of one from the
// list. Grouped mobs stop wandering, so a group stays together. Only the
// room's own spawns take part (a mob that wandered in, or a travel
// encounter's pair, is left as it is). It is called after the room's spawn
// pass, on the game loop.
func (r *Room) FormSpawnGroups() {
	spawned := map[int]bool{}
	for _, e := range r.SpawnInfo {
		if e.InstanceId > 0 {
			spawned[e.InstanceId] = true
		}
	}
	ours := fmt.Sprintf("spawn:%d:", r.RoomId)
	var ms []groupable
	byID := map[int]*mobs.Mob{}
	for _, instanceId := range r.mobs {
		mob := mobs.GetInstance(instanceId)
		if !groupsHere(mob) {
			continue
		}
		if mob.SpawnGroup == "" && !spawned[instanceId] {
			continue
		}
		if mob.SpawnGroup != "" && !strings.HasPrefix(mob.SpawnGroup, ours) {
			continue
		}
		byID[instanceId] = mob
		fighting := mob.Character.Aggro != nil || battle.Engaged(instanceId)
		ms = append(ms, groupable{InstanceId: instanceId, MobId: int(mob.MobId), Group: mob.SpawnGroup, Fighting: fighting})
	}
	if len(ms) == 0 {
		return
	}
	plan := planSpawnGroups(ms, func() string {
		r.spawnGroupSeq++
		return fmt.Sprintf("spawn:%d:%d", r.RoomId, r.spawnGroupSeq)
	})
	ids := make([]int, 0, len(plan.Assign))
	for id := range plan.Assign {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		byID[id].SpawnGroup = plan.Assign[id]
		byID[id].MaxWander = 0
	}

	pool := r.hostilePool()
	for _, g := range plan.TopUp {
		lone := 0
		for _, m := range byID {
			if m.SpawnGroup == g {
				lone = int(m.MobId)
				break
			}
		}
		entry, ok := topUpEntry(pool, lone)
		if !ok {
			entry = SpawnInfo{MobId: lone}
		}
		mob := r.spawnMob(entry)
		if mob == nil {
			continue
		}
		mob.Hostile = true // a copy of a hostile mob, or from a hostile entry
		mob.SpawnGroup = g
		mob.MaxWander = 0
		r.mobs = append(r.mobs, mob.InstanceId)
		roomManager.roomsWithMobs[r.RoomId] = len(r.mobs)
	}
}
