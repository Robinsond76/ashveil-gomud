package enemyparty

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 32c: enemy groups by name. A Group is a party with its name. Players
// start a fight by naming a group (FindGroup), and the first member they
// strike is chosen for them (FirstAim).

// Group is one enemy party in a room, named.
type Group struct {
	Party  mobparty.Party
	Naming mobparty.Naming
	Name   string // as listed in the room: "a second band of ruffians" for the second of that name
}

// Solo reports whether the group is a lone mob, named by itself.
func (g Group) Solo() bool { return g.Naming.Solo }

// Groups lists the room's enemy groups in room order, named, repeats
// numbered as the room lists them.
func Groups(room *rooms.Room) []Group {
	if room == nil {
		return nil
	}
	sums := summaries(room)
	byId := make(map[int]mobparty.MobSummary, len(sums))
	for _, s := range sums {
		byId[s.InstanceId] = s
	}
	seen := map[string]int{}
	var out []Group
	for _, p := range mobparty.Assemble(sums) {
		p = groundParty(room, p)
		members := make([]mobparty.MobSummary, len(p.Members))
		for i, id := range p.Members {
			members[i] = byId[id]
		}
		n := mobparty.NameGroup(members)
		if !n.Solo && !named(members) {
			// A group that formed outside a room's spawn list (a script's or
			// an admin's spawn, a company's quarry) is named the first time
			// it is seen, and keeps the name while members fall.
			for _, id := range p.Members {
				if m := mobs.GetInstance(id); m != nil {
					m.GroupName = n.Name
				}
			}
		}
		name := n.Name
		if !n.Solo {
			seen[n.Name]++
			name = mobparty.WithOrdinal(n.Name, seen[n.Name])
		}
		out = append(out, Group{Party: p, Naming: n, Name: name})
	}
	return out
}

// named reports whether any member already carries its group's name.
func named(members []mobparty.MobSummary) bool {
	for _, m := range members {
		if m.GroupName != "" {
			return true
		}
	}
	return false
}

// GroupOf is the group holding instanceId.
func GroupOf(room *rooms.Room, instanceId int) (Group, bool) {
	for _, g := range Groups(room) {
		for _, id := range g.Party.Members {
			if id == instanceId {
				return g, true
			}
		}
	}
	return Group{}, false
}

// FindGroup finds the group search names: exactly one of its words first,
// then the start of one ("ruff"). A lone mob is named by its own name, as
// room.FindByName matches it. "ruffians#2" is the second group so named.
// A group none of whose members the viewer can see isn't found; a hidden
// member is otherwise part of its group. A member's own name never names a
// group of more than one.
func FindGroup(room *rooms.Room, search string) (Group, bool) {
	name, nth := util.GetMatchNumber(search)
	if name == "" {
		return Group{}, false
	}
	groups := visibleGroups(room)
	pick := func(match func(Group) bool) (Group, bool) {
		n := 0
		for _, g := range groups {
			if match(g) {
				n++
				if n == nth {
					return g, true
				}
			}
		}
		return Group{}, false
	}
	if g, ok := FindGroupNamed(room, search); ok {
		return g, true
	}
	// A lone mob, by its name as the room matches names.
	playerId, mobId := room.FindByName(search)
	if mobId == 0 && playerId > 0 {
		return Group{}, false // a player of that name, not a group (32c review)
	}
	if mobId > 0 {
		for _, g := range groups {
			if g.Solo() && g.Party.Members[0] == mobId {
				return g, true
			}
		}
		return Group{}, false // a member of a larger group: not a group's name
	}
	return pick(func(g Group) bool { return !g.Solo() && g.Naming.MatchesPrefix(name) })
}

// Keyword is what a player types to name g, found back by FindGroupNamed:
// its keyword, with "#2" when an earlier visible group also answers to
// that word ("a band of big rats" answers to "rats" too, 32c review). A
// lone mob goes by its own name.
func Keyword(room *rooms.Room, g Group) string {
	if g.Solo() {
		return g.Naming.Keyword
	}
	n := 0
	for _, other := range visibleGroups(room) {
		if !other.Solo() && other.Naming.Matches(g.Naming.Keyword) {
			n++
		}
		if other.Party.ID == g.Party.ID {
			break
		}
	}
	if n > 1 {
		return fmt.Sprintf(`%s#%d`, g.Naming.Keyword, n)
	}
	return g.Naming.Keyword
}

// FindGroupNamed finds a group of more than one by exactly one of its
// words ("ruffians", "band", "ruffians#2"), and nothing else: no lone mob,
// no partial word. look uses it after checking the word names no thing
// there, so a group never hides a thing of the same name.
func FindGroupNamed(room *rooms.Room, search string) (Group, bool) {
	name, nth := util.GetMatchNumber(search)
	if name == "" {
		return Group{}, false
	}
	n := 0
	for _, g := range visibleGroups(room) {
		if !g.Solo() && g.Naming.Matches(name) {
			n++
			if n == nth {
				return g, true
			}
		}
	}
	return Group{}, false
}

// Visible lists the group's living members the viewer can see (hidden ones
// are left out), in its formation order: front row first.
func (g Group) Visible() []*mobs.Mob {
	type placed struct {
		m        *mobs.Mob
		row, col int
	}
	var ps []placed
	for _, id := range g.Party.Members {
		m := mobs.GetInstance(id)
		if m == nil || m.Character.Health < 1 || m.Character.HasBuffFlag("hidden") {
			continue
		}
		row, col, _ := g.Party.Formation.Find(mobparty.MemberKeyFor(id))
		ps = append(ps, placed{m, row, col})
	}
	sort.SliceStable(ps, func(i, j int) bool {
		if ps[i].row != ps[j].row {
			return ps[i].row < ps[j].row
		}
		return ps[i].col < ps[j].col
	})
	out := make([]*mobs.Mob, len(ps))
	for i, p := range ps {
		out[i] = p.m
	}
	return out
}

// HealthWord says how hurt a fighter looks: unhurt, scratched, wounded,
// badly wounded, near death, or down.
func HealthWord(health, max int) string {
	if health < 1 {
		return "down"
	}
	if max < 1 {
		max = 1
	}
	switch pct := health * 100 / max; {
	case pct >= 100:
		return "unhurt"
	case pct >= 75:
		return "scratched"
	case pct >= 50:
		return "wounded"
	case pct >= 25:
		return "badly wounded"
	}
	return "near death"
}

// visibleGroups are the room's groups with at least one member not hidden.
func visibleGroups(room *rooms.Room) []Group {
	var out []Group
	for _, g := range Groups(room) {
		for _, id := range g.Party.Members {
			if m := mobs.GetInstance(id); m != nil && !m.Character.HasBuffFlag("hidden") {
				out = append(out, g)
				break
			}
		}
	}
	return out
}
