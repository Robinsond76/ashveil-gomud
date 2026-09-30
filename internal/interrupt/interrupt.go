// Package interrupt holds Ashveil's rules for broken chants and shield
// counters (Phase 30d1). It is pure: the combat loop (internal/hooks)
// supplies the facts and the dice, and applies what these rules decide.
//
// A chant breaks whenever the chanter takes physical damage: a weapon blow
// that does 1 or more damage (the owner's rule, 2026-09-30). There is no
// pressure meter, threshold, or concentration roll.
//
// A shield counter is a reaction: when an attack on a shield-bearer misses
// (a dodge included), the bearer may slam its shield back into the
// attacker, and the bash may stun.
package interrupt

// The counter's odds, as package values so tests can make them certain.
var (
	// BashChance is the percent of fumbled blows a bearer counters.
	BashChance = 50
	// StunChance is the percent of bashes that stun.
	StunChance = 25
)

// BashDamage is the bash's damage die: 1 to BashDamage.
const BashDamage = 4

// Breaks reports whether a blow breaks its target's chant: it hit, did at
// least 1 damage, and the target was chanting.
func Breaks(hit bool, damage int, chanting bool) bool {
	return hit && damage >= 1 && chanting
}

// Refund is the mana a company caster gets back when its spell is broken:
// half the spell's cost, rounded down.
func Refund(cost int) int {
	if cost <= 0 {
		return 0
	}
	return cost / 2
}

// Counter is what decides whether a blow's target can counter it.
type Counter struct {
	Missed    bool // the blow missed or was dodged
	Melee     bool // not a bow or sling
	SameRoom  bool // the attacker stands in the bearer's room
	Shield    bool // the bearer holds a shield it can raise
	Able      bool // standing: alive, able to fight, not down or stunned
	Chanting  bool // the bearer is chanting a spell
	Countered bool // the bearer has countered this combat round already
}

// CanCounter reports whether the bearer may counter the blow.
func CanCounter(c Counter) bool {
	return c.Missed && c.Melee && c.SameRoom && c.Shield && c.Able && !c.Chanting && !c.Countered
}

// Bash is a counter that landed.
type Bash struct {
	Damage int
	Stun   bool
}

// RollCounter rolls a counter: whether the bearer bashes, its damage, and
// whether it stuns. roll(n) returns 0..n-1 (util.Rand).
func RollCounter(roll func(int) int) (Bash, bool) {
	if roll(100) >= BashChance {
		return Bash{}, false
	}
	b := Bash{Damage: roll(BashDamage) + 1}
	b.Stun = roll(100) < StunChance
	return b, true
}
