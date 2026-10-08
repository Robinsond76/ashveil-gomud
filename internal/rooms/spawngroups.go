package rooms

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
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
	Straggler  bool // not spawned by this room's list: it wandered in
}

// spawnGroupPlan is what the planner decided: the group each mob joins
// (an ungrouped mob, or a lone survivor regrouping), and the groups to top
// up by one.
type spawnGroupPlan struct {
	Assign map[int]string
	TopUp  []string
}

// planSpawnGroups decides who joins which group, the way a broken unit
// regroups in Ogre Battle:
//   - a lone survivor of a group, idle now, joins the smallest other idle
//     group with room;
//   - an ungrouped mob that isn't fighting joins the smallest idle group
//     with room;
//   - the room's own ungrouped spawns left over form new groups as even as
//     they can be, and a new group of one is topped up. A straggler that
//     wandered in never starts a group: with none to join it stays as it is.
//
// A group in a fight is never joined or reinforced, and an ungrouped mob in
// a fight waits until it is over. newID names a new group.
func planSpawnGroups(ms []groupable, newID func() string) spawnGroupPlan {
	plan := spawnGroupPlan{Assign: map[int]string{}}
	size := map[string]int{}
	fighting := map[string]bool{}
	members := map[string][]int{}
	var order []string
	var loose []groupable
	for _, m := range ms {
		if m.Group == "" {
			if !m.Fighting {
				loose = append(loose, m)
			}
			continue
		}
		if _, ok := size[m.Group]; !ok {
			order = append(order, m.Group)
		}
		size[m.Group]++
		members[m.Group] = append(members[m.Group], m.InstanceId)
		fighting[m.Group] = fighting[m.Group] || m.Fighting
	}

	// smallestIdle is the smallest idle group with room, other than skip.
	smallestIdle := func(skip string) string {
		best := ""
		for _, g := range order {
			if g == skip || size[g] == 0 || fighting[g] || size[g] >= mobparty.MaxPartySize {
				continue
			}
			if best == "" || size[g] < size[best] {
				best = g
			}
		}
		return best
	}

	for _, g := range order {
		if size[g] != 1 || fighting[g] {
			continue
		}
		if to := smallestIdle(g); to != "" {
			plan.Assign[members[g][0]] = to
			size[to]++
			size[g] = 0
		}
	}

	var left []int
	for _, m := range loose {
		if best := smallestIdle(""); best != "" {
			plan.Assign[m.InstanceId] = best
			size[best]++
			continue
		}
		if !m.Straggler {
			left = append(left, m.InstanceId)
		}
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
// list, and regroups lone survivors and stragglers into the room's idle
// groups (planSpawnGroups). Grouped mobs stop wandering, so a group stays
// together. Another room's group, or a travel encounter's pair, is left as
// it is. It is called after the room's spawn pass, on the game loop.
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
		if mob.SpawnGroup != "" && !strings.HasPrefix(mob.SpawnGroup, ours) {
			continue // another room's group, or a travel encounter's
		}
		byID[instanceId] = mob
		fighting := mob.Character.Aggro != nil || battle.Engaged(instanceId)
		ms = append(ms, groupable{InstanceId: instanceId, MobId: int(mob.MobId), Group: mob.SpawnGroup, Fighting: fighting,
			Straggler: mob.SpawnGroup == "" && !spawned[instanceId]})
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
	joined := map[int]bool{}
	for _, id := range ids {
		byID[id].SpawnGroup = plan.Assign[id]
		byID[id].MaxWander = 0
		joined[id] = true
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
		joined[mob.InstanceId] = true
	}

	r.mixSpawnGroups(ours, pool, joined)
	r.nameSpawnGroups(ours, joined)
}

// mixSwap is one member of a one-kind group to replace with a pool entry.
type mixSwap struct {
	InstanceId int
	Entry      int // index into the pool
}

// planMixes picks, for each idle group of three or more that is all one
// kind, its last member to swap for a pool entry of another kind (Phase 89:
// a group is not a row of the same creature). pick returns 0..n-1. A group
// that is fighting, already mixed, or smaller is left; so is any group when
// the pool has no other kind.
func planMixes(groups map[string][]groupable, order []string, pool []SpawnInfo, pick func(n int) int) []mixSwap {
	var swaps []mixSwap
	for _, g := range order {
		members := groups[g]
		if len(members) < 3 {
			continue
		}
		skip := false
		for _, m := range members {
			skip = skip || m.Fighting || m.MobId != members[0].MobId // fighting, or already mixed
		}
		if skip {
			continue
		}
		var other []int
		for i, e := range pool {
			if e.MobId != members[0].MobId {
				other = append(other, i)
			}
		}
		if len(other) == 0 {
			continue
		}
		swaps = append(swaps, mixSwap{InstanceId: members[len(members)-1].InstanceId, Entry: other[pick(len(other))]})
	}
	return swaps
}

// mixSpawnGroups makes the room's idle one-kind groups of three or more
// mixed, swapping a member for another kind from the room's spawn list.
// The replaced member's spawn entry goes on cooldown like a killed one, and
// the newcomer joins the group the way a top-up does.
func (r *Room) mixSpawnGroups(prefix string, pool []SpawnInfo, joined map[int]bool) {
	if len(pool) < 2 {
		return
	}
	groups := map[string][]groupable{}
	var order []string
	for _, instanceId := range r.mobs {
		mob := mobs.GetInstance(instanceId)
		if mob == nil || !groupsHere(mob) || !strings.HasPrefix(mob.SpawnGroup, prefix) {
			continue
		}
		if _, ok := groups[mob.SpawnGroup]; !ok {
			order = append(order, mob.SpawnGroup)
		}
		groups[mob.SpawnGroup] = append(groups[mob.SpawnGroup], groupable{InstanceId: instanceId, MobId: int(mob.MobId),
			Group: mob.SpawnGroup, Fighting: mob.Character.Aggro != nil || battle.Engaged(instanceId)})
	}
	for _, sw := range planMixes(groups, order, pool, util.Rand) {
		old := mobs.GetInstance(sw.InstanceId)
		if old == nil {
			continue
		}
		mob := r.spawnMob(pool[sw.Entry])
		if mob == nil {
			continue
		}
		group, name, desc := old.SpawnGroup, old.GroupName, old.GroupDesc
		for i, e := range r.SpawnInfo {
			if e.InstanceId == sw.InstanceId {
				r.SpawnInfo[i].InstanceId = 0
				r.SpawnInfo[i].DespawnedRound = util.GetRoundCount()
			}
		}
		r.RemoveMob(sw.InstanceId)
		mobs.DestroyInstance(sw.InstanceId)
		mob.Hostile = true
		mob.SpawnGroup, mob.GroupName, mob.GroupDesc = group, name, desc
		mob.MaxWander = 0
		r.mobs = append(r.mobs, mob.InstanceId)
		roomManager.roomsWithMobs[r.RoomId] = len(r.mobs)
		joined[mob.InstanceId] = true
	}
}

// nameSpawnGroups gives each of the room's spawn groups its name (Phase
// 32c), once: a group keeps the name its standing members carry, and a mob
// that just joined it (a regrouped survivor, a top-up) takes that name. A
// group with no name yet takes an authored one from a member's spawn entry
// (groupname), else a generated one ("a band of ruffians").
func (r *Room) nameSpawnGroups(prefix string, joined map[int]bool) {
	entryOf := map[int]SpawnInfo{}
	for _, e := range r.SpawnInfo {
		if e.InstanceId > 0 {
			entryOf[e.InstanceId] = e
		}
	}
	members := map[string][]*mobs.Mob{}
	var order []string
	for _, instanceId := range r.mobs {
		mob := mobs.GetInstance(instanceId)
		if mob == nil || !strings.HasPrefix(mob.SpawnGroup, prefix) || mob.Character.Health < 1 {
			continue
		}
		if _, ok := members[mob.SpawnGroup]; !ok {
			order = append(order, mob.SpawnGroup)
		}
		members[mob.SpawnGroup] = append(members[mob.SpawnGroup], mob)
	}
	for _, g := range order {
		ms := members[g]
		name, desc := "", ""
		for _, m := range ms {
			if !joined[m.InstanceId] && m.GroupName != "" {
				name, desc = m.GroupName, m.GroupDesc
				break
			}
		}
		if name == "" {
			for _, m := range ms {
				if e, ok := entryOf[m.InstanceId]; ok && e.GroupName != "" {
					name, desc = e.GroupName, e.GroupDesc
					break
				}
			}
		}
		if name == "" {
			summaries := make([]mobparty.MobSummary, len(ms))
			for i, m := range ms {
				summaries[i] = GroupSummary(m)
			}
			name = mobparty.Generate(summaries).Name
		}
		for _, m := range ms {
			m.GroupName, m.GroupDesc = name, desc
		}
	}
}

// GroupSummary is a mob as group naming and grouping see it (Phase 32c).
func GroupSummary(mob *mobs.Mob) mobparty.MobSummary {
	return mobparty.MobSummary{
		InstanceId: mob.InstanceId,
		SpawnGroup: mob.SpawnGroup,
		Groups:     mob.Groups,
		EHP:        float64(mob.Character.HealthMax.Value),
		Name:       mob.Character.Name,
		Noun:       mob.CollectiveNoun(),
		GroupName:  mob.GroupName,
		GroupDesc:  mob.GroupDesc,
		Hostile:    mob.Hostile,
	}
}
