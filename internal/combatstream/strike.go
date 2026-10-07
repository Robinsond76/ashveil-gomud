package combatstream

import (
	"fmt"
	"strings"
)

// Phase 62: battle lines that explain themselves. A Strike is one weapon
// strike's roll with its parts, recorded by the combat engine as it
// resolves the strike (never recomputed afterwards), so the breakdown a
// player reads is the engine's own numbers.

// Strike is one weapon strike of an attack round.
type Strike struct {
	// Chance is the chance in 100 the strike was rolled against after
	// every modifier; Base is the chance before Modifier (darkness, a
	// second weapon) and Bonus (company chemistry) moved it. Roll is what
	// was rolled, 0-99: the strike landed when Roll < Chance. Roll is -1
	// and Chance 0 for a strike that needed no roll (see Auto).
	Chance   int
	Base     int
	Modifier int
	Bonus    int
	Roll     int
	Hit      bool
	// Auto names why a strike needed no roll: "perfect shot", or
	// "harmless" for a body with no natural weapon.
	Auto string

	// Defense is what stopped a strike that hit (Blocked, Parried or
	// Dodged, "" for none) and DefenseChance the chance it was rolled at.
	Defense       string
	DefenseChance int

	// Quality is "glancing", "solid" or "telling" for a landed blow.
	Quality string
	// Rolled is the weapon's dice and bonuses, Raw the damage once the
	// quality, crit and named modifiers had acted, Armor the armor rating
	// the blow met, Reduced what armor and other shields took off in all
	// and Damage what got through.
	Rolled  int
	Raw     int
	Armor   int
	Reduced int
	Damage  int
	Crit    bool

	// Notes name the modifiers that acted, in plain words ("backstab",
	// "sharpened edge +2").
	Notes []string
}

// Explain turns a strike into short plain lines: what it needed, what
// stopped or shaped it, and what it did.
func (s Strike) Explain() []string {
	var out []string
	switch {
	case s.Auto != "":
		out = append(out, "No roll needed ("+s.Auto+").")
	case s.Roll >= 0:
		line := fmt.Sprintf("To hit: %d in 100, rolled %d (%s)", s.Chance, s.Roll+1, landedWord(s.Hit))
		if parts := s.chanceParts(); parts != "" {
			line += ". " + parts
		}
		out = append(out, line+".")
	}
	if !s.Hit {
		if s.Auto == "harmless" {
			out = append(out, "It has nothing to hurt with.")
		}
		return out
	}
	if s.Defense != "" {
		line := fmt.Sprintf("Then it was %s, a %d in 100 chance", s.Defense, s.DefenseChance)
		return append(out, line+"; no damage.")
	}
	if s.Defense == "" && s.DefenseChance > 0 {
		out = append(out, fmt.Sprintf("Defence: %d in 100 to turn it aside, and it was not.", s.DefenseChance))
	}
	if s.Quality != "" && s.Quality != "solid" {
		out = append(out, "A "+s.Quality+" blow.")
	}
	if s.Crit {
		out = append(out, "A critical hit.")
	}
	if s.Raw > 0 || s.Damage > 0 {
		line := fmt.Sprintf("Damage: %d before armor", s.Raw)
		if s.Armor > 0 {
			line += fmt.Sprintf(", armor %d", s.Armor)
		}
		if s.Reduced > 0 {
			line += fmt.Sprintf(" took %d", s.Reduced)
		}
		out = append(out, fmt.Sprintf("%s, %d got through.", line, s.Damage))
	}
	for _, n := range s.Notes {
		out = append(out, n+".")
	}
	return out
}

func landedWord(hit bool) string {
	if hit {
		return "it landed"
	}
	return "it missed"
}

// chanceParts says what moved the chance off its base.
func (s Strike) chanceParts() string {
	parts := []string{fmt.Sprintf("skill and speed gave %d", s.Base)}
	if s.Modifier != 0 {
		parts = append(parts, fmt.Sprintf("%+d from darkness or a second weapon", s.Modifier))
	}
	if s.Bonus != 0 {
		parts = append(parts, fmt.Sprintf("%+d from company chemistry", s.Bonus))
	}
	if len(parts) == 1 {
		return ""
	}
	return strings.Join(parts, ", ")
}

// Headline is a strike round's one-line cause for a log, "missed, 38 in
// 100" or "dodged, 31 in 100", for the line's tooltip or a compact log.
func Headline(strikes []Strike) string {
	if len(strikes) == 0 {
		return ""
	}
	parts := make([]string, 0, len(strikes))
	for _, s := range strikes {
		switch {
		case s.Auto != "":
			parts = append(parts, s.Auto)
		case !s.Hit:
			parts = append(parts, fmt.Sprintf("missed, %d in 100", s.Chance))
		case s.Defense != "":
			parts = append(parts, fmt.Sprintf("%s, %d in 100", s.Defense, s.DefenseChance))
		default:
			parts = append(parts, fmt.Sprintf("%d damage after armor %d", s.Damage, s.Armor))
		}
	}
	return strings.Join(parts, "; ")
}
