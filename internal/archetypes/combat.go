package archetypes

import (
	"fmt"
	"math"
	"strings"
)

// Phase 35a2 (skill over hit points): an archetype's fighting profile.
// Attack and Evasion grow by its rates a level; HPStart is its head start
// in health; ArmorTraining is the heaviest bulk it wears without penalty;
// ShieldSizes and WeaponClasses are hard rules on what it may hold.

// Bulk names, lightest first.
const (
	BulkLight  = "light"
	BulkMedium = "medium"
	BulkHeavy  = "heavy"
)

// Shield sizes, smallest first. ShieldNone in an archetype's list means it
// carries no shield at all.
const (
	ShieldBuckler = "buckler"
	ShieldNormal  = "shield"
	ShieldTower   = "tower"
	ShieldNone    = "none"
)

// BulkRank orders bulk: light (or none) 0, medium 1, heavy 2; -1 for an
// unknown name.
func BulkRank(bulk string) int {
	switch strings.ToLower(strings.TrimSpace(bulk)) {
	case "", BulkLight:
		return 0
	case BulkMedium:
		return 1
	case BulkHeavy:
		return 2
	}
	return -1
}

// ValidShieldSize reports whether a shield size names one of the three.
func ValidShieldSize(size string) bool {
	switch size {
	case ShieldBuckler, ShieldNormal, ShieldTower:
		return true
	}
	return false
}

// Profile is an archetype's fighting profile.
type Profile struct {
	Name          string
	AttackRate    float64
	EvasionRate   float64
	HPStart       int
	ArmorTraining string
	ShieldSizes   []string
	WeaponClasses []string
}

// Gear is what CanWield needs to know about an item, without the engine's
// item types: whether it is a shield (and its size) or a weapon (and its
// class).
type Gear struct {
	Shield      bool
	ShieldSize  string
	Weapon      bool
	WeaponClass string
}

// validateCombat normalizes and checks the Phase 35a2 fields.
func (a *Archetype) validateCombat() error {
	for _, v := range []float64{a.AttackRate, a.EvasionRate} {
		if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("%w: %q has an invalid attack or evasion rate", ErrInvalidArchetype, a.ID)
		}
	}
	if a.HPStart < 0 {
		return fmt.Errorf("%w: %q has a negative HPStart", ErrInvalidArchetype, a.ID)
	}
	a.ArmorTraining = strings.ToLower(strings.TrimSpace(a.ArmorTraining))
	if a.ArmorTraining == "" {
		a.ArmorTraining = BulkHeavy
	}
	if BulkRank(a.ArmorTraining) < 0 {
		return fmt.Errorf("%w: %q armor training %q", ErrInvalidArchetype, a.ID, a.ArmorTraining)
	}
	a.ShieldSizes = normalizeList(a.ShieldSizes)
	for _, s := range a.ShieldSizes {
		if s == ShieldNone {
			if len(a.ShieldSizes) > 1 {
				return fmt.Errorf("%w: %q lists none with other shield sizes", ErrInvalidArchetype, a.ID)
			}
			continue
		}
		if !ValidShieldSize(s) {
			return fmt.Errorf("%w: %q shield size %q", ErrInvalidArchetype, a.ID, s)
		}
	}
	a.WeaponClasses = normalizeList(a.WeaponClasses)
	return nil
}

// Profile is the archetype's fighting profile.
func (a Archetype) Profile() Profile {
	return Profile{
		Name:          a.Name,
		AttackRate:    a.AttackRate,
		EvasionRate:   a.EvasionRate,
		HPStart:       a.HPStart,
		ArmorTraining: a.ArmorTraining,
		ShieldSizes:   append([]string(nil), a.ShieldSizes...),
		WeaponClasses: append([]string(nil), a.WeaponClasses...),
	}
}

// CanWield applies the profile's shield and weapon rules: an empty list
// allows anything; reason is player-facing when refused.
func (p Profile) CanWield(g Gear) (bool, string) {
	plural := p.Name + "s"
	if p.Name == "" {
		plural = "Your class"
	}
	if g.Shield && len(p.ShieldSizes) > 0 {
		size := strings.ToLower(strings.TrimSpace(g.ShieldSize))
		if size == "" {
			size = ShieldNormal
		}
		if !contains(p.ShieldSizes, size) {
			if p.ShieldSizes[0] == ShieldNone {
				return false, plural + " don't carry shields."
			}
			return false, plural + " carry only " + joinPlural(p.ShieldSizes) + "."
		}
	}
	if g.Weapon && len(p.WeaponClasses) > 0 && !contains(p.WeaponClasses, g.WeaponClass) {
		return false, plural + " fight with " + joinPlural(p.WeaponClasses) + "."
	}
	return true, ""
}

// joinPlural is "staffs, rods and maces".
func joinPlural(list []string) string {
	words := make([]string, len(list))
	for i, w := range list {
		words[i] = w + "s"
	}
	if len(words) == 1 {
		return words[0]
	}
	return strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
}

// CombatProvider is optionally implemented by the provider (Phase 35a2).
type CombatProvider interface {
	CombatProfile(archetypeID string) (Profile, bool)
}

// CombatProfile is an archetype's fighting profile; ok is false without a
// provider or for an unknown id.
func CombatProfile(archetypeID string) (Profile, bool) {
	if archetypeID == "" {
		return Profile{}, false
	}
	if cp, ok := current().(CombatProvider); ok {
		return cp.CombatProfile(archetypeID)
	}
	return Profile{}, false
}

// CanWield applies an archetype's shield and weapon rules. An unknown or
// empty archetype (enemies, no provider) may hold anything.
func CanWield(archetypeID string, g Gear) (bool, string) {
	p, ok := CombatProfile(archetypeID)
	if !ok {
		return true, ""
	}
	return p.CanWield(g)
}
