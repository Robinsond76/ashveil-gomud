package rooms

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// hostileMobDisplay is the already-resolved view of one hostile room mob
// that roomdetails.go's mob loop hands to groupedMobDisplay: everything
// needed to render either its own line or fold it into its group's line,
// without re-touching mobs.GetInstance or the engine.
type hostileMobDisplay struct {
	summary mobparty.MobSummary
	display string // fully rendered mob name (colors, adjectives, quest alert, etc.)
}

// groupedMobDisplay renders hostile room mobs grouped by mobparty.Assemble
// (Phase 32c): a lone mob with no group name keeps its own name, returned
// in solo for the "Also here" list; every other group gets a line of its
// own, returned in groups: "A band of ruffians (4): two ruffians, a
// cutpurse, and a rat." doing, when set, says what a group (by its member
// ids) is doing, appended in brackets ("fighting you").
func groupedMobDisplay(entries []hostileMobDisplay, doing func(members []int) string) (solo []string, groups []string) {
	summaries := make([]mobparty.MobSummary, len(entries))
	byInstance := make(map[int]hostileMobDisplay, len(entries))
	for i, e := range entries {
		summaries[i] = e.summary
		byInstance[e.summary.InstanceId] = e
	}

	seen := map[string]int{}
	for _, p := range mobparty.Assemble(summaries) {
		members := make([]mobparty.MobSummary, len(p.Members))
		names := make([]string, len(p.Members))
		for i, id := range p.Members {
			members[i] = byInstance[id].summary
			names[i] = byInstance[id].summary.Name
		}
		n := mobparty.NameGroup(members)
		if n.Solo {
			solo = append(solo, byInstance[p.Members[0]].display)
			continue
		}
		seen[n.Name]++
		groups = append(groups, groupLine(mobparty.WithOrdinal(n.Name, seen[n.Name]), names, n, doingOf(doing, p.Members)))
	}
	return solo, groups
}

func doingOf(doing func([]int) string, members []int) string {
	if doing == nil {
		return ""
	}
	return doing(members)
}

// groupLine is one group's line in the room. A group of one kind whose
// name already says what it is ("a swarm of rats") needs no list.
func groupLine(name string, names []string, n mobparty.Naming, doing string) string {
	line := fmt.Sprintf(`<ansi fg="mobname">%s</ansi> (%d)`, mobparty.Capitalize(name), len(names))
	oneKind := true
	for _, m := range names {
		if m != names[0] {
			oneKind = false
		}
	}
	if !oneKind || n.Authored || mobparty.Plural(names[0]) != n.Kind {
		line += ": " + mobparty.ListKinds(names)
	}
	if doing != "" {
		line += " (" + doing + ")"
	}
	return line + "."
}

// GroupDoing says what a group, by its members, is doing as viewerId sees
// it: "fighting you", "fighting Brom", "waiting" (set on the viewer while
// they fight another group), or "" when it is idle.
func GroupDoing(viewerId int, members []int) string {
	has := func(b battle.Battle) bool {
		for _, id := range members {
			if b.Has(id) {
				return true
			}
		}
		return false
	}
	if b, ok := battle.Current(viewerId); ok && has(b) {
		return "fighting you"
	}
	for _, uid := range battle.Players() {
		if uid == viewerId {
			continue
		}
		if b, ok := battle.Current(uid); ok && has(b) {
			if u := users.GetByUserId(uid); u != nil {
				return "fighting " + u.Character.Name
			}
			return "fighting"
		}
	}
	_, busy := battle.Current(viewerId)
	for _, id := range members {
		m := mobs.GetInstance(id)
		if m == nil || m.Character.Aggro == nil || m.Character.Health < 1 {
			continue
		}
		if a := m.Character.Aggro; a.UserId == viewerId {
			if busy {
				return "waiting"
			}
			return "fighting you"
		} else if a.UserId > 0 {
			if u := users.GetByUserId(a.UserId); u != nil {
				return "fighting " + u.Character.Name
			}
		}
	}
	return ""
}

// DoingPhrase turns GroupDoing into a clause for a sentence: "idle",
// "fighting you", "waiting its turn".
func DoingPhrase(doing string) string {
	switch {
	case doing == "":
		return "idle"
	case doing == "waiting":
		return "waiting its turn"
	case strings.HasPrefix(doing, "fighting"):
		return doing
	}
	return doing
}
