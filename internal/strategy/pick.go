package strategy

import "sort"

// Foe is one living, visible member of the enemy group a character is in
// battle with.
type Foe struct {
	ID        int
	HP, MaxHP int
	Row, Col  int  // its place in the group's formation; row 0 is the front
	Reachable bool // the character can strike it from where it stands (11c)
	Leader    bool // the group's leader: its toughest, placed first
	// StrikesPct is the health, in percent, of the member of our side it
	// is striking, or -1 when it strikes none of us.
	StrikesPct int
	// Chanting is a foe chanting a spell now; Caster one that casts
	// (knows spells, or, for a company member, a healer or caster role).
	// The casters rule (Phase 30c).
	Chanting, Caster bool
}

// Pick chooses a target by rule (the owner's rule 6). The rule chooses
// among the foes the character can reach (every foe, for a spell:
// ignoreReach). When the rule's own choice can't be made among them (the
// leader, the player's target, or a defended ally's attacker is out of
// reach; nobody is striking us), it takes the nearest foe it can reach.
// With none in reach it takes the front-most foe: a blow at the back is
// caught by the front (11c). assistID is the player's target, for Assist.
// ok is false only when there are no foes.
func Pick(rule Rule, foes []Foe, assistID int, ignoreReach bool) (int, bool) {
	if len(foes) == 0 {
		return 0, false
	}
	ordered := make([]Foe, len(foes))
	copy(ordered, foes)
	byFormation(ordered)

	pool := ordered
	if !ignoreReach {
		pool = nil
		for _, f := range ordered {
			if f.Reachable {
				pool = append(pool, f)
			}
		}
		if len(pool) == 0 {
			return ordered[0].ID, true
		}
	}
	if id, ok := choose(rule, pool, assistID); ok {
		return id, true
	}
	return pool[0].ID, true // the nearest
}

// byFormation orders foes front row first, then left to right.
func byFormation(fs []Foe) {
	sort.SliceStable(fs, func(i, j int) bool {
		if fs[i].Row != fs[j].Row {
			return fs[i].Row < fs[j].Row
		}
		return fs[i].Col < fs[j].Col
	})
}

// Choose is the rule's own choice among pool, with no fallback: ok is false
// when the rule has none (no leader, no player's target, nobody striking
// us among them). Ties go front row first, then left.
func Choose(rule Rule, pool []Foe, assistID int) (int, bool) {
	if len(pool) == 0 {
		return 0, false
	}
	ordered := make([]Foe, len(pool))
	copy(ordered, pool)
	byFormation(ordered)
	return choose(rule, ordered, assistID)
}

// choose applies a rule to foes already in formation order; the first of
// equals wins, so ties go front row first, then left.
func choose(rule Rule, pool []Foe, assistID int) (int, bool) {
	best := func(better func(a, b Foe) bool) (int, bool) {
		b := -1
		for i, f := range pool {
			if b < 0 || better(f, pool[b]) {
				b = i
			}
		}
		if b < 0 {
			return 0, false
		}
		return pool[b].ID, true
	}
	switch rule {
	case Strongest:
		return best(func(a, b Foe) bool { return a.HP > b.HP })
	case Wounded:
		return best(func(a, b Foe) bool { return fraction(a.HP, a.MaxHP) < fraction(b.HP, b.MaxHP) })
	case Nearest:
		return pool[0].ID, true
	case Furthest:
		return best(func(a, b Foe) bool { return a.Row > b.Row })
	case Leader:
		for _, f := range pool {
			if f.Leader {
				return f.ID, true
			}
		}
		return 0, false
	case Casters:
		// Casters first (Phase 35d): a chanting caster, else any caster,
		// else the weakest foe.
		for _, f := range pool {
			if f.Chanting {
				return f.ID, true
			}
		}
		for _, f := range pool {
			if f.Caster {
				return f.ID, true
			}
		}
		return best(func(a, b Foe) bool { return a.HP < b.HP })
	case Assist:
		for _, f := range pool {
			if assistID > 0 && f.ID == assistID {
				return f.ID, true
			}
		}
		return 0, false
	case Defend:
		b := -1
		for i, f := range pool {
			if f.StrikesPct < 0 {
				continue
			}
			if b < 0 || f.StrikesPct < pool[b].StrikesPct {
				b = i
			}
		}
		if b < 0 {
			return 0, false
		}
		return pool[b].ID, true
	}
	// Weakest, and anything unknown.
	return best(func(a, b Foe) bool { return a.HP < b.HP })
}

// fraction is health in thousandths of the maximum.
func fraction(hp, max int) int {
	if max < 1 {
		max = 1
	}
	return hp * 1000 / max
}

// Percent is health as a whole percentage of the maximum.
func Percent(hp, max int) int {
	if max < 1 {
		max = 1
	}
	return hp * 100 / max
}
