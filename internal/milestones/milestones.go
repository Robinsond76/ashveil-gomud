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
// Phase 38b delivered talents, promotion and the advanced ranks; the elite
// step (promotion at 30, ranks and talents to 60) is delivered per lineage:
// 38c1 for warriors and clerics, 38c2 for rogues and rangers, 38c3 for
// wizards and witches.
var eliteShipped = []string{"warrior", "cleric", "rogue", "ranger", "wizard", "witch"}

var schedule = [...]Milestone{
	{3, "second class option", false, []string{"wizard", "cleric", "witch", "halberdier", "dollmaster"}},
	{5, "talent", true, nil},
	{10, "class promotion", true, nil},
	{15, "talent", true, nil},
	{20, "advanced signature", true, nil},
	{25, "talent", true, nil},
	{30, "elite promotion", false, eliteShipped},
	{35, "elite rank and talent", false, eliteShipped},
	{40, "elite rank", false, eliteShipped},
	{45, "elite rank and talent", false, eliteShipped},
	{50, "elite rank", false, eliteShipped},
	{55, "elite rank and talent", false, eliteShipped},
	{60, "elite capstone", false, eliteShipped},
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
