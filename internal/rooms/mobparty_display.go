package rooms

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/mobparty"
)

// hostileMobDisplay is the already-resolved view of one hostile room mob
// that roomdetails.go's mob loop hands to groupedMobDisplay: everything
// needed to render either its own line or fold it into a party's aggregate
// line, without re-touching mobs.GetInstance or the engine.
type hostileMobDisplay struct {
	instanceId int
	groups     []string
	rawName    string // undecorated Character.Name, used for the mixed-party check
	display    string // fully rendered mob name (colors, adjectives, quest alert, etc.)
}

// groupedMobDisplay renders hostile room mobs grouped by mobparty.Assemble:
// a party of one keeps its own individually rendered line; a multi-member
// party collapses to one aggregate line (e.g. "a pack of 3 goblins").
//
// EHP/DPS are left at zero here deliberately: internal/rooms cannot import
// internal/combat (internal/combat already imports internal/rooms, so the
// reverse would cycle), and display grouping only needs party membership,
// never the front/mid/back Formation ordering — that ordering matters once
// something actually reads Formation for combat purposes, which is out of
// scope for Phase 11a's room-display integration.
func groupedMobDisplay(entries []hostileMobDisplay) []string {
	summaries := make([]mobparty.MobSummary, len(entries))
	displayByInstance := make(map[int]string, len(entries))
	rawNameByInstance := make(map[int]string, len(entries))

	for i, e := range entries {
		summaries[i] = mobparty.MobSummary{InstanceId: e.instanceId, Groups: e.groups}
		displayByInstance[e.instanceId] = e.display
		rawNameByInstance[e.instanceId] = e.rawName
	}

	lines := make([]string, 0, len(entries))
	for _, p := range mobparty.Assemble(summaries) {
		if len(p.Members) == 1 {
			lines = append(lines, displayByInstance[p.Members[0]])
			continue
		}
		lines = append(lines, describeParty(p, rawNameByInstance))
	}

	return lines
}

// describeParty builds one aggregate line for a multi-member party. Members
// that don't all share the same base name collapse to a generic label.
//
// ponytail: naive "+s"/"+es" pluralization and a flat "a pack of N X"
// phrasing — good enough for the first grouped listing; swap in real
// pluralization/party-noun-by-race if players start noticing "wolfs".
func describeParty(p mobparty.Party, rawNameByInstance map[int]string) string {
	name := ""
	mixed := false
	for _, id := range p.Members {
		n := rawNameByInstance[id]
		if name == "" {
			name = n
		} else if n != name {
			mixed = true
		}
	}

	if mixed || name == "" {
		return fmt.Sprintf("a pack of %d creatures", len(p.Members))
	}
	return fmt.Sprintf("a pack of %d %s", len(p.Members), pluralize(name))
}

func pluralize(name string) string {
	if name == "" {
		return name
	}
	switch name[len(name)-1] {
	case 's', 'x', 'z':
		return name + "es"
	default:
		return name + "s"
	}
}
