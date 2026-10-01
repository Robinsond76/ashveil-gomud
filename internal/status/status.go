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
)

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
}

// Ids lists every status's buff id.
func Ids() []int {
	return []int{Bleeding, Staggered, KnockedDown, ArmorBroken, Exposed, Burning, Overloaded, Stunned, Hobbled}
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

// Grounded reports whether c is knocked down or stunned now (a live
// status): a guardian so held can't step in (Phase 30c2).
func Grounded(c *characters.Character) bool {
	grounded := false
	each(c, func(b *buffs.Buff, s *Spec) {
		if !b.Expired() && (s.Id == KnockedDown || s.Id == Stunned) {
			grounded = true
		}
	})
	return grounded
}
