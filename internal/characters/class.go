package characters

import (
	"slices"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

// Phase 38b: a character's class (the advanced or elite route promoted into
// from its archetype) and talents. A player's are the archetype registry's,
// a companion's are copied onto its mob by the company module; everything
// they give is derived from them and the character's current level.

// SetClassState records a companion's class and talents on its runtime
// character; the company record is the durable source.
func (c *Character) SetClassState(class string, talents []string) {
	c.HPClass = class
	c.HPTalents = slices.Clone(talents)
}

// ClassState is the character's class id ("" before promotion) and talents.
func (c *Character) ClassState() (string, []string) {
	if c == nil {
		return "", nil
	}
	if c.userId > 0 {
		s := classes.PlayerClass(c.userId)
		return s.Class, s.Talents
	}
	return c.HPClass, c.HPTalents
}

// ClassEffects are the character's class benefits at its current level:
// its route's ranks reached and the talents it has earned. Nil for a
// character with neither.
func (c *Character) ClassEffects() classes.Effects {
	class, talents := c.ClassState()
	if class == "" && len(talents) == 0 {
		return nil
	}
	if c.fxValid && c.fxClass == class && c.fxLevel == c.Level && slices.Equal(c.fxTalents, talents) {
		return c.fx
	}
	c.fx = classes.EffectsFor(class, c.Level, talents)
	c.fxClass, c.fxLevel, c.fxTalents, c.fxValid = class, c.Level, slices.Clone(talents), true
	return c.fx
}

// classPct raises a value by a percent class effect.
func classPct(value, pct int) int {
	if pct == 0 {
		return value
	}
	return value + (value*pct+50)/100
}

// SpellCost is a spell's mana cost for this character (Phase 38b): the
// spell's own, less the class's spell discount, and for a restoration
// spell more by the class's heal cost. A spell never costs less than 1.
func (c *Character) SpellCost(sp *spells.SpellData) int {
	if sp == nil {
		return 0
	}
	cost := sp.Cost
	fx := c.ClassEffects()
	if cost <= 0 || fx == nil {
		return cost
	}
	if pct := fx.Int(classes.SpellCost); pct != 0 {
		cost -= (cost*pct + 50) / 100
	}
	if sp.School == spells.SchoolRestoration {
		cost = classPct(cost, fx.Int(classes.HealCost))
	}
	return max(1, cost)
}

// HealCostPct is the percent a character's healing spells cost more
// (negative for less) when its company patches itself: its class's heal
// cost and spell discount, and for patching its patch discount.
func (c *Character) HealCostPct(patching bool) int {
	fx := c.ClassEffects()
	pct := fx.Int(classes.HealCost) - fx.Int(classes.SpellCost)
	if patching {
		pct -= fx.Int(classes.PatchCost)
	}
	return pct
}

// ClassAura is what an ally's class aura gives a character for a combat
// round: Evasion points and a percent less damage.
type ClassAura struct {
	Evasion int
	Resolve int
}
