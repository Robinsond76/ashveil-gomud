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
	UseRaise   Use = "raise"    // raises a fallen foe as a thrall (a Necromancer, Phase 38c3)

	// Phase 39c: a Shaman's spells.
	UseWeather Use = "weather" // calls a battle weather: Fog, Chill Wind or Rain
	UseStorm   Use = "storm"   // Lightning, a heavy bolt at one foe (a second for a Stormcaller)

	// Phase 38d: a Sorcerer's burst.
	UseBurst Use = "burst" // Arcane Lance, one heavy, costly bolt at one foe

	// Phase 39g: an Alchemist's flasks.
	UseCure  Use = "cure"  // an antidote: takes poison and bleeding off an ally
	UseFlame Use = "flame" // a Fire Flask at a foe and the one beside it
)

// ParseUse reads a use from config.
func ParseUse(s string) (Use, bool) {
	switch u := Use(strings.ToLower(strings.TrimSpace(s))); u {
	case UseHeal, UseHealAll, UseAttack, UseAttackAll, UseHex, UseBigHeal, UseRejuv, UseGrove, UseWard, UseBark, UseBless, UseSiphon, UseSummon, UseRaise, UseWeather, UseStorm, UseCure, UseFlame, UseBurst:
		return u, true
	}
	return "", false
}

// Spell is a spell cast automatically, what it's for, and its mana cost.
// Flask (Phase 39g) is the flasks it uses up, an Alchemist's cost instead of
// mana.
type Spell struct {
	ID    string
	Use   Use
	Cost  int
	Flask int
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
		{ID: "raisefallen", Use: UseRaise},
		// Phase 39c: the Shaman's weather, in the order tried (Rain first, as
		// Lightning feeds on it), then Lightning, then Gust.
		{ID: "rain", Use: UseWeather},
		{ID: "chillwind", Use: UseWeather},
		{ID: "callfog", Use: UseWeather},
		{ID: "lightning", Use: UseStorm},
		{ID: "gust", Use: UseAttack},
		{ID: "stoneskin", Use: UseBark},
		// Phase 38d: the Sorcerer's Lance, ahead of the plain attack spells.
		{ID: "arcanelance", Use: UseBurst},
		// Phase 39g: the Alchemist's flasks. Each costs a flask, not mana.
		{ID: "draught", Use: UseHeal},
		{ID: "antidote", Use: UseCure},
		{ID: "tonic", Use: UseBless},
		{ID: "fireflask", Use: UseFlame},
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
	// Afflicted (Phase 39g) is an ally with poison or bleeding on it, which
	// an antidote takes off.
	Afflicted bool
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
	// Boss (Phase 38b review) is whether a boss stands among the foes: a
	// summoner calls for a boss even when fewer than SummonFoes stand.
	Boss bool
	// HealBelow is the healing threshold, a percent of each ally's wound
	// limit (Phase 30c tactics); 0 means DefaultHealing.
	HealBelow int
	// MaxMana and Reserve (a percent, Phase 33e): an attack spell is cast
	// only while Reserve percent of MaxMana would remain. Heals ignore it.
	MaxMana, Reserve int
	// CanRaise (Phase 38c3) is whether a foe has fallen that the character
	// may raise as a thrall now: a Necromancer with a raise left.
	CanRaise bool
	// CanHex (Phase 38a) reports whether a hex has a foe worth casting it
	// at: one that doesn't already carry its status, isn't immune to it,
	// and (Blight) heals. Nil means every hex may be cast.
	CanHex func(spellID string) bool
	// Weather (Phase 39c) is the battle's weather now ("" for none); a
	// Shaman calls one only when none is up. CanWeather reports whether a
	// weather spell is worth casting (a fog or chill needs a foe that
	// shoots or casts; rain needs Lightning to feed). Nil means always.
	Weather    string
	CanWeather func(spellID string) bool
	// Flasks (Phase 39g) is the flasks the character's satchel holds. A
	// spell that costs flasks is cast only while it holds enough; a Fire
	// Flask only while more than FlaskKeep remain, so a heal is never left
	// without one.
	Flasks int
	// Hold (Phase 61) is a battle order to hold mana: no attack spell
	// (Lightning included) this round; heals, hexes, buffs, summons and
	// weather go on.
	Hold bool
	// LanceFoes (Phase 84) is the most foes the character looses an Arcane
	// Lance at; against more it sparks the group. 0 means no limit (a High
	// Sorcerer, or a caster with no class).
	LanceFoes int
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
	Raise                       // raise a fallen foe as a thrall (the caster is its own target)
	Weather                     // call a battle weather (the caster stands for it; Phase 39c)
	Storm                       // Lightning at a foe, and a second for a chain (Phase 39c)
	Flame                       // a Fire Flask at a foe and the one beside it (Phase 39g)
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
		if !ok || s.Mana < sp.Cost || s.Flasks < sp.Flask {
			return Spell{}, false
		}
		if use == UseFlame && s.Flasks <= FlaskKeep {
			return Spell{}, false
		}
		if s.Hold && (use == UseAttack || use == UseAttackAll || use == UseBurst || use == UseStorm) {
			return Spell{}, false
		}
		if (use == UseAttack || use == UseAttackAll || use == UseBurst) && s.Reserve > 0 && (s.Mana-sp.Cost)*100 < s.Reserve*s.MaxMana {
			return Spell{}, false
		}
		return sp, true
	}
	// Phase 38b: a summoner calls its summon first, as the battle opens,
	// when the battle is worth it (three or more foes, or a boss) and the
	// call leaves its mana reserve.
	if !s.Summoned && (s.Foes >= SummonFoes || (s.Boss && s.Foes >= 1)) && (s.Role == Healer || s.Role == Caster || s.Role == Controller) {
		if sp, ok := affordable(UseSummon); ok && (s.Reserve <= 0 || (s.Mana-sp.Cost)*100 >= s.Reserve*s.MaxMana) {
			return Action{Kind: Summon, Spell: sp.ID}
		}
	}
	// Phase 38c3: a Necromancer raises a foe that has fallen.
	if s.CanRaise && (s.Role == Healer || s.Role == Caster || s.Role == Controller) {
		if sp, ok := affordable(UseRaise); ok && (s.Reserve <= 0 || (s.Mana-sp.Cost)*100 >= s.Reserve*s.MaxMana) {
			return Action{Kind: Raise, Spell: sp.ID}
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
		// Phase 39g: an antidote first, while no one is in danger.
		if worstFrac >= CureAbove {
			if act, ok := tryCure(s, affordable); ok {
				return act
			}
		}
		if worstFrac < BigHealBelow {
			if sp, ok := affordable(UseBigHeal); ok {
				return Action{Kind: Heal, Spell: sp.ID, Ally: worst}
			}
		}
		if hurt >= GroveHurt {
			if sp, ok := affordable(UseGrove); ok {
				return Action{Kind: Row, Spell: sp.ID, Ally: worst}
			}
		}
		if hurt >= 2 {
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
		spare := func(sp Spell) bool {
			return s.Reserve <= 0 || (s.Mana-sp.Cost)*100 >= s.Reserve*s.MaxMana
		}
		// Phase 39c: a Shaman calls a weather first, when none is up.
		if s.Weather == "" {
			for _, sp := range s.Spells {
				if sp.Use != UseWeather || s.Knows == nil || !s.Knows(sp.ID) || s.Mana < sp.Cost || !spare(sp) {
					continue
				}
				if s.CanWeather != nil && !s.CanWeather(sp.ID) {
					continue
				}
				return Action{Kind: Weather, Spell: sp.ID}
			}
		}
		// Phase 38b: a Theurgist wards the company before it casts; Phase
		// 39c: an Earthspeaker turns Stoneskin on an ally.
		if act, ok := tryBuffs(s, affordable, spare, UseWard, UseBark); ok {
			return act
		}
		// Phase 38c3: a Warlock's Life Drain, while an ally is hurt enough to
		// want it (a drain heals the most hurt ally).
		if sp, ok := affordable(UseSiphon); ok && spare(sp) {
			for _, a := range s.Allies {
				if a.HP >= 1 && !a.Pending && a.HP*1000 < DrainBelow*a.MaxHP {
					return Action{Kind: Drain, Spell: sp.ID, Ally: -1}
				}
			}
		}
		// Phase 39c: Lightning, the Shaman's heavy bolt.
		if sp, ok := affordable(UseStorm); ok {
			return Action{Kind: Storm, Spell: sp.ID}
		}
		// Phase 38d: the Sorcerer's Arcane Lance, while its mana lasts.
		// Phase 84: against a crowd the Sorcerer sparks the group instead,
		// as the plain wizard does.
		if sp, ok := affordable(UseBurst); ok && LanceFits(s.LanceFoes, s.Foes) {
			return Action{Kind: Attack, Spell: sp.ID}
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

// LanceFits (Phase 84) reports whether an Arcane Lance suits this many foes
// for a caster whose limit is lanceFoes (0: any number). The Lance is a
// single bolt, so a Sorcerer that loosed it at a full group spent a two-round
// chant on one foe while the plain wizard's Shower of Sparks hit them all:
// it read 12 points under the wizard at level 50. Limited to three foes it
// is level with the wizard at levels 25, 40 and 50 (docs/plans/
// 2026-10-07-phase-84-sorcerer-balance.md).
func LanceFits(lanceFoes, foes int) bool {
	return lanceFoes <= 0 || foes <= lanceFoes
}

// SummonFoes is how many foes make a battle worth a summon (faith routes
// design: three or more, or a boss).
const SummonFoes = 3

// BigHealBelow and RejuvAbove are the shares of health (in thousandths) under
// which a healer reaches for its heavy heal, and from which it prefers a
// heal over time (Phase 38b).
const (
	// GroveHurt is how many hurt allies make a Grove's long chant worth it
	// (Phase 38c1 review: two, now that a Grove chants 2 rounds and blooms).
	GroveHurt = 2
	// GroveBelow (thousandths of health) is the scratch an idle Elder Druid
	// sows a Grove for (Phase 38c1 review).
	GroveBelow = 850
	// DrainBelow (thousandths of health) is the wound a Warlock's Life Drain
	// is worth casting for (Phase 38c3).
	DrainBelow   = 750
	BigHealBelow = 400
	RejuvAbove   = 350
	// CureAbove (thousandths of health) is the share of health the most
	// hurt ally must still have for an Alchemist to turn to an antidote
	// before a heal (Phase 39g).
	CureAbove = 500
)

// FlaskKeep is the flasks an Alchemist holds back from its fire, for heals
// (Phase 39g).
const FlaskKeep = 2

// idleHealer is what a healer does when no one needs healing: while foes
// stand, put up a class buff on someone without it, hobble a foe, or drain
// one; otherwise swing. Each keeps the mana reserve.
func idleHealer(s Situation, affordable func(Use) (Spell, bool)) Action {
	if s.Foes < 1 {
		return Action{Kind: Swing}
	}
	if act, ok := tryCure(s, affordable); ok {
		return act
	}
	spare := func(sp Spell) bool {
		return s.Reserve <= 0 || (s.Mana-sp.Cost)*100 >= s.Reserve*s.MaxMana
	}
	// Phase 38c1 review: a Grove is a heal over time, so an Elder Druid
	// sows it before anyone is in danger, on the most hurt of two or more
	// scratched allies who carry no Rejuvenation.
	if sp, ok := affordable(UseGrove); ok && spare(sp) {
		scratched, worst := 0, -1
		for i, a := range s.Allies {
			if a.HP < 1 || a.Pending || a.Rejuv || a.HP*1000 >= GroveBelow*a.MaxHP {
				continue
			}
			scratched++
			if worst < 0 || fraction(a.HP, a.MaxHP) < fraction(s.Allies[worst].HP, s.Allies[worst].MaxHP) {
				worst = i
			}
		}
		if scratched >= 2 {
			return Action{Kind: Row, Spell: sp.ID, Ally: worst}
		}
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
	// Phase 39g: an Alchemist with flasks to spare throws fire.
	if sp, ok := affordable(UseFlame); ok {
		return Action{Kind: Flame, Spell: sp.ID}
	}
	return Action{Kind: Swing}
}

// tryCure is an antidote at the first ally that carries poison or bleeding,
// when the character has one (Phase 39g).
func tryCure(s Situation, affordable func(Use) (Spell, bool)) (Action, bool) {
	sp, ok := affordable(UseCure)
	if !ok {
		return Action{}, false
	}
	for i, a := range s.Allies {
		if a.HP >= 1 && a.Afflicted && !a.Pending {
			return Action{Kind: Buff, Spell: sp.ID, Ally: i}, true
		}
	}
	return Action{}, false
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
