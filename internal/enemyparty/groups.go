package enemyparty

import (
	"sort"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
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
		members := make([]mobparty.MobSummary, len(p.Members))
		for i, id := range p.Members {
			members[i] = byId[id]
		}
		n := mobparty.NameGroup(members)
		name := n.Name
		if !n.Solo {
			seen[n.Name]++
			name = mobparty.WithOrdinal(n.Name, seen[n.Name])
		}
		out = append(out, Group{Party: p, Naming: n, Name: name})
	}
	return out
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
	if g, ok := pick(func(g Group) bool { return !g.Solo() && g.Naming.Matches(name) }); ok {
		return g, true
	}
	// A lone mob, by its name as the room matches names.
	if _, mobId := room.FindByName(search); mobId > 0 {
		for _, g := range groups {
			if g.Solo() && g.Party.Members[0] == mobId {
				return g, true
			}
		}
		return Group{}, false // a member of a larger group: not a group's name
	}
	return pick(func(g Group) bool { return !g.Solo() && g.Naming.MatchesPrefix(name) })
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

// FirstAim is the member of g a player strikes first (Phase 32c, the
// owner's rule 6: a character's strategy sets its target, and if it can't
// reach that one it attacks one it can reach, by formation). Until 32d adds
// strategies, the one strategy is 29a's: the weakest living, visible member
// the player can reach from their place in their company's formation; a
// player with no place may strike anyone. With no one in reach, the
// frontmost member (a blow at the back is caught by the front, 11c).
func FirstAim(g Group, userId int, c *characters.Character) (int, bool) {
	alive := Alive(g.Party)
	col, placed := 0, false
	if f, ok := company.FormationFor(userId); ok {
		_, col, placed = f.Find(company.LeaderMemberKey)
	}
	reach := combat.ResolveReach(c, false)

	type cand struct {
		id, hp, row, col int
	}
	var cands []cand
	for _, id := range g.Party.Members {
		m := mobs.GetInstance(id)
		if m == nil || m.Character.Health < 1 || m.Character.HasBuffFlag("hidden") {
			continue
		}
		row, mcol, _ := g.Party.Formation.Find(mobparty.MemberKeyFor(id))
		cands = append(cands, cand{id: id, hp: m.Character.Health, row: row, col: mcol})
	}
	if len(cands) == 0 {
		return 0, false
	}
	// By formation: front row first, then left to right; the weakest wins.
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].row != cands[j].row {
			return cands[i].row < cands[j].row
		}
		return cands[i].col < cands[j].col
	})
	best := -1
	for i, cd := range cands {
		if placed && !formationcombat.Legal(col, g.Party.Formation, mobparty.MemberKeyFor(cd.id), alive, reach) {
			continue
		}
		if best < 0 || cd.hp < cands[best].hp {
			best = i
		}
	}
	if best < 0 {
		return cands[0].id, true
	}
	return cands[best].id, true
}
