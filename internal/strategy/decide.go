package strategy

import "strings"

// Use is what an automatic spell is for.
type Use string

const (
	UseHeal      Use = "heal"       // heals one
	UseHealAll   Use = "heal-all"   // heals the company
	UseAttack    Use = "attack"     // harms one foe
	UseAttackAll Use = "attack-all" // harms the foe's group
	UseHex       Use = "hex"        // a Witch's hex, which takes foes' turns away (Phase 38a)

	// Phase 38b: class spells.
	UseBigHeal Use = "big-heal" // a heavy single heal, for an ally in trouble
	UseRejuv   Use = "rejuv"    // heals one over rounds
	UseGrove   Use = "grove"    // heals over rounds a whole formation row
	UseWard    Use = "ward"     // absorbs an ally's next blows
	UseBark    Use = "bark"     // armor for an ally (or a row)
	UseBless   Use = "bless"    // Attack and Evasion for an ally
	UseSiphon  Use = "siphon"   // drains a foe and heals the most hurt ally
	UseSummon  Use = "summon"   // calls the class summon at the start of a battle
)

// ParseUse reads a use from config.
func ParseUse(s string) (Use, bool) {
	switch u := Use(strings.ToLower(strings.TrimSpace(s))); u {
	case UseHeal, UseHealAll, UseAttack, UseAttackAll, UseHex, UseBigHeal, UseRejuv, UseGrove, UseWard, UseBark, UseBless, UseSiphon, UseSummon:
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
		// Phase 38a: the Witch's hexes, in the order tried, then its curse.
		{ID: "binding", Use: UseHex},
		{ID: "slumber", Use: UseHex},
		{ID: "earthbind", Use: UseHex},
		{ID: "frailty", Use: UseHex},
		{ID: "leaden", Use: UseHex},
		{ID: "miasma", Use: UseHex},
		{ID: "dread", Use: UseHex},
		{ID: "blight", Use: UseHex},
		{ID: "hex", Use: UseAttack},
		// Phase 38b: the class spells (a character casts one only if its
		// class has taught it).
		{ID: "greaterheal", Use: UseBigHeal},
		{ID: "rejuvenation", Use: UseRejuv},
		{ID: "grove", Use: UseGrove},
		{ID: "siphon", Use: UseSiphon},
		{ID: "ward", Use: UseWard},
		{ID: "arcaneward", Use: UseWard},
		{ID: "barkskin", Use: UseBark},
		{ID: "bless", Use: UseBless},
		{ID: "entangle", Use: UseHex},
		{ID: "callhost", Use: UseSummon},
		{ID: "bindfiend", Use: UseSummon},
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
	// Downed is a player at or below 0 health but not dead: still healed
	// (the most hurt of all). A companion at 0 is dead and isn't.
	Downed bool
	// Pending is an ally a heal already covers (Phase 33e): a chant in
	// progress, or another healer's choice this round. It is not healed
	// again.
	Pending bool
	// Phase 38b: what the ally already carries this battle, so a buff goes
	// to someone without it.
	Warded, Barked, Rejuv, Blessed bool
}

// Situation is what a character's role decides from, each round.
type Situation struct {
	Role   Role
	Mana   int
	Knows  func(spellID string) bool
	Spells []Spell // the automatic spells, in order, with costs
	Allies []Ally  // the side's members here, the character included
	Foes   int     // the battle's foes standing
	// Summoned (Phase 38b) is whether the character has called its summon
	// this battle (or has none to call).
	Summoned bool
	// HealBelow is the healing threshold, a percent of each ally's wound
	// limit (Phase 30c tactics); 0 means DefaultHealing.
	HealBelow int
	// MaxMana and Reserve (a percent, Phase 33e): an attack spell is cast
	// only while Reserve percent of MaxMana would remain. Heals ignore it.
	MaxMana, Reserve int
	// CanHex (Phase 38a) reports whether a hex has a foe worth casting it
	// at: one that doesn't already carry its status, isn't immune to it,
	// and (Blight) heals. Nil means every hex may be cast.
	CanHex func(spellID string) bool
}

// ActionKind is what a character does this round.
type ActionKind int

const (
	Swing     ActionKind = iota // strike its aim
	Heal                        // heal Allies[Ally]
	HealAll                     // heal the side
	Attack                      // a spell at its target
	AttackAll                   // a spell at the group
	Hex                         // a hex at the foes it reaches (Phase 38a)
	Buff                        // a class buff on Allies[Ally] (Phase 38b)
	Row                         // a spell on the formation row of Allies[Ally]
	Drain                       // Siphon at the foes it reaches
	Summon                      // call the class summon (the caster is its own target)
)

// Action is a role's decision. Spell is the spell to cast (for all but
// Swing); Ally indexes Situation.Allies for Heal.
type Action struct {
	Kind  ActionKind
	Spell string
	Ally  int
}

// Decide chooses what a character does this round by its role:
//   - a healer heals anyone below the healing threshold (HealBelow, half
//     by default; the group heal when two or more are, else the single
//     heal on the most hurt), else swings;
//     An ally a heal already covers (Pending) is skipped (Phase 33e);
//   - a caster casts its area spell when two or more foes stand, else its
//     single-target spell, else swings;
//   - a controller casts the first of its hexes that has a foe worth it
//     (CanHex), else its single-target attack spell (Withering Hex), else
//     swings (Phase 38a);
//   - a fighter swings.
//
// A spell is cast only when it is configured, known, and paid for, and an
// attack spell only while it leaves the member's mana reserve (heals and
// hexes ignore the reserve: they are what it is kept for).
func Decide(s Situation) Action {
	affordable := func(use Use) (Spell, bool) {
		sp, ok := SpellFor(s.Spells, use, s.Knows)
		if !ok || s.Mana < sp.Cost {
			return Spell{}, false
		}
		if (use == UseAttack || use == UseAttackAll) && s.Reserve > 0 && (s.Mana-sp.Cost)*100 < s.Reserve*s.MaxMana {
			return Spell{}, false
		}
		return sp, true
	}
	// Phase 38b: a summoner calls its summon first, as the battle opens.
	if !s.Summoned && s.Foes >= 1 && (s.Role == Healer || s.Role == Caster || s.Role == Controller) {
		if sp, ok := affordable(UseSummon); ok {
			return Action{Kind: Summon, Spell: sp.ID}
		}
	}
	switch s.Role {
	case Healer:
		below := s.HealBelow
		if below <= 0 {
			below = DefaultHealing
		}
		hurt, worst := 0, -1
		for i, a := range s.Allies {
			if (a.HP < 1 && !a.Downed) || a.Pending || a.HP*100 >= below*a.MaxHP {
				continue
			}
			hurt++
			if worst < 0 || fraction(a.HP, a.MaxHP) < fraction(s.Allies[worst].HP, s.Allies[worst].MaxHP) {
				worst = i
			}
		}
		if hurt == 0 {
			return idleHealer(s, affordable)
		}
		worstFrac := fraction(s.Allies[worst].HP, s.Allies[worst].MaxHP)
		if worstFrac < BigHealBelow {
			if sp, ok := affordable(UseBigHeal); ok {
				return Action{Kind: Heal, Spell: sp.ID, Ally: worst}
			}
		}
		if hurt >= 2 {
			if sp, ok := affordable(UseGrove); ok {
				return Action{Kind: Row, Spell: sp.ID, Ally: worst}
			}
			if sp, ok := affordable(UseHealAll); ok {
				return Action{Kind: HealAll, Spell: sp.ID}
			}
		}
		if worstFrac >= RejuvAbove && !s.Allies[worst].Rejuv {
			if sp, ok := affordable(UseRejuv); ok {
				return Action{Kind: Heal, Spell: sp.ID, Ally: worst}
			}
		}
		if s.Foes >= 1 && !s.Allies[worst].Downed {
			if sp, ok := affordable(UseSiphon); ok {
				return Action{Kind: Drain, Spell: sp.ID, Ally: -1}
			}
		}
		if sp, ok := affordable(UseHeal); ok {
			return Action{Kind: Heal, Spell: sp.ID, Ally: worst}
		}
		if sp, ok := affordable(UseHealAll); ok {
			return Action{Kind: HealAll, Spell: sp.ID}
		}
	case Controller:
		if s.Foes < 1 {
			return Action{Kind: Swing}
		}
		for _, sp := range s.Spells {
			if sp.Use != UseHex || s.Knows == nil || !s.Knows(sp.ID) || s.Mana < sp.Cost {
				continue
			}
			if s.CanHex != nil && !s.CanHex(sp.ID) {
				continue
			}
			return Action{Kind: Hex, Spell: sp.ID}
		}
		if sp, ok := affordable(UseAttack); ok {
			return Action{Kind: Attack, Spell: sp.ID}
		}
	case Caster:
		if s.Foes < 1 {
			return Action{Kind: Swing}
		}
		// Phase 38b: a Theurgist wards the company before it casts.
		spare := func(sp Spell) bool {
			return s.Reserve <= 0 || (s.Mana-sp.Cost)*100 >= s.Reserve*s.MaxMana
		}
		if act, ok := tryBuffs(s, affordable, spare, UseWard); ok {
			return act
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

// BigHealBelow and RejuvAbove are the shares of health (in thousandths) under
// which a healer reaches for its heavy heal, and from which it prefers a
// heal over time (Phase 38b).
const (
	BigHealBelow = 400
	RejuvAbove   = 350
)

// idleHealer is what a healer does when no one needs healing: while foes
// stand, put up a class buff on someone without it, hobble a foe, or drain
// one; otherwise swing. Each keeps the mana reserve.
func idleHealer(s Situation, affordable func(Use) (Spell, bool)) Action {
	if s.Foes < 1 {
		return Action{Kind: Swing}
	}
	spare := func(sp Spell) bool {
		return s.Reserve <= 0 || (s.Mana-sp.Cost)*100 >= s.Reserve*s.MaxMana
	}
	if act, ok := tryBuffs(s, affordable, spare, UseWard, UseBark, UseBless); ok {
		return act
	}
	for _, sp := range s.Spells {
		if sp.Use != UseHex || s.Knows == nil || !s.Knows(sp.ID) || s.Mana < sp.Cost || !spare(sp) {
			continue
		}
		if s.CanHex != nil && !s.CanHex(sp.ID) {
			continue
		}
		return Action{Kind: Hex, Spell: sp.ID}
	}
	if sp, ok := affordable(UseSiphon); ok && spare(sp) {
		return Action{Kind: Drain, Spell: sp.ID, Ally: -1}
	}
	return Action{Kind: Swing}
}

// tryBuffs is the first of the buffs, in order, that a known, affordable
// spell can put on an ally who lacks it.
func tryBuffs(s Situation, affordable func(Use) (Spell, bool), spare func(Spell) bool, uses ...Use) (Action, bool) {
	for _, use := range uses {
		sp, ok := affordable(use)
		if !ok || !spare(sp) {
			continue
		}
		for i, a := range s.Allies {
			if a.HP < 1 || a.Pending {
				continue
			}
			if (use == UseWard && a.Warded) || (use == UseBark && a.Barked) || (use == UseBless && a.Blessed) {
				continue
			}
			return Action{Kind: Buff, Spell: sp.ID, Ally: i}, true
		}
	}
	return Action{}, false
}
