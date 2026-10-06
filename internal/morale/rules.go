// Package morale contains deterministic morale rules; adapters own runtime state.
package morale

import "fmt"

type Outcome uint8

const (
	Hold Outcome = iota
	Yield
	Flee
	Hesitate
)

// ValidTemperament accepts blank for backward-compatible unbreakable behavior.
func ValidTemperament(s string) bool {
	switch s {
	case "", "unbreakable", "steadfast", "wary", "craven", "skittish":
		return true
	}
	return false
}
func Validate(s string) error {
	if !ValidTemperament(s) {
		return fmt.Errorf("unknown morale temperament %q", s)
	}
	return nil
}
func Enemy(s string, roll int) Outcome {
	hold, yield := 100, 0
	switch s {
	case "steadfast":
		hold, yield = 85, 10
	case "wary":
		hold, yield = 65, 25
	case "craven":
		hold, yield = 40, 40
	case "skittish":
		hold, yield = 65, 0
	}
	if roll < 0 || roll >= 100 || roll < hold {
		return Hold
	}
	if roll < hold+yield {
		return Yield
	}
	return Flee
}

// EnemyHunted is Enemy for a foe a hunter has struck (Phase 38c2): its flee
// band shrinks by penalty points but never below 5 when it could flee at all,
// and what is taken off the band becomes Hold.
func EnemyHunted(s string, roll, penalty int) Outcome {
	out := Enemy(s, roll)
	if out != Flee || roll < 0 || roll >= 100 {
		return out
	}
	fleeBand := 0
	for r := 0; r < 100; r++ {
		if Enemy(s, r) == Flee {
			fleeBand++
		}
	}
	keep := max(5, fleeBand-penalty)
	if roll >= 100-keep {
		return Flee
	}
	return Hold
}

// Group tracks consumed triggers. Departures are not deaths.
type Group struct {
	Leader, Specialist, Size int
	Seen                     uint8
}
type Member struct {
	ID, HP, Max  int
	Dead, Active bool
}

func (g *Group) Check(ms []Member) bool {
	var triggers uint8
	dead, active := 0, 0
	var last Member
	for _, m := range ms {
		if m.Dead {
			dead++
			if m.ID == g.Leader {
				triggers |= 1
			}
			if m.ID == g.Specialist {
				triggers |= 2
			}
		}
		if m.Active && m.HP > 0 {
			active++
			last = m
		}
	}
	if g.Size > 0 && dead*2 >= g.Size {
		triggers |= 4
	}
	if active == 1 && last.Max > 0 && last.HP*4 <= last.Max {
		triggers |= 8
	}
	fresh := triggers &^ g.Seen
	g.Seen |= triggers
	return fresh != 0
}
func Losing(start, incapacitated, hp, maximum int) bool {
	return start > 0 && (incapacitated*2 >= start || maximum > 0 && hp*4 <= maximum)
}
func Nerve(loyalty int, weakChemistry bool, roll int) Outcome {
	if roll < 0 || roll >= 100 {
		return Hold
	}
	if loyalty < 25 {
		if roll < 20 {
			return Hesitate
		}
		if roll < 25 {
			return Flee
		}
	} else if weakChemistry && roll < 10 {
		return Hesitate
	}
	return Hold
}
func Reaction(alignment int, spare bool) int {
	if alignment >= 25 {
		if spare {
			return 2
		}
		return -1
	}
	if alignment <= -25 {
		if !spare {
			return 2
		}
		return -1
	}
	return 0
}
