// Package flasks is the Alchemist's satchel (Phase 39g). An Alchemist heals
// and strengthens its company from flasks, not from prayer or mana: each
// flask is a spell that costs one flask (spells.SpellData.Flask) instead of
// mana, and a flask thrown is used up. The satchel's size grows with level
// and rank; what is spent is recorded on the character
// (Character.FlasksSpent), so a new Alchemist starts with a full satchel and
// a restart or copyover keeps what was thrown. A camp rest (or the brew
// command) refills it from reagents, a pack item, one reagent a flask. None
// of it advances world time.
package flasks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// Lineage is the archetype id of the Alchemist.
const Lineage = "alchemist"

// ReagentItemID is the reagent: what a flask is brewed from.
const ReagentItemID = 95

// Satchel numbers: six flasks at level 1, one more every three levels (16
// at level 30).
const (
	BaseCapacity   = 6
	LevelsPerFlask = 3
	// Keep is the flasks an Alchemist holds back from its fire: it throws
	// a Fire Flask only while more than this many remain, so a heal is
	// never left without a flask.
	Keep = 2
)

// IsAlchemist reports whether the character is an Alchemist.
func IsAlchemist(c *characters.Character) bool {
	return c != nil && c.ArchetypeID() == Lineage
}

// Capacity is how many flasks the character's satchel holds: 0 for anyone
// but an Alchemist.
func Capacity(c *characters.Character) int {
	if !IsAlchemist(c) {
		return 0
	}
	n := BaseCapacity + max(c.Level, 1)/LevelsPerFlask
	if fx := c.ClassEffects(); fx != nil {
		n += fx.Int(classes.FlaskCap)
	}
	return n
}

// CapacityAt is a satchel's size at a level, with no rank or talent.
func CapacityAt(level int) int { return BaseCapacity + max(level, 1)/LevelsPerFlask }

// Remaining is the flasks the character still carries.
func Remaining(c *characters.Character) int {
	return max(0, Capacity(c)-c.FlasksSpent)
}

// Missing is how many flasks the satchel lacks.
func Missing(c *characters.Character) int {
	if !IsAlchemist(c) {
		return 0
	}
	return min(max(0, c.FlasksSpent), Capacity(c))
}

// Spend uses up one flask, false when the satchel is empty.
func Spend(c *characters.Character) bool {
	if Remaining(c) < 1 {
		return false
	}
	c.FlasksSpent++
	return true
}

// Reagents counts the reagents a character carries.
func Reagents(c *characters.Character) int {
	n := 0
	for _, itm := range c.Items {
		if itm.ItemId == ReagentItemID {
			n++
		}
	}
	return n
}

// TakeReagents removes up to n reagents from a character's pack and returns
// the ones taken.
func TakeReagents(c *characters.Character, n int) []items.Item {
	var taken []items.Item
	for i := len(c.Items) - 1; i >= 0 && len(taken) < n; i-- {
		if c.Items[i].ItemId == ReagentItemID {
			taken = append(taken, c.Items[i])
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
		}
	}
	return taken
}

// Brewing is one Alchemist's refill: how many flasks it brewed.
type Brewing struct {
	Char   *characters.Character
	Brewed int
}

// Brew refills each Alchemist's satchel from the leader's reagents, one
// reagent a flask, the first Alchemist in the list first. It returns the
// reagents taken from the leader's pack and how many flasks each Alchemist
// brewed (zero for one that needed none or got none).
func Brew(leader *characters.Character, chars []*characters.Character) ([]items.Item, []Brewing) {
	have := Reagents(leader)
	out := make([]Brewing, len(chars))
	used := 0
	for i, c := range chars {
		out[i].Char = c
		// A satchel that shrank (a rank or talent changed) is never owed
		// more than it holds.
		if IsAlchemist(c) {
			c.FlasksSpent = min(c.FlasksSpent, Capacity(c))
		}
		n := min(Missing(c), have-used)
		if n < 1 {
			continue
		}
		used += n
		c.FlasksSpent -= n
		out[i].Brewed = n
	}
	if used == 0 {
		return nil, out
	}
	return TakeReagents(leader, used), out
}

// NeedsBrewing reports whether any of the characters' satchels lack flasks.
func NeedsBrewing(chars ...*characters.Character) bool {
	for _, c := range chars {
		if Missing(c) > 0 {
			return true
		}
	}
	return false
}

// Satchel is a satchel's state in a short line: "5 of 8 flasks".
func Satchel(c *characters.Character) (have, size int) {
	return Remaining(c), Capacity(c)
}
