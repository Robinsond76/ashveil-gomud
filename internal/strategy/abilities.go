package strategy

import (
	"strconv"
	"strings"
)

// Phase 33e: automatic class abilities. A member's archetype (a companion)
// or trained skill (a player) unlocks an ability its turn uses on its own
// when the ability's condition holds, as spells are cast by role. Pure
// rules here; internal/hooks resolves them in the combat round.

// Ability is an automatic class ability.
type Ability string

const (
	// Tackle (warrior): knock its foe down; the whole turn.
	Tackle Ability = "tackle"
	// OpeningStrike (rogue): the round's first blow that lands on a foe
	// that is down, stunned, staggered, or exposed is a critical hit.
	OpeningStrike Ability = "opening-strike"
	// AimedShot (ranger): the round's first shot that lands is a critical
	// hit (and so leaves the foe exposed).
	AimedShot Ability = "aimed-shot"
)

// AbilitySpec is an ability's unlock, cooldown, and text.
type AbilitySpec struct {
	ID        Ability
	Name      string
	Archetype string // a companion of this archetype has it
	Skill     string // a player with this skill (level 1+) has it
	Cooldown  int    // combat rounds before it can be used again
	When      string // its condition, for the strategy command
	Does      string // its effect
}

// Abilities is every ability, in the order a member tries them.
var Abilities = []AbilitySpec{
	{ID: Tackle, Name: "Tackle", Archetype: "warrior", Skill: "brawling", Cooldown: 4,
		When: "its foe is on its feet and it fights hand to hand",
		Does: "knocks the foe down (it loses its next action), breaking a chant or wind-up; the whole turn"},
	{ID: OpeningStrike, Name: "Opening Strike", Archetype: "rogue", Skill: "skulduggery", Cooldown: 2,
		When: "its foe is knocked down, stunned, staggered, or exposed, and it wields a blade or claws",
		Does: "its first blow that lands this round is a critical hit"},
	{ID: AimedShot, Name: "Aimed Shot", Archetype: "ranger", Skill: "track", Cooldown: 3,
		When: "it has a shooting weapon and its foe is not already exposed",
		Does: "its first shot that lands this round is a critical hit, leaving the foe exposed"},
}

// SpecOf is an ability's spec.
func SpecOf(id Ability) (AbilitySpec, bool) {
	for _, a := range Abilities {
		if a.ID == id {
			return a, true
		}
	}
	return AbilitySpec{}, false
}

// CompanionAbilities are the abilities a companion of an archetype has.
func CompanionAbilities(archetype string) []Ability {
	archetype = strings.ToLower(strings.TrimSpace(archetype))
	var out []Ability
	for _, a := range Abilities {
		if archetype != "" && a.Archetype == archetype {
			out = append(out, a.ID)
		}
	}
	return out
}

// PlayerAbilities are the abilities a player has: each one whose skill
// they have trained.
func PlayerAbilities(skillLevel func(skill string) int) []Ability {
	var out []Ability
	for _, a := range Abilities {
		if skillLevel != nil && skillLevel(a.Skill) > 0 {
			out = append(out, a.ID)
		}
	}
	return out
}

// Names are the abilities' names, in order.
func Names(list []Ability) []string {
	out := make([]string, 0, len(list))
	for _, id := range list {
		if spec, ok := SpecOf(id); ok {
			out = append(out, spec.Name)
		}
	}
	return out
}

// WeaponKind is what a member strikes with, for an ability's condition.
type WeaponKind int

const (
	Unarmed  WeaponKind = iota // fists or claws of its own
	Melee                      // a hand-to-hand weapon
	Shooting                   // a shooting weapon
)

// AbilitySituation is what an ability is chosen from, for one member's
// turn. The caller has already checked the member may act at all (alive,
// aimed at a standing foe of its battle, not chanting or waiting).
type AbilitySituation struct {
	Known []Ability
	// Off is the member's strategy turning abilities off.
	Off bool
	// Ready reports whether an ability's cooldown is spent.
	Ready func(Ability) bool
	// Weapon is what it strikes with; Backstab is true when it wields a
	// weapon and every wielded weapon is a backstab kind.
	Weapon   WeaponKind
	Backstab bool
	// The foe's live statuses.
	FoeDown, FoeStunned, FoeStaggered, FoeExposed bool
}

// DecideAbility is the ability a member uses this turn, if any: the first
// it knows, is ready, and whose condition holds.
func DecideAbility(s AbilitySituation) (Ability, bool) {
	if s.Off {
		return "", false
	}
	for _, id := range s.Known {
		if s.Ready != nil && !s.Ready(id) {
			continue
		}
		switch id {
		case Tackle:
			if s.Weapon != Shooting && !s.FoeDown && !s.FoeStunned {
				return id, true
			}
		case OpeningStrike:
			if s.Backstab && (s.FoeDown || s.FoeStunned || s.FoeStaggered || s.FoeExposed) {
				return id, true
			}
		case AimedShot:
			if s.Weapon == Shooting && !s.FoeExposed {
				return id, true
			}
		}
	}
	return "", false
}

// TackleChance is a tackle's chance in 100: the tackler's Speed against
// the foe's Perception, plus 20, held to 20–80 (the manual command's
// formula).
func TackleChance(speed, foePerception int) int {
	c := speed - foePerception + 20
	if c < 20 {
		return 20
	}
	if c > 80 {
		return 80
	}
	return c
}

// MaxReserve is the highest mana reserve, in percent.
const MaxReserve = 90

// ParseReserve reads a mana reserve: a whole percent, 0 to MaxReserve
// ("30" or "30%").
func ParseReserve(s string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(s), "%"))
	if err != nil || n < 0 || n > MaxReserve {
		return 0, false
	}
	return n, true
}
