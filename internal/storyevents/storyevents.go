// Package storyevents is Phase 60's rules: story events, short scenes shown
// as a page of text with choices, which a company walks into at a cliff, a
// stranger's fire or a ruined shrine. It is GoMud-free and pure: the YAML
// shapes, their validation, which member a choice falls to, and the text.
// modules/storyevents owns the saved state, the triggers, the commands and
// the world effects.
//
// Events are data. A file under `events/` (embedded in the module, or in the
// world's own `events` folder) holds one or more events; later phases hang
// their own hooks on the same shapes: member tags (a background, phase 72),
// company flags (towns that remember, phase 68) and the outcome list.
package storyevents

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/lifestory"
	"github.com/GoMudEngine/GoMud/internal/opinions"
)

// Trigger kinds.
const (
	// TriggerRoom fires when the company walks (or a journey arrives) into
	// a given room.
	TriggerRoom = "room"
	// TriggerTag fires on entering any room carrying a tag; a lair's door is
	// a tagged room.
	TriggerTag = "tag"
	// TriggerArrival fires when a journey ends in a zone: the road's own
	// event, at a chance.
	TriggerArrival = "arrival"
	// TriggerCamp fires when a camp rest finishes in a zone.
	TriggerCamp = "camp"
)

// Outcome kinds.
const (
	OutcomeWound    = "wound"     // a lasting wound, a share of max health
	OutcomeAilment  = "ailment"   // chill, gut ache or fever
	OutcomeNeed     = "need"      // hunger, thirst or fatigue drain
	OutcomeItem     = "item"      // supplies or an item gained into the cargo
	OutcomeLoseItem = "lose_item" // supplies or an item taken from the cargo
	OutcomeGold     = "gold"      // the leader's gold, gained or lost
	OutcomeLoyalty  = "loyalty"   // companions' loyalty, up or down
	OutcomeBattle   = "battle"    // a fight with a named group
	OutcomeMove     = "move"      // the company is carried to a room
	OutcomeFlag     = "flag"      // a company flag, for later events and hooks
)

// Who an outcome lands on.
const (
	WhoActor  = "actor"  // the member who took the choice (the default)
	WhoLeader = "leader" // the player
	WhoAll    = "all"    // every member present
	WhoRandom = "random" // one member present, by chance
)

// Limits that keep an event from becoming a gold or item farm, and a
// page graph from looping forever.
const (
	MaxGold        = 500 // most gold one outcome may give or take
	MaxItemCount   = 10
	MaxWoundPct    = 50
	MaxNeed        = 60
	MaxLoyalty     = 25
	MaxRiskPct     = 95
	MaxChoices     = 6
	MaxFoes        = 5
	FoeLevelMax    = 100
	DefaultStart   = "start"
	maxTextLen     = 2000
	maxLabelLen    = 90
	cooldownMaxMin = 60 * 24 * 30
)

var (
	idPattern      = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,47}$`)
	pictureKeyExpr = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,47}$`)
)

// Event is one authored story event.
type Event struct {
	ID    string `yaml:"id"`
	Title string `yaml:"title"`
	// Picture is a key for an illustration (`static/images/events/<key>.png`
	// in the web client); optional, and pages may name their own.
	Picture  string    `yaml:"picture,omitempty"`
	Triggers []Trigger `yaml:"triggers"`
	// CooldownMinutes is the real time before the same company may see the
	// event again; zero means once per company, ever.
	CooldownMinutes int `yaml:"cooldown_minutes,omitempty"`
	// MinLevel and MaxLevel bound the company's level (the leader's) the
	// event may open for; zero is open.
	MinLevel int `yaml:"min_level,omitempty"`
	MaxLevel int `yaml:"max_level,omitempty"`
	// Require is checked against the company before the event opens (a flag
	// an earlier event set, say).
	Require CompanyRequirement `yaml:"require,omitempty"`
	// Start is the first page's id (default "start").
	Start string          `yaml:"start,omitempty"`
	Pages map[string]Page `yaml:"pages"`
}

// Trigger is one way an event opens.
type Trigger struct {
	Kind string `yaml:"kind"`
	Room int    `yaml:"room,omitempty"`
	Tag  string `yaml:"tag,omitempty"`
	Zone string `yaml:"zone,omitempty"`
	// Chance is the percent chance the trigger opens the event when it
	// fires; zero means always.
	Chance int `yaml:"chance,omitempty"`
}

// Page is one page: a picture, a paragraph and choices.
type Page struct {
	Text    string   `yaml:"text"`
	Picture string   `yaml:"picture,omitempty"`
	Choices []Choice `yaml:"choices"`
}

// Choice is one option on a page.
type Choice struct {
	Label string `yaml:"label"`
	// Hint is what a locked choice says it needs, when the requirement does
	// not read well on its own.
	Hint    string      `yaml:"hint,omitempty"`
	Require Requirement `yaml:"require,omitempty"`
	// Risk, when set, lets the choice fail: its Fail outcomes and text run
	// instead.
	Risk *Risk `yaml:"risk,omitempty"`
	// Stance, when set, is the kind of choice this is to a companion's eyes
	// (Phase 64 opinions): kindness, greed, courage, prudence, reverence or
	// cunning. Taking the choice, whether or not its risk falls, lets the
	// companions with the company say what they think.
	Stance string `yaml:"stance,omitempty"`
	// Text is what happens, shown after the choice; {who} is the member who
	// took it.
	Text string    `yaml:"text,omitempty"`
	Do   []Outcome `yaml:"do,omitempty"`
	// FailText and Fail run when the risk falls on the company.
	FailText string    `yaml:"fail_text,omitempty"`
	Fail     []Outcome `yaml:"fail,omitempty"`
	// Next is the follow-on page's id; empty ends the event. FailNext is the
	// page after a failure; empty ends the event.
	Next     string `yaml:"next,omitempty"`
	FailNext string `yaml:"fail_next,omitempty"`
}

// Risk is a choice's chance of going wrong: Pct, less LessPerLevel for each
// rank of the member who took it (their skill level when the choice asks
// for a skill, else their level).
type Risk struct {
	Pct          int `yaml:"pct"`
	LessPerLevel int `yaml:"less_per_level,omitempty"`
}

// RiskPct is the failure chance for a member of rank.
func (r Risk) RiskPct(rank int) int {
	p := r.Pct - r.LessPerLevel*max(rank, 0)
	return min(max(p, 0), MaxRiskPct)
}

// Outcome is one effect of a choice. Only the fields of its Kind are read.
type Outcome struct {
	Kind string `yaml:"kind"`
	Who  string `yaml:"who,omitempty"`

	Pct     int    `yaml:"pct,omitempty"`     // wound: share of max health
	Ailment string `yaml:"ailment,omitempty"` // ailment: chill, gutache, fever
	Stat    string `yaml:"stat,omitempty"`    // need: hunger, thirst, fatigue
	Amount  int    `yaml:"amount,omitempty"`  // need drain, gold, loyalty (signed)
	Item    int    `yaml:"item,omitempty"`    // item, lose_item
	Count   int    `yaml:"count,omitempty"`   // item, lose_item (default 1)
	Foes    []Foe  `yaml:"foes,omitempty"`    // battle
	Room    int    `yaml:"room,omitempty"`    // move
	Flag    string `yaml:"flag,omitempty"`    // flag
	// Line is shown with the outcome, when it has something to say; {who}
	// is the member it landed on.
	Line string `yaml:"line,omitempty"`
}

// Foe is one enemy of a battle outcome.
type Foe struct {
	Mob int `yaml:"mob"`
	// Level of the foe; zero takes the zone's lowest band level, or the
	// leader's level in a zone with no band.
	Level int `yaml:"level,omitempty"`
}

// Requirement gates a choice: every set field must hold. The member fields
// pick who takes the choice (the best qualified member present); the
// company fields must hold for the company as a whole.
type Requirement struct {
	// Member fields.
	Skill       string   `yaml:"skill,omitempty"`
	SkillLevel  int      `yaml:"skill_level,omitempty"` // default 1 with Skill
	Classes     []string `yaml:"classes,omitempty"`     // any of these archetypes or advanced classes
	Personality string   `yaml:"personality,omitempty"`
	Alignment   string   `yaml:"alignment,omitempty"` // good, neutral or evil
	MinLevel    int      `yaml:"min_level,omitempty"`
	Leader      bool     `yaml:"leader,omitempty"` // only the player
	Tag         string   `yaml:"tag,omitempty"`    // a member tag (a background, later)
	// CompanyRequirement fields.
	CompanyRequirement `yaml:",inline"`
}

// CompanyRequirement is the part of a requirement the company meets as a
// whole.
type CompanyRequirement struct {
	Item    int    `yaml:"item,omitempty"` // in the company's shared cargo
	Gold    int    `yaml:"gold,omitempty"`
	Flag    string `yaml:"flag,omitempty"`
	NotFlag string `yaml:"not_flag,omitempty"`
}

// MemberSet reports whether any member field is set.
func (r Requirement) MemberSet() bool {
	return r.Skill != "" || len(r.Classes) > 0 || r.Personality != "" || r.Alignment != "" ||
		r.MinLevel > 0 || r.Leader || r.Tag != ""
}

// Free reports whether the requirement asks for nothing.
func (r Requirement) Free() bool {
	return !r.MemberSet() && r.CompanyRequirement == CompanyRequirement{}
}

// Facts is what a requirement reads of one member.
type Facts struct {
	// Key is the member's key (the leader's or a companion's).
	Key         string
	Name        string
	Leader      bool
	Archetype   string
	Class       string
	Personality string
	Alignment   int
	Level       int
	Skills      map[string]int
	Tags        []string
}

// AlignmentBand is good, neutral or evil, at the thresholds banter and the
// rest of the game use.
func AlignmentBand(alignment int) string {
	switch {
	case alignment >= 20:
		return "good"
	case alignment <= -20:
		return "evil"
	}
	return "neutral"
}

// Rank is how well a member does what the requirement asks: its skill level
// when it names a skill, else the member's level.
func (r Requirement) Rank(f Facts) int {
	if r.Skill != "" {
		return f.Skills[strings.ToLower(r.Skill)]
	}
	return f.Level
}

// MeetsMember reports whether one member qualifies.
func (r Requirement) MeetsMember(f Facts) bool {
	if r.Leader && !f.Leader {
		return false
	}
	if r.Skill != "" {
		need := r.SkillLevel
		if need < 1 {
			need = 1
		}
		if f.Skills[strings.ToLower(r.Skill)] < need {
			return false
		}
	}
	if len(r.Classes) > 0 {
		match := false
		for _, c := range r.Classes {
			if strings.EqualFold(c, f.Archetype) || strings.EqualFold(c, f.Class) {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	if r.Personality != "" && !strings.EqualFold(r.Personality, f.Personality) {
		return false
	}
	if r.Alignment != "" && !strings.EqualFold(r.Alignment, AlignmentBand(f.Alignment)) {
		return false
	}
	if r.MinLevel > 0 && f.Level < r.MinLevel {
		return false
	}
	if r.Tag != "" {
		found := false
		for _, t := range f.Tags {
			if strings.EqualFold(t, r.Tag) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Company is what a requirement reads of the company as a whole.
type Company struct {
	Gold  int
	Items map[int]int
	Flags map[string]bool
}

// Meets reports whether the company satisfies the company fields.
func (c CompanyRequirement) Meets(co Company) bool {
	if c.Gold > 0 && co.Gold < c.Gold {
		return false
	}
	if c.Item > 0 && co.Items[c.Item] < 1 {
		return false
	}
	if c.Flag != "" && !co.Flags[c.Flag] {
		return false
	}
	if c.NotFlag != "" && co.Flags[c.NotFlag] {
		return false
	}
	return true
}

// Best picks who takes a choice: among the members who qualify, the highest
// rank, then the highest level; ties go to the earlier in the list (the
// leader first), so the pick never wobbles. ok is false when nobody does or
// the company fields fail.
func Best(r Requirement, members []Facts, co Company) (Facts, bool) {
	if !r.CompanyRequirement.Meets(co) {
		return Facts{}, false
	}
	var best Facts
	found := false
	for _, f := range members {
		if !r.MeetsMember(f) {
			continue
		}
		if !found || r.Rank(f) > r.Rank(best) || (r.Rank(f) == r.Rank(best) && f.Level > best.Level) {
			best, found = f, true
		}
	}
	return best, found
}

// Describe says what a requirement asks, for a locked choice.
func (r Requirement) Describe() string {
	var parts []string
	if r.Skill != "" {
		level := max(r.SkillLevel, 1)
		parts = append(parts, fmt.Sprintf("someone with %s %d", r.Skill, level))
	}
	if len(r.Classes) > 0 {
		parts = append(parts, "a "+strings.Join(r.Classes, " or a "))
	}
	if r.Personality != "" {
		parts = append(parts, "someone "+r.Personality)
	}
	if r.Alignment != "" {
		parts = append(parts, "someone of "+r.Alignment+" heart")
	}
	if r.MinLevel > 0 {
		parts = append(parts, fmt.Sprintf("a member of level %d", r.MinLevel))
	}
	if r.Leader {
		parts = append(parts, "you")
	}
	if r.Tag != "" {
		if name, ok := lifestory.TagName(r.Tag); ok {
			parts = append(parts, "life story: "+name)
		} else {
			parts = append(parts, "someone who is "+r.Tag)
		}
	}
	if r.Gold > 0 {
		parts = append(parts, fmt.Sprintf("%d gold", r.Gold))
	}
	if r.Item > 0 {
		parts = append(parts, "something you do not carry")
	}
	if r.Flag != "" || r.NotFlag != "" {
		parts = append(parts, "something the company has not done")
	}
	return strings.Join(parts, ", ")
}

// Fill puts a member's name in a text's {who}.
func Fill(text, who string) string { return strings.ReplaceAll(text, "{who}", who) }

// StartPage is the event's first page id.
func (e Event) StartPage() string {
	if e.Start != "" {
		return e.Start
	}
	return DefaultStart
}

// Lookups lets validation check ids the package cannot see. A nil func
// accepts everything.
type Lookups struct {
	Item    func(id int) bool
	Mob     func(id int) bool
	Room    func(id int) bool
	Skill   func(id string) bool
	Ailment func(kind string) bool
	// Class accepts an archetype or class id; Personality a temperament.
	Class       func(id string) bool
	Personality func(name string) bool
}

func (l Lookups) item(id int) bool { return l.Item == nil || l.Item(id) }
func (l Lookups) mob(id int) bool  { return l.Mob == nil || l.Mob(id) }
func (l Lookups) room(id int) bool { return l.Room == nil || l.Room(id) }

// Validate lists every problem with an event; an event with any is not
// used. The rules keep a scene always leavable (a free choice on every
// page), its pages acyclic and all reachable, and its outcomes small.
func (e Event) Validate(l Lookups) []string {
	var errs []string
	bad := func(format string, args ...any) { errs = append(errs, fmt.Sprintf(format, args...)) }
	if !idPattern.MatchString(e.ID) {
		bad("id %q must be lowercase words joined by dashes", e.ID)
	}
	if strings.TrimSpace(e.Title) == "" {
		bad("title is empty")
	}
	if e.Picture != "" && !pictureKeyExpr.MatchString(e.Picture) {
		bad("picture %q must be a key of lowercase letters, digits and dashes", e.Picture)
	}
	if e.CooldownMinutes < 0 || e.CooldownMinutes > cooldownMaxMin {
		bad("cooldown_minutes %d is out of range", e.CooldownMinutes)
	}
	if e.MinLevel < 0 || e.MaxLevel < 0 || (e.MaxLevel > 0 && e.MinLevel > e.MaxLevel) {
		bad("min_level/max_level %d/%d is not a range", e.MinLevel, e.MaxLevel)
	}
	if len(e.Triggers) == 0 {
		bad("no triggers")
	}
	for i, t := range e.Triggers {
		switch t.Kind {
		case TriggerRoom:
			if t.Room <= 0 || !l.room(t.Room) {
				bad("trigger %d: room %d does not exist", i+1, t.Room)
			}
		case TriggerTag:
			if strings.TrimSpace(t.Tag) == "" {
				bad("trigger %d: tag is empty", i+1)
			}
		case TriggerArrival, TriggerCamp:
			if strings.TrimSpace(t.Zone) == "" {
				bad("trigger %d: %s needs a zone", i+1, t.Kind)
			}
		default:
			bad("trigger %d: unknown kind %q", i+1, t.Kind)
		}
		if t.Chance < 0 || t.Chance > 100 {
			bad("trigger %d: chance %d is not a percent", i+1, t.Chance)
		}
	}
	if _, ok := e.Pages[e.StartPage()]; !ok {
		bad("start page %q does not exist", e.StartPage())
	}
	ids := make([]string, 0, len(e.Pages))
	for id := range e.Pages {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		errs = append(errs, e.validatePage(id, e.Pages[id], l)...)
	}
	errs = append(errs, e.validateGraph()...)
	return errs
}

func (e Event) validatePage(id string, p Page, l Lookups) []string {
	var errs []string
	bad := func(format string, args ...any) {
		errs = append(errs, fmt.Sprintf("page %q: "+format, append([]any{id}, args...)...))
	}
	if strings.TrimSpace(p.Text) == "" || len(p.Text) > maxTextLen {
		bad("text must be 1 to %d characters", maxTextLen)
	}
	if p.Picture != "" && !pictureKeyExpr.MatchString(p.Picture) {
		bad("picture %q must be a key of lowercase letters, digits and dashes", p.Picture)
	}
	if len(p.Choices) == 0 || len(p.Choices) > MaxChoices {
		bad("needs 1 to %d choices", MaxChoices)
	}
	free := false
	for i, c := range p.Choices {
		n := i + 1
		if strings.TrimSpace(c.Label) == "" || len(c.Label) > maxLabelLen {
			bad("choice %d: label must be 1 to %d characters", n, maxLabelLen)
		}
		if c.Require.Free() && !spendsOrGambles(c) {
			free = true
		}
		for _, cls := range c.Require.Classes {
			if l.Class != nil && !l.Class(strings.ToLower(cls)) {
				bad("choice %d: unknown class %q", n, cls)
			}
		}
		if pers := c.Require.Personality; pers != "" && l.Personality != nil && !l.Personality(strings.ToLower(pers)) {
			bad("choice %d: unknown personality %q", n, pers)
		}
		if e.CooldownMinutes > 0 && (gains(c.Do) || gains(c.Fail)) {
			bad("choice %d: an event that returns cannot give gold, items or loyalty", n)
		}
		if c.Require.Skill != "" && l.Skill != nil && !l.Skill(strings.ToLower(c.Require.Skill)) {
			bad("choice %d: unknown skill %q", n, c.Require.Skill)
		}
		if a := c.Require.Alignment; a != "" && a != "good" && a != "neutral" && a != "evil" {
			bad("choice %d: alignment must be good, neutral or evil", n)
		}
		if c.Stance != "" && !opinions.IsStance(c.Stance) {
			bad("choice %d: stance %q must be one of %s", n, c.Stance, strings.Join(opinions.Stances(), ", "))
		}
		if c.Require.SkillLevel < 0 || c.Require.MinLevel < 0 || c.Require.Gold < 0 {
			bad("choice %d: requirement numbers cannot be negative", n)
		}
		if c.Require.Item > 0 && !l.item(c.Require.Item) {
			bad("choice %d: required item %d does not exist", n, c.Require.Item)
		}
		if c.Risk != nil {
			if c.Risk.Pct < 1 || c.Risk.Pct > MaxRiskPct || c.Risk.LessPerLevel < 0 {
				bad("choice %d: risk pct must be 1 to %d", n, MaxRiskPct)
			}
			if len(c.Fail) == 0 && c.FailText == "" {
				bad("choice %d: a risk needs a fail text or fail outcomes", n)
			}
		} else if len(c.Fail) > 0 || c.FailText != "" || c.FailNext != "" {
			bad("choice %d: fail outcomes need a risk", n)
		}
		for _, next := range []string{c.Next, c.FailNext} {
			if next != "" {
				if _, ok := e.Pages[next]; !ok {
					bad("choice %d: next page %q does not exist", n, next)
				}
			}
		}
		if hasBattle(c.Do) && c.Next != "" {
			bad("choice %d: a battle ends the event, so it cannot have a next page", n)
		}
		if hasBattle(c.Fail) && c.FailNext != "" {
			bad("choice %d: a battle on failure ends the event, so it cannot have a fail_next page", n)
		}
		for _, list := range [][]Outcome{c.Do, c.Fail} {
			for j, o := range list {
				for _, msg := range o.validate(l) {
					bad("choice %d outcome %d: %s", n, j+1, msg)
				}
			}
		}
		if len(c.Text) > maxTextLen || len(c.FailText) > maxTextLen {
			bad("choice %d: text is longer than %d characters", n, maxTextLen)
		}
	}
	if !free && len(p.Choices) > 0 {
		bad("needs a choice anyone can always take (no requirement, no cost, no risk)")
	}
	return errs
}

// gains reports whether outcomes give the company something it could farm
// from an event that returns: gold, items or loyalty.
func gains(list []Outcome) bool {
	for _, o := range list {
		switch o.Kind {
		case OutcomeItem:
			return true
		case OutcomeGold, OutcomeLoyalty:
			if o.Amount > 0 {
				return true
			}
		}
	}
	return false
}

func hasBattle(list []Outcome) bool {
	for _, o := range list {
		if o.Kind == OutcomeBattle {
			return true
		}
	}
	return false
}

// spendsOrGambles reports whether a choice with no requirement still costs
// something that could be missing (gold or an item) or can fail: a way out
// must always be there.
func spendsOrGambles(c Choice) bool {
	if c.Risk != nil {
		return true
	}
	for _, o := range c.Do {
		switch o.Kind {
		case OutcomeLoseItem:
			return true
		case OutcomeGold:
			if o.Amount < 0 {
				return true
			}
		}
	}
	return false
}

func (o Outcome) validate(l Lookups) []string {
	var errs []string
	bad := func(format string, args ...any) { errs = append(errs, fmt.Sprintf(format, args...)) }
	switch o.Who {
	case "", WhoActor, WhoLeader, WhoAll, WhoRandom:
	default:
		bad("who %q must be actor, leader, all or random", o.Who)
	}
	switch o.Kind {
	case OutcomeWound:
		if o.Pct < 1 || o.Pct > MaxWoundPct {
			bad("wound pct must be 1 to %d", MaxWoundPct)
		}
	case OutcomeAilment:
		if l.Ailment != nil && !l.Ailment(o.Ailment) {
			bad("unknown ailment %q", o.Ailment)
		}
		if o.Ailment == "" {
			bad("ailment is empty")
		}
	case OutcomeNeed:
		switch o.Stat {
		case "hunger", "thirst", "fatigue":
		default:
			bad("need stat must be hunger, thirst or fatigue")
		}
		if o.Amount < 1 || o.Amount > MaxNeed {
			bad("need amount must be 1 to %d", MaxNeed)
		}
	case OutcomeItem, OutcomeLoseItem:
		if o.Item <= 0 || !l.item(o.Item) {
			bad("item %d does not exist", o.Item)
		}
		if o.Count < 0 || o.Count > MaxItemCount {
			bad("count must be 1 to %d", MaxItemCount)
		}
	case OutcomeGold:
		if o.Amount == 0 || o.Amount > MaxGold || o.Amount < -MaxGold {
			bad("gold amount must be 1 to %d, gained or lost", MaxGold)
		}
	case OutcomeLoyalty:
		if o.Amount == 0 || o.Amount > MaxLoyalty || o.Amount < -MaxLoyalty {
			bad("loyalty amount must be 1 to %d, up or down", MaxLoyalty)
		}
	case OutcomeBattle:
		if len(o.Foes) < 2 || len(o.Foes) > MaxFoes {
			bad("a battle needs 2 to %d foes", MaxFoes)
		}
		for _, f := range o.Foes {
			if f.Mob <= 0 || !l.mob(f.Mob) {
				bad("foe mob %d does not exist", f.Mob)
			}
			if f.Level < 0 || f.Level > FoeLevelMax {
				bad("foe level %d is out of range", f.Level)
			}
		}
	case OutcomeMove:
		if o.Room <= 0 || !l.room(o.Room) {
			bad("move room %d does not exist", o.Room)
		}
	case OutcomeFlag:
		if !idPattern.MatchString(o.Flag) {
			bad("flag %q must be lowercase words joined by dashes", o.Flag)
		}
	default:
		bad("unknown kind %q", o.Kind)
	}
	return errs
}

// validateGraph checks every page is reachable from the start and that
// follow-on pages never loop (a loop of free choices could repeat an
// outcome without end).
func (e Event) validateGraph() []string {
	var errs []string
	if _, ok := e.Pages[e.StartPage()]; !ok {
		return nil
	}
	const (
		white = iota
		grey
		black
	)
	colour := map[string]int{}
	var cyclic []string
	var visit func(id string)
	visit = func(id string) {
		colour[id] = grey
		p := e.Pages[id]
		for _, c := range p.Choices {
			for _, next := range []string{c.Next, c.FailNext} {
				if next == "" {
					continue
				}
				if _, ok := e.Pages[next]; !ok {
					continue
				}
				switch colour[next] {
				case white:
					visit(next)
				case grey:
					cyclic = append(cyclic, id+" -> "+next)
				}
			}
		}
		colour[id] = black
	}
	visit(e.StartPage())
	sort.Strings(cyclic)
	for _, c := range cyclic {
		errs = append(errs, "pages loop: "+c)
	}
	ids := make([]string, 0, len(e.Pages))
	for id := range e.Pages {
		if colour[id] == white {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		errs = append(errs, fmt.Sprintf("page %q cannot be reached from %q", id, e.StartPage()))
	}
	return errs
}

// Matches reports whether a trigger of the given kind fires for a room and
// zone with these tags. Kind TriggerRoom matches by room id, TriggerTag by
// tag; TriggerArrival and TriggerCamp by zone.
func (t Trigger) Matches(kind string, roomID int, zone string, tags []string) bool {
	if t.Kind != kind {
		return false
	}
	switch kind {
	case TriggerRoom:
		return t.Room == roomID
	case TriggerTag:
		for _, tag := range tags {
			if strings.EqualFold(strings.TrimSpace(tag), t.Tag) {
				return true
			}
		}
		return false
	case TriggerArrival, TriggerCamp:
		return strings.EqualFold(t.Zone, zone)
	}
	return false
}
