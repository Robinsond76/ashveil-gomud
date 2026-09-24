package camping

import "time"

// Tier is a rest bonus tier (Phase 23a). Higher tiers are better: a camp
// rest gives Rested, an inn stay Well Rested.
type Tier int

const (
	TierNone Tier = iota
	TierRested
	TierWellRested
)

// Valid reports a grantable tier.
func (t Tier) Valid() bool { return t == TierRested || t == TierWellRested }

func (t Tier) String() string {
	switch t {
	case TierRested:
		return "Rested"
	case TierWellRested:
		return "Well Rested"
	}
	return "none"
}

// Decide applies tier exclusivity to one member: whether granting grant to
// a member holding held gives anything, and which lower tiers it removes.
// A higher tier is never downgraded; the same tier refreshes.
func Decide(held, grant Tier) (bool, []Tier) {
	if !grant.Valid() || held > grant {
		return false, nil
	}
	var remove []Tier
	for t := TierRested; t < grant; t++ {
		if held == t {
			remove = append(remove, t)
		}
	}
	return true, remove
}

// OwedGrant is a tier owed to a companion that had no live mob when its
// company's rest bonus was granted.
type OwedGrant struct {
	BuffID       int       `yaml:"buff_id"`
	Tier         Tier      `yaml:"tier"`
	ExpiresAtUTC time.Time `yaml:"expires_at_utc"`
}

// Expired reports whether the tier would no longer be active at now.
func (o OwedGrant) Expired(now time.Time) bool {
	return !now.UTC().Before(o.ExpiresAtUTC)
}

// Valid rejects an entry that could not have come from a grant.
func (o OwedGrant) Valid() bool {
	return o.BuffID > 0 && o.Tier.Valid() && !o.ExpiresAtUTC.IsZero()
}

// MergeOwed returns the entry a companion is owed after a new grant: the
// new one, unless an unexpired old one is a higher tier.
func MergeOwed(old OwedGrant, hadOld bool, next OwedGrant, now time.Time) OwedGrant {
	if hadOld && !old.Expired(now) && old.Tier > next.Tier {
		return old
	}
	return next
}

const defaultRoundSeconds = 4

// RoundsFor converts a duration to whole rounds, rounding up, at least one.
func RoundsFor(d time.Duration, roundSeconds int) int {
	if roundSeconds < 1 {
		roundSeconds = defaultRoundSeconds
	}
	per := time.Duration(roundSeconds) * time.Second
	rounds := int((d + per - 1) / per)
	if rounds < 1 {
		return 1
	}
	return rounds
}

// RemainingRounds is the rounds left until expires, rounded up, or 0 once
// expired.
func RemainingRounds(expires, now time.Time, roundSeconds int) int {
	left := expires.Sub(now.UTC())
	if left <= 0 {
		return 0
	}
	return RoundsFor(left, roundSeconds)
}
