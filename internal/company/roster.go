package company

// Phase 32a2: per-player recruit rosters. Each leader has their own
// generated candidates at each settlement recruiter; each stays a random
// while, then leaves and another takes the slot. Rosters are brought up
// to date when read, from the world round, which is only ever read here.

import (
	"strings"
)

// Candidate is one generated recruit on a leader's roster.
type Candidate struct {
	// Key is the given name, lower case: the word players type.
	Key           string `yaml:"key"`
	Name          string `yaml:"name"`
	Archetype     string `yaml:"archetype"`
	MobTemplateID int    `yaml:"mob_template_id"`
	Level         int    `yaml:"level"`
	// Alignment is on the engine scale, like Disposition.Alignment.
	Alignment int    `yaml:"alignment"`
	Price     int    `yaml:"price"`
	Trait     string `yaml:"trait,omitempty"`
	// Arrived and Leaves are world rounds.
	Arrived uint64 `yaml:"arrived"`
	Leaves  uint64 `yaml:"leaves"`
}

// Roster is one leader's generated candidates at one recruiter room.
type Roster struct {
	RoomID     int         `yaml:"room_id"`
	Candidates []Candidate `yaml:"candidates,omitempty"`
	// Openings are the rounds at which slots emptied by a hire fill again.
	Openings []uint64 `yaml:"openings,omitempty"`
}

func (r Roster) clone() Roster {
	r.Candidates = append([]Candidate(nil), r.Candidates...)
	r.Openings = append([]uint64(nil), r.Openings...)
	return r
}

// Find matches a candidate by key, then exact name, then a unique name
// substring, ignoring case.
func (r Roster) Find(selector string) (Candidate, bool) {
	selector = strings.ToLower(strings.TrimSpace(selector))
	if selector == "" {
		return Candidate{}, false
	}
	for _, c := range r.Candidates {
		if c.Key == selector || strings.ToLower(c.Name) == selector {
			return c, true
		}
	}
	var partial []Candidate
	for _, c := range r.Candidates {
		if strings.Contains(strings.ToLower(c.Name), selector) {
			partial = append(partial, c)
		}
	}
	if len(partial) == 1 {
		return partial[0], true
	}
	return Candidate{}, false
}

// RosterArchetype is one archetype a roster can generate.
type RosterArchetype struct {
	Archetype     string
	MobTemplateID int
	Weight        int
	// PricePercent scales the price (100 when 0).
	PricePercent int
}

// RosterRules are the roster knobs, all in world rounds and gold.
type RosterRules struct {
	Size                 int
	StayMin, StayMax     uint64
	RefillMin, RefillMax uint64
	PriceBase            int
	PricePerLevel        int
	// AlignmentMin and AlignmentMax bound a candidate's alignment.
	AlignmentMin, AlignmentMax int
	// BynamePercent is the chance a name gets a byname ("Hild Marrow").
	BynamePercent int
	Archetypes    []RosterArchetype
	GivenNames    []string
	Bynames       []string
	Traits        []string
}

// RosterContext is what a refresh needs from the world.
type RosterContext struct {
	Now         uint64
	LeaderLevel int
	// Taken are given names (lower case) not to reuse: the company's and
	// the recruiter's regulars'.
	Taken map[string]bool
	// Weights overrides the rules' archetype weights by archetype, when
	// set (a recruiter's own mix).
	Weights map[string]int
}

// Rand is the randomness a roster uses: Intn returns [0, n).
type Rand interface{ Intn(n int) int }

func between(rng Rand, lo, hi uint64) uint64 {
	if hi <= lo {
		return lo
	}
	return lo + uint64(rng.Intn(int(hi-lo+1)))
}

// RefreshRoster brings a roster up to date at ctx.Now: candidates whose
// stay is over leave, openings that are due fill, and a new roster fills
// every slot. A newcomer arrives when its slot emptied; if its stay would
// already be over, it is part way through a stay instead, so a long
// absence costs one step per slot. A new roster's candidates are part way
// through their stays too, so faces leave one at a time. changed reports
// whether anything changed. It never reads or advances the clock itself.
func RefreshRoster(r Roster, rules RosterRules, ctx RosterContext, rng Rand) (Roster, bool) {
	out := r.clone()
	changed := false
	var kept []Candidate
	var vacated []uint64
	for _, c := range out.Candidates {
		if c.Leaves <= ctx.Now {
			vacated = append(vacated, c.Leaves)
			changed = true
			continue
		}
		kept = append(kept, c)
	}
	var pending []uint64
	for _, at := range out.Openings {
		if at <= ctx.Now {
			vacated = append(vacated, at)
			changed = true
			continue
		}
		pending = append(pending, at)
	}
	taken := map[string]bool{}
	for k, v := range ctx.Taken {
		if v {
			taken[strings.ToLower(k)] = true
		}
	}
	for _, c := range kept {
		taken[c.Key] = true
	}
	// A smaller RosterSize takes effect at once: the newest faces and the
	// latest openings go first.
	size := max(rules.Size, 0)
	if len(kept) > size {
		kept = kept[:size]
		changed = true
	}
	if len(kept)+len(pending) > size {
		pending = pending[:size-len(kept)]
		changed = true
	}
	slots := size - len(kept) - len(pending)
	for i := 0; i < slots; i++ {
		c, ok := generateCandidate(rules, ctx, taken, rng)
		if !ok {
			break
		}
		stay := between(rng, rules.StayMin, rules.StayMax)
		if stay < 1 {
			stay = 1
		}
		fresh := i >= len(vacated)
		var arrived uint64
		if !fresh {
			arrived = vacated[i]
		}
		if fresh || arrived+stay <= ctx.Now {
			// Part way through its stay: at least half the shortest stay
			// still to go, so no one leaves the moment they're seen.
			floor := min(rules.StayMin/2, stay)
			remaining := between(rng, max(floor, 1), stay)
			c.Leaves = ctx.Now + remaining
			if ctx.Now+remaining >= stay {
				c.Arrived = ctx.Now + remaining - stay
			}
		} else {
			c.Arrived = arrived
			c.Leaves = arrived + stay
		}
		taken[c.Key] = true
		kept = append(kept, c)
		changed = true
	}
	out.Candidates = kept
	out.Openings = pending
	return out, changed
}

// HireFromRoster takes a candidate off the roster and opens its slot to
// fill again after the refill delay.
func HireFromRoster(r Roster, key string, rules RosterRules, now uint64, rng Rand) (Roster, bool) {
	out := r.clone()
	for i, c := range out.Candidates {
		if c.Key != key {
			continue
		}
		out.Candidates = append(out.Candidates[:i], out.Candidates[i+1:]...)
		out.Openings = append(out.Openings, now+max(between(rng, rules.RefillMin, rules.RefillMax), 1))
		return out, true
	}
	return r, false
}

// CandidatePrice is what a candidate of this level and archetype asks.
func CandidatePrice(rules RosterRules, a RosterArchetype, level int) int {
	percent := a.PricePercent
	if percent <= 0 {
		percent = 100
	}
	price := (rules.PriceBase + rules.PricePerLevel*level) * percent / 100
	return max(price, 0)
}

func generateCandidate(rules RosterRules, ctx RosterContext, taken map[string]bool, rng Rand) (Candidate, bool) {
	archetype, ok := pickArchetype(rules, ctx, rng)
	if !ok {
		return Candidate{}, false
	}
	var names []string
	for _, n := range rules.GivenNames {
		n = strings.TrimSpace(n)
		if n != "" && !taken[strings.ToLower(n)] {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		return Candidate{}, false
	}
	given := names[rng.Intn(len(names))]
	name := given
	if len(rules.Bynames) > 0 && rng.Intn(100) < rules.BynamePercent {
		if by := strings.TrimSpace(rules.Bynames[rng.Intn(len(rules.Bynames))]); by != "" {
			name = given + " " + by
		}
	}
	level := max(ctx.LeaderLevel+rng.Intn(3)-1, 1)
	alignment := rules.AlignmentMin
	if rules.AlignmentMax > rules.AlignmentMin {
		alignment += rng.Intn(rules.AlignmentMax - rules.AlignmentMin + 1)
	}
	trait := ""
	if len(rules.Traits) > 0 {
		trait = strings.TrimSpace(rules.Traits[rng.Intn(len(rules.Traits))])
	}
	return Candidate{
		Key:           strings.ToLower(given),
		Name:          name,
		Archetype:     archetype.Archetype,
		MobTemplateID: archetype.MobTemplateID,
		Level:         level,
		Alignment:     ClampAlignment(alignment),
		Price:         CandidatePrice(rules, archetype, level),
		Trait:         trait,
	}, true
}

func pickArchetype(rules RosterRules, ctx RosterContext, rng Rand) (RosterArchetype, bool) {
	weight := func(a RosterArchetype) int {
		if ctx.Weights != nil {
			return ctx.Weights[a.Archetype]
		}
		return a.Weight
	}
	total := 0
	for _, a := range rules.Archetypes {
		if a.MobTemplateID > 0 && weight(a) > 0 {
			total += weight(a)
		}
	}
	if total == 0 {
		return RosterArchetype{}, false
	}
	roll := rng.Intn(total)
	for _, a := range rules.Archetypes {
		if a.MobTemplateID <= 0 || weight(a) <= 0 {
			continue
		}
		if roll < weight(a) {
			return a, true
		}
		roll -= weight(a)
	}
	return RosterArchetype{}, false
}

// Roster is the leader's roster at a recruiter room, if any.
func (r Record) Roster(roomID int) (Roster, bool) {
	for _, ros := range r.Rosters {
		if ros.RoomID == roomID {
			return ros.clone(), true
		}
	}
	return Roster{}, false
}

// SetRoster stores the leader's roster at its room, replacing any there.
func (r *Record) SetRoster(roster Roster) {
	for i, ros := range r.Rosters {
		if ros.RoomID == roster.RoomID {
			r.Rosters[i] = roster.clone()
			return
		}
	}
	r.Rosters = append(r.Rosters, roster.clone())
}

// PutRoster stores a leader's roster, creating the record if needed.
func (r *Registry) PutRoster(leaderUserID int, roster Roster) error {
	if leaderUserID <= 0 {
		return ErrInvalidLeader
	}
	record, _ := r.Get(leaderUserID)
	record.LeaderUserID = leaderUserID
	record.SetRoster(roster)
	r.Put(record)
	return nil
}
