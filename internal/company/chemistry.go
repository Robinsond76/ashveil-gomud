package company

// Phase 24: company chemistry. The longer a band stays together, the better
// it fights. Each member keeps a durable count of the rounds it has served
// with the band; a band's tier comes from the average service of the
// members together, so a new recruit dilutes it until they settle in.

// Service is one member's accumulated time with the band.
type Service struct {
	Member MemberKey `yaml:"member"`
	Rounds int       `yaml:"rounds"`
	// LastRound is the last global round charged to this member, so a round
	// is counted once however many times it is seen.
	LastRound uint64 `yaml:"last_round"`
	// Saved is Rounds as of the last successful company save (or load).
	// Tiers are worked out from it only, so no tier is used or shown before
	// it is on disk.
	Saved int `yaml:"-"`
}

// Tier numbers: 0 is no tier.
const (
	TierNone = iota
	TierFamiliar
	TierTrusted
	TierSworn
)

// counterResetRounds is how far the round counter must go back before a
// member stops waiting for it to catch up (a reset counter file).
const counterResetRounds = 900

// ChemistryRules are the tier thresholds (in average rounds served) and hit
// bonuses (in percentage points), Familiar, Trusted, Sworn.
type ChemistryRules struct {
	TierRounds [3]int
	TierBonus  [3]int
}

// DefaultChemistryRules are 1, 3, and 7 game days of 900 rounds, for +2,
// +4, and +6 points.
func DefaultChemistryRules() ChemistryRules {
	return ChemistryRules{TierRounds: [3]int{900, 2700, 6300}, TierBonus: [3]int{2, 4, 6}}
}

// Valid reports whether the thresholds strictly increase from at least 1
// and the bonuses are 0..10 and never decrease.
func (r ChemistryRules) Valid() bool {
	prevRounds, prevBonus := 0, 0
	for i := range r.TierRounds {
		if r.TierRounds[i] <= prevRounds || r.TierBonus[i] < prevBonus || r.TierBonus[i] > 10 {
			return false
		}
		prevRounds, prevBonus = r.TierRounds[i], r.TierBonus[i]
	}
	return true
}

// Tier is the tier reached at rounds of (average) service.
func (r ChemistryRules) Tier(rounds int) int {
	tier := TierNone
	for i, threshold := range r.TierRounds {
		if rounds >= threshold {
			tier = i + 1
		}
	}
	return tier
}

// Bonus is a tier's hit bonus in percentage points.
func (r ChemistryRules) Bonus(tier int) int {
	if tier < TierFamiliar || tier > TierSworn {
		return 0
	}
	return r.TierBonus[tier-1]
}

// Progress is the next tier after rounds and the whole percent of the way
// there from the current tier. At the top tier next is TierNone.
func (r ChemistryRules) Progress(rounds int) (next, percent int) {
	tier := r.Tier(rounds)
	if tier == TierSworn {
		return TierNone, 100
	}
	from := 0
	if tier > TierNone {
		from = r.TierRounds[tier-1]
	}
	to := r.TierRounds[tier]
	return tier + 1, (rounds - from) * 100 / (to - from)
}

// TierName is a tier's name for players.
func TierName(tier int) string {
	switch tier {
	case TierFamiliar:
		return "Familiar"
	case TierTrusted:
		return "Trusted"
	case TierSworn:
		return "Sworn"
	}
	return "Strangers"
}

// chargeable reports whether round has not yet been charged.
func (s Service) chargeable(round uint64) bool {
	return round > s.LastRound || s.LastRound-round > counterResetRounds
}

// ChargeService adds one round of service for member, creating its entry at
// 0 if it has none. charged is false when round was already charged.
func (r *Record) ChargeService(member MemberKey, round uint64) (charged bool) {
	for i := range r.Service {
		s := &r.Service[i]
		if s.Member != member {
			continue
		}
		if !s.chargeable(round) {
			return false
		}
		s.Rounds++
		s.LastRound = round
		return true
	}
	r.Service = append(r.Service, Service{Member: member, Rounds: 1, LastRound: round})
	return true
}

// FindService returns member's service entry.
func (r Record) FindService(member MemberKey) (Service, bool) {
	for _, s := range r.Service {
		if s.Member == member {
			return s, true
		}
	}
	return Service{}, false
}

// MarkServiceSaved records that every member's rounds are on disk.
func (r *Record) MarkServiceSaved() {
	for i := range r.Service {
		r.Service[i].Saved = r.Service[i].Rounds
	}
}

// BandAverage is the average service of members together, from saved
// rounds (durable is true) or current ones; a member with no entry counts
// as 0. ok is false for fewer than two members: a lone member is no band.
func (r Record) BandAverage(members []MemberKey, durable bool) (average int, ok bool) {
	if len(members) < 2 {
		return 0, false
	}
	total := 0
	for _, member := range members {
		if s, found := r.FindService(member); found {
			if durable {
				total += s.Saved
			} else {
				total += s.Rounds
			}
		}
	}
	return total / len(members), true
}

// BandTier is the tier of members together, from saved rounds.
func (r Record) BandTier(members []MemberKey, rules ChemistryRules) int {
	average, ok := r.BandAverage(members, true)
	if !ok {
		return TierNone
	}
	return rules.Tier(average)
}

// pruneService drops entries for members no longer in the record and
// normalizes the rest: rounds at least 0, one entry per member (a duplicate
// merges into the first, keeping the most rounds and the latest round).
func pruneService(service []Service, valid map[MemberKey]bool) []Service {
	var kept []Service
	index := map[MemberKey]int{}
	for _, s := range service {
		if !valid[s.Member] {
			continue
		}
		s.Rounds = max(s.Rounds, 0)
		s.Saved = min(max(s.Saved, 0), s.Rounds)
		if i, dup := index[s.Member]; dup {
			kept[i].Rounds = max(kept[i].Rounds, s.Rounds)
			kept[i].Saved = max(kept[i].Saved, s.Saved)
			kept[i].LastRound = max(kept[i].LastRound, s.LastRound)
			continue
		}
		index[s.Member] = len(kept)
		kept = append(kept, s)
	}
	return kept
}
