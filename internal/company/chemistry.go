package company

// Phase 24: company chemistry. Members who serve together long enough fight
// better together. A bond is kept per unordered pair of member keys and
// counts the rounds the pair spent eligible together.

// Bond is one pair's accumulated shared service. A is the lesser key.
type Bond struct {
	A      MemberKey `yaml:"a"`
	B      MemberKey `yaml:"b"`
	Rounds int       `yaml:"rounds"`
	// LastRound is the last global round charged to this bond, so a round
	// is counted once however many times it is seen.
	LastRound uint64 `yaml:"last_round"`
}

// Tier numbers: 0 is no tier.
const (
	TierNone = iota
	TierFamiliar
	TierTrusted
	TierSworn
)

// counterResetRounds is how far the round counter must go back before a
// bond stops waiting for it to catch up (a reset counter file).
const counterResetRounds = 900

// ChemistryRules are the tier thresholds (in shared rounds) and hit bonuses
// (in percentage points), Familiar, Trusted, Sworn.
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

// Tier is the tier reached after rounds of shared service.
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

// BondPair orders two keys the way a Bond stores them.
func BondPair(a, b MemberKey) (MemberKey, MemberKey) {
	if b < a {
		return b, a
	}
	return a, b
}

// Partner is the other member of a bond involving key.
func (b Bond) Partner(key MemberKey) (MemberKey, bool) {
	switch key {
	case b.A:
		return b.B, true
	case b.B:
		return b.A, true
	}
	return "", false
}

// chargeable reports whether round has not yet been charged.
func (b Bond) chargeable(round uint64) bool {
	return round > b.LastRound || b.LastRound-round > counterResetRounds
}

// ChargeBond adds one shared round for the pair, creating its bond at 0 if
// it has none. It returns the rounds before and after; charged is false when
// round was already charged.
func (r *Record) ChargeBond(a, b MemberKey, round uint64) (before, after int, charged bool) {
	a, b = BondPair(a, b)
	if a == b {
		return 0, 0, false
	}
	for i := range r.Bonds {
		bond := &r.Bonds[i]
		if bond.A != a || bond.B != b {
			continue
		}
		if !bond.chargeable(round) {
			return bond.Rounds, bond.Rounds, false
		}
		before = bond.Rounds
		bond.Rounds++
		bond.LastRound = round
		return before, bond.Rounds, true
	}
	r.Bonds = append(r.Bonds, Bond{A: a, B: b, Rounds: 1, LastRound: round})
	return 0, 1, true
}

// FindBond returns the pair's bond.
func (r Record) FindBond(a, b MemberKey) (Bond, bool) {
	a, b = BondPair(a, b)
	for _, bond := range r.Bonds {
		if bond.A == a && bond.B == b {
			return bond, true
		}
	}
	return Bond{}, false
}

// BestBond is key's highest-tier bond whose partner passes present (all
// partners when present is nil). Ties go to the most rounds. ok is false
// when key has no such bond.
func BestBond(bonds []Bond, key MemberKey, present func(MemberKey) bool, rules ChemistryRules) (best Bond, tier int, ok bool) {
	for _, bond := range bonds {
		partner, involved := bond.Partner(key)
		if !involved || (present != nil && !present(partner)) {
			continue
		}
		t := rules.Tier(bond.Rounds)
		if !ok || t > tier || (t == tier && bond.Rounds > best.Rounds) {
			best, tier, ok = bond, t, true
		}
	}
	return best, tier, ok
}

// pruneBonds drops bonds whose members are not both valid.
func pruneBonds(bonds []Bond, valid map[MemberKey]bool) []Bond {
	var kept []Bond
	for _, bond := range bonds {
		if valid[bond.A] && valid[bond.B] && bond.A != bond.B {
			kept = append(kept, bond)
		}
	}
	return kept
}
