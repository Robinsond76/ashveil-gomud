// Package classes is the GoMud-free class graph (Phase 38b): the advanced
// and elite classes a character promotes into from its base lineage (the
// archetype it was created with), the ranks each earns every five levels,
// and the talents chosen every ten levels from level 5.
//
// A character's lineage never changes (its archetype). Its class is the
// promoted route on top of it, and everything a class gives is derived from
// the class and the character's current level, never saved: a level lost to
// death removes a rank until it is regained.
//
// The table is code, not config: the numbers are balance values the combat
// code reads by key (see Effects).
package classes

import (
	"fmt"
	"sort"
	"strings"
)

// Tier is how far up a lineage a class stands.
type Tier int

const (
	TierBase     Tier = iota // the archetype a character starts with
	TierAdvanced             // level 10
	TierElite                // level 30
)

// Promotion levels, and the gate an alignment-locked route checks.
const (
	AdvancedLevel = 10
	EliteLevel    = 30
	MaxRankLevel  = 60
	// GateAlignment is how far from neutral a good or evil route needs
	// its individual to stand: +30 or higher, or −30 or lower.
	GateAlignment = 30
)

// TalentLevels are the levels a talent is chosen at.
var TalentLevels = []int{5, 15, 25, 35, 45, 55}

// Gate is the alignment a route asks for, checked at each promotion.
type Gate int

const (
	GateAny  Gate = iota // any alignment
	GateGood             // +30 or higher
	GateEvil             // −30 or lower
)

// Allows reports whether an individual's alignment meets the gate.
func (g Gate) Allows(alignment int) bool {
	switch g {
	case GateGood:
		return alignment >= GateAlignment
	case GateEvil:
		return alignment <= -GateAlignment
	}
	return true
}

// Label is the gate in a player's words.
func (g Gate) Label() string {
	switch g {
	case GateGood:
		return fmt.Sprintf("alignment +%d or higher", GateAlignment)
	case GateEvil:
		return fmt.Sprintf("alignment -%d or lower", GateAlignment)
	}
	return "any alignment"
}

// Rank is one benefit a route gains at a level. Set is applied to the
// route's effects when the level is reached: a key's value replaces an
// earlier rank's, so a table reads as final values (uses per rest: 2, then
// 3, then 4).
type Rank struct {
	Level int
	Name  string
	Text  string
	Set   Effects
	// Spells are the spells the rank teaches: a player learns them at the
	// level (owning the spell is the marker), a companion casts them from
	// it.
	Spells []string
}

// Class is one advanced or elite route.
type Class struct {
	ID      string
	Name    string
	Lineage string // the base archetype it grows from
	Tier    Tier
	Parent  string // the advanced class an elite continues; empty for advanced
	Gate    Gate
	Role    string // one line, for the path listing
	Ranks   []Rank
	// Planned marks a route whose mechanics are not delivered yet: it is
	// listed but cannot be chosen.
	Planned bool
}

// Effects are a character's class benefits by key (the keys are the
// constants in effects.go). A missing key is 0.
type Effects map[string]int

// Int is an effect's value, 0 when absent.
func (e Effects) Int(key string) int { return e[key] }

// Has reports whether an effect is in force.
func (e Effects) Has(key string) bool { return e[key] != 0 }

// Clone copies the effects.
func (e Effects) Clone() Effects {
	out := make(Effects, len(e))
	for k, v := range e {
		out[k] = v
	}
	return out
}

func normalize(id string) string { return strings.ToLower(strings.TrimSpace(id)) }

var (
	byID    = map[string]Class{}
	order   []string
	lineage = map[string][]string{} // lineage -> advanced ids, in table order
)

func register(c Class) {
	c.ID = normalize(c.ID)
	if _, dup := byID[c.ID]; dup {
		panic("classes: duplicate class " + c.ID)
	}
	byID[c.ID] = c
	order = append(order, c.ID)
	if c.Tier == TierAdvanced {
		lineage[c.Lineage] = append(lineage[c.Lineage], c.ID)
	}
}

// Get returns a class by id.
func Get(id string) (Class, bool) {
	c, ok := byID[normalize(id)]
	return c, ok
}

// All lists every class in table order.
func All() []Class {
	out := make([]Class, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return out
}

// Lineages lists the base archetypes that have routes, sorted.
func Lineages() []string {
	out := make([]string, 0, len(lineage))
	for l := range lineage {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}

// Advanced lists a lineage's advanced classes in table order.
func Advanced(lineageID string) []Class {
	var out []Class
	for _, id := range lineage[normalize(lineageID)] {
		out = append(out, byID[id])
	}
	return out
}

// Elite is the elite class an advanced class continues into.
func Elite(advancedID string) (Class, bool) {
	advancedID = normalize(advancedID)
	for _, id := range order {
		if c := byID[id]; c.Tier == TierElite && c.Parent == advancedID {
			return c, true
		}
	}
	return Class{}, false
}

// Path is the classes a character on a route has been through, base side
// first: the advanced class, then the elite one.
func Path(id string) []Class {
	c, ok := Get(id)
	if !ok {
		return nil
	}
	if c.Tier == TierElite {
		if parent, ok := Get(c.Parent); ok {
			return []Class{parent, c}
		}
	}
	return []Class{c}
}

// LineageOf is the base archetype a class grows from.
func LineageOf(id string) (string, bool) {
	c, ok := Get(id)
	return c.Lineage, ok
}

// Effects are the class benefits in force for a character of a class at a
// level, with the talents it has chosen: every rank of its route up to the
// level (an elite class carries its advanced class's), then the talents'
// additive effects. Only as many talents as the level has earned count.
func EffectsFor(classID string, level int, talentIDs []string) Effects {
	return EffectsForLineage("", classID, level, talentIDs)
}

// RanksReached are the route's ranks a character at a level has earned,
// in level order, across its whole path.
func RanksReached(classID string, level int) []Rank {
	var out []Rank
	for _, c := range Path(classID) {
		for _, r := range c.Ranks {
			if level >= r.Level {
				out = append(out, r)
			}
		}
	}
	return out
}

// NextRank is the next rank a character of a class has yet to reach, if
// any.
func NextRank(classID string, level int) (Rank, bool) {
	for _, c := range Path(classID) {
		for _, r := range c.Ranks {
			if level < r.Level {
				return r, true
			}
		}
	}
	return Rank{}, false
}

// SpellsAt are the spells a character of a class knows at a level from its
// route, across its whole path.
func SpellsAt(classID string, level int) []string {
	var out []string
	for _, r := range RanksReached(classID, level) {
		out = append(out, r.Spells...)
	}
	return out
}

// SpellLocked reports whether a spell is one the class's route teaches at a
// rank the character's level has not reached (Phase 38b review): a player
// keeps a rank spell in the spellbook after a death costs the level, but
// can't cast it until the level is regained.
func SpellLocked(classID string, level int, spellID string) bool {
	if classID == "" {
		return false
	}
	taught := false
	for _, c := range Path(classID) {
		for _, r := range c.Ranks {
			for _, id := range r.Spells {
				if id != spellID {
					continue
				}
				if level >= r.Level {
					return false
				}
				taught = true
			}
		}
	}
	return taught
}
