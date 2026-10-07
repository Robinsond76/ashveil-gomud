// Package errands is Phase 70's errands: a companion sitting out the
// formation is sent from a town on an escort, a hunt or a scouting job of a
// chosen real-time length, and comes back with modest gold, an item, a
// rumour of a lair, or a wound. The package is pure: kinds, lengths, the
// saved Errand, and Resolve, which turns a saved errand into its outcome.
// modules/company owns the saved state, the commands and the world.
//
// Everything is real time (Unix seconds). Nothing here reads or advances the
// world clock.
package errands

import (
	"fmt"
	"math/rand"
	"strings"
)

// Kind is the sort of job.
type Kind string

const (
	Escort Kind = "escort" // walking a caravan: steady pay, safe for one who matches the zone
	Hunt   Kind = "hunt"   // clearing the road: better spoils, a wound's risk
	Scout  Kind = "scout"  // reading the country: rumours first, little coin
)

// KindInfo is a kind's player-facing words.
type KindInfo struct {
	Kind   Kind
	Label  string // "an escort job"
	Gerund string // "escorting"
	Blurb  string
}

// Kinds lists the jobs in the order the views show them.
var Kinds = []KindInfo{
	{Escort, "an escort job", "escorting", "walks a caravan down the road: steady pay, and safe for one who matches the zone"},
	{Hunt, "a hunt", "hunting", "clears game and vermin off the road: better spoils, and a wound's risk"},
	{Scout, "a scouting job", "scouting", "reads the country: mostly rumours of lairs, little coin, some risk"},
}

// KindByWord resolves what a player typed ("escort", "hunt", "scout").
func KindByWord(word string) (Kind, bool) {
	word = strings.ToLower(strings.TrimSpace(word))
	for _, k := range Kinds {
		if word == string(k.Kind) || (len(word) >= 3 && strings.HasPrefix(string(k.Kind), word)) {
			return k.Kind, true
		}
	}
	return "", false
}

// Info is the kind's words.
func (k Kind) Info() KindInfo {
	for _, i := range Kinds {
		if i.Kind == k {
			return i
		}
	}
	return KindInfo{Kind: k, Label: "an errand", Gerund: "on an errand"}
}

// Length is how long the errand takes.
type Length string

const (
	Short  Length = "short"
	Medium Length = "medium"
	Long   Length = "long"
)

// LengthInfo is a length's real-time span and pay.
type LengthInfo struct {
	Length  Length
	Seconds int64
	Label   string // "half an hour"
	PayPct  int    // the pay against a short errand's, in percent
}

// Lengths lists the spans, shortest first. Pay grows slower than time, so a
// long errand is a convenience for a session's gap, never a better wage.
var Lengths = []LengthInfo{
	{Short, 30 * 60, "half an hour", 100},
	{Medium, 2 * 60 * 60, "two hours", 250},
	{Long, 8 * 60 * 60, "eight hours", 600},
}

// LengthByWord resolves "short", "medium", "long", or the span itself
// ("30m", "2h", "8h").
func LengthByWord(word string) (Length, bool) {
	word = strings.ToLower(strings.TrimSpace(word))
	switch word {
	case "short", "30m", "half", "30":
		return Short, true
	case "medium", "med", "2h", "2":
		return Medium, true
	case "long", "8h", "8":
		return Long, true
	}
	return "", false
}

// Info is the length's span and pay.
func (l Length) Info() LengthInfo {
	for _, i := range Lengths {
		if i.Length == l {
			return i
		}
	}
	return Lengths[0]
}

// Errand is a companion's saved errand. It carries everything Resolve needs,
// frozen when the companion left, so a restart, a level change in the band,
// or a second look can never change the outcome.
type Errand struct {
	Kind      Kind   `yaml:"kind"`
	Length    Length `yaml:"length"`
	Zone      string `yaml:"zone,omitempty"`
	StartedAt int64  `yaml:"started_at"`
	ReturnsAt int64  `yaml:"returns_at"`
	// Level is the companion's level when it left; BandLow and BandHigh the
	// zone's recommended band then (both the level when the zone names none).
	Level    int `yaml:"level"`
	BandLow  int `yaml:"band_low"`
	BandHigh int `yaml:"band_high"`
	// MaxHealth is its health maximum, for the size of a wound.
	MaxHealth int `yaml:"max_health,omitempty"`
	// Seed fixes every roll of the outcome.
	Seed int64 `yaml:"seed"`
	// Placed, Row and Col are the formation cell the companion left; it
	// takes that cell again on its return when the cell is still free.
	Placed bool `yaml:"placed,omitempty"`
	Row    int  `yaml:"row,omitempty"`
	Col    int  `yaml:"col,omitempty"`
}

// Due reports whether the errand is over at now.
func (e Errand) Due(now int64) bool { return now >= e.ReturnsAt }

// Remaining is the seconds left at now (0 once due).
func (e Errand) Remaining(now int64) int64 {
	if left := e.ReturnsAt - now; left > 0 {
		return left
	}
	return 0
}

// New starts an errand at now. A band that is not valid becomes the
// companion's level, so a zone that names no band still pays.
func New(kind Kind, length Length, zone string, now int64, level, bandLow, bandHigh, maxHealth int, seed int64) Errand {
	level = max(level, 1)
	if bandLow < 1 || bandHigh < bandLow {
		bandLow, bandHigh = level, level
	}
	return Errand{
		Kind: kind, Length: length, Zone: zone,
		StartedAt: now, ReturnsAt: now + length.Info().Seconds,
		Level: level, BandLow: bandLow, BandHigh: bandHigh,
		MaxHealth: max(maxHealth, 0), Seed: seed,
	}
}

// OutcomeKind is what the companion comes back with.
type OutcomeKind string

const (
	Gold   OutcomeKind = "gold"
	Item   OutcomeKind = "item"
	Rumour OutcomeKind = "rumour"
	Wound  OutcomeKind = "wound"
)

// Outcome is what an errand came to. Gold is the pay the errand earned:
// paid as it is for a Gold outcome; for an Item, the module pays part of it
// as a find, valued at the item's full worth, and the rest in coin, so an
// item is never a way to turn an errand into more than its gold. Roll picks among the module's
// options (which item, which lair) and is stable for the errand.
type Outcome struct {
	Kind     OutcomeKind
	Gold     int
	WoundPct int
	Roll     int
}

// WoundPct is the share of its maximum health a wound holds back.
const WoundPct = 20

// Pay is what the errand earns before its outcome is rolled: 3 gold per
// level of the band's middle and 1 per level of the companion, a kind's
// share, and a length's share. A level-8 companion in a 7-9 band earns 32
// for a short escort, 44 on a hunt, 19 scouting.
func Pay(e Errand) int {
	mid := (e.BandLow + e.BandHigh) / 2
	base := 3*mid + e.Level
	kindPct := map[Kind]int{Escort: 100, Hunt: 140, Scout: 60}[e.Kind]
	if kindPct == 0 {
		kindPct = 100
	}
	return max(1, base*kindPct/100*e.Length.Info().PayPct/100)
}

// WoundRisk is the chance in percent that the companion comes back
// wounded: a kind's own risk, eight more for each level it stands under the
// band, half as much when it outlevels the zone, capped at 60.
func WoundRisk(e Errand) int {
	base := map[Kind]int{Escort: 0, Hunt: 15, Scout: 8}[e.Kind]
	risk := base
	if e.Level < e.BandLow {
		risk += 8 * (e.BandLow - e.Level)
	} else if e.Level > e.BandHigh {
		risk /= 2
	}
	return min(max(risk, 0), 60)
}

// weights are the shares of gold, item and rumour among the outcomes that
// are not a wound.
var weights = map[Kind][3]int{
	Escort: {60, 25, 15},
	Hunt:   {45, 40, 15},
	Scout:  {20, 15, 65},
}

// Resolve is the errand's outcome. hasLair says whether the zone has a lair
// to hear a rumour of; without one a rumour pays in gold. The same errand
// always resolves the same way.
func Resolve(e Errand, hasLair bool) Outcome {
	rng := rand.New(rand.NewSource(e.Seed))
	woundRoll := rng.Intn(100)
	pick := rng.Intn(100)
	roll := rng.Intn(1 << 20)
	if woundRoll < WoundRisk(e) {
		return Outcome{Kind: Wound, WoundPct: WoundPct, Roll: roll}
	}
	pay := Pay(e)
	w, ok := weights[e.Kind]
	if !ok {
		w = weights[Escort]
	}
	switch {
	case pick < w[0]:
		return Outcome{Kind: Gold, Gold: pay, Roll: roll}
	case pick < w[0]+w[1]:
		return Outcome{Kind: Item, Gold: pay, Roll: roll}
	case hasLair:
		return Outcome{Kind: Rumour, Roll: roll}
	}
	return Outcome{Kind: Gold, Gold: pay, Roll: roll}
}

// Span says a number of seconds in words ("about 40 minutes", "3 hours").
func Span(seconds int64) string {
	switch {
	case seconds <= 60:
		return "a minute"
	case seconds < 90*60:
		return fmt.Sprintf("about %d minutes", (seconds+30)/60)
	case seconds < 36*3600:
		return fmt.Sprintf("about %d hours", (seconds+1800)/3600)
	}
	return fmt.Sprintf("about %d days", (seconds+43200)/86400)
}
