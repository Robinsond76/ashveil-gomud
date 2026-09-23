package archetypes

// Phase 17b utility-skill math: effective utility levels, company
// best-member resolution, skill checks, and trap-disarm expiry. Pure and
// GoMud-free; modules/archetype resolves the live members and rolls.

// MaxUtilityLevel caps an effective utility level.
const MaxUtilityLevel = SkillLevels

// CheckBaseline is added to a check's difficulty so that a d100 roll of 50
// on an evenly matched check succeeds exactly.
const CheckBaseline = 50

// PlayerUtilityLevel is a player's level (0..4) in a utility: their level
// in the utility's mapped skill, but only if their archetype performs it.
func PlayerUtilityLevel(a Archetype, chosen bool, utility string, skillLevel int) int {
	if !chosen || !a.HasUtility(utility) || skillLevel <= 0 {
		return 0
	}
	if skillLevel > MaxUtilityLevel {
		return MaxUtilityLevel
	}
	return skillLevel
}

// CompanionUtilityLevel is a companion's level (0..4) in a utility, derived
// from its archetype's companion levels and its character level.
func CompanionUtilityLevel(a Archetype, known bool, utility string, characterLevel int) int {
	if !known || !a.HasUtility(utility) {
		return 0
	}
	return a.CompanionSkillLevel(characterLevel)
}

// UtilityMember is one present company member considered for a utility.
type UtilityMember struct {
	IsLeader    bool
	CompanionID int
	Level       int
}

// BestMember picks the member with the highest utility level. Ties go to
// the leader, then to the lowest companion id, so the choice is
// deterministic. ok is false when nobody has the utility.
func BestMember(members []UtilityMember) (UtilityMember, bool) {
	var best UtilityMember
	found := false
	for _, m := range members {
		if m.Level <= 0 {
			continue
		}
		if !found || better(m, best) {
			best, found = m, true
		}
	}
	return best, found
}

func better(a, b UtilityMember) bool {
	if a.Level != b.Level {
		return a.Level > b.Level
	}
	if a.IsLeader != b.IsLeader {
		return a.IsLeader
	}
	return a.CompanionID < b.CompanionID
}

// CheckScore is a member's score for a sense or disarm check.
func CheckScore(level, perLevel, perception int) int {
	return level*perLevel + perception/4
}

// CheckMargin is how far a check beat (>= 0) or missed (< 0) its target:
// score plus a 1..100 roll against difficulty × factor plus CheckBaseline.
func CheckMargin(score, roll, difficulty, factor int) int {
	return score + roll - (difficulty*factor + CheckBaseline)
}

// DisarmedUntil is the round a trap disarmed at round now re-arms.
func DisarmedUntil(now uint64, rounds int) uint64 {
	if rounds <= 0 {
		return now
	}
	return now + uint64(rounds)
}

// TrapArmedAt reports whether a trap disarmed until round `until` is armed
// at round now. until 0 means never disarmed.
func TrapArmedAt(until, now uint64) bool {
	return until == 0 || now >= until
}
