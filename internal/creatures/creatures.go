// Package creatures holds the family rules of creature recruits (Phase 38e):
// companions that are not people. A creature takes one company slot and one
// formation cell like any companion, but its family decides how it is kept:
// what it eats, how it recovers, whether it can lose heart, and what it can
// carry. The package is GoMud-free; modules and hooks ask it by archetype id
// (a creature's archetype is its species, set when it is recruited).
//
// Two families ship as the pilot. A Hound is a biological creature: it eats,
// drinks and tires, heals as any companion does, and can lose faith in a
// company it disagrees with. A Stone Golem is a construct: it needs no food,
// drink or rest, never loses heart (it is bound, not loyal), does not mend
// on its own, and is repaired with stone mortar instead.
package creatures

import "strings"

// Kind is how a family is kept.
type Kind string

const (
	// Biological creatures eat, drink and tire, mend on their own and in
	// camp, and can lose loyalty like any companion.
	Biological Kind = "biological"
	// Construct creatures need no food, drink or rest, never lose heart
	// and never mend on their own: they are repaired.
	Construct Kind = "construct"
)

// Hound and StoneGolem are the pilot species' archetype ids.
const (
	Hound      = "hound"
	StoneGolem = "stone-golem"
)

// RepairItemID is the item a repair spends: stone mortar (one use).
const RepairItemID = 290

// RepairPct is the share of a golem's maximum health one mortar restores.
const RepairPct = 50

// Family is one creature species' upkeep rules.
type Family struct {
	ID   string
	Name string
	Kind Kind
	// Role is the one line a recruiter listing and `company inspect` give.
	Role string
	// Gear names what the creature may wear, for the listing and help.
	Gear string
	// Innate names its own attack, for the listing and help.
	Innate string
}

var families = map[string]Family{
	Hound: {
		ID: Hound, Name: "Hound", Kind: Biological,
		Role:   "a pursuit hunter: it runs down foes that are hurt, exposed or knocked down",
		Gear:   "a collar and a harness",
		Innate: "two bites a turn (1d4 each)",
	},
	StoneGolem: {
		ID: StoneGolem, Name: "Stone Golem", Kind: Construct,
		Role:   "a formation anchor: slow, heavily armored, and it shelters the row it stands in",
		Gear:   "nothing but its own stone",
		Innate: "a stone fist (2d5)",
	},
}

// ForArchetype is the family of a creature's archetype id.
func ForArchetype(archetype string) (Family, bool) {
	f, ok := families[strings.ToLower(strings.TrimSpace(archetype))]
	return f, ok
}

// Is reports whether an archetype id names a creature species.
func Is(archetype string) bool {
	_, ok := ForArchetype(archetype)
	return ok
}

// All lists the families in id order.
func All() []Family {
	return []Family{families[Hound], families[StoneGolem]}
}

// NeedsUpkeep is whether the family eats, drinks and tires.
func (f Family) NeedsUpkeep() bool { return f.Kind == Biological }

// Bound is whether the family is held by bond rather than loyalty: it never
// drifts, never loses heart and never deserts, and passes no judgement on
// the company's deeds.
func (f Family) Bound() bool { return f.Kind == Construct }

// Repaired is whether the family is repaired with mortar rather than mended
// by rest and time.
func (f Family) Repaired() bool { return f.Kind == Construct }

// MendsOnItsOwn is whether the family heals between fights on the idle beat
// and from a camp rest or an inn stay.
func (f Family) MendsOnItsOwn() bool { return f.Kind == Biological }

// KindOf is the creature kind of an archetype id, or "" for an ordinary
// companion.
func KindOf(archetype string) Kind {
	f, _ := ForArchetype(archetype)
	return f.Kind
}

// RepairedHealth is a repair's result: the health a golem has after one
// mortar, never above its limit and never lowering it.
func RepairedHealth(health, limit, healthMax int) int {
	if limit < 1 || healthMax < 1 {
		return health
	}
	got := health + max(1, healthMax*RepairPct/100)
	if got > limit {
		got = limit
	}
	return max(got, health)
}

// CanWear is whether a wearer of an archetype may wear gear cut for the given
// species (an item's `wornby`). A creature wears only gear cut for its own
// species; anyone else may not wear gear cut for a creature. The reason is
// written for the one who asked to equip it.
func CanWear(archetype string, wornBy []string) (bool, string) {
	f, creature := ForArchetype(archetype)
	cut := len(wornBy) > 0
	switch {
	case creature && !cut:
		return false, "A " + strings.ToLower(f.Name) + " wears only gear cut for it: " + f.Gear + "."
	case creature:
		for _, who := range wornBy {
			if strings.ToLower(strings.TrimSpace(who)) == f.ID {
				return true, ""
			}
		}
		return false, "That is cut for another kind of creature, not a " + strings.ToLower(f.Name) + "."
	case cut:
		return false, "That is cut for a creature, not for a person."
	}
	return true, ""
}
