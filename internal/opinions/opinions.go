// Package opinions is Phase 64's rules: what each companion personality
// thinks of the leader's choices. It is GoMud-free and pure: the kinds of
// choice, who likes or dislikes which, the one line a companion says, the
// size and limits of the loyalty nudge, and a small observer seam. The
// company module owns the saved memory and applies the nudge through the
// real loyalty record; sources (mercy, camps, sales, story events) report a
// Choice through internal/company and never import the module.
//
// Personalities are the six banter rolls (stoic, cheerful, grim, boastful,
// wry, devout). Alignment already moves loyalty on mercy (Phase 27), so a
// companion that alignment reacted to stays quiet here: one reaction per
// choice per companion.
package opinions

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// Kind is one kind of choice a companion has an opinion on.
type Kind string

const (
	Spare     Kind = "spare"      // spared a yielded foe
	Execute   Kind = "execute"    // executed a yielded foe
	Inn       Kind = "inn"        // paid for beds at an inn
	Rough     Kind = "rough"      // rested at a rough camp
	Share     Kind = "share"      // fed a companion from the leader's own pack
	SellRelic Kind = "sell-relic" // sold a relic to a merchant
	// Story stances: tagged on a story event's choice (Phase 60).
	Kindness  Kind = "kindness"  // helped someone at a cost
	Greed     Kind = "greed"     // took the coin or the spoils
	Courage   Kind = "courage"   // took the bold, risky way
	Prudence  Kind = "prudence"  // took the careful way, or turned back
	Reverence Kind = "reverence" // honoured the dead, a shrine or an oath
	Cunning   Kind = "cunning"   // lied, tricked or slipped by
)

// Info describes one kind for the player: Label finishes "likes", "dislikes"
// and "approved of", and Words are what a story author or player may type.
type Info struct {
	Kind     Kind
	Label    string
	Cooldown time.Duration
	Story    bool // may be tagged on a story event choice
}

// Kinds is every kind, in the order they are listed.
var Kinds = []Info{
	{Spare, "mercy to the beaten", 10 * time.Minute, false},
	{Execute, "executing the beaten", 10 * time.Minute, false},
	{Inn, "a bed at an inn", 2 * time.Hour, false},
	{Rough, "a rough camp", 2 * time.Hour, false},
	{Share, "sharing your own food", 2 * time.Hour, false},
	{SellRelic, "selling relics", 30 * time.Minute, false},
	{Kindness, "kindness at a cost", 10 * time.Minute, true},
	{Greed, "taking the spoils", 10 * time.Minute, true},
	{Courage, "the bold way", 10 * time.Minute, true},
	{Prudence, "the careful way", 10 * time.Minute, true},
	{Reverence, "honouring the dead and oaths", 10 * time.Minute, true},
	{Cunning, "lies and tricks", 10 * time.Minute, true},
}

// InfoOf is a kind's description; false for an unknown kind.
func InfoOf(k Kind) (Info, bool) {
	for _, i := range Kinds {
		if i.Kind == k {
			return i, true
		}
	}
	return Info{}, false
}

// Label is a kind's phrase, or the kind itself when it is unknown.
func Label(k Kind) string {
	if i, ok := InfoOf(k); ok {
		return i.Label
	}
	return string(k)
}

// IsStance reports whether a story event may tag a choice with this kind.
func IsStance(s string) bool {
	i, ok := InfoOf(Kind(s))
	return ok && i.Story
}

// Stances is the story-tag names, for error messages and authors.
func Stances() []string {
	var out []string
	for _, i := range Kinds {
		if i.Story {
			out = append(out, string(i.Kind))
		}
	}
	return out
}

// Verdicts.
const (
	Dislikes = -1
	Neutral  = 0
	Likes    = 1
)

// leanings are what each personality likes (+1) or dislikes (-1). Every
// personality weighs at least four kinds; any other it shrugs at.
var leanings = map[string]map[Kind]int{
	"stoic": {
		Rough: Likes, Prudence: Likes, Share: Likes, Reverence: Likes,
		Inn: Dislikes, Greed: Dislikes, Cunning: Dislikes,
	},
	"cheerful": {
		Spare: Likes, Inn: Likes, Share: Likes, Kindness: Likes, Courage: Likes,
		Execute: Dislikes, Greed: Dislikes, Rough: Dislikes,
	},
	"grim": {
		Execute: Likes, Rough: Likes, Prudence: Likes, SellRelic: Likes,
		Spare: Dislikes, Inn: Dislikes, Kindness: Dislikes, Courage: Dislikes,
	},
	"boastful": {
		Courage: Likes, Inn: Likes, Greed: Likes, SellRelic: Likes, Execute: Likes,
		Prudence: Dislikes, Rough: Dislikes, Reverence: Dislikes,
	},
	"wry": {
		Cunning: Likes, Greed: Likes, Inn: Likes, SellRelic: Likes,
		Courage: Dislikes, Reverence: Dislikes, Kindness: Dislikes, Rough: Dislikes,
	},
	"devout": {
		Spare: Likes, Kindness: Likes, Reverence: Likes, Share: Likes,
		Execute: Dislikes, SellRelic: Dislikes, Greed: Dislikes, Cunning: Dislikes,
	},
}

// Verdict is what a personality thinks of a kind: Likes, Dislikes or Neutral.
func Verdict(personality string, k Kind) int {
	return leanings[strings.ToLower(personality)][k]
}

// Leanings are the kinds a personality likes and dislikes, in list order.
func Leanings(personality string) (likes, dislikes []Kind) {
	for _, i := range Kinds {
		switch Verdict(personality, i.Kind) {
		case Likes:
			likes = append(likes, i.Kind)
		case Dislikes:
			dislikes = append(dislikes, i.Kind)
		}
	}
	return likes, dislikes
}

// The loyalty nudge. It is small and bounded, so opinions colour loyalty
// without becoming a way to farm it or to drive a companion away:
// approval never lifts loyalty past Ceiling, disapproval never lowers it
// under Floor (30, above the battle nerve line of 25, so opinions alone never
// make a companion hesitate or desert), and a
// companion speaks on a kind again only after that kind's Cooldown.
const (
	Nudge   = 2
	Ceiling = 80
	Floor   = 30
	// MaxNotes is how many recent reactions a companion remembers.
	MaxNotes = 6
)

// Choice is one choice the leader made, reported by its real source.
type Choice struct {
	Kind Kind
	// Op names this one choice (a mercy token, a story operation), so a
	// retried report changes nothing.
	Op string
	// Witnesses are the companions who saw it; nil means every companion
	// present with the leader.
	Witnesses []int
	// Subject is what it was about, for the companion's memory ("the
	// Hollow Bandit").
	Subject string
}

// Reaction is one companion's answer to a choice.
type Reaction struct {
	CompanionID int
	Name        string
	Personality string
	Kind        Kind
	// Verdict is Likes or Dislikes.
	Verdict int
	// Line is what the companion says, ready to show.
	Line string
	// Delta is the loyalty change made (zero at the ceiling or floor).
	Delta int
}

// Observer hears every batch of reactions, so Phase 65 (bonds) can set one
// companion's verdict beside another's without this package knowing it.
type Observer func(leaderUserID int, c Choice, reactions []Reaction)

var (
	observersMu sync.RWMutex
	observers   []Observer
)

// Observe registers an observer.
func Observe(o Observer) {
	observersMu.Lock()
	defer observersMu.Unlock()
	observers = append(observers, o)
}

// Announce tells every observer about a batch of reactions.
func Announce(leaderUserID int, c Choice, reactions []Reaction) {
	if len(reactions) == 0 {
		return
	}
	observersMu.RLock()
	list := append([]Observer(nil), observers...)
	observersMu.RUnlock()
	for _, o := range list {
		o(leaderUserID, c, reactions)
	}
}

// Words lists kinds as a sentence part: "a rough camp, selling relics".
func Words(kinds []Kind) string {
	parts := make([]string, 0, len(kinds))
	for _, k := range kinds {
		parts = append(parts, Label(k))
	}
	return strings.Join(parts, ", ")
}

// KindsSorted is every known kind id, sorted, for tests and validation.
func KindsSorted() []string {
	out := make([]string, 0, len(Kinds))
	for _, i := range Kinds {
		out = append(out, string(i.Kind))
	}
	sort.Strings(out)
	return out
}
