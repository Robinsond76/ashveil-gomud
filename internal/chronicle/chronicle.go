// Package chronicle is Phase 63's company chronicle: a capped, durable log of
// the deeds a company is known for (who joined and left, who fell, which
// bosses it killed, which relics it found, mercy and executions,
// promotions, defeats, story event choices). It is GoMud-free: the entry
// shapes, the prose, the filter and the query seam live here, the saved
// state and the game's hooks in modules/chronicle.
//
// Other code records a deed with Record and reads the log with Query, Count,
// Has and Total, never importing the module. Phases that read the chronicle
// (opinions, relic awakenings, towns that remember, errands, creeds, bounty
// boards, Hardcore) use only those functions.
package chronicle

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Kind is the sort of deed an entry records.
type Kind string

const (
	Joined    Kind = "joined"    // a companion took the company's coin or word
	Dismissed Kind = "dismissed" // the leader sent a companion away
	Deserted  Kind = "deserted"  // a companion lost faith and left
	Fell      Kind = "fell"      // a member died
	Raised    Kind = "raised"    // a fallen companion was raised
	Lost      Kind = "lost"      // a fallen companion was not raised in time
	Defeated  Kind = "defeated"  // the company was beaten (a defeat scenario)
	Boss      Kind = "boss"      // a boss, a lair's master, was killed
	Relic     Kind = "relic"     // a relic was found
	Spared    Kind = "spared"    // mercy: a yielded foe was let go
	Executed  Kind = "executed"  // a yielded foe was put to death
	Promoted  Kind = "promoted"  // a member took an advanced or elite class
	Story     Kind = "story"     // a story event ended on a choice
	Awakened  Kind = "awakened"  // a relic woke a new power (Phase 67)
)

// KindInfo is a kind's player-facing name and filter words.
type KindInfo struct {
	Kind  Kind
	Label string   // "Mercy" in the panel's filter
	Words []string // what `chronicle <word>` accepts
}

// Kinds lists every kind in the order the views show them.
var Kinds = []KindInfo{
	{Joined, "Joined", []string{"joined", "joins", "recruits"}},
	{Dismissed, "Dismissed", []string{"dismissed", "dismissals"}},
	{Deserted, "Deserted", []string{"deserted", "desertions"}},
	{Fell, "Fallen", []string{"fell", "fallen", "deaths", "death"}},
	{Raised, "Raised", []string{"raised", "resurrections"}},
	{Lost, "Lost", []string{"lost"}},
	{Defeated, "Defeats", []string{"defeated", "defeats", "defeat"}},
	{Boss, "Bosses", []string{"boss", "bosses", "lairs", "lair"}},
	{Relic, "Relics", []string{"relic", "relics"}},
	{Spared, "Mercy", []string{"spared", "mercy"}},
	{Executed, "Executions", []string{"executed", "executions", "execution"}},
	{Promoted, "Promotions", []string{"promoted", "promotions", "promotion"}},
	{Story, "Stories", []string{"story", "stories", "events"}},
	{Awakened, "Awakenings", []string{"awakened", "awakenings", "awakening"}},
}

// KindByWord resolves what a player typed to a kind.
func KindByWord(word string) (Kind, bool) {
	word = strings.ToLower(strings.TrimSpace(word))
	for _, k := range Kinds {
		for _, w := range k.Words {
			if w == word {
				return k.Kind, true
			}
		}
	}
	return "", false
}

// Valid reports whether k is a known kind.
func (k Kind) Valid() bool {
	for _, i := range Kinds {
		if i.Kind == k {
			return true
		}
	}
	return false
}

// Entry is one deed. Members and Subject are display names; Ref is the
// stable key later phases match on ("mob:12", "item:50033", "event:cliff",
// "class:samurai", "scenario:captured"), so a renamed thing still matches.
// Keys are the members' company keys ("leader", "companion:7"), in the
// order of Members, so a later phase can tell apart two companions with one
// name; a companion's id is never reused within a company.
type Entry struct {
	Seq     int      `yaml:"seq" json:"seq"`
	At      int64    `yaml:"at" json:"at"` // real time, Unix seconds
	Kind    Kind     `yaml:"kind" json:"kind"`
	Members []string `yaml:"members,omitempty" json:"members,omitempty"`
	Keys    []string `yaml:"keys,omitempty" json:"keys,omitempty"`
	Subject string   `yaml:"subject,omitempty" json:"subject,omitempty"`
	Detail  string   `yaml:"detail,omitempty" json:"detail,omitempty"`
	Place   string   `yaml:"place,omitempty" json:"place,omitempty"`
	Ref     string   `yaml:"ref,omitempty" json:"ref,omitempty"`
}

// Who is the members named in a sentence: "Mara", "Mara and Tobin",
// "Mara, Tobin and Ysolde". Nobody reads "The company".
func Who(members []string) string {
	names := make([]string, 0, len(members))
	for _, m := range members {
		if m = strings.TrimSpace(m); m != "" {
			names = append(names, m)
		}
	}
	switch len(names) {
	case 0:
		return "The company"
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// was agrees with Who: "was" for one or none, "were" for several.
func was(members []string) string {
	n := 0
	for _, m := range members {
		if strings.TrimSpace(m) != "" {
			n++
		}
	}
	if n > 1 {
		return "were"
	}
	return "was"
}

// whoMid is Who for the middle of a sentence.
func whoMid(members []string) string {
	if w := Who(members); w != "The company" {
		return w
	}
	return "the company"
}

func (e Entry) at() string {
	if e.Place == "" {
		return ""
	}
	return " at " + e.Place
}

func orThing(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

// Prose is the deed as one plain sentence, no markup.
func Prose(e Entry) string {
	who := Who(e.Members)
	switch e.Kind {
	case Joined:
		return fmt.Sprintf("%s joined the company%s.", who, e.at())
	case Dismissed:
		return fmt.Sprintf("%s %s sent away%s.", who, was(e.Members), e.at())
	case Deserted:
		return fmt.Sprintf("%s lost faith in the company and left%s.", who, e.at())
	case Fell:
		if e.Subject != "" {
			return fmt.Sprintf("%s fell to %s%s.", who, e.Subject, e.at())
		}
		return fmt.Sprintf("%s fell%s.", who, e.at())
	case Raised:
		return fmt.Sprintf("%s %s raised from the dead%s.", who, was(e.Members), e.at())
	case Lost:
		return fmt.Sprintf("%s %s lost for good; no one raised them in time.", who, was(e.Members))
	case Defeated:
		if e.Detail != "" {
			return fmt.Sprintf("The company was beaten%s. %s", e.at(), sentence(e.Detail))
		}
		return fmt.Sprintf("The company was beaten%s.", e.at())
	case Boss:
		return fmt.Sprintf("%s slew %s%s.", who, orThing(e.Subject, "a lair's master"), e.at())
	case Relic:
		if e.Detail != "" {
			return fmt.Sprintf("%s found %s on %s%s.", who, orThing(e.Subject, "a relic"), e.Detail, e.at())
		}
		return fmt.Sprintf("%s found %s%s.", who, orThing(e.Subject, "a relic"), e.at())
	case Spared:
		return fmt.Sprintf("%s showed mercy to %s%s.", who, orThing(e.Subject, "a beaten foe"), e.at())
	case Executed:
		return fmt.Sprintf("%s put %s to the sword%s.", who, orThing(e.Subject, "a beaten foe"), e.at())
	case Promoted:
		return fmt.Sprintf("%s became %s%s.", who, withArticle(orThing(e.Subject, "something more")), e.at())
	case Story:
		title := orThing(e.Subject, "a strange scene")
		if e.Detail != "" {
			return fmt.Sprintf("At %s, %s chose: %s.", title, whoMid(e.Members), strings.TrimRight(sentence(e.Detail), ".!?"))
		}
		return fmt.Sprintf("The company met with %s%s.", title, e.at())
	case Awakened:
		relic := orThing(e.Subject, "relic")
		if e.Detail != "" {
			return fmt.Sprintf("%s's %s awoke to %s%s.", who, relic, strings.TrimRight(e.Detail, ".!?"), e.at())
		}
		return fmt.Sprintf("%s's %s awoke%s.", who, relic, e.at())
	}
	return who + " did something worth remembering."
}

// sentence makes s start with a capital letter and end with a stop.
func sentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.ToUpper(s[:1]) + s[1:]
	if !strings.HasSuffix(s, ".") && !strings.HasSuffix(s, "!") && !strings.HasSuffix(s, "?") {
		s += "."
	}
	return s
}

func withArticle(s string) string {
	low := strings.ToLower(s)
	switch {
	case strings.HasPrefix(low, "a "), strings.HasPrefix(low, "an "), strings.HasPrefix(low, "the "):
		return s
	case strings.ContainsAny(low[:1], "aeiou"):
		return "an " + s
	}
	return "a " + s
}

// Ago is how long before now a deed happened, in words.
func Ago(at, now int64) string {
	d := now - at
	switch {
	case d < 90:
		return "just now"
	case d < 3600:
		return plural(int(d/60), "minute") + " ago"
	case d < 36*3600:
		return plural(int((d+1800)/3600), "hour") + " ago"
	}
	return plural(int((d+43200)/86400), "day") + " ago"
}

func plural(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

// Filter narrows a query. Zero fields match everything.
type Filter struct {
	Kinds  []Kind // any of these kinds
	Ref    string // this exact reference
	Member string // a deed naming this member (case-insensitive)
	Key    string // a deed naming the member with this company key
	Since  int64  // at or after this Unix time
	Limit  int    // at most this many, newest first; 0 is no limit
}

// Matches reports whether e passes the filter (Limit aside).
func (f Filter) Matches(e Entry) bool {
	if len(f.Kinds) > 0 {
		found := false
		for _, k := range f.Kinds {
			if k == e.Kind {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	if f.Ref != "" && f.Ref != e.Ref {
		return false
	}
	if f.Since > 0 && e.At < f.Since {
		return false
	}
	if f.Key != "" {
		found := false
		for _, k := range e.Keys {
			if k == f.Key {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	if f.Member != "" {
		found := false
		for _, m := range e.Members {
			if strings.EqualFold(m, f.Member) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// MaxEntries is how many deeds a company's log keeps; the oldest go first.
// Tally keeps counting what has gone.
const MaxEntries = 300

// Log is one company's chronicle: the deeds, oldest first, and a running
// count of every deed ever recorded by kind (never trimmed).
type Log struct {
	Entries []Entry      `yaml:"entries,omitempty"`
	Tally   map[Kind]int `yaml:"tally,omitempty"`
	NextSeq int          `yaml:"next_seq,omitempty"`
}

// Clone is a copy that shares nothing with l.
func (l Log) Clone() Log {
	out := l
	out.Entries = make([]Entry, len(l.Entries))
	for i, e := range l.Entries {
		e.Members = append([]string(nil), e.Members...)
		e.Keys = append([]string(nil), e.Keys...)
		out.Entries[i] = e
	}
	if l.Tally != nil {
		out.Tally = make(map[Kind]int, len(l.Tally))
		for k, v := range l.Tally {
			out.Tally[k] = v
		}
	}
	return out
}

// Add appends a deed, numbering it, and trims the log to MaxEntries. It
// returns the stored entry.
func (l *Log) Add(e Entry) Entry {
	l.NextSeq++
	e.Seq = l.NextSeq
	l.Entries = append(l.Entries, e)
	if over := len(l.Entries) - MaxEntries; over > 0 {
		l.Entries = append([]Entry(nil), l.Entries[over:]...)
	}
	if l.Tally == nil {
		l.Tally = map[Kind]int{}
	}
	l.Tally[e.Kind]++
	return e
}

// Query is the deeds that pass f, newest first.
func (l Log) Query(f Filter) []Entry {
	var out []Entry
	for i := len(l.Entries) - 1; i >= 0; i-- {
		if f.Matches(l.Entries[i]) {
			out = append(out, l.Entries[i])
			if f.Limit > 0 && len(out) >= f.Limit {
				break
			}
		}
	}
	return out
}

// Provider is what holds the chronicles. The module implements it; Memory
// is a plain one for tests.
type Provider interface {
	// Record adds a deed to a company's (its leader's user id) chronicle.
	// A zero At is stamped with the time now; an empty Place with where the
	// leader stands.
	Record(leaderUserID int, e Entry)
	// Log is a copy of the company's chronicle.
	Log(leaderUserID int) Log
}

var (
	mu       sync.RWMutex
	provider Provider
)

// SetProvider installs the module. Nil removes it.
func SetProvider(p Provider) {
	mu.Lock()
	defer mu.Unlock()
	provider = p
}

func current() Provider {
	mu.RLock()
	defer mu.RUnlock()
	return provider
}

var observers []func(leaderUserID int, e Entry)

// OnRecord calls fn for every deed a caller records, after it is kept (or
// when no module is installed). Phase 67's relic awakenings watch the deeds
// this way, so a lair's master slain advances a relic from the same record
// the chronicle keeps. fn must not block and may record deeds of its own.
func OnRecord(fn func(leaderUserID int, e Entry)) {
	mu.Lock()
	defer mu.Unlock()
	observers = append(observers, fn)
}

// Record notes a deed. It does nothing while no module is installed, so a
// caller never needs to check.
func Record(leaderUserID int, e Entry) {
	if leaderUserID <= 0 || !e.Kind.Valid() {
		return
	}
	if p := current(); p != nil {
		p.Record(leaderUserID, e)
	}
	mu.RLock()
	watchers := observers
	mu.RUnlock()
	for _, fn := range watchers {
		fn(leaderUserID, e)
	}
}

// Query is the deeds that pass f, newest first.
func Query(leaderUserID int, f Filter) []Entry {
	if p := current(); p != nil {
		return p.Log(leaderUserID).Query(f)
	}
	return nil
}

// Count is how many of the kept deeds pass f (Limit ignored).
func Count(leaderUserID int, f Filter) int {
	f.Limit = 0
	return len(Query(leaderUserID, f))
}

// Has reports whether a kept deed passes f.
func Has(leaderUserID int, f Filter) bool {
	f.Limit = 1
	return len(Query(leaderUserID, f)) > 0
}

// Total is how many deeds of a kind were ever recorded, including those the
// log has since dropped.
func Total(leaderUserID int, k Kind) int {
	if p := current(); p != nil {
		return p.Log(leaderUserID).Tally[k]
	}
	return 0
}

// Memory is an in-memory Provider for tests of other packages.
type Memory struct {
	mu   sync.Mutex
	logs map[int]Log
	Now  func() time.Time
}

// NewMemory is an empty Memory.
func NewMemory() *Memory { return &Memory{logs: map[int]Log{}, Now: time.Now} }

func (m *Memory) Record(leaderUserID int, e Entry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e.At == 0 {
		e.At = m.Now().Unix()
	}
	l := m.logs[leaderUserID]
	l.Add(e)
	m.logs[leaderUserID] = l
}

func (m *Memory) Log(leaderUserID int) Log {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.logs[leaderUserID].Clone()
}
