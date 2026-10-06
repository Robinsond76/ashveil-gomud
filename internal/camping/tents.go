package camping

import "strings"

// Phase 52 tents. A company pitches one tent per camp, chosen from the ones
// it carries (`camp tent`). Each trades something for something: the
// canvas tent is the plain shelter (Phase 40a3), the others lean one way.
// The numbers are multipliers in percent; 100 changes nothing.

// TentKind names a tent. The zero value is no tent.
type TentKind string

const (
	TentCanvas      TentKind = "canvas"
	TentFur         TentKind = "fur"
	TentCamouflaged TentKind = "camouflaged"
	TentLarge       TentKind = "large"
)

// Tent is one tent's effects on a rest.
type Tent struct {
	Kind   TentKind
	ItemID int
	// Name is the item's name; Short is the word the commands take.
	Name  string
	Short string
	// FullShelter ends the weather's penalty to a rest's fatigue recovery
	// (the canvas tent only halves it).
	FullShelter bool
	// RaidPct and ThiefPct scale the chance that raiders or thieves come
	// to a rest (rolled when the rest begins).
	RaidPct  int
	ThiefPct int
	// RestedPct scales how long the Rested buff lasts.
	RestedPct int
	// WellRested upgrades a finished rest's buff from Rested to Well Rested
	// (the buff only: wounds and vitals follow the camp rest's own rules).
	WellRested bool
	// Effect is the one line the camp views and help print.
	Effect string
}

// Tents lists every tent, plain canvas first. The order is also the default
// pick when a company carries several and has chosen none.
var Tents = []Tent{
	{Kind: TentCanvas, ItemID: 46, Name: "oiled canvas tent", Short: "canvas", RaidPct: 100, ThiefPct: 100, RestedPct: 100,
		Effect: "shelter: halves the weather's penalty to a rest, and keeps the cold off"},
	{Kind: TentFur, ItemID: 300, Name: "fur-lined tent", Short: "fur", FullShelter: true, RaidPct: 100, ThiefPct: 100, RestedPct: 100,
		Effect: "no weather penalty to a rest at all, and keeps the cold off"},
	{Kind: TentCamouflaged, ItemID: 301, Name: "camouflaged tent", Short: "camouflaged", RaidPct: 50, ThiefPct: 50, RestedPct: 50,
		Effect: "raiders and thieves come half as often, but a light, wary sleep: Rested lasts half as long"},
	{Kind: TentLarge, ItemID: 302, Name: "large pavilion tent", Short: "large", RaidPct: 150, ThiefPct: 150, RestedPct: 100, WellRested: true,
		Effect: "everyone who sleeps wakes Well Rested, but the camp is easy to find: raiders and thieves come half again as often"},
}

// TentOf is the tent of kind. An empty or unknown kind is the canvas tent,
// which is what a tent meant before there were kinds.
func TentOf(kind TentKind) Tent {
	for _, t := range Tents {
		if t.Kind == kind {
			return t
		}
	}
	return Tents[0]
}

// ParseTent reads a command word ("fur", "fur-lined tent", "large").
func ParseTent(word string) (TentKind, bool) {
	word = strings.ToLower(strings.TrimSpace(word))
	for _, t := range Tents {
		if word == string(t.Kind) || word == t.Short || word == t.Name {
			return t.Kind, true
		}
	}
	return "", false
}

// PickTent is the tent a camp pitches: the chosen one if the company still
// carries it, else the first carried in Tents order. ok is false when it
// carries none.
func PickTent(carried []TentKind, choice TentKind) (TentKind, bool) {
	has := func(k TentKind) bool {
		for _, c := range carried {
			if c == k {
				return true
			}
		}
		return false
	}
	if choice != "" && has(choice) {
		return choice, true
	}
	for _, t := range Tents {
		if has(t.Kind) {
			return t.Kind, true
		}
	}
	return "", false
}

// ScaleChance applies a tent's percent multiplier to a chance in percent,
// capped at 100.
func ScaleChance(chancePct, tentPct int) int {
	if tentPct <= 0 {
		tentPct = 100
	}
	return min(100, (chancePct*tentPct+50)/100)
}

// RestedRounds scales a Rested duration, in rounds, by RestedPct.
func (t Tent) RestedRounds(rounds int) int {
	pct := t.RestedPct
	if pct <= 0 {
		pct = 100
	}
	return max(1, (rounds*pct+50)/100)
}

// WithArticle is the tent's name with "a" or "an" in front.
func (t Tent) WithArticle() string {
	if strings.ContainsRune("aeiou", rune(t.Name[0])) {
		return "an " + t.Name
	}
	return "a " + t.Name
}
