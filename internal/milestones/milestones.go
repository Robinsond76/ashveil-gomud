// Package milestones describes upcoming progression choices without granting them.
package milestones

import "fmt"

type Milestone struct {
	Level   int
	Text    string
	Shipped bool
}

// Flip Shipped only in the phase that delivers the corresponding choice.
var schedule = [...]Milestone{
	{3, "second class option", false},
	{5, "talent", false},
	{10, "class promotion", false},
	{15, "talent", false},
	{20, "advanced signature", false},
	{25, "talent", false},
	{30, "elite promotion", false},
}

func Next(level int) Milestone {
	for _, m := range schedule {
		if m.Level > level {
			if !m.Shipped {
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
