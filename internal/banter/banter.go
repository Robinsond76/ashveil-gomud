// Package banter is the company's camp talk and after-battle chatter
// (Ashveil Phase 49). Companions say a line or two to each other, drawn
// from a pool of authored lines tagged by archetype, personality, alignment
// and context, so a company feels like comrades. The package is pure: the
// company module supplies who is talking and what just happened, and
// shows the result.
package banter

import (
	"embed"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed data/*.yaml
var dataFiles embed.FS

// OptionKey is the player config option that turns banter off: a bool,
// on when unset.
const OptionKey = "banter"

// Enabled reads the saved option: on unless explicitly false.
func Enabled(saved any) bool {
	if on, ok := saved.(bool); ok {
		return on
	}
	return true
}

// Contexts a line can be said in.
const (
	CtxCamp     = "camp"     // settling in by the fire
	CtxRested   = "rested"   // a rest is over
	CtxWin      = "win"      // any won battle
	CtxClose    = "close"    // a won battle that was a close call
	CtxFall     = "fall"     // a won battle in which someone fell
	CtxFlawless = "flawless" // a won battle nobody was hurt much in
)

// Personalities are the temperaments a companion is rolled with.
var Personalities = []string{"stoic", "cheerful", "grim", "boastful", "wry", "devout"}

// Verbs is how each personality delivers a line.
var verbs = map[string]string{
	"stoic": "says", "cheerful": "laughs", "grim": "mutters",
	"boastful": "boasts", "wry": "remarks", "devout": "murmurs",
}

// Groups are archetype families a line may be tagged with instead of one
// archetype.
var groups = map[string]string{
	"warrior": "martial", "halberdier": "martial", "samurai": "martial",
	"wizard": "arcane", "witch": "arcane", "dollmaster": "arcane",
	"cleric": "faith", "shaman": "faith",
	"rogue": "skirmisher", "ranger": "skirmisher", "gryphon-rider": "skirmisher",
}

// GroupOf is an archetype's family ("" when it has none).
func GroupOf(archetype string) string { return groups[strings.ToLower(archetype)] }

// ValidPersonality reports whether p is a known personality.
func ValidPersonality(p string) bool { return verbs[p] != "" }

// Bucket maps an engine alignment (-100..100) to good, neutral, or evil,
// at the same lawful and misguided thresholds the rest of the game uses.
func Bucket(alignment int) string {
	switch {
	case alignment >= 20:
		return "good"
	case alignment <= -20:
		return "evil"
	}
	return "neutral"
}

// Line is one authored line.
type Line struct {
	ID   string   `yaml:"id,omitempty"`
	Text string   `yaml:"text"`
	Ctx  []string `yaml:"ctx,omitempty"`
	// Arch, Pers and Align are required tags: the speaker must have one
	// of the values listed for each that is set. Arch takes an archetype
	// or a group.
	Arch  []string `yaml:"arch,omitempty"`
	Pers  []string `yaml:"pers,omitempty"`
	Align []string `yaml:"align,omitempty"`
	// Prefer lists tags (of any kind) that make the line likelier.
	Prefer []string `yaml:"prefer,omitempty"`
	// Reply names the line this one answers. A reply is said only after
	// that line, by another member.
	Reply string `yaml:"reply,omitempty"`

	needsOther  bool
	needsFallen bool
	idx         int
}

// Member is a company member who may speak.
type Member struct {
	ID          int
	Name        string
	Archetype   string
	Personality string
	Alignment   int
}

func (m Member) short() string {
	name := strings.TrimSpace(m.Name)
	if i := strings.IndexByte(name, ' '); i > 0 && name[0] >= 'A' && name[0] <= 'Z' {
		return name[:i]
	}
	return name
}

func (m Member) has(tag string) bool {
	return tag == m.Archetype || tag == GroupOf(m.Archetype) || tag == m.Personality || tag == Bucket(m.Alignment)
}

// Said is one spoken line.
type Said struct {
	Member int // the speaker's Member.ID
	Name   string
	Text   string // the line, names filled in
	Verb   string
	LineID string
}

// Pool is the loaded set of lines.
type Pool struct {
	lines   []Line
	byID    map[string]int
	replies map[string][]int
}

// Load reads and checks the embedded line files.
func Load() (*Pool, error) {
	entries, err := dataFiles.ReadDir("data")
	if err != nil {
		return nil, err
	}
	var all []Line
	for _, e := range entries {
		data, err := dataFiles.ReadFile("data/" + e.Name())
		if err != nil {
			return nil, err
		}
		var lines []Line
		if err := yaml.Unmarshal(data, &lines); err != nil {
			return nil, fmt.Errorf("banter %s: %w", e.Name(), err)
		}
		all = append(all, lines...)
	}
	return NewPool(all)
}

// NewPool checks lines and indexes them.
func NewPool(lines []Line) (*Pool, error) {
	p := &Pool{byID: map[string]int{}, replies: map[string][]int{}}
	for i, l := range lines {
		if strings.TrimSpace(l.Text) == "" {
			return nil, fmt.Errorf("banter line %d has no text", i)
		}
		for _, p := range l.Pers {
			if !ValidPersonality(p) {
				return nil, fmt.Errorf("banter %q: unknown personality %q", l.Text, p)
			}
		}
		for _, a := range l.Align {
			if a != "good" && a != "neutral" && a != "evil" {
				return nil, fmt.Errorf("banter %q: unknown alignment %q", l.Text, a)
			}
		}
		l.needsOther = strings.Contains(l.Text, "{other}")
		l.needsFallen = strings.Contains(l.Text, "{fallen}")
		l.idx = i
		if l.ID == "" {
			h := fnv.New32a()
			h.Write([]byte(l.Text))
			l.ID = fmt.Sprintf("t%08x", h.Sum32())
		}
		if _, dup := p.byID[l.ID]; dup {
			return nil, fmt.Errorf("banter: duplicate line id %q", l.ID)
		}
		p.byID[l.ID] = len(p.lines)
		p.lines = append(p.lines, l)
	}
	for i, l := range p.lines {
		if l.Reply == "" {
			continue
		}
		if _, ok := p.byID[l.Reply]; !ok {
			return nil, fmt.Errorf("banter %q answers unknown line %q", l.ID, l.Reply)
		}
		p.replies[l.Reply] = append(p.replies[l.Reply], i)
	}
	return p, nil
}

// Len is the number of lines.
func (p *Pool) Len() int { return len(p.lines) }

// Lines returns a copy of the lines, for tests and tools.
func (p *Pool) Lines() []Line { return append([]Line(nil), p.lines...) }

// Rand is the randomness used: Intn returns [0, n).
type Rand interface{ Intn(n int) int }

// Request is one exchange to make.
type Request struct {
	// Contexts, most specific first. Each is tried in turn until lines
	// exist for the first speaker.
	Contexts []string
	// Members who can speak. Nobody speaks alone: fewer than two ends it.
	Members []Member
	// Leader is the player's name, for {leader}.
	Leader string
	// Fallen names a member who fell, for {fallen}.
	Fallen string
	// Recent is the line ids a member has said lately, by Member.ID, which
	// are not repeated.
	Recent map[int]map[string]bool
}

// matches reports whether m satisfies a line's required tags.
func (l Line) matches(m Member) bool {
	if len(l.Arch) > 0 && !anyHas(m, l.Arch, func(m Member, a string) bool { return a == m.Archetype || a == GroupOf(m.Archetype) }) {
		return false
	}
	if len(l.Pers) > 0 && !contains(l.Pers, m.Personality) {
		return false
	}
	if len(l.Align) > 0 && !contains(l.Align, Bucket(m.Alignment)) {
		return false
	}
	return true
}

func anyHas(m Member, tags []string, f func(Member, string) bool) bool {
	for _, t := range tags {
		if f(m, t) {
			return true
		}
	}
	return false
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// weight is how likely a matching line is: each required tag and each
// preferred tag the speaker has makes it likelier.
func (l Line) weight(m Member) int {
	w := 1
	if len(l.Arch) > 0 {
		w += 2
	}
	if len(l.Pers) > 0 {
		w += 2
	}
	if len(l.Align) > 0 {
		w += 2
	}
	for _, t := range l.Prefer {
		if m.has(t) {
			w += 2
		}
	}
	return w
}

func inContext(l Line, ctx string) bool { return contains(l.Ctx, ctx) }

// pick draws from candidates by weight; ok is false with none.
func (p *Pool) pick(rng Rand, m Member, req Request, candidates []int) (int, bool) {
	total := 0
	weights := make([]int, len(candidates))
	for i, c := range candidates {
		weights[i] = p.lines[c].weight(m)
		total += weights[i]
	}
	if total == 0 {
		return 0, false
	}
	n := rng.Intn(total)
	for i, w := range weights {
		if n < w {
			return candidates[i], true
		}
		n -= w
	}
	return 0, false
}

// usable lists line indexes speaker m may say in ctx: its tags match, it
// was not said lately, and the people it names exist. replies selects
// reply lines answering one prompt (prompt ""), or ordinary lines.
func (p *Pool) usable(m Member, req Request, ctx, prompt string, others int) []int {
	var out []int
	for i, l := range p.lines {
		if !inContext(l, ctx) || !l.matches(m) {
			continue
		}
		if prompt == "" && l.Reply != "" || prompt != "" && l.Reply != prompt {
			continue
		}
		if l.needsOther && others < 1 || l.needsFallen && req.Fallen == "" {
			continue
		}
		if req.Recent[m.ID][l.ID] {
			continue
		}
		out = append(out, i)
	}
	return out
}

func (p *Pool) say(l Line, m Member, others []Member, rng Rand, req Request) Said {
	text := l.Text
	if l.needsOther && len(others) > 0 {
		o := others[rng.Intn(len(others))]
		text = strings.ReplaceAll(text, "{other}", o.short())
	}
	text = strings.ReplaceAll(text, "{fallen}", req.Fallen)
	text = strings.ReplaceAll(text, "{leader}", req.Leader)
	text = strings.ReplaceAll(text, "{name}", m.short())
	verb := verbs[m.Personality]
	if verb == "" {
		verb = "says"
	}
	return Said{Member: m.ID, Name: m.Name, Text: text, Verb: verb, LineID: l.ID}
}

func without(ms []Member, id int) []Member {
	var out []Member
	for _, m := range ms {
		if m.ID != id {
			out = append(out, m)
		}
	}
	return out
}

// Exchange builds a short conversation of two to four lines among two or
// three members, or nil when the pool has nothing to say for them. Some
// openers have replies: the second voice then answers the first.
func (p *Pool) Exchange(rng Rand, req Request) []Said {
	if len(req.Members) < 2 {
		return nil
	}
	members := append([]Member(nil), req.Members...)
	sort.Slice(members, func(i, j int) bool { return members[i].ID < members[j].ID })
	for i := len(members) - 1; i > 0; i-- { // shuffle
		j := rng.Intn(i + 1)
		members[i], members[j] = members[j], members[i]
	}
	size := 2
	if len(members) >= 3 && rng.Intn(3) == 0 {
		size = 3
	}
	members = members[:size]

	for _, ctx := range req.Contexts {
		if said := p.exchangeIn(rng, req, members, ctx); len(said) >= 2 {
			return said
		}
	}
	return nil
}

func (p *Pool) exchangeIn(rng Rand, req Request, members []Member, ctx string) []Said {
	// The opener is the first member with something to say.
	for _, opener := range members {
		others := without(members, opener.ID)
		cands := p.usable(opener, req, ctx, "", len(others))
		line, ok := p.pick(rng, opener, req, cands)
		if !ok {
			continue
		}
		first := p.lines[line]
		said := []Said{p.say(first, opener, others, rng, req)}
		// The others answer: with a reply when the opener has one they can
		// say, otherwise with a line of their own.
		last := first
		for _, m := range others {
			rest := without(members, m.ID)
			var cands []int
			if replies := p.replies[last.ID]; len(replies) > 0 {
				cands = p.repliesFor(m, req, last.ID, len(rest))
			}
			if len(cands) == 0 {
				cands = p.usable(m, req, ctx, "", len(rest))
			}
			l, ok := p.pick(rng, m, req, cands)
			if !ok {
				continue
			}
			said = append(said, p.say(p.lines[l], m, rest, rng, req))
			last = p.lines[l]
			if len(said) == 3 {
				break
			}
		}
		if len(said) < 2 {
			continue
		}
		// Sometimes the opener has a last word.
		if len(said) < 4 && rng.Intn(3) == 0 {
			rest := without(members, opener.ID)
			var cands []int
			for _, c := range p.usable(opener, req, ctx, "", len(rest)) {
				if p.lines[c].ID != first.ID {
					cands = append(cands, c)
				}
			}
			if l, ok := p.pick(rng, opener, req, cands); ok {
				said = append(said, p.say(p.lines[l], opener, rest, rng, req))
			}
		}
		return said
	}
	return nil
}

// repliesFor lists the lines speaker m can say answering prompt (a reply
// is usable wherever its prompt was said).
func (p *Pool) repliesFor(m Member, req Request, prompt string, others int) []int {
	var out []int
	for _, i := range p.replies[prompt] {
		l := p.lines[i]
		if !l.matches(m) || l.needsOther && others < 1 || l.needsFallen && req.Fallen == "" || req.Recent[m.ID][l.ID] {
			continue
		}
		out = append(out, i)
	}
	return out
}

// Format is an exchange as game text, one line per voice, each with the
// speaker's name colored.
func Format(said []Said) string {
	var b strings.Builder
	for i, s := range said {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, `<ansi fg="cyan">%s</ansi> %s, "%s"`, s.Name, s.Verb, s.Text)
	}
	return b.String()
}

// PersonalityFor is the personality used for a member that has none saved:
// a fixed pick from its ID, so it never changes between exchanges.
func PersonalityFor(id int) string {
	if id < 0 {
		id = -id
	}
	return Personalities[id%len(Personalities)]
}
