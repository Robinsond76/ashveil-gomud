// Package strategy holds Ashveil's battle strategies (Phase 32d): the role
// and target rule each character (a player, or one of their companions)
// fights by once a battle starts. A battle plays out on its own (the
// owner's rule 5); strategies are how the player sets it up beforehand.
//
// This package is GoMud-free: target rules pick among plain Foe values
// (Pick), roles decide an action from a plain Situation (Decide), and the
// durable settings live in modules/strategy behind the Provider seam.
package strategy

import (
	"errors"
	"strings"
)

// Role is what a character does in a battle.
type Role string

const (
	Fighter Role = "fighter" // swings at its aim
	Healer  Role = "healer"  // heals anyone below the healing threshold, else swings
	Caster  Role = "caster"  // casts its attack spell while mana lasts, else swings
	// Guardian swings as a fighter, and steps in to take a blow meant for
	// its ward (Phase 30c2). Never a default.
	Guardian Role = "guardian"
	// Controller hexes foes to take their turns away (Phase 38a: the
	// Witch), then casts its weak damage curse, else swings.
	Controller Role = "controller"
)

// Roles in the order they are listed.
var Roles = []Role{Fighter, Healer, Caster, Guardian, Controller}

// Rule is how a character picks its target.
type Rule string

const (
	Weakest   Rule = "weakest"   // the least health left (Ogre Battle's Weak)
	Strongest Rule = "strongest" // the most health left (Ogre Battle's Strong)
	Wounded   Rule = "wounded"   // the lowest health fraction (FF12's "HP lowest")
	Nearest   Rule = "nearest"   // the front-most (FF12's "nearest")
	Furthest  Rule = "furthest"  // the back-most (FF12's "furthest")
	Leader    Rule = "leader"    // the group's leader, its toughest (Ogre Battle's Leader)
	Casters   Rule = "casters"   // spell-casters, one chanting first (Phase 30c)
	Assist    Rule = "assist"    // the player's own target (FF12's "party leader's target")
	Defend    Rule = "defend"    // the foe striking our most hurt (FF12's "foe targeting ally")
)

// Rules in the order they are listed.
var Rules = []Rule{Weakest, Strongest, Wounded, Nearest, Furthest, Leader, Casters, Assist, Defend}

var roleAliases = map[string]Role{
	"fight": Fighter, "fighter": Fighter, "melee": Fighter,
	"heal": Healer, "healer": Healer,
	"cast": Caster, "caster": Caster,
	// Phase 30c2 (the owner's decision 10): "guard" is the guardian, no
	// longer the defend rule.
	"guard": Guardian, "guardian": Guardian, "protector": Guardian,
	"control": Controller, "controller": Controller, "hex": Controller, "hexer": Controller,
}

var ruleAliases = map[string]Rule{
	"weak": Weakest, "strong": Strongest, "hurt": Wounded,
	"near": Nearest, "front": Nearest, "far": Furthest, "back": Furthest,
	"focus": Assist, "protect": Defend,
	"mages": Casters, "spellcasters": Casters,
}

// ParseRole reads a role or its alias.
func ParseRole(s string) (Role, bool) {
	r, ok := roleAliases[strings.ToLower(strings.TrimSpace(s))]
	return r, ok
}

// ParseRule reads a target rule or its alias.
func ParseRule(s string) (Rule, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	for _, r := range Rules {
		if string(r) == s {
			return r, true
		}
	}
	r, ok := ruleAliases[s]
	return r, ok
}

// Strategy is one character's role and target rule. A blank field means
// the character's default (Default), so only a change needs storing.
type Strategy struct {
	Role Role `yaml:"role,omitempty"`
	Rule Rule `yaml:"rule,omitempty"`
	// Ward is the member key a guardian guards (Phase 30c2); blank for
	// the most hurt. Kept only for a guardian.
	Ward string `yaml:"ward,omitempty"`
	// NoAbilities turns the member's automatic class abilities off (Phase
	// 33e); they are on by default.
	NoAbilities bool `yaml:"no_abilities,omitempty"`
	// Reserve is the percent of its maximum mana an attack spell must
	// leave (Phase 33e); heals ignore it. 0 by default.
	Reserve int `yaml:"reserve,omitempty"`
}

// IsZero reports whether nothing is set: the character fights by its
// default.
func (s Strategy) IsZero() bool {
	return s.Role == "" && s.Rule == "" && s.Ward == "" && !s.NoAbilities && s.Reserve == 0
}

// DefaultRole is an archetype's role: a cleric heals, a wizard casts, a
// witch controls, everyone else fights.
func DefaultRole(archetype string) Role {
	switch strings.ToLower(archetype) {
	case "cleric":
		return Healer
	case "wizard":
		return Caster
	case "witch":
		return Controller
	}
	return Fighter
}

// Default is the strategy a character of this archetype fights by until
// it is changed: its archetype's role, and the weakest foe it can reach
// (the rule every fight used before 32d).
func Default(archetype string) Strategy {
	return Strategy{Role: DefaultRole(archetype), Rule: Weakest}
}

// Resolve fills blank fields from the archetype's default.
func (s Strategy) Resolve(archetype string) Strategy {
	d := Default(archetype)
	if s.Role == "" {
		s.Role = d.Role
	}
	if s.Rule == "" {
		s.Rule = d.Rule
	}
	return s
}

// ErrPlayerAssist is returned for a player set to assist: they are the one
// assisted.
var ErrPlayerAssist = errors.New("only a companion can assist you")

// ValidFor checks a strategy for the player (isPlayer) or a companion.
func (s Strategy) ValidFor(isPlayer bool) error {
	if isPlayer && s.Rule == Assist {
		return ErrPlayerAssist
	}
	return nil
}

// ReaimsEachRound reports whether the rule follows something that moves
// (the player's target, or who is striking whom), so it is read again
// every round rather than only when a target falls or can't be reached.
func (r Rule) ReaimsEachRound() bool { return r == Assist || r == Defend }

// Describe says what the rule goes for, for the strategy command.
func (r Rule) Describe() string {
	switch r {
	case Weakest:
		return "the foe with the least health left"
	case Strongest:
		return "the foe with the most health left"
	case Wounded:
		return "the most badly hurt foe, to finish it"
	case Nearest:
		return "the front-most foe"
	case Furthest:
		return "the back-most foe"
	case Leader:
		return "their leader, the toughest of them"
	case Casters:
		return "their spell-casters, one chanting first"
	case Assist:
		return "whatever you are striking"
	case Defend:
		return "the foe striking whichever of you is most hurt"
	}
	return string(r)
}

// Describe says what the role does, for the strategy command.
func (r Role) Describe() string {
	switch r {
	case Fighter:
		return "fights with weapon or fists"
	case Healer:
		return "heals anyone below the healing threshold, else fights"
	case Caster:
		return "casts an attack spell while mana lasts, else fights"
	case Guardian:
		return "fights, and steps in to take a blow meant for its ward"
	case Controller:
		return "hexes foes to take their turns away, then curses or fights"
	}
	return string(r)
}
