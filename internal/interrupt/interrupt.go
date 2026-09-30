// Package interrupt holds Ashveil's rules for broken chants and shield
// counters (Phase 30d1). It is pure: the combat loop (internal/hooks)
// supplies the facts and the dice, and applies what these rules decide.
//
// Only a weapon blow that does 1 or more damage can break a chant (the
// owner's rule, 2026-09-30), and it does so by chance (Phase 30d1b): heavy
// force (a crit, a stagger, a knockdown, a stun, a shield bash) always
// breaks it; any other blow breaks it more often the harder it lands,
// against the chanter's max health. There is no pressure meter.
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

// The chance a blow that isn't heavy breaks a chant is held between these.
const (
	BreakChanceMin = 40
	BreakChanceMax = 90
)

// CanBreak reports whether a blow may break its target's chant: it hit,
// did at least 1 damage, and the target was chanting. BreakChance says
// how likely it is.
func CanBreak(hit bool, damage int, chanting bool) bool {
	return hit && damage >= 1 && chanting
}

// BreakChance is the percent chance a blow of damage breaks the chant of
// a chanter with maxHP: 100 for heavy force, otherwise 40 plus 2 for each
// percent of the chanter's max health the blow took, held to 40-90. A
// nick is about 40%; a quarter of the chanter's health is 90%. A blow of
// no damage never breaks.
func BreakChance(damage, maxHP int, heavy bool) int {
	if damage < 1 {
		return 0
	}
	if heavy {
		return 100
	}
	if maxHP < 1 {
		maxHP = 1
	}
	chance := BreakChanceMin + 2*(damage*100/maxHP)
	return min(max(chance, BreakChanceMin), BreakChanceMax)
}

// RollBreak rolls a break at chance percent. roll(n) returns 0..n-1
// (util.Rand); a certain outcome doesn't roll.
func RollBreak(chance int, roll func(int) int) bool {
	if chance >= 100 {
		return true
	}
	if chance <= 0 {
		return false
	}
	return roll(100) < chance
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
