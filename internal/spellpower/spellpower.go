// Package spellpower is the one source of a spell's size (Phase 35b): a
// flat base, a dice roll, and bonuses for the caster's level and
// Mysticism, read from the spell's YAML `power` block. Spell scripts,
// `heal wounds`, after-battle patching, the level-up report and the
// balance harness all read it, so they cannot drift apart.
//
// It is pure: callers supply the caster's level, its Mysticism and the
// dice. Skill (a damage spell's SpellFactor) and gear (a heal's
// HealFactor) are applied by the caller.
package spellpower

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Power is a spell's size:
//
//	base + dice + floor(level / leveldiv) + floor(mysticism / mysticismdiv)
//
// then times percent/100 (healall's 55% of a Minor Heal). A zero divisor
// adds nothing; a zero percent is 100.
type Power struct {
	Base         int    `yaml:"base"`
	Dice         string `yaml:"dice,omitempty"`
	LevelDiv     int    `yaml:"leveldiv,omitempty"`
	MysticismDiv int    `yaml:"mysticismdiv,omitempty"`
	Percent      int    `yaml:"percent,omitempty"`
}

// ParseDice reads "2d4" as (2, 4). "" is no dice.
func ParseDice(s string) (qty, sides int, err error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, 0, nil
	}
	q, sd, ok := strings.Cut(s, "d")
	if !ok {
		return 0, 0, fmt.Errorf("dice %q: want NdM", s)
	}
	if qty, err = strconv.Atoi(q); err != nil || qty < 1 {
		return 0, 0, fmt.Errorf("dice %q: bad count", s)
	}
	if sides, err = strconv.Atoi(sd); err != nil || sides < 1 {
		return 0, 0, fmt.Errorf("dice %q: bad sides", s)
	}
	return qty, sides, nil
}

// Validate rejects a power block that could not roll.
func (p Power) Validate() error {
	if _, _, err := ParseDice(p.Dice); err != nil {
		return err
	}
	if p.Base < 0 || p.LevelDiv < 0 || p.MysticismDiv < 0 || p.Percent < 0 || p.Percent > 1000 {
		return fmt.Errorf("power: negative or out-of-range field in %+v", p)
	}
	if p.Base == 0 && p.Dice == "" {
		return fmt.Errorf("power: neither base nor dice")
	}
	return nil
}

// DiceQS is the power's dice as (count, sides); (0, 0) when it has none.
func (p Power) DiceQS() (int, int) {
	q, s, _ := ParseDice(p.Dice)
	return q, s
}

// Flat is everything but the dice: base plus the level and Mysticism
// bonuses.
func (p Power) Flat(level, mysticism int) int {
	f := p.Base
	if p.LevelDiv > 0 {
		f += max(level, 0) / p.LevelDiv
	}
	if p.MysticismDiv > 0 {
		f += max(mysticism, 0) / p.MysticismDiv
	}
	return f
}

// pct is the share applied after the roll, as a fraction.
func (p Power) pct() float64 {
	if p.Percent == 0 {
		return 1
	}
	return float64(p.Percent) / 100
}

// Raw is the unrounded size of one roll: (flat + dice) times the share.
// roll(n) returns 0..n-1 (util.Rand).
func (p Power) Raw(level, mysticism int, roll func(int) int) float64 {
	q, s := p.DiceQS()
	t := p.Flat(level, mysticism)
	for i := 0; i < q; i++ {
		t += roll(s) + 1
	}
	return float64(t) * p.pct()
}

// Roll is one roll, rounded down, at least 1.
func (p Power) Roll(level, mysticism int, roll func(int) int) int {
	return max(1, int(math.Floor(p.Raw(level, mysticism, roll))))
}

// Range is the smallest and largest Roll at a level and Mysticism.
func (p Power) Range(level, mysticism int) (lo, hi int) {
	q, s := p.DiceQS()
	f := p.Flat(level, mysticism)
	lo = max(1, int(math.Floor(float64(f+q)*p.pct())))
	hi = max(1, int(math.Floor(float64(f+q*s)*p.pct())))
	return lo, hi
}

// Mean is the average Raw at a level and Mysticism.
func (p Power) Mean(level, mysticism int) float64 {
	q, s := p.DiceQS()
	return (float64(p.Flat(level, mysticism)) + float64(q)*float64(s+1)/2) * p.pct()
}
