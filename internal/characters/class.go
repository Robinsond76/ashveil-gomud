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
	Block   int // block chance points (a Knight guarding a ward)
}

// ClassRT is a character's class state for the battle it is in: nothing in
// it is saved, and a fight's end clears all of it but the Lay on Hands
// uses, which come back with rest.
type ClassRT struct {
	Ward, WardCap int  // blows a ward absorbs, and the most it takes from each
	Bark, Thorns  int  // Barkskin's armor and the damage a striker takes
	Rejuv, Per    int  // rounds of Rejuvenation left and its heal each round
	ShieldUsed    bool // Divine Shield has been spent this battle
	OathUsed      int  // Blood Oath blows spent this battle
	Intim         int  // Attack this foe loses against anyone but IntimOwner
	IntimOwner    *ClassRT
	Cleansed      map[string]bool
	Guards        int // an Angel's Guard uses spent
	Hands         int // Lay on Hands uses since the last rest
	Summoned      bool
	Bless         int // rounds of Bless left
}

// RTState is the character's class battle state, made on first use.
func (c *Character) RTState() *ClassRT {
	if c.RT == nil {
		c.RT = &ClassRT{}
	}
	return c.RT
}

// EndFightRT clears the battle's class state, keeping what rest restores.
func (c *Character) EndFightRT() {
	if c.RT == nil {
		return
	}
	c.RT = &ClassRT{Hands: c.RT.Hands}
	c.Aura = ClassAura{}
}

// AbsorbWard takes a ward's share of one blow, and returns what is left.
func (c *Character) AbsorbWard(dmg int) (left, absorbed int) {
	if c.RT == nil || c.RT.Ward <= 0 || dmg <= 0 {
		return dmg, 0
	}
	absorbed = min(dmg, max(c.RT.WardCap, 0))
	c.RT.Ward--
	return dmg - absorbed, absorbed
}

// ShieldBlow spends a Divine Shield on a blow, once a battle.
func (c *Character) ShieldBlow() bool {
	if c.RT == nil || c.RT.ShieldUsed || !c.ClassEffects().Has(classes.DivineShield) {
		return false
	}
	c.RT.ShieldUsed = true
	return true
}

// Bless is a cleric's blessing: Attack and Evasion points while it lasts.
const BlessPoints = 5

func (c *Character) blessPoints() int {
	if c.RT != nil && c.RT.Bless > 0 {
		return BlessPoints
	}
	return 0
}

// RestClass renews what rest gives back to a class: Lay on Hands uses.
func (c *Character) RestClass() {
	if c.RT != nil {
		c.RT.Hands = 0
	}
}
