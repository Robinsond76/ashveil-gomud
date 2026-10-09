// Package bonds is Phase 65's rules: how two companions feel about each
// other. It is GoMud-free and pure: the value's range and its words, what
// each source of change is worth and how far it may push, the affinity of
// two temperaments, and what a friendship or a rivalry does in a battle.
// The company module owns the saved value per pair and applies it; combat
// reads it through internal/company and never imports the module.
//
// A bond is one number from -100 to 100 per pair of companions. Time
// together (camps, battles, banter) moves it only so far, up to +50 or down
// to -25: a deeper friendship needs one saving the other, and a rivalry
// needs real clashes (splitting over the leader's choices, a refused
// guard), so a company that merely travels together settles into trust or
// wariness and never into a feud.
package bonds

import (
	"strings"
	"time"
)

// The range and the lines in it.
const (
	Min = -100
	Max = 100

	// FriendAt is "trusts": friends step in for each other (Guards).
	FriendAt = 25
	// CloseAt is "close", the most that time together alone can reach.
	CloseAt = 50
	// KinAt is "like kin": a friend steps in twice a battle.
	KinAt = 75
	// WaryAt is "wary of": a mild coldness with no effect in battle.
	WaryAt = -25
	// RivalAt is "can't stand": a rival will not guard the other.
	RivalAt = -50
	// WarnAt warns the leader that one of a rivalry will leave.
	WarnAt = -85
	// LeaveAt is where, once warned, one of the pair leaves.
	LeaveAt = -100
	// MendAt is how far a warned rivalry must recover to be forgiven its
	// warning (and warned again if it falls back).
	MendAt = -60
)

// Source is where a change in a bond came from.
type Source string

const (
	Camp    Source = "camp"    // a camp rest shared: temperaments decide the sign
	Battle  Source = "battle"  // a battle won side by side
	Talk    Source = "talk"    // a line of banter between friends or rivals
	Opinion Source = "opinion" // agreeing or clashing over the leader's choice
	Rescue  Source = "rescue"  // one stepped in for the other in a battle
	Refusal Source = "refusal" // a rival let the other take the blow
	Rite    Source = "rite"    // the company held rites for one of their own (Phase 74)
)

// Rule is one source's worth: Cooldown is the real time before it may move
// the same pair again.
type Rule struct {
	Cooldown time.Duration
}

var rules = map[Source]Rule{
	Camp:    {30 * time.Minute},
	Battle:  {10 * time.Minute},
	Talk:    {30 * time.Minute},
	Opinion: {10 * time.Minute},
	Rescue:  {time.Hour},
	Refusal: {time.Hour},
	Rite:    {time.Hour},
}

// Sources is every source, in the order the help lists them.
var Sources = []Source{Camp, Battle, Talk, Opinion, Rescue, Refusal, Rite}

// CooldownOf is a source's cooldown; zero for an unknown source.
func CooldownOf(s Source) time.Duration { return rules[s].Cooldown }

// Known reports whether s is a source.
func Known(s Source) bool { _, ok := rules[s]; return ok }

// Rescue and refusal sizes. Camp, battle, talk and opinion deltas are one
// or two points, and come from Affinity and the callers.
const (
	RescueGain  = 3
	RefusalLoss = 2
)

// Apply is value after delta from source s. Rising sources stop at +50
// (Rescue at +100). Time together (Camp, Battle, Talk) falls no lower than
// -25, "wary of"; the real clashes, an Opinion split and a Refusal, reach
// -100 (Phase 65 review: a clashing pair that only camped together reached
// "can't stand" and a guardian then refused its ward with no act of the
// leader's behind it). A value already past a limit is not pulled back by
// a push the other way.
func Apply(value, delta int, s Source) int {
	hi, lo := CloseAt, WaryAt
	switch s {
	case Rescue:
		hi = Max
	case Refusal, Opinion:
		lo = Min
	}
	next := value + delta
	switch {
	case delta > 0 && next > hi:
		next = max(value, hi)
	case delta < 0 && next < lo:
		next = min(value, lo)
	}
	return max(Min, min(Max, next))
}

// Tiers.
const (
	TierBitter    = -3 // cannot bear
	TierRival     = -2 // can't stand
	TierWary      = -1 // wary of
	TierStrangers = 0
	TierFriend    = 1 // trusts
	TierClose     = 2 // close to
	TierKin       = 3 // like kin
)

// TierOf is a value's tier.
func TierOf(v int) int {
	switch {
	case v >= KinAt:
		return TierKin
	case v >= CloseAt:
		return TierClose
	case v >= FriendAt:
		return TierFriend
	case v <= WarnAt:
		return TierBitter
	case v <= RivalAt:
		return TierRival
	case v <= WaryAt:
		return TierWary
	}
	return TierStrangers
}

// IsFriend and IsRival are the two lines that matter in a battle.
func IsFriend(v int) bool { return v >= FriendAt }
func IsRival(v int) bool  { return v <= RivalAt }

// Feeling is what one of the pair feels about the other, finishing
// "Merek ...": "trusts", "is wary of", "can't stand".
func Feeling(v int) string {
	switch TierOf(v) {
	case TierKin:
		return "is like kin to"
	case TierClose:
		return "is close to"
	case TierFriend:
		return "trusts"
	case TierWary:
		return "is wary of"
	case TierRival:
		return "can't stand"
	case TierBitter:
		return "cannot bear"
	}
	return "is still getting to know"
}

// Together is the pair as a phrase finishing "Merek and Ysolde ...".
func Together(v int) string {
	switch TierOf(v) {
	case TierKin:
		return "are like kin"
	case TierClose:
		return "are close"
	case TierFriend:
		return "trust each other"
	case TierWary:
		return "are wary of each other"
	case TierRival:
		return "can't stand each other"
	case TierBitter:
		return "cannot bear each other"
	}
	return "are still getting to know each other"
}

// Guards is how many times a battle a friend steps in for a friend at low
// health (see GuardBelowPct); none for a pair that isn't friends.
func Guards(v int) int {
	switch {
	case v >= KinAt:
		return 2
	case v >= FriendAt:
		return 1
	}
	return 0
}

// GuardBelowPct is the share of its health, in percent, at or under which
// a friend's ward is worth stepping in for, and CompanyGuards is the most
// bond steps a company makes in one battle, all together (a company of
// friends is not a company of guardians).
const (
	GuardBelowPct = 40
	CompanyGuards = 2
)

// pair is an unordered pair of personalities.
type pair struct{ a, b string }

func pr(a, b string) pair {
	if a > b {
		a, b = b, a
	}
	return pair{a, b}
}

var suits = map[pair]bool{
	pr("stoic", "devout"): true, pr("stoic", "grim"): true,
	pr("cheerful", "devout"): true, pr("cheerful", "boastful"): true,
	pr("boastful", "wry"): true, pr("wry", "grim"): true,
}

var clashes = map[pair]bool{
	pr("cheerful", "grim"): true, pr("devout", "wry"): true,
	pr("stoic", "boastful"): true, pr("devout", "boastful"): true,
}

// AlignmentGap is the distance in alignment points at which two companions
// rub against each other, whatever their temperaments.
const AlignmentGap = 70

// Affinity is what a shared camp is worth to two companions: 2 for
// temperaments that suit, 1 for the rest, -1 for ones that clash, and one
// less again when their alignments are far apart (a saint and a villain).
func Affinity(a, b string, alignA, alignB int) int {
	a, b = strings.ToLower(a), strings.ToLower(b)
	n := 1
	switch p := pr(a, b); {
	case suits[p]:
		n = 2
	case clashes[p]:
		n = -1
	}
	gap := alignA - alignB
	if gap < 0 {
		gap = -gap
	}
	if gap >= AlignmentGap {
		n--
	}
	return n
}

// BattleGain is what a battle won side by side is worth to a pair whose
// camp affinity is aff: one point, but none for a pair that rubs.
func BattleGain(aff int) int {
	if aff > 0 {
		return 1
	}
	return 0
}

// OpinionDelta is what two verdicts on one choice do to a pair: both liked
// or both disliked it, +1; one liked it and the other didn't, -1. Verdicts
// are opinions.Likes (1) and opinions.Dislikes (-1).
func OpinionDelta(a, b int) int {
	switch {
	case a == 0 || b == 0:
		return 0
	case a == b:
		return 1
	}
	return -1
}

// Warning reports what a pair's value means for a rivalry that has (warned)
// been warned: Warn is true when an unwarned pair is at WarnAt or worse,
// Leave when a warned pair is at LeaveAt, and Mended when a warned pair has
// recovered to MendAt or better (its warning is cleared). A warning always
// comes first, so a leader has been told before anyone leaves.
func Warning(value int, warned bool) (warn, leave, mended bool) {
	switch {
	case warned && value >= MendAt:
		return false, false, true
	case warned && value <= LeaveAt:
		return false, true, false
	case !warned && value <= WarnAt:
		return true, false, false
	}
	return false, false, false
}
