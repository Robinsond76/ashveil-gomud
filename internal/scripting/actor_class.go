package scripting

import (
	"slices"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/summons"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 38b: what class spells ask of the engine. Their numbers come from
// the class effects (ClassEffect) and the spell's power block; the state
// they set lives in the character's class battle state, which combat reads.

// ClassEffect is a class effect's value for the actor at its level (0 when
// it has none).
func (a ScriptActor) ClassEffect(key string) int {
	if a.characterRecord == nil {
		return 0
	}
	return a.characterRecord.ClassEffects().Int(key)
}

// GrantWard puts a ward on the actor: the next blows blows are each
// absorbed up to cap. False when it already carries one.
func (a ScriptActor) GrantWard(cap, blows int) bool {
	c := a.characterRecord
	if c == nil || c.Health < 1 || cap < 1 || blows < 1 {
		return false
	}
	rt := c.RTState()
	if rt.Ward > 0 && !rt.WardSigil { // Phase 54 review: a sigil's small ward gives way
		return false
	}
	rt.Ward, rt.WardCap, rt.WardSigil = blows, cap, false
	return true
}

// GrantBark gives the actor Barkskin: armor for the battle, and a foe that
// strikes it takes thorns damage. False when it already has it.
func (a ScriptActor) GrantBark(armor, thorns int) bool {
	c := a.characterRecord
	if c == nil || c.Health < 1 {
		return false
	}
	rt := c.RTState()
	if rt.Bark > 0 {
		return false
	}
	rt.Bark, rt.Thorns = armor, thorns
	return true
}

// StartRejuv starts the actor healing over rounds rounds, total points in
// all (at least 1 a round). False when it is already rejuvenating.
func (a ScriptActor) StartRejuv(rounds, total int) bool {
	c := a.characterRecord
	if c == nil || c.Health < 1 || rounds < 1 {
		return false
	}
	rt := c.RTState()
	if rt.Rejuv > 0 {
		return false
	}
	rt.Rejuv, rt.Per = rounds, max(1, total/rounds)
	return true
}

// GrantBless blesses the actor for rounds combat rounds. False when it is
// already blessed.
func (a ScriptActor) GrantBless(rounds int) bool {
	c := a.characterRecord
	if c == nil || c.Health < 1 {
		return false
	}
	rt := c.RTState()
	if rt.Bless > 0 {
		return false
	}
	rt.Bless = rounds
	return true
}

// CleanseOne removes one harmful combat status from the actor, once per
// battle for each cleansing spell, and returns its word ("" for none).
func (a ScriptActor) CleanseOne(source string) string {
	c := a.characterRecord
	if c == nil || c.Health < 1 {
		return ""
	}
	rt := c.RTState()
	if rt.Cleansed[source] {
		return ""
	}
	word := status.CleanseOne(c)
	if word == "" {
		return ""
	}
	if rt.Cleansed == nil {
		rt.Cleansed = map[string]bool{}
	}
	rt.Cleansed[source] = true
	return word
}

// HurtAllies are the living members of the actor's company in its room that
// are below their wound limit, the most hurt (by share of health) first.
// others leaves the actor itself out.
func (a ScriptActor) HurtAllies(others bool) []*ScriptActor {
	return a.companyAllies(others, true)
}

// AllAllies are every living member of the actor's company in its room,
// the most hurt (by share of health) first (Phase 38c3).
func (a ScriptActor) AllAllies(others bool) []*ScriptActor {
	return a.companyAllies(others, false)
}

func (a ScriptActor) companyAllies(others, hurtOnly bool) []*ScriptActor {
	c := a.characterRecord
	if c == nil {
		return nil
	}
	leader := a.userId
	if leader == 0 {
		leader = c.GetCharmedUserId()
	}
	if leader == 0 {
		return nil
	}
	type hurt struct {
		actor *ScriptActor
		frac  float64
	}
	var found []hurt
	consider := func(x *ScriptActor) {
		if x == nil || x.characterRecord == nil || x.characterRecord.Health < 1 || x.characterRecord.RoomId != c.RoomId {
			return
		}
		if others && x.userId == a.userId && x.mobInstanceId == a.mobInstanceId {
			return
		}
		limit := max(1, x.characterRecord.HealthLimit())
		if hurtOnly && x.characterRecord.Health >= limit {
			return
		}
		found = append(found, hurt{x, float64(x.characterRecord.Health) / float64(limit)})
	}
	if u := users.GetByUserId(leader); u != nil {
		consider(GetUser(leader))
	}
	if room := rooms.LoadRoom(c.RoomId); room != nil {
		for _, id := range room.GetMobs(rooms.FindCharmed) {
			if l, _, ok := company.LeaderAndKeyForInstance(id); ok && l == leader && mobs.GetInstance(id) != nil {
				consider(GetMob(id))
			}
		}
	}
	sort.SliceStable(found, func(i, j int) bool { return found[i].frac < found[j].frac })
	out := make([]*ScriptActor, len(found))
	for i, f := range found {
		out[i] = f.actor
	}
	return out
}

// MostHurtAlly is the first of HurtAllies, nil when no one is hurt.
func (a ScriptActor) MostHurtAlly(others bool) *ScriptActor {
	if all := a.HurtAllies(others); len(all) > 0 {
		return all[0]
	}
	return nil
}

// RowAllies is the living members of the actor's company that stand in the
// actor's formation row, and in the rows more rows hurt after it (a Grove
// covers one or two). An actor outside the formation covers only itself.
func (a ScriptActor) RowAllies(rows int) []*ScriptActor {
	c := a.characterRecord
	self := []*ScriptActor{GetActor(a.userId, a.mobInstanceId)}
	if c == nil {
		return nil
	}
	leader := a.userId
	if leader == 0 {
		leader = c.GetCharmedUserId()
	}
	f, ok := company.FormationFor(leader)
	if !ok {
		return self
	}
	type member struct {
		actor *ScriptActor
		row   int
		hurt  bool
	}
	var members []member
	add := func(x *ScriptActor, key company.MemberKey) {
		if x == nil || x.characterRecord == nil || x.characterRecord.Health < 1 || x.characterRecord.RoomId != c.RoomId {
			return
		}
		row, _, placed := f.Find(key)
		if !placed {
			row = -1
		}
		members = append(members, member{x, row, x.characterRecord.Health < x.characterRecord.HealthLimit()})
	}
	if u := users.GetByUserId(leader); u != nil {
		add(GetUser(leader), company.LeaderMemberKey)
	}
	selfRow := -1
	if room := rooms.LoadRoom(c.RoomId); room != nil {
		for _, id := range room.GetMobs(rooms.FindCharmed) {
			if l, key, ok := company.LeaderAndKeyForInstance(id); ok && l == leader && mobs.GetInstance(id) != nil {
				add(GetMob(id), key)
			}
		}
	}
	for _, m := range members {
		if m.actor.userId == a.userId && m.actor.mobInstanceId == a.mobInstanceId {
			selfRow = m.row
		}
	}
	if selfRow < 0 {
		return self
	}
	chosen := []int{selfRow}
	for len(chosen) < rows {
		best, bestHurt := -1, 0
		counts := map[int]int{}
		for _, m := range members {
			if m.row >= 0 && m.hurt {
				counts[m.row]++
			}
		}
		for r, n := range counts {
			if n > bestHurt && !slices.Contains(chosen, r) || (n == bestHurt && best >= 0 && r < best && !slices.Contains(chosen, r)) {
				best, bestHurt = r, n
			}
		}
		if best < 0 {
			break
		}
		chosen = append(chosen, best)
	}
	var out []*ScriptActor
	for _, m := range members {
		if slices.Contains(chosen, m.row) {
			out = append(out, m.actor)
		}
	}
	return out
}

// Summon calls the actor's summon (its class's Summon effect: 1 an Angel,
// 2 a Demon) into the room. False when it has none, already called it this
// battle, or has no company to call it for.
func (a ScriptActor) Summon() bool {
	if a.characterRecord == nil {
		return false
	}
	kind := summons.Angel
	switch a.characterRecord.ClassEffects().Int(classes.Summon) {
	case 1:
	case 2:
		kind = summons.Demon
	default:
		return false
	}
	_, err := summons.Call(summons.Caller{UserID: a.userId, MobID: a.mobInstanceId}, kind)
	return err == nil
}
