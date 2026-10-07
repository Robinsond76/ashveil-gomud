// Package stance holds Ashveil's weapon stances (Phase 69): one alternate
// way to use a weapon family, chosen per company member between battles,
// that trades one strength for another. A great weapon hits harder and
// misses more, a shield turns away more blows and swings less, a bow draws
// faster and hits lighter, a dagger finds more weak spots and cuts shallower.
//
// This package is GoMud-free: it names the stances, says what each needs,
// and what it changes. The durable choice lives in modules/strategy and
// internal/combat reads the effect.
package stance

import "strings"

// Stance is a stance's key as stored and typed. Blank is no stance.
type Stance string

const (
	None  Stance = ""
	Heavy Stance = "heavy" // great weapon: harder blows, more misses
	Wall  Stance = "wall"  // shield: more blocks, fewer attacks
	Quick Stance = "quick" // bow: more shots, lighter ones
	Keen  Stance = "keen"  // dagger: more critical hits, shallower cuts
)

// Effect is what a stance changes, in the units combat already uses.
type Effect struct {
	Hit       int // points added to the chance to hit (negative: a penalty)
	DamagePct int // percent added to the damage of a landed blow
	Crit      int // points added to the chance of a critical hit
	TempoPct  int // percent added to the member's turns per round
	Block     int // points added to the chance a shield blocks a blow
}

// IsZero reports whether the effect changes nothing.
func (e Effect) IsZero() bool { return e == Effect{} }

// Def is one stance.
type Def struct {
	Key    Stance
	Name   string // as a player reads it: "Heavy blows"
	Family string // what it is for: "great weapon"
	Needs  string // what the member must hold, in words
	Effect Effect
	// Gain and Cost say the trade in words, for the command and help.
	Gain, Cost string
}

// Defs is every stance, in the order they are listed.
var Defs = []Def{
	{Key: Heavy, Name: "Heavy blows", Family: "great weapon", Needs: "a two-handed weapon (not a staff or a bow)",
		Effect: Effect{Hit: -15, DamagePct: 30}, Gain: "blows land 30% harder", Cost: "15 points less likely to hit"},
	{Key: Wall, Name: "Shield wall", Family: "shield", Needs: "a shield",
		Effect: Effect{TempoPct: -30, Block: 12}, Gain: "12 points more likely to block a blow", Cost: "30% fewer turns each round"},
	{Key: Quick, Name: "Quick draw", Family: "bow", Needs: "a bow",
		Effect: Effect{TempoPct: 25, DamagePct: -15}, Gain: "25% more turns each round", Cost: "arrows land 15% lighter"},
	{Key: Keen, Name: "Keen edge", Family: "dagger", Needs: "a dagger",
		Effect: Effect{Crit: 10, DamagePct: -10}, Gain: "10 points more likely to land a critical hit", Cost: "blows land 10% shallower"},
}

var aliases = map[string]Stance{
	"heavy": Heavy, "heavy-blows": Heavy, "great": Heavy, "greatweapon": Heavy, "power": Heavy,
	"wall": Wall, "shield-wall": Wall, "shieldwall": Wall, "shield": Wall, "turtle": Wall,
	"quick": Quick, "quick-draw": Quick, "quickdraw": Quick, "bow": Quick, "fast": Quick,
	"keen": Keen, "keen-edge": Keen, "keenedge": Keen, "dagger": Keen, "edge": Keen,
}

// Parse reads a stance from what a player typed; "heavy blows" and
// "heavy-blows" both read.
func Parse(word string) (Stance, bool) {
	s, ok := aliases[strings.ReplaceAll(strings.ToLower(strings.TrimSpace(word)), " ", "-")]
	return s, ok
}

// Valid reports whether s is a stance (or none).
func (s Stance) Valid() bool {
	if s == None {
		return true
	}
	_, ok := Lookup(s)
	return ok
}

// Lookup is a stance's definition.
func Lookup(s Stance) (Def, bool) {
	for _, d := range Defs {
		if d.Key == s {
			return d, true
		}
	}
	return Def{}, false
}

// Gear is what a member holds, as a stance reads it.
type Gear struct {
	// TwoHanded is a melee weapon held in both hands; Class and Family are
	// that weapon's, lowercase.
	TwoHanded bool
	Shooting  bool
	Class     string
	Family    string
	// Shield is a shield (or other armored offhand) in the offhand.
	Shield bool
}

// Fits reports whether the gear can use the stance.
func Fits(s Stance, g Gear) bool {
	switch s {
	case Heavy:
		return g.TwoHanded && !g.Shooting && g.Class != "staff" && g.Class != "rod"
	case Wall:
		return g.Shield
	case Quick:
		return g.Shooting && g.Class == "bow"
	case Keen:
		return !g.Shooting && (g.Class == "dagger" || g.Family == "dagger")
	}
	return false
}

// Available is the stances the gear can use, in list order.
func Available(g Gear) []Stance {
	var out []Stance
	for _, d := range Defs {
		if Fits(d.Key, g) {
			out = append(out, d.Key)
		}
	}
	return out
}

// EffectFor is what the stance does for a member holding the gear: nothing
// without its weapon.
func EffectFor(s Stance, g Gear) Effect {
	if !Fits(s, g) {
		return Effect{}
	}
	d, _ := Lookup(s)
	return d.Effect
}

// Describe is the stance in one line: its name, its trade, and what it needs.
func (d Def) Describe() string {
	return d.Name + " (" + d.Family + "): " + d.Gain + ", but " + d.Cost + "."
}

// Scale applies a percent change to a damage figure, never taking a landed
// blow below 1.
func Scale(damage, pct int) int {
	if pct == 0 || damage <= 0 {
		return damage
	}
	return max(1, (damage*(100+pct)+50)/100)
}
