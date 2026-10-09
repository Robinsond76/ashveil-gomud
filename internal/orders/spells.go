package orders

import "github.com/GoMudEngine/GoMud/internal/strategy"

// Caster is what a member can cast, for carrying out an order: the
// automatic spells (with the member's own costs), those it knows, and what
// it can pay.
type Caster struct {
	Spells []strategy.Spell
	Knows  func(spellID string) bool
	Mana   int
	Flasks int
	// LanceFoes (Phase 84) is the most foes it looses a Lance at (0: any).
	LanceFoes int
}

func (c Caster) affordable(sp strategy.Spell) bool {
	return c.Knows != nil && c.Knows(sp.ID) && c.Mana >= sp.Cost && c.Flasks >= sp.Flask
}

func (c Caster) first(use strategy.Use) (strategy.Spell, bool) {
	for _, sp := range c.Spells {
		if sp.Use == use && c.affordable(sp) {
			return sp, true
		}
	}
	return strategy.Spell{}, false
}

// Heal is the spell a heal order casts on an ally with this share of its
// health (thousandths): the heavy heal for one in trouble, else the plain
// heal, else the heavy one, else a heal over time.
func (c Caster) Heal(share int) (strategy.Spell, bool) {
	if share < strategy.BigHealBelow {
		if sp, ok := c.first(strategy.UseBigHeal); ok {
			return sp, true
		}
	}
	for _, use := range []strategy.Use{strategy.UseHeal, strategy.UseBigHeal, strategy.UseRejuv} {
		if sp, ok := c.first(use); ok {
			return sp, true
		}
	}
	return strategy.Spell{}, false
}

// Attack is the attack spell an order casts, in the order a caster prefers
// them: Lightning, the Arcane Lance (not at a crowd past its LanceFoes limit,
// unless aimed at one foe), a spell at the whole group (only when
// foes is two or more, and never for a single target), then the plain
// single-target spell, then the group spell as a last resort. The
// strategy.ActionKind says how the caller aims it.
func (c Caster) Attack(foes int, single bool) (strategy.Spell, strategy.ActionKind, bool) {
	if sp, ok := c.first(strategy.UseStorm); ok {
		return sp, strategy.Storm, true
	}
	// Phase 84: against a crowd the Lance gives way to the group spell, as
	// in strategy.Decide; an order aimed at one foe keeps it.
	if sp, ok := c.first(strategy.UseBurst); ok && (single || strategy.LanceFits(c.LanceFoes, foes)) {
		return sp, strategy.Attack, true
	}
	if !single && foes >= 2 {
		if sp, ok := c.first(strategy.UseAttackAll); ok {
			return sp, strategy.AttackAll, true
		}
	}
	if sp, ok := c.first(strategy.UseAttack); ok {
		return sp, strategy.Attack, true
	}
	if !single {
		if sp, ok := c.first(strategy.UseAttackAll); ok {
			return sp, strategy.AttackAll, true
		}
	}
	return strategy.Spell{}, 0, false
}

// KnowsAttack reports whether the member knows any attack spell (paid for
// or not): a hold order only matters to one that does.
func (c Caster) KnowsAttack() bool {
	for _, sp := range c.Spells {
		switch sp.Use {
		case strategy.UseAttack, strategy.UseAttackAll, strategy.UseBurst, strategy.UseStorm:
			if c.Knows != nil && c.Knows(sp.ID) {
				return true
			}
		}
	}
	return false
}
