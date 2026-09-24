package company

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
)

// Alignment and loyalty bounds. Alignment uses the engine's −100..100
// scale (characters.AlignmentMinimum..AlignmentMaximum), which is also the
// scale players see through DisplayAlignment.
const (
	MinAlignment = int(characters.AlignmentMinimum)
	MaxAlignment = int(characters.AlignmentMaximum)
	MinLoyalty   = 0
	MaxLoyalty   = 100
)

// Disposition is a companion's durable alignment and loyalty (Phase 21a).
// A companion record holds it by pointer; the pointer is replaced, never
// written through, so copies of a record stay independent.
type Disposition struct {
	Alignment int `yaml:"alignment"`
	Loyalty   int `yaml:"loyalty"`
}

func (d Disposition) clamped() Disposition {
	return Disposition{Alignment: ClampAlignment(d.Alignment), Loyalty: clampInt(d.Loyalty, MinLoyalty, MaxLoyalty)}
}

// AlignmentRules are the Phase 21a balance knobs, in alignment points
// (engine and displayed points are the same).
type AlignmentRules struct {
	DriftStep           int // points a companion moves toward the rest of the company per tick
	LoyaltyToleranceGap int // a larger gap to the rest of the company costs loyalty
	LoyaltyLoss         int
	LoyaltyGain         int
	StartLoyalty        int // a new recruit's loyalty
	LoyaltyWarnBelow    int // falling below this warns the leader
	RecruitMaxGap       int // a candidate further than this from the company average won't join
}

// DefaultAlignmentRules returns the design's defaults.
func DefaultAlignmentRules() AlignmentRules {
	return AlignmentRules{
		DriftStep:           2,
		LoyaltyToleranceGap: 60,
		LoyaltyLoss:         5,
		LoyaltyGain:         2,
		StartLoyalty:        70,
		LoyaltyWarnBelow:    25,
		RecruitMaxGap:       60,
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ClampAlignment bounds an alignment to the engine scale.
func ClampAlignment(alignment int) int {
	return clampInt(alignment, MinAlignment, MaxAlignment)
}

// DisplayAlignment is an engine alignment as players see it: −100 is most
// evil, 0 neutral, 100 most good.
func DisplayAlignment(alignment int) int {
	return ClampAlignment(alignment)
}

// AlignmentBand is the engine's name for an alignment (neutral, good, ...).
func AlignmentBand(alignment int) string {
	return characters.AlignmentToString(int8(ClampAlignment(alignment)))
}

// AverageAlignment is the mean of values rounded half away from zero, or 0
// for none.
func AverageAlignment(values []int) int {
	if len(values) == 0 {
		return 0
	}
	sum := 0
	for _, v := range values {
		sum += v
	}
	n := len(values)
	if sum >= 0 {
		return (2*sum + n) / (2 * n)
	}
	return -((-2*sum + n) / (2 * n))
}

// DriftToward moves value up to step points toward target without passing
// it, clamped to the alignment scale.
func DriftToward(value, target, step int) int {
	if step <= 0 {
		return ClampAlignment(value)
	}
	switch {
	case target > value:
		value = min(value+step, target)
	case target < value:
		value = max(value-step, target)
	}
	return ClampAlignment(value)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// MemberAlignment is one companion's alignment state for a tick.
type MemberAlignment struct {
	ID        int
	Alignment int
	Loyalty   int
}

// TickResult is the outcome of one drift tick.
type TickResult struct {
	Members   []MemberAlignment // same order as the input
	Warned    []int             // companion IDs whose loyalty fell below the warning line this tick
	Deserters []int             // companion IDs at 0 loyalty
}

// RestAverage is the average alignment of the rest of the company as seen
// by members[self]: the leader and every other companion.
func RestAverage(leader int, members []MemberAlignment, self int) int {
	values := make([]int, 0, len(members))
	values = append(values, leader)
	for i, m := range members {
		if i != self {
			values = append(values, m.Alignment)
		}
	}
	return AverageAlignment(values)
}

// TickAlignment runs one drift tick: every companion moves DriftStep (at
// most half its gap) toward the average of the rest of the company, and
// gains or loses loyalty by whether its gap to that average (before the
// drift) is within LoyaltyToleranceGap. A companion deserts when an uneasy
// tick leaves it at 0 loyalty. All targets come from the pre-tick values.
// The leader's alignment is an input only.
func TickAlignment(leader int, members []MemberAlignment, rules AlignmentRules) TickResult {
	result := TickResult{Members: make([]MemberAlignment, len(members))}
	for i, m := range members {
		target := RestAverage(leader, members, i)
		before := clampInt(m.Loyalty, MinLoyalty, MaxLoyalty)
		loyalty := before
		gap := absInt(ClampAlignment(m.Alignment) - target)
		uneasy := gap > rules.LoyaltyToleranceGap
		if uneasy {
			loyalty -= rules.LoyaltyLoss
		} else {
			loyalty += rules.LoyaltyGain
		}
		loyalty = clampInt(loyalty, MinLoyalty, MaxLoyalty)
		// Move at most half the gap, so members pulling on each other meet
		// instead of swapping places every tick.
		result.Members[i] = MemberAlignment{
			ID:        m.ID,
			Alignment: DriftToward(m.Alignment, target, min(rules.DriftStep, gap/2)),
			Loyalty:   loyalty,
		}
		switch {
		case uneasy && loyalty <= MinLoyalty:
			result.Deserters = append(result.Deserters, m.ID)
		case before >= rules.LoyaltyWarnBelow && loyalty < rules.LoyaltyWarnBelow:
			result.Warned = append(result.Warned, m.ID)
		}
	}
	return result
}

// CanRecruit reports whether a candidate's alignment is close enough to
// the company average to join.
func CanRecruit(candidate, companyAverage int, rules AlignmentRules) bool {
	return absInt(ClampAlignment(candidate)-ClampAlignment(companyAverage)) <= rules.RecruitMaxGap
}

// SetDisposition replaces a companion's disposition, clamped to range.
func (r *Registry) SetDisposition(leaderUserID, companionID int, d Disposition) error {
	record, ok := r.Get(leaderUserID)
	if !ok {
		return ErrUnknownMember
	}
	for i, c := range record.Companions {
		if c.ID != companionID {
			continue
		}
		clamped := d.clamped()
		record.Companions[i].Disposition = &clamped
		r.Put(record)
		return nil
	}
	return ErrUnknownMember
}
