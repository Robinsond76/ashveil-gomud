package strategy

import "strings"

// Use is what an automatic spell is for.
type Use string

const (
	UseHeal      Use = "heal"       // heals one
	UseHealAll   Use = "heal-all"   // heals the company
	UseAttack    Use = "attack"     // harms one foe
	UseAttackAll Use = "attack-all" // harms the foe's group
)

// ParseUse reads a use from config.
func ParseUse(s string) (Use, bool) {
	switch u := Use(strings.ToLower(strings.TrimSpace(s))); u {
	case UseHeal, UseHealAll, UseAttack, UseAttackAll:
		return u, true
	}
	return "", false
}

// Spell is a spell cast automatically, what it's for, and its mana cost.
type Spell struct {
	ID   string
	Use  Use
	Cost int
}

// DefaultAutoSpells is the shipped list of automatic spells (costs are
// filled in from the spell files by the caller). No other spell is ever
// cast automatically.
func DefaultAutoSpells() []Spell {
	return []Spell{
		{ID: "heal", Use: UseHeal},
		{ID: "healall", Use: UseHealAll},
		{ID: "mm", Use: UseAttack},
		{ID: "sparks", Use: UseAttackAll},
	}
}

// SpellFor is the first spell of a use that the character knows.
func SpellFor(spells []Spell, use Use, knows func(string) bool) (Spell, bool) {
	for _, sp := range spells {
		if sp.Use == use && knows != nil && knows(sp.ID) {
			return sp, true
		}
	}
	return Spell{}, false
}

// Ally is one member of the character's side (the player or a companion)
// here, for a healer.
type Ally struct {
	HP, MaxHP int
}

// Situation is what a character's role decides from, each round.
type Situation struct {
	Role   Role
	Mana   int
	Knows  func(spellID string) bool
	Spells []Spell // the automatic spells, in order, with costs
	Allies []Ally  // the side's members here, the character included
	Foes   int     // the battle's foes standing
}

// ActionKind is what a character does this round.
type ActionKind int

const (
	Swing     ActionKind = iota // strike its aim
	Heal                        // heal Allies[Ally]
	HealAll                     // heal the side
	Attack                      // a spell at its target
	AttackAll                   // a spell at the group
)

// Action is a role's decision. Spell is the spell to cast (for all but
// Swing); Ally indexes Situation.Allies for Heal.
type Action struct {
	Kind  ActionKind
	Spell string
	Ally  int
}

// Decide chooses what a character does this round by its role:
//   - a healer heals anyone below half health (the group heal when two or
//     more are, else the single heal on the most hurt), else swings;
//   - a caster casts its area spell when two or more foes stand, else its
//     single-target spell, else swings;
//   - a fighter swings.
//
// A spell is cast only when it is configured, known, and paid for.
func Decide(s Situation) Action {
	affordable := func(use Use) (Spell, bool) {
		sp, ok := SpellFor(s.Spells, use, s.Knows)
		if !ok || s.Mana < sp.Cost {
			return Spell{}, false
		}
		return sp, true
	}
	switch s.Role {
	case Healer:
		hurt, worst := 0, -1
		for i, a := range s.Allies {
			if a.HP < 1 || a.HP*2 >= a.MaxHP {
				continue
			}
			hurt++
			if worst < 0 || fraction(a.HP, a.MaxHP) < fraction(s.Allies[worst].HP, s.Allies[worst].MaxHP) {
				worst = i
			}
		}
		if hurt == 0 {
			return Action{Kind: Swing}
		}
		if hurt >= 2 {
			if sp, ok := affordable(UseHealAll); ok {
				return Action{Kind: HealAll, Spell: sp.ID}
			}
		}
		if sp, ok := affordable(UseHeal); ok {
			return Action{Kind: Heal, Spell: sp.ID, Ally: worst}
		}
		if sp, ok := affordable(UseHealAll); ok {
			return Action{Kind: HealAll, Spell: sp.ID}
		}
	case Caster:
		if s.Foes < 1 {
			return Action{Kind: Swing}
		}
		if s.Foes >= 2 {
			if sp, ok := affordable(UseAttackAll); ok {
				return Action{Kind: AttackAll, Spell: sp.ID}
			}
		}
		if sp, ok := affordable(UseAttack); ok {
			return Action{Kind: Attack, Spell: sp.ID}
		}
		if sp, ok := affordable(UseAttackAll); ok {
			return Action{Kind: AttackAll, Spell: sp.ID}
		}
	}
	return Action{Kind: Swing}
}
