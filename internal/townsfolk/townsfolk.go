// Package townsfolk is Phase 68's towns that remember: town NPCs tagged as
// talkers mention what a company has done, once per deed per player, and
// otherwise fall back to a line about the weather or the hour, or say
// nothing. It is GoMud-free: the line shapes, validation, the choice of a
// line and the speaking seam live here; saved memory, data loading and the
// `townsfolk` command are in modules/townsfolk.
//
// Lines are data (YAML), keyed by chronicle deed kind (and optionally a
// stable ref) and by the NPC's townsfolk tags, so a replacement world brings
// its own. A deed line is told once per player per deed; the engine's idle
// chatter limits (internal/mobs/chatter.go) still apply on top.
package townsfolk

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"gopkg.in/yaml.v2"
)

const (
	// DefaultDays is how long a deed stays worth mentioning.
	DefaultDays = 14
	// MaxDays caps a line's own window.
	MaxDays = 90
	// MaxText is the longest line.
	MaxText = 240
)

var (
	idPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,47}$`)
	refPattern  = regexp.MustCompile(`^(mob|item|event|class|scenario):[a-z0-9][a-z0-9_-]{0,47}$`)
	placeholder = regexp.MustCompile(`\{[^{}]*\}`)
)

// Placeholders a line may use: the members named in the deed, its subject
// (the boss, the relic, the event) and the place it happened.
var knownPlaceholders = map[string]bool{"{who}": true, "{subject}": true, "{place}": true}

// Line is one thing a townsperson may say. With a Kind it speaks of a deed;
// without one it is a state line (weather, hour) the NPC falls back to.
type Line struct {
	ID   string `yaml:"id"`
	Text string `yaml:"text"`

	// Who says it. Tags are townsfolk tags on the mob template (`townsfolk:`);
	// no Tags means any talker. Zones, when given, limits it to NPCs in them.
	Tags  []string `yaml:"tags,omitempty"`
	Zones []string `yaml:"zones,omitempty"`

	// A deed line.
	Kind chronicle.Kind `yaml:"kind,omitempty"`
	Ref  string         `yaml:"ref,omitempty"`  // only this stable ref ("mob:12"); a ref line beats a plain kind line
	Days int            `yaml:"days,omitempty"` // how long the deed is worth mentioning (default 14)

	// A state line: any of these weather names (lowercase), and/or a time.
	Weather []string `yaml:"weather,omitempty"`
	Time    string   `yaml:"time,omitempty"` // "day" or "night"

	// Conditions and marks, for any line. Flag and NoFlag read company flags
	// (the marks story events leave); MemberTag needs a member named in the
	// deed to carry a tag (Phase 72 backgrounds register tags); Sets leaves a
	// company flag when a deed line is told, so a later line or scene can
	// build on it.
	Flag      string `yaml:"flag,omitempty"`
	NoFlag    string `yaml:"no_flag,omitempty"`
	MemberTag string `yaml:"member_tag,omitempty"`
	Sets      string `yaml:"sets,omitempty"`
}

// IsDeed reports whether the line speaks of a deed.
func (l Line) IsDeed() bool { return l.Kind != "" }

// Window is the line's deed window in seconds.
func (l Line) Window() int64 {
	d := l.Days
	if d <= 0 {
		d = DefaultDays
	}
	return int64(d) * 24 * 3600
}

// Validate lists what is wrong with a line; empty means usable.
func (l Line) Validate() []string {
	var p []string
	if !idPattern.MatchString(l.ID) {
		p = append(p, "id must be lowercase words joined by dashes")
	}
	text := strings.TrimSpace(l.Text)
	switch {
	case text == "":
		p = append(p, "text is empty")
	case len(text) > MaxText:
		p = append(p, fmt.Sprintf("text is over %d characters", MaxText))
	case strings.ContainsAny(text, "\n\r;"):
		p = append(p, "text must be one line without semicolons")
	}
	for _, ph := range placeholder.FindAllString(l.Text, -1) {
		if !knownPlaceholders[ph] {
			p = append(p, fmt.Sprintf("unknown placeholder %s", ph))
		}
	}
	if l.IsDeed() {
		if !l.Kind.Valid() {
			p = append(p, fmt.Sprintf("unknown deed kind %q", l.Kind))
		}
		if l.Ref != "" && !refPattern.MatchString(l.Ref) {
			p = append(p, fmt.Sprintf("ref %q must look like mob:12", l.Ref))
		}
		if len(l.Weather) > 0 || l.Time != "" {
			p = append(p, "a deed line has no weather or time")
		}
		if l.Days < 0 || l.Days > MaxDays {
			p = append(p, fmt.Sprintf("days must be 0 (default %d) to %d", DefaultDays, MaxDays))
		}
	} else {
		if l.Ref != "" || l.Days != 0 || l.Sets != "" || l.MemberTag != "" {
			p = append(p, "ref, days, sets and member_tag belong to a deed line (give it a kind)")
		}
		if len(l.Weather) == 0 && l.Time == "" {
			p = append(p, "a line needs a kind, or a weather or time to speak of")
		}
		if strings.Contains(l.Text, "{") {
			p = append(p, "a state line has no placeholders")
		}
	}
	if l.Time != "" && l.Time != "day" && l.Time != "night" {
		p = append(p, `time must be "day" or "night"`)
	}
	for _, w := range l.Weather {
		if w != strings.ToLower(strings.TrimSpace(w)) || w == "" {
			p = append(p, fmt.Sprintf("weather %q must be lowercase", w))
		}
	}
	for _, f := range []string{l.Flag, l.NoFlag, l.Sets} {
		if f != "" && !idPattern.MatchString(f) {
			p = append(p, fmt.Sprintf("flag %q must be lowercase words joined by dashes", f))
		}
	}
	if l.MemberTag != "" && !idPattern.MatchString(l.MemberTag) {
		p = append(p, fmt.Sprintf("member_tag %q must be lowercase words joined by dashes", l.MemberTag))
	}
	for _, t := range append(append([]string(nil), l.Tags...), l.Zones...) {
		if strings.TrimSpace(t) == "" {
			p = append(p, "a tag or zone is blank")
		}
	}
	return p
}

// Parse reads one file: a YAML list of lines. Unknown fields are errors.
func Parse(data []byte) ([]Line, error) {
	var list []Line
	if err := yaml.UnmarshalStrict(data, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// Catalog is the usable lines in id order.
type Catalog struct{ lines []Line }

// NewCatalog keeps the sound lines; a duplicate id keeps the first. The
// returned problems name each rejected line.
func NewCatalog(list []Line) (Catalog, []string) {
	var c Catalog
	var problems []string
	seen := map[string]bool{}
	for _, l := range list {
		if seen[l.ID] {
			problems = append(problems, fmt.Sprintf("line %q: defined twice; the first is used", l.ID))
			continue
		}
		if errs := l.Validate(); len(errs) > 0 {
			for _, e := range errs {
				problems = append(problems, fmt.Sprintf("line %q: %s", l.ID, e))
			}
			continue
		}
		seen[l.ID] = true
		c.lines = append(c.lines, l)
	}
	sort.Slice(c.lines, func(i, j int) bool { return c.lines[i].ID < c.lines[j].ID })
	return c, problems
}

// Lines lists the catalog in id order.
func (c Catalog) Lines() []Line { return append([]Line(nil), c.lines...) }

// Len is how many lines the catalog holds.
func (c Catalog) Len() int { return len(c.lines) }

// NPC is the talker: its template id, its townsfolk tags and its zone.
type NPC struct {
	MobID int
	Tags  []string
	Zone  string
}

func (n NPC) hasTag(t string) bool {
	for _, x := range n.Tags {
		if strings.EqualFold(x, t) {
			return true
		}
	}
	return false
}

// saidBy reports whether the NPC may say the line: its tags and zone.
func (l Line) saidBy(n NPC) bool {
	if len(l.Tags) > 0 {
		ok := false
		for _, t := range l.Tags {
			if n.hasTag(t) {
				ok = true
			}
		}
		if !ok {
			return false
		}
	}
	if len(l.Zones) > 0 {
		ok := false
		for _, z := range l.Zones {
			if strings.EqualFold(z, n.Zone) {
				ok = true
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

// LeaderKey is the leader's company key in a deed's Keys (chronicle and
// story events use the same one).
const LeaderKey = "leader"

// Context is what one listener's choice reads. Entries are the company's
// deeds, newest first.
type Context struct {
	NPC     NPC
	Now     int64
	Entries []chronicle.Entry
	// Leader is the listener's character name. A deed that names no member
	// (a boss slain, a relic found, a mercy answer: the company's deeds) is
	// the leader's: {who} says it and member_tag reads the leader's tags.
	Leader string
	// Heard reports whether the player was already told of this deed.
	Heard func(seq int) bool
	// Flag reports a company flag.
	Flag func(flag string) bool
	// MemberTag reports whether a member (by company key) carries a tag.
	MemberTag func(key, tag string) bool
	// Weather is the lowercase weather name in the NPC's zone ("" unknown).
	Weather string
	Night   bool
	// Rand picks 0..n-1; nil takes the first.
	Rand func(n int) int
}

func (c Context) pick(n int) int {
	if c.Rand == nil || n <= 1 {
		return 0
	}
	return c.Rand(n)
}

func (c Context) flag(f string) bool { return c.Flag != nil && c.Flag(f) }

// conditionsMet checks a line's flag and member-tag conditions.
func (l Line) conditionsMet(c Context, e *chronicle.Entry) bool {
	if l.Flag != "" && !c.flag(l.Flag) {
		return false
	}
	if l.NoFlag != "" && c.flag(l.NoFlag) {
		return false
	}
	if l.MemberTag != "" {
		if e == nil || c.MemberTag == nil {
			return false
		}
		ok := false
		for _, k := range e.Keys {
			if c.MemberTag(k, l.MemberTag) {
				ok = true
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

// Choice is what an NPC says to a player.
type Choice struct {
	Line  Line
	Entry *chronicle.Entry // the deed told; nil for a state line
	Text  string
}

// companyDeed gives a deed that names no member to the leader, so lines
// read "Mara put down the ogre" rather than "The company put down".
func (c Context) companyDeed(e chronicle.Entry) chronicle.Entry {
	if len(e.Members) == 0 && len(e.Keys) == 0 && strings.TrimSpace(c.Leader) != "" {
		e.Members = []string{c.Leader}
		e.Keys = []string{LeaderKey}
	}
	return e
}

// Choose picks what the NPC says to one player: the newest unheard deed
// that has a line this NPC may say (a ref-specific line beats a plain kind
// line), else a state line, else nothing.
func (cat Catalog) Choose(c Context) (Choice, bool) {
	for i := range c.Entries {
		e := c.companyDeed(c.Entries[i])
		if c.Heard != nil && c.Heard(e.Seq) {
			continue
		}
		var specific, general []Line
		for _, l := range cat.lines {
			if !l.IsDeed() || l.Kind != e.Kind || !l.saidBy(c.NPC) {
				continue
			}
			if c.Now-e.At > l.Window() || !l.conditionsMet(c, &e) {
				continue
			}
			switch {
			case l.Ref == "":
				general = append(general, l)
			case l.Ref == e.Ref:
				specific = append(specific, l)
			}
		}
		pool := specific
		if len(pool) == 0 {
			pool = general
		}
		if len(pool) == 0 {
			continue
		}
		l := pool[c.pick(len(pool))]
		return Choice{Line: l, Entry: &e, Text: Fill(l.Text, e)}, true
	}
	var states []Line
	for _, l := range cat.lines {
		if l.IsDeed() || !l.saidBy(c.NPC) || !l.conditionsMet(c, nil) {
			continue
		}
		if len(l.Weather) > 0 {
			ok := false
			for _, w := range l.Weather {
				if c.Weather != "" && w == strings.ToLower(c.Weather) {
					ok = true
				}
			}
			if !ok {
				continue
			}
		}
		if (l.Time == "night" && !c.Night) || (l.Time == "day" && c.Night) {
			continue
		}
		states = append(states, l)
	}
	if len(states) == 0 {
		return Choice{}, false
	}
	l := states[c.pick(len(states))]
	return Choice{Line: l, Text: l.Text}, true
}

// Fill puts a deed's members, subject and place into a line.
func Fill(text string, e chronicle.Entry) string {
	place := strings.TrimSpace(e.Place)
	if place == "" {
		place = "these parts"
	}
	subject := strings.TrimSpace(e.Subject)
	if subject == "" {
		subject = "it"
	}
	// A semicolon would split the NPC's command in two.
	clean := strings.NewReplacer(";", ",", "\n", " ", "\r", " ")
	r := strings.NewReplacer("{who}", clean.Replace(chronicle.Who(e.Members)), "{subject}", clean.Replace(subject), "{place}", clean.Replace(place))
	return r.Replace(text)
}

// DaysAgo is how many whole days ago a deed happened, for views.
func DaysAgo(at, now int64) int {
	if now <= at {
		return 0
	}
	return int((now - at) / (24 * 3600))
}

// Speech is what an NPC says: To is the player it is said to, as a target
// the say commands take ("@12"; "" says it to the room).
type Speech struct {
	To   string
	Text string
	// Said, when set, records the telling. The caller calls Confirm once the
	// line has really been let through, so a line the chatter limits hold
	// back is not used up.
	Said func()
}

// Confirm records that the speech was said.
func (s Speech) Confirm() {
	if s.Said != nil {
		s.Said()
	}
}

// Provider is implemented by modules/townsfolk.
type Provider interface {
	// Speak decides what the NPC says to the players who hear it (user ids).
	// Nothing is recorded until the caller confirms the speech. ok is false
	// when it has nothing to say.
	Speak(npc NPC, listeners []int) (Speech, bool)
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

// Speak asks the installed module what the NPC says. Without a module it
// says nothing, so a caller never needs to check.
func Speak(npc NPC, listeners []int) (Speech, bool) {
	mu.RLock()
	p := provider
	mu.RUnlock()
	if p == nil || len(listeners) == 0 || len(npc.Tags) == 0 {
		return Speech{}, false
	}
	return p.Speak(npc, listeners)
}
