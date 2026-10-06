package strategy

import (
	"math"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
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
	// Sweep (halberdier, Phase 39a): one swing strikes its foe and the
	// foes beside it in the same row, each at part of the damage.
	Sweep Ability = "sweep"
	// Brace (halberdier): holds the turn; the first foe to strike into the
	// halberdier's place takes a held blow at once.
	Brace Ability = "brace"
	// The Doll Master's four (Phase 39d). Puppet Strike is its ordinary
	// action: the doll strikes in its Master's place. Guard String, Tangle and
	// Emergency Splice are resolved by internal/hooks/combat_doll.go; they are
	// listed here so the strategy, company and capability views show them.
	PuppetStrike Ability = "puppet-strike"
	GuardString  Ability = "guard-string"
	Tangle       Ability = "tangle"
	Splice       Ability = "emergency-splice"
	// The Beast Tamer's three (Phase 39e). Sic is part of the Tamer's own
	// turn; Rally and Pack Sense are resolved by internal/hooks/combat_beast.go
	// and listed here so the strategy, company and capability views show them.
	Sic       Ability = "sic"
	Rally     Ability = "rally"
	PackSense Ability = "pack-sense"
)

// AbilitySpec is an ability's unlock, cooldown, and text.
type AbilitySpec struct {
	ID        Ability
	Name      string
	Archetype string // a companion of this archetype has it
	Skill     string // a player with this skill (level 1+) has it
	Cooldown  int    // combat rounds before it can be used again
	MinLevel  int    // the character level it comes at (0: from level 1)
	When      string // its condition, for the strategy command
	Does      string // its effect
}

// Abilities is every ability, in the order a member tries them.
var Abilities = []AbilitySpec{
	{ID: Tackle, Name: "Tackle", Archetype: "warrior", Skill: "brawling", Cooldown: 4,
		When: "its foe is on its feet and within hand-to-hand reach, and it has no bow or sling",
		Does: "knocks the foe down (it loses its next action), breaking a chant or wind-up; the whole turn"},
	{ID: OpeningStrike, Name: "Opening Strike", Archetype: "rogue", Skill: "skulduggery", Cooldown: 2,
		When: "its foe is knocked down, stunned, staggered, or exposed, and it wields a blade or claws",
		Does: "its first blow that lands this round is a critical hit"},
	{ID: AimedShot, Name: "Aimed Shot", Archetype: "ranger", Skill: "track", Cooldown: 3,
		When: "it has a shooting weapon and its foe is not already exposed",
		Does: "its first shot that lands this round is a critical hit, leaving the foe exposed"},
	{ID: Sweep, Name: "Sweep", Archetype: "halberdier", Skill: "polearm", Cooldown: 3,
		When: "it wields a melee weapon, and its foe stands in a row with another foe",
		Does: "one swing strikes its foe and one foe beside it in the same row (every foe in the row from level 8), each at 90% of the damage and each defending separately; the whole turn"},
	{ID: Brace, Name: "Brace", Archetype: "halberdier", Skill: "polearm", Cooldown: 1, MinLevel: 3,
		When: "its Sweep is not ready (or has no second foe to strike), it wields a melee weapon, and a foe is striking at its place in the line",
		Does: "holds its turn; the first foe that strikes it takes a held blow at once, 25% harder than an ordinary one"},
	{ID: PuppetStrike, Name: "Puppet Strike", Archetype: "dollmaster", Skill: "puppetry",
		When: "its doll stands and a foe is within the doll's reach",
		Does: "the doll strikes in its Master's place, with the doll's own weapon and Attack; the Master's own blow is not struck"},
	{ID: GuardString, Name: "Guard String", Archetype: "dollmaster", Skill: "puppetry", MinLevel: 5,
		When: "a foe is about to strike the most hurt ally beside the doll",
		Does: "the doll steps in and takes the blow, twice a battle (three times from level 8)"},
	{ID: Tangle, Name: "Tangle", Archetype: "dollmaster", Skill: "puppetry", Cooldown: 3, MinLevel: 12,
		When: "the doll strikes a foe that is not already tangled",
		Does: "strings snag the foe the doll strikes and push its action meter back by half a turn (a quarter for a boss); the same foe can't be tangled again for 2 rounds"},
	{ID: Splice, Name: "Emergency Splice", Archetype: "dollmaster", Skill: "puppetry", MinLevel: 18,
		When: "the doll would break",
		Does: "once a battle the doll stands back up at 25% health, and its Master loses its next turn"},
	{ID: Sic, Name: "Sic", Archetype: "beasttamer", Skill: "taming",
		When: "its beast stands and the Tamer has a foe",
		Does: "sends the beast at the Tamer's foe with +10 Attack on its strike; the Tamer's own blow is still struck, with a whip that reaches like a polearm"},
	{ID: Rally, Name: "Rally", Archetype: "beasttamer", Skill: "taming", MinLevel: 3,
		When: "its beast is below the company's healing threshold",
		Does: "heals the beast for a Minor Heal's worth with no mana, twice a battle"},
	{ID: PackSense, Name: "Pack Sense", Archetype: "beasttamer", Skill: "taming", MinLevel: 8,
		When: "always, while its beast stands",
		Does: "the Tamer has +5 Evasion"},
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

// AtLevel keeps the abilities a character of this level has: one that
// comes at a level (Phase 39a: Brace at 3) is left out below it.
func AtLevel(list []Ability, level int) []Ability {
	out := make([]Ability, 0, len(list))
	for _, id := range list {
		if spec, ok := SpecOf(id); ok && spec.MinLevel > level {
			continue
		}
		out = append(out, id)
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
	// Close is true when the foe is within hand-to-hand reach (no
	// reaching weapon needed), for a tackle.
	Close bool
	// The foe's live statuses.
	FoeDown, FoeStunned, FoeStaggered, FoeExposed bool
	// FoeDownQueued: another member's tackle in this ability pass has just
	// succeeded, but the knockdown is not live yet. It stops a second
	// tackle and opens nothing else (30g6a).
	FoeDownQueued bool
	// Ambush opens any foe (Phase 38b: a Mercenary-turned-Scout's Ambush,
	// in the battle's first rounds).
	Ambush bool
	// SweepFoes is how many foes a sweep at this foe would strike: the foe
	// itself and those beside it in its row (Phase 39a).
	SweepFoes int
	// Struck is true when a foe is striking at the member (or, for a
	// Vanguard, at its column) and no brace is held yet: a brace has
	// something to answer (Phase 39a).
	Struck bool
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
			if s.Weapon != Shooting && s.Close && !s.FoeDown && !s.FoeDownQueued && !s.FoeStunned {
				return id, true
			}
		case OpeningStrike:
			if s.Backstab && (s.Ambush || s.FoeDown || s.FoeStunned || s.FoeStaggered || s.FoeExposed) {
				return id, true
			}
		case AimedShot:
			if s.Weapon == Shooting && !s.FoeExposed {
				return id, true
			}
		case Sweep:
			if s.Weapon == Melee && s.SweepFoes >= 2 {
				return id, true
			}
		case Brace:
			if s.Weapon == Melee && s.Struck {
				return id, true
			}
		}
	}
	return "", false
}

// OpeningStrikeBonus is the damage an Opening Strike adds to its blow
// (Phase 35a2): 2 + level/6, about half an ordinary hit.
func OpeningStrikeBonus(level int) int {
	return 2 + max(level, 0)/6
}

// AimedShotBonus is the damage an Aimed Shot adds to its blow (Phase 35b):
// 2 + level/3. The shot is already a sure critical hit, so it grows in
// damage rather than crit chance.
func AimedShotBonus(level int) int {
	return 2 + max(level, 0)/3
}

// TackleExtraRounds is the rounds a tackle's knockdown lasts beyond its
// own (Phase 35b): 1 from level 20.
func TackleExtraRounds(level int) int {
	if level >= 20 {
		return 1
	}
	return 0
}

// Tackle chance bounds, in 100 (30g6 amendment): even Speed and Perception
// give TackleEven; a full stat edge moves it to a bound.
const (
	TackleMin  = 20
	TackleEven = 40
	TackleMax  = 80
)

// TackleChance is a tackle's chance in 100: the tackler's Speed edge over
// the foe's Perception (the combat StatEdgeSpan), plus its skill edge
// (Attack against the foe's Evasion, Phase 35a2), held to −1..1, moves
// TackleEven toward TackleMin or TackleMax. The manual command and
// automatic tackles share it.
func TackleChance(speed, foePerception int, skillEdge float64) int {
	span := float64(configs.GetCombatConfig().StatEdgeSpan)
	if span <= 0 || math.IsNaN(span) || math.IsInf(span, 0) {
		span = 10
	}
	if math.IsNaN(skillEdge) {
		skillEdge = 0
	}
	edge := max(-1, min(1, (float64(speed)-float64(foePerception))/span+skillEdge))
	c := float64(TackleEven)
	if edge >= 0 {
		c += edge * (TackleMax - TackleEven)
	} else {
		c += edge * (TackleEven - TackleMin)
	}
	return max(TackleMin, min(TackleMax, int(math.Floor(c+1e-9))))
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

// Sweep numbers (Phase 39a). A sweep strikes each foe at SweepPct of a
// blow's damage; from SweepRowLevel it reaches the whole row.
const (
	SweepPct      = 90
	SweepRowLevel = 8
	BracePct      = 125
)

// HookLevel is the level a halberdier's glaive hit can knock down a
// leaping foe; HookChance is its chance in 100: 20, and 40 from level 20.
const HookLevel = 6

func HookChance(level int) int {
	switch {
	case level >= 20:
		return 40
	case level >= HookLevel:
		return 20
	}
	return 0
}

// SweepWide reports whether a halberdier's sweep reaches every foe in its
// target's row, not only one beside it.
func SweepWide(level int) bool { return level >= SweepRowLevel }
