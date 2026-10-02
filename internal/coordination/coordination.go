// Package coordination is Ashveil's enemy coordination (Phase 33i2): how
// well an enemy group fights together, by tier. A group's tier comes from
// its members' average level when its battle begins, unless a member's
// template sets one outright (the highest such setting wins). Coordination
// changes only the group's choices: whom it aims at, when its healers
// heal, and how often its guardians step in. Never stats, damage, health,
// actions, or reach.
//
// The package is pure: the combat round (internal/hooks) reads a tier's
// Spec and applies it; the assessment names its Word.
package coordination

import (
	"fmt"
	"strings"
)

// Tier is how well a group fights together, 1 (a rabble) to 4 (a veteran
// company). 0 is no tier: not an enemy group.
type Tier int

const (
	None    Tier = 0
	Rabble  Tier = 1
	Band    Tier = 2
	Drilled Tier = 3
	Veteran Tier = 4
)

// MaxTier is the highest tier.
const MaxTier = Veteran

// Spec is what a tier does in a battle.
type Spec struct {
	Tier Tier
	// Word is how the assessment names it ("a drilled company").
	Word string
	// FocusPct is the percent of a group's fighters (rounded up) who
	// take the group's focus: the foe its leader is aiming at.
	FocusPct int
	// NoiseFloor is the least percent of re-aims that take a random foe;
	// a member's own targeting noise counts when higher.
	NoiseFloor int
	// NoNoise drops all noise, the member's own included.
	NoNoise bool
	// HealBelow is the percent of health under which its healers heal.
	HealBelow int
	// HealsPerRound caps the heals its healers start in a round; 0 is no
	// cap (they still don't double up on one patient).
	HealsPerRound int
	// Guards is the times its guardians may step in, in all, in a battle.
	Guards int
	// CastersFirst has the leader pick the focus by the casters rule when
	// the other side has a caster.
	CastersFirst bool
	// SpellsAtFocus has its casters cast at the focus.
	SpellsAtFocus bool
	// BreakHeals switches the focus to a foe chanting a heal.
	BreakHeals bool
	// Announce says the focus aloud when it changes.
	Announce bool
}

var specs = map[Tier]Spec{
	Rabble:  {Tier: Rabble, Word: "a rabble", FocusPct: 0, NoiseFloor: 30, HealBelow: 30, HealsPerRound: 1, Guards: 0},
	Band:    {Tier: Band, Word: "a band", FocusPct: 50, NoiseFloor: 15, HealBelow: 50, Guards: 1, Announce: true},
	Drilled: {Tier: Drilled, Word: "a drilled company", FocusPct: 100, NoiseFloor: 5, HealBelow: 60, Guards: 2, CastersFirst: true, SpellsAtFocus: true, Announce: true},
	Veteran: {Tier: Veteran, Word: "a veteran company", FocusPct: 100, NoNoise: true, HealBelow: 70, Guards: 2, CastersFirst: true, SpellsAtFocus: true, BreakHeals: true, Announce: true},
}

// SpecOf is a tier's spec. A tier out of range reads as the nearest one.
func SpecOf(t Tier) Spec {
	if t < Rabble {
		t = Rabble
	}
	if t > MaxTier {
		t = MaxTier
	}
	return specs[t]
}

// ForLevel is the tier of a group whose members average level.
func ForLevel(level int) Tier {
	switch {
	case level >= 45:
		return Veteran
	case level >= 25:
		return Drilled
	case level >= 10:
		return Band
	}
	return Rabble
}

// Of is a group's tier: the highest tier its members' templates set
// outright (explicit, 0 for none), else the tier of their average level,
// rounded down. A group with no members is a rabble.
func Of(levels []int, explicit []int) Tier {
	best := None
	for _, e := range explicit {
		if t := Tier(e); t > best && t <= MaxTier {
			best = t
		}
	}
	if best > None {
		return best
	}
	if len(levels) == 0 {
		return Rabble
	}
	sum := 0
	for _, l := range levels {
		sum += max(l, 0)
	}
	return ForLevel(sum / len(levels))
}

// FocusCount is how many of a group's fighters take its focus.
func FocusCount(t Tier, fighters int) int {
	if fighters < 1 {
		return 0
	}
	return (fighters*SpecOf(t).FocusPct + 99) / 100
}

// Noise is the targeting noise a member re-aims with: its own, raised to
// the tier's floor, or none at all for a tier without noise.
func Noise(t Tier, own int) int {
	s := SpecOf(t)
	if s.NoNoise {
		return 0
	}
	return min(max(own, s.NoiseFloor), 100)
}

// ValidateTier checks a template's coordination: 0 (by level) or 1–4.
func ValidateTier(n int) error {
	if n < 0 || n > int(MaxTier) {
		return fmt.Errorf("coordination %d: want 1 to %d, or none", n, MaxTier)
	}
	return nil
}

// Roles an enemy template may give.
var Roles = []string{"fighter", "healer", "caster", "guardian"}

// ValidateRole checks a template's role: blank (a fighter) or one of Roles.
func ValidateRole(role string) error {
	r := strings.ToLower(strings.TrimSpace(role))
	if r == "" {
		return nil
	}
	for _, ok := range Roles {
		if r == ok {
			return nil
		}
	}
	return fmt.Errorf("role %q: want one of %s", role, strings.Join(Roles, ", "))
}

// ValidateWounds checks a template's wounds: blank (light wounds) or none.
func ValidateWounds(w string) error {
	switch strings.ToLower(strings.TrimSpace(w)) {
	case "", "none":
		return nil
	}
	return fmt.Errorf("wounds %q: want none, or leave it out", w)
}
