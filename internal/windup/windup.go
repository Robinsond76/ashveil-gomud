// Package windup holds Ashveil's physical wind-ups (Phase 30d2): a heavy
// blow a foe spends turns preparing, in plain view, before it lands. It is
// pure: the combat loop (internal/hooks) keeps who is winding up, rolls
// the dice, and tells the lines; internal/combat resolves the blow.
//
// A wind-up breaks only on heavier force (the owner's rule, 2026-09-30): a
// critical hit that lands, or a blow that staggers, knocks down, or stuns
// (internal/interrupt's BreaksWindUp). Ordinary blows and shield bashes
// never break one.
package windup

import "strings"

// Cooldown is how many of its turns a foe swings normally after a
// wind-up lands, is wasted, or is broken, before it may start another.
const Cooldown = 2

// Ability is one wind-up. Its lines take the tokens {actor} (the foe's
// label), {his} (its possessive), {weapon} (its weapon's simple name),
// and {target} (whom it aims at).
type Ability struct {
	Id         string
	Name       string
	Rounds     int  // turns spent winding up before it lands
	Multiplier int  // the weapon's rolled damage is multiplied, before armor
	KnockDown  bool // a blow that gets through the armor knocks the target down

	Telegraph string // the room, when it starts
	Release   string // the room, as it swings (the swing's own line follows)
	Broken    string // the room, when heavy force breaks it
	Lost      string // the room, when a status not from a blow costs it the turn
	Wasted    string // the room, when nothing is left to strike
}

var abilities = map[string]Ability{
	"crushing-blow": {
		Id:         "crushing-blow",
		Name:       "Crushing Blow",
		Rounds:     1,
		Multiplier: 2,
		KnockDown:  true,
		Telegraph:  "{actor} plants {his} feet and drags {his} {weapon} up over {his} shoulder, eyes on {target}.",
		Release:    "{actor} brings {his} {weapon} down with all {his} weight.",
		Broken:     "{actor} staggers, and the blow dies before it can fall.",
		Wasted:     "{actor} lets {his} {weapon} fall, with nothing left to strike.",
		Lost:       "The blow {actor} was winding up is lost.",
	},
}

// Get returns a registered ability.
func Get(id string) (Ability, bool) {
	a, ok := abilities[id]
	return a, ok
}

// RollStart rolls whether a foe starts a wind-up it has chance percent of
// its turns to start. roll(n) returns 0..n-1 (util.Rand); a certain
// outcome doesn't roll.
func RollStart(chance int, roll func(int) int) bool {
	if chance <= 0 {
		return false
	}
	if chance >= 100 {
		return true
	}
	return roll(100) < chance
}

// Render fills an ability's line.
func Render(line, actor, his, weapon, target string) string {
	return strings.NewReplacer("{actor}", actor, "{his}", his, "{weapon}", weapon, "{target}", target).Replace(line)
}
