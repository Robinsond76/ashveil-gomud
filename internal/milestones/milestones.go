// Package milestones describes upcoming progression choices without granting them.
package milestones

import "fmt"

type Milestone struct {
	Level   int
	Text    string
	Shipped bool
	// ShippedFor lists the archetypes for which the milestone is already
	// delivered when Shipped is false (Phase 35b: casters' level-3 spell).
	ShippedFor []string
}

// Flip Shipped only in the phase that delivers the corresponding choice.
var schedule = [...]Milestone{
	{3, "second class option", false, []string{"wizard", "cleric"}},
	{5, "talent", false, nil},
	{10, "class promotion", false, nil},
	{15, "talent", false, nil},
	{20, "advanced signature", false, nil},
	{25, "talent", false, nil},
	{30, "elite promotion", false, nil},
}

// Next is the next milestone after level for a character of the archetype
// ("" for none), marked "(coming)" when it isn't delivered for them yet.
func Next(level int, archetype string) Milestone {
	for _, m := range schedule {
		if m.Level > level {
			shipped := m.Shipped
			for _, a := range m.ShippedFor {
				shipped = shipped || a == archetype
			}
			m.Shipped = shipped
			m.ShippedFor = nil
			if !shipped {
				m.Text += " (coming)"
			}
			return m
		}
	}
	return Milestone{}
}

func (m Milestone) String() string {
	if m.Level == 0 {
		return "No upcoming milestone announced."
	}
	return fmt.Sprintf("level %d: %s", m.Level, m.Text)
}
