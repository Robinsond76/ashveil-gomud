// Package status is Ashveil's combat status effects (Phase 30a): bleeding,
// stagger, knockdown, and the rest. A status is an ordinary buff whose
// duration is counted in combat rounds: the combat loop calls Tick once per
// combat round, and nothing here ever advances a game round or the world
// clock. The package holds the rules and the text; internal/hooks wires it
// into the combat round.
package status

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// Buff ids of the statuses (_datafiles/world/default/buffs).
const (
	Bleeding    = 1100
	Staggered   = 1101
	KnockedDown = 1102
	ArmorBroken = 1103
	Exposed     = 1104
	Burning     = 1105
	Overloaded  = 1106
	Stunned     = 1107
	Hobbled     = 1108
	Asleep      = 1109 // Phase 38a: the Witch's hexes
	Paralyzed   = 1110
	Blighted    = 1111
	// Phase 43b: weapon poisons (items.Poisons), one at a time per victim.
	Bitterleaf = 1120
	Leechbane  = 1121
	Leadroot   = 1122
	Mirethorn  = 1123
)

// Buff flags the statuses carry.
const (
	FlagCombatStatus    = "combat-status"     // ticked by the combat round, cleared at fight end
	FlagLoseActions     = "lose-actions"      // loses every action while alive
	FlagLoseFirstAction = "lose-first-action" // loses its next action only
	FlagArmorBroken     = "armor-broken"
	FlagExposed         = "exposed"
	FlagNoDodge         = "no-dodge" // no active defense at all: block, parry, or dodge (stunned)
	FlagNoBlock         = "no-block" // can't block with a shield (stunned)
	FlagAsleep          = "asleep"   // blows against it hit more often; damage wakes it (38a)
	FlagBlighted        = "blighted" // healing it receives is halved (38a)
	FlagPoison          = "poison"   // curepoison and cleansing take it off
	FlagWeaponPoison    = "weapon-poison"
	FlagLeechbane       = "leechbane" // healing it receives is cut by a quarter (43b)
	FlagLeadroot        = "leadroot"  // physical damage it deals is cut by 15% (43b)
	FlagMirethorn       = "mirethorn" // its dodge chance is down 10 points (43b)
)

// AsleepHitBonus is the points added to the chance to hit a sleeper.
const AsleepHitBonus = 25

// ExposedCritBonus is the points added to the crit chance against an
// exposed target.
const ExposedCritBonus = 25

// Spec is one status's rules and text. Lines are tagged names' templates:
// the *You forms are second person, the *Other forms take the holder's name
// as their one %s.
type Spec struct {
	Id     int
	Word   string // as the hit line and tick parentheses name it
	Damage int    // per stack, per tick

	TickYou, TickOther string // a damaging tick
	LoseYou, LoseOther string // a lost action
	EndYou, EndOther   string // expiry
}

var specs = map[int]*Spec{
	Bleeding: {Id: Bleeding, Word: "bleeding", Damage: 1,
		TickYou: "You bleed.", TickOther: "%s bleeds.",
		EndYou: "Your bleeding stops.", EndOther: "%s's bleeding stops."},
	Staggered: {Id: Staggered, Word: "staggered",
		LoseYou: "You reel, and lose your action.", LoseOther: "%s reels, and loses the action.",
		EndYou: "You steady yourself.", EndOther: "%s steadies."},
	KnockedDown: {Id: KnockedDown, Word: "knocked down",
		LoseYou: "You scramble up off the ground, and lose your action.", LoseOther: "%s scrambles up off the ground, and loses the action.",
		EndYou: "You are back on your feet.", EndOther: "%s is back on their feet."},
	ArmorBroken: {Id: ArmorBroken, Word: "armor broken",
		EndYou: "Your armor hangs together again.", EndOther: "%s's armor hangs together again."},
	Exposed: {Id: Exposed, Word: "exposed",
		EndYou: "You close your guard.", EndOther: "%s closes their guard."},
	Burning: {Id: Burning, Word: "burning", Damage: 2,
		TickYou: "Fire eats at you.", TickOther: "Fire eats at %s.",
		EndYou: "The flames die.", EndOther: "The flames on %s die."},
	Overloaded: {Id: Overloaded, Word: "overloaded",
		EndYou: "The charge drains out of you.", EndOther: "The charge drains out of %s."},
	Stunned: {Id: Stunned, Word: "stunned",
		LoseYou: "You stand stunned, and lose your action.", LoseOther: "%s stands stunned, and loses the action.",
		EndYou: "Your head clears.", EndOther: "%s's head clears."},
	Hobbled: {Id: Hobbled, Word: "hobbled",
		EndYou: "Your legs answer you again.", EndOther: "%s's legs answer again."},
	Asleep: {Id: Asleep, Word: "asleep",
		LoseYou: "You sleep on, and lose your action.", LoseOther: "%s sleeps on, and loses the action.",
		EndYou: "You wake with a start.", EndOther: "%s wakes with a start."},
	Paralyzed: {Id: Paralyzed, Word: "paralyzed",
		LoseYou: "You cannot move, and lose your action.", LoseOther: "%s cannot move, and loses the action.",
		EndYou: "Your limbs answer you again.", EndOther: "%s's limbs answer again."},
	Blighted: {Id: Blighted, Word: "blighted",
		EndYou: "The blight lifts from you.", EndOther: "The blight lifts from %s."},
	Bitterleaf: {Id: Bitterleaf, Word: "bitterleaf", Damage: 1,
		TickYou: "Bitterleaf burns in your veins.", TickOther: "Bitterleaf burns in %s's veins.",
		EndYou: "The bitterleaf runs out of your blood.", EndOther: "The bitterleaf runs out of %s's blood."},
	Leechbane: {Id: Leechbane, Word: "leechbane",
		EndYou: "The leechbane thins out of your blood.", EndOther: "The leechbane thins out of %s's blood."},
	Leadroot: {Id: Leadroot, Word: "leadroot",
		EndYou: "Your arms feel your own again.", EndOther: "%s's arms look their own again."},
	Mirethorn: {Id: Mirethorn, Word: "mirethorn",
		EndYou: "Your feet find themselves again.", EndOther: "%s's feet find themselves again."},
}

// Ids lists every status's buff id.
func Ids() []int {
	return []int{Bleeding, Staggered, KnockedDown, ArmorBroken, Exposed, Burning, Overloaded, Stunned, Hobbled, Asleep, Paralyzed, Blighted, Bitterleaf, Leechbane, Leadroot, Mirethorn}
}

// Get is the status with buff id, or nil for a buff that is not one.
func Get(buffId int) *Spec { return specs[buffId] }

// Word names a status for the hit line ("bleeding"), or "" for a buff that
// is not one.
func Word(buffId int) string {
	if s := specs[buffId]; s != nil {
		return s.Word
	}
	return ""
}

// Change is what one combat round did to one status on a holder.
type Change struct {
	Spec    *Spec
	Damage  int  // health the tick took (0 for a status that does none)
	Stacks  int  // stacks at the tick
	Expired bool // the status ended this tick
}

// alive is a live status buff and its spec.
func each(c *characters.Character, fn func(b *buffs.Buff, s *Spec)) {
	for _, b := range c.Buffs.List {
		if s := specs[b.BuffId]; s != nil {
			fn(b, s)
		}
	}
}

func stacks(b *buffs.Buff) int {
	if b.Stacks < 1 {
		return 1
	}
	return b.Stacks
}

// Tick advances every live status on c by one combat round: its count falls
// by one and its damage (per stack) comes off c's health. A status whose
// count reaches zero is reported Expired and left for the buff pruner.
func Tick(c *characters.Character) []Change {
	if c.CombatWithdrawn {
		return nil
	}
	var out []Change
	each(c, func(b *buffs.Buff, s *Spec) {
		if b.Expired() {
			return
		}
		b.TriggersLeft--
		ch := Change{Spec: s, Stacks: stacks(b), Expired: b.Expired()}
		if s.Damage > 0 {
			ch.Damage = s.Damage * ch.Stacks
			c.Health -= ch.Damage
		}
		out = append(out, ch)
	})
	// An ended status's stat mods (a knockdown's speed) stop now, not at
	// the next prune.
	for _, ch := range out {
		if ch.Expired {
			c.RecalculateStats()
			break
		}
	}
	return out
}

// LostAction reports whether c loses this combat round's action to a status,
// and which: a lose-actions status costs every action while it lasts, a
// lose-first-action status only the first round after it was applied. A
// status costs nothing before its first tick, so a blow that lands this
// round takes the victim's action from the next round, not a half-spent one.
func LostAction(c *characters.Character) (*Spec, bool) {
	var found *Spec
	each(c, func(b *buffs.Buff, s *Spec) {
		if found != nil || b.Expired() {
			return
		}
		spec := buffs.GetBuffSpec(b.BuffId)
		if spec == nil {
			return
		}
		for _, f := range spec.Flags {
			switch {
			case f == FlagLoseActions && b.TriggersLeft < b.TriggersInitial:
				found = s
			case f == FlagLoseFirstAction && b.TriggersLeft == b.TriggersInitial-1:
				found = s
			}
		}
	})
	return found, found != nil
}

// Has reports whether c carries any live status.
func Has(c *characters.Character) bool {
	live := false
	each(c, func(b *buffs.Buff, s *Spec) {
		if !b.Expired() {
			live = true
		}
	})
	return live
}

// Clear ends every status on c at once, and reports how many it ended.
func Clear(c *characters.Character) int {
	var ids []int
	each(c, func(b *buffs.Buff, s *Spec) {
		if !b.Expired() {
			ids = append(ids, b.BuffId)
		}
	})
	for _, id := range ids {
		c.RemoveBuff(id)
	}
	return len(ids)
}

// Roll picks a whole number in [0, n).
type Roll func(n int) int

// CritEffect is the buffs a critical hit with a weapon of subtype puts on
// its target. A weapon's own crit buffs (override) win when it has any.
func CritEffect(subtype items.ItemSubType, override []int, roll Roll) []int {
	if len(override) > 0 {
		return override
	}
	switch subtype {
	case items.Slashing, items.Claws:
		return []int{Bleeding}
	case items.Stabbing:
		return []int{Bleeding, Bleeding} // deep bleeding: two stacks at once
	case items.Bludgeoning:
		return []int{Staggered}
	case items.Cleaving:
		switch r := roll(10); {
		case r < 4:
			return []int{KnockedDown}
		case r < 8:
			return []int{ArmorBroken}
		default:
			return []int{Stunned}
		}
	case items.Shooting:
		return []int{Exposed}
	case items.Whipping:
		return []int{Hobbled}
	}
	return nil
}

// Words names the statuses among buffIds, once each, in order, for the hit
// line's parentheses. A buff that is not a status is left out.
func Words(buffIds []int) []string {
	var out []string
	seen := map[int]bool{}
	for _, id := range buffIds {
		if w := Word(id); w != "" && !seen[id] {
			seen[id] = true
			out = append(out, w)
		}
	}
	return out
}

// Live reports whether c carries the status with buff id now.
func Live(c *characters.Character, id int) bool {
	live := false
	each(c, func(b *buffs.Buff, s *Spec) {
		if s.Id == id && !b.Expired() {
			live = true
		}
	})
	return live
}

// Wake ends a sleeper's sleep at once (Phase 38a: the first damage it takes
// wakes it) and reports whether it was asleep. Paralysis is not broken by
// damage.
func Wake(c *characters.Character) bool {
	if !Live(c, Asleep) {
		return false
	}
	c.RemoveBuff(Asleep)
	return true
}

// Grounded reports whether c is knocked down, stunned, asleep or paralyzed
// now (a live status): a guardian so held can't step in (Phase 30c2, 38a).
func Grounded(c *characters.Character) bool {
	grounded := false
	each(c, func(b *buffs.Buff, s *Spec) {
		if !b.Expired() && (s.Id == KnockedDown || s.Id == Stunned || s.Id == Asleep || s.Id == Paralyzed) {
			grounded = true
		}
	})
	return grounded
}

// cleansable are the statuses a cleansing heal or touch takes off, worst
// first.
var cleansable = []int{Stunned, Paralyzed, Asleep, KnockedDown, Hobbled, Burning, Bleeding, Bitterleaf, Leechbane, Leadroot, Mirethorn, Blighted, ArmorBroken, Exposed, Staggered}

// CleanseOne ends the worst harmful status c carries and returns its word,
// or "" when it has none.
func CleanseOne(c *characters.Character) string {
	for _, id := range cleansable {
		if Live(c, id) {
			c.RemoveBuff(id)
			if s := Get(id); s != nil {
				return s.Word
			}
			return "status"
		}
	}
	return ""
}

// PoisonLive reports whether c carries a live weapon-poison effect (Phase
// 43b): a victim carries one at a time.
func PoisonLive(c *characters.Character) bool {
	for _, id := range []int{Bitterleaf, Leechbane, Leadroot, Mirethorn} {
		if Live(c, id) {
			return true
		}
	}
	return false
}

// IsWeaponPoison reports whether buffId is a weapon poison's status.
func IsWeaponPoison(buffId int) bool {
	return buffId == Bitterleaf || buffId == Leechbane || buffId == Leadroot || buffId == Mirethorn
}
