package strategy

// Phase 30c2: the guardian. A guardian fights as a fighter, and steps in
// to take a blow meant for its ward: the member its strategy names
// (Strategy.Ward), or, with none, the most hurt member it can reach. The
// guard itself (counts, the blow) is the combat round's; this is the
// choice of ward, over plain values.

// Guarded is one living member of the guardian's company here, in
// formation order, as a guardian sees it.
type Guarded struct {
	Key       string // its member key
	HP, MaxHP int
	InReach   bool // within one column of the guardian (GuardReach)
}

// GuardWard is whom guardian guards now among members: its set ward when
// that member is here and in reach, else, with no ward set or the ward
// gone, the most hurt member in reach by health fraction (ties by order),
// and only someone below full health. ok is false when it guards no one.
// A guardian never guards itself.
func GuardWard(guardian, ward string, members []Guarded) (string, bool) {
	if ward != "" && ward != guardian {
		for _, m := range members {
			if m.Key == ward {
				if !m.InReach || m.HP < 1 {
					return "", false
				}
				return m.Key, true
			}
		}
	}
	if ward == guardian {
		return "", false
	}
	best := -1
	for i, m := range members {
		if m.Key == guardian || !m.InReach || m.HP < 1 || m.MaxHP < 1 || m.HP >= m.MaxHP {
			continue
		}
		if best < 0 || fraction(m.HP, m.MaxHP) < fraction(members[best].HP, members[best].MaxHP) {
			best = i
		}
	}
	if best < 0 {
		return "", false
	}
	return members[best].Key, true
}
