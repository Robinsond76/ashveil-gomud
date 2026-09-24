package camping

import "sort"

// Phase 23b whetstones. A whetstone use sharpens one company member's
// dull blades, however many they carry. Members are sharpened leader
// first, then companions by company number, until the uses run out.

// LeaderID is the leader's member ID in a sharpening plan; companions use
// their company IDs, which start at 1.
const LeaderID = 0

// Bladed reports whether a weapon subtype takes an edge.
func Bladed(subtype string) bool {
	switch subtype {
	case "slashing", "cleaving", "stabbing":
		return true
	}
	return false
}

// SharpenMember is one member's equipped bladed weapons, counted as dull
// (no edge) and sharp (an edge left).
type SharpenMember struct {
	ID      int
	Name    string
	Present bool
	Dull    int
	Sharp   int
}

// SharpenOutcome is what a pass does for one member.
type SharpenOutcome int

const (
	Sharpened    SharpenOutcome = iota // dull blades sharpened, one use spent
	AlreadySharp                       // every blade already has an edge
	NoBlade                            // no bladed weapon equipped
	NotHere                            // not present; nothing spent
	LeftOut                            // the stones ran out first
)

// SharpenEntry is one member's outcome.
type SharpenEntry struct {
	Member  SharpenMember
	Outcome SharpenOutcome
}

// SharpenPlan is an ordered pass over the company.
type SharpenPlan struct {
	Entries   []SharpenEntry
	UsesSpent int
}

// PlanSharpen orders members leader first then by ID and decides each
// one's outcome with uses whetstone uses available.
func PlanSharpen(members []SharpenMember, uses int) SharpenPlan {
	ordered := append([]SharpenMember(nil), members...)
	sort.SliceStable(ordered, func(a, b int) bool { return ordered[a].ID < ordered[b].ID })
	plan := SharpenPlan{Entries: make([]SharpenEntry, 0, len(ordered))}
	for _, member := range ordered {
		outcome := Sharpened
		switch {
		case !member.Present:
			outcome = NotHere
		case member.Dull == 0 && member.Sharp > 0:
			outcome = AlreadySharp
		case member.Dull == 0:
			outcome = NoBlade
		case plan.UsesSpent >= uses:
			outcome = LeftOut
		default:
			plan.UsesSpent++
		}
		plan.Entries = append(plan.Entries, SharpenEntry{Member: member, Outcome: outcome})
	}
	return plan
}

// Names lists, in order, the members with outcome.
func (p SharpenPlan) Names(outcome SharpenOutcome) []string {
	var names []string
	for _, e := range p.Entries {
		if e.Outcome == outcome {
			names = append(names, e.Member.Name)
		}
	}
	return names
}

// Count is how many members have outcome.
func (p SharpenPlan) Count(outcome SharpenOutcome) int {
	return len(p.Names(outcome))
}
