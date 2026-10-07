// Package appearance is Ashveil Phase 72a's character looks: a short list of
// bands per trait (age, height, build, face, skin, eyes, hair, voice, marks)
// loaded from the world's looks.yaml, and the composer that turns a
// character's picks into the prose `look` shows. Looks are words only; they
// have no stat, combat or carry effect. The package knows nothing of
// characters or users: callers pass the picks, the pronoun forms and the race
// name.
package appearance

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// Trait ids the composer reads. A looks file must define every one.
const (
	TraitPronouns  = `pronouns`
	TraitAge       = `age`
	TraitHeight    = `height`
	TraitBuild     = `build`
	TraitFace      = `face`
	TraitSkin      = `skin`
	TraitEyes      = `eyes`
	TraitHairColor = `haircolor`
	TraitHairStyle = `hairstyle`
	TraitVoice     = `voice`
	TraitMark      = `mark`

	// KeyLine is the free line a player adds, KeyMark2 the second mark.
	KeyLine  = `line`
	KeyMark2 = `mark2`

	// NoMark is the mark option that ends the list.
	NoMark = `none`

	defaultLineMax = 160
	defaultMaxMark = 2
)

// SingleTraits are the traits every player answers once, in order. Marks
// and the free line follow them.
var SingleTraits = []string{
	TraitPronouns, TraitAge, TraitHeight, TraitBuild, TraitFace, TraitSkin,
	TraitEyes, TraitHairColor, TraitHairStyle, TraitVoice,
}

// Option is one band of a trait.
type Option struct {
	ID     string   `yaml:"id"`
	Name   string   `yaml:"name"`
	Adj    string   `yaml:"adj,omitempty"`
	After  string   `yaml:"after,omitempty"`
	Phrase string   `yaml:"phrase,omitempty"`
	Noun   string   `yaml:"noun,omitempty"`
	Color  string   `yaml:"color,omitempty"`
	Races  []string `yaml:"races,omitempty"`
}

// Trait is one question with its bands.
type Trait struct {
	ID      string   `yaml:"id"`
	Title   string   `yaml:"title"`
	Options []Option `yaml:"options"`
}

// Data is the whole looks file.
type Data struct {
	LineMax  int     `yaml:"linemax"`
	MaxMarks int     `yaml:"maxmarks"`
	Traits   []Trait `yaml:"traits"`
}

// Looks are a character's picks: trait id to option id, plus KeyMark2 and
// KeyLine.
type Looks map[string]string

// Pronouns are the forms the composer fills placeholders with.
type Pronouns struct {
	Subject    string
	Object     string
	Possessive string
}

var (
	mu        sync.RWMutex
	loaded    *Data
	suspended bool
)

var (
	idPattern          = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	colorPattern       = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	placeholderPattern = regexp.MustCompile(`\{[A-Za-z]+\}`)
)

// Load reads looks.yaml from the world data files. A missing file leaves
// the package empty (Available is false), so a world without it simply has
// no looks step.
func Load() error {
	path := string(configs.GetFilePathsConfig().DataFiles) + `/looks.yaml`
	if _, err := os.Stat(path); err != nil {
		mu.Lock()
		loaded = nil
		mu.Unlock()
		return nil
	}
	raw, err := util.ReadFile(path)
	if err != nil {
		return fmt.Errorf("appearance: %w", err)
	}
	return LoadBytes(raw)
}

// LoadBytes parses and validates a looks file and makes it current.
func LoadBytes(raw []byte) error {
	var d Data
	if err := yaml.Unmarshal(raw, &d); err != nil {
		return fmt.Errorf("appearance: %w", err)
	}
	if err := d.Validate(); err != nil {
		return err
	}
	SetData(&d)
	return nil
}

// SetData installs d as the current data (tests, and Load).
func SetData(d *Data) {
	mu.Lock()
	loaded = d
	mu.Unlock()
}

// Suspend makes the world look as if it had no looks data until the
// returned function runs, for tests of flows that run without the steps.
func Suspend() func() {
	mu.Lock()
	suspended = true
	mu.Unlock()
	return func() {
		mu.Lock()
		suspended = false
		mu.Unlock()
	}
}

// Current is the loaded data, or nil when the world has no looks.yaml. It
// loads the file on first use.
func Current() *Data {
	mu.RLock()
	d, off := loaded, suspended
	mu.RUnlock()
	if off {
		return nil
	}
	if d != nil {
		return d
	}
	if err := Load(); err != nil {
		return nil
	}
	mu.RLock()
	defer mu.RUnlock()
	return loaded
}

// Available reports whether the world defines looks.
func Available() bool { return Current() != nil }

// Validate rejects a malformed file rather than guessing at it.
func (d *Data) Validate() error {
	for _, id := range append(append([]string{}, SingleTraits...), TraitMark) {
		if d.Trait(id) == nil {
			return fmt.Errorf("appearance: missing trait %q", id)
		}
	}
	seenTrait := map[string]bool{}
	for _, t := range d.Traits {
		if !idPattern.MatchString(t.ID) {
			return fmt.Errorf("appearance: bad trait id %q", t.ID)
		}
		if seenTrait[t.ID] {
			return fmt.Errorf("appearance: duplicate trait %q", t.ID)
		}
		seenTrait[t.ID] = true
		if strings.TrimSpace(t.Title) == `` || len(t.Options) == 0 {
			return fmt.Errorf("appearance: trait %q needs a title and options", t.ID)
		}
		seen := map[string]bool{}
		for _, o := range t.Options {
			if !idPattern.MatchString(o.ID) || strings.TrimSpace(o.Name) == `` {
				return fmt.Errorf("appearance: trait %q has an option without a valid id and name", t.ID)
			}
			if seen[o.ID] {
				return fmt.Errorf("appearance: trait %q repeats option %q", t.ID, o.ID)
			}
			seen[o.ID] = true
			if o.Color != `` && !colorPattern.MatchString(o.Color) {
				return fmt.Errorf("appearance: %s/%s colour %q is not #rrggbb", t.ID, o.ID, o.Color)
			}
			for _, text := range []string{o.Adj, o.After, o.Phrase, o.Noun} {
				if bad := unknownPlaceholder(text); bad != `` {
					return fmt.Errorf("appearance: %s/%s uses unknown placeholder %s", t.ID, o.ID, bad)
				}
			}
			switch t.ID {
			case TraitPronouns:
				if o.Noun == `` {
					return fmt.Errorf("appearance: pronoun option %q needs a noun", o.ID)
				}
			case TraitSkin, TraitHairColor:
				if o.Color == `` {
					return fmt.Errorf("appearance: %s/%s needs a colour", t.ID, o.ID)
				}
			case TraitHairStyle, TraitVoice:
				if o.Phrase == `` {
					return fmt.Errorf("appearance: %s/%s needs a phrase", t.ID, o.ID)
				}
			}
		}
		if t.ID == TraitMark {
			if !seen[NoMark] {
				return fmt.Errorf("appearance: the mark trait needs a %q option", NoMark)
			}
			for _, o := range t.Options {
				if o.ID != NoMark && o.Phrase == `` {
					return fmt.Errorf("appearance: mark %q needs a phrase", o.ID)
				}
			}
		}
		if t.ID == TraitHairColor {
			for _, o := range t.Options {
				if o.Adj == `` {
					return fmt.Errorf("appearance: hair colour %q needs an adj (its colour word)", o.ID)
				}
			}
		}
	}
	return nil
}

var knownPlaceholders = map[string]bool{
	`{their}`: true, `{they}`: true, `{them}`: true,
	`{Their}`: true, `{They}`: true, `{Them}`: true, `{colour}`: true,
}

func unknownPlaceholder(text string) string {
	for _, m := range placeholderPattern.FindAllString(text, -1) {
		if !knownPlaceholders[m] {
			return m
		}
	}
	return ``
}

// Trait finds a trait by id.
func (d *Data) Trait(id string) *Trait {
	for i := range d.Traits {
		if d.Traits[i].ID == id {
			return &d.Traits[i]
		}
	}
	return nil
}

// Line is how long the free line may be.
func (d *Data) Line() int {
	if d.LineMax > 0 {
		return d.LineMax
	}
	return defaultLineMax
}

// Marks is how many marks a character may carry (1 or 2).
func (d *Data) Marks() int {
	if d.MaxMarks >= 1 && d.MaxMarks <= defaultMaxMark {
		return d.MaxMarks
	}
	return defaultMaxMark
}

// Options are a trait's bands open to a race (a lowercase race name; empty
// offers every band). Options naming no race are open to all.
func (d *Data) Options(traitID, race string) []Option {
	t := d.Trait(traitID)
	if t == nil {
		return nil
	}
	race = strings.ToLower(strings.TrimSpace(race))
	out := make([]Option, 0, len(t.Options))
	for _, o := range t.Options {
		if len(o.Races) > 0 {
			ok := false
			for _, r := range o.Races {
				if strings.EqualFold(r, race) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		out = append(out, o)
	}
	return out
}

// Option finds a band by id.
func (d *Data) Option(traitID, optionID string) (Option, bool) {
	if t := d.Trait(traitID); t != nil {
		for _, o := range t.Options {
			if o.ID == optionID {
				return o, true
			}
		}
	}
	return Option{}, false
}

// Match picks a band from a typed answer: its 1-based number, id or name.
func Match(options []Option, response string) (Option, bool) {
	response = strings.TrimSpace(response)
	if response == `` {
		return Option{}, false
	}
	if n, err := strconv.Atoi(response); err == nil {
		if n >= 1 && n <= len(options) {
			return options[n-1], true
		}
		return Option{}, false
	}
	for _, o := range options {
		if strings.EqualFold(o.ID, response) || strings.EqualFold(o.Name, response) {
			return o, true
		}
	}
	return Option{}, false
}

// Check reports the first problem with a set of picks for a race: a trait
// left out, a band that doesn't exist or isn't open to the race, or a free
// line that is too long.
func (d *Data) Check(l Looks, race string) error {
	for _, id := range SingleTraits {
		if err := d.checkPick(l, id, race, true); err != nil {
			return err
		}
	}
	if err := d.checkPick(l, TraitMark, race, true); err != nil {
		return err
	}
	if v := l[KeyMark2]; v != `` {
		if d.Marks() < 2 {
			return fmt.Errorf("appearance: only %d mark allowed", d.Marks())
		}
		if l[TraitMark] == NoMark {
			return fmt.Errorf("appearance: a second mark needs a first")
		}
		if err := d.checkPick(l, KeyMark2, race, false); err != nil {
			return err
		}
	}
	if len(l[KeyLine]) > d.Line() {
		return fmt.Errorf("appearance: the free line is over %d characters", d.Line())
	}
	return nil
}

func (d *Data) checkPick(l Looks, key, race string, required bool) error {
	traitID := key
	if key == KeyMark2 {
		traitID = TraitMark
	}
	v := l[key]
	if v == `` {
		if required {
			return fmt.Errorf("appearance: %s is not chosen", key)
		}
		return nil
	}
	for _, o := range d.Options(traitID, race) {
		if o.ID == v {
			return nil
		}
	}
	return fmt.Errorf("appearance: %q is not an option for %s", v, key)
}

// Colors are the skin and hair colours (#rrggbb) of a set of picks, for
// the sprites; either is "" when not chosen.
func (d *Data) Colors(l Looks) (skin, hair string) {
	return d.Color(TraitSkin, l), d.Color(TraitHairColor, l)
}

// Color is the colour hex of a skin or hair colour pick, or "".
func (d *Data) Color(traitID string, l Looks) string {
	o, _ := d.Option(traitID, l[traitID])
	return o.Color
}

// Compose writes the picks into prose: one sentence of body and one of
// each voice and mark, then the player's own line. A trait left out is
// skipped, so a partial pick set still reads cleanly (the live preview).
func (d *Data) Compose(l Looks, p Pronouns) string {
	pick := func(traitID string) Option {
		o, _ := d.Option(traitID, l[traitID])
		return o
	}
	fill := func(s string, colour string) string {
		if s == `` {
			return ``
		}
		s = strings.NewReplacer(
			`{colour}`, colour,
			`{their}`, p.Possessive, `{Their}`, upperFirst(p.Possessive),
			`{they}`, p.Subject, `{They}`, upperFirst(p.Subject),
			`{them}`, p.Object, `{Them}`, upperFirst(p.Object),
		).Replace(s)
		return strings.TrimSpace(s)
	}

	noun := fill(pick(TraitPronouns).Noun, ``)
	if noun == `` {
		noun = `person`
	}

	adjs := []string{}
	for _, id := range []string{TraitHeight, TraitBuild, TraitAge, TraitFace} {
		if a := fill(pick(id).Adj, ``); a != `` {
			adjs = append(adjs, a)
		}
	}
	sentence := noun
	if len(adjs) > 0 {
		sentence = strings.Join(adjs, `, `) + ` ` + noun
	}
	if after := fill(pick(TraitAge).After, ``); after != `` {
		sentence += ` ` + after
	}

	clauses := []string{}
	for _, id := range []string{TraitEyes, TraitSkin} {
		if c := fill(pick(id).Phrase, ``); c != `` {
			clauses = append(clauses, c)
		}
	}
	if style := pick(TraitHairStyle); style.ID != `` {
		colour := pick(TraitHairColor).Adj
		if c := fill(style.Phrase, colour); c != `` {
			clauses = append(clauses, c)
		}
	}
	if len(clauses) > 0 {
		sentence += `, ` + strings.Join(clauses, `, `)
	}

	article := `A`
	if startsWithVowel(sentence) {
		article = `An`
	}
	out := []string{article + ` ` + sentence + `.`}

	if v := fill(pick(TraitVoice).Phrase, ``); v != `` {
		out = append(out, v)
	}
	for _, key := range []string{TraitMark, KeyMark2} {
		if l[key] == `` || l[key] == NoMark {
			continue
		}
		o, ok := d.Option(TraitMark, l[key])
		if !ok {
			continue
		}
		if s := fill(o.Phrase, ``); s != `` {
			out = append(out, s)
		}
	}
	if line := strings.TrimSpace(l[KeyLine]); line != `` {
		out = append(out, line)
	}
	return strings.Join(out, ` `)
}

func upperFirst(s string) string {
	if s == `` {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func startsWithVowel(s string) bool {
	if s == `` {
		return false
	}
	return strings.ContainsRune(`aeiouAEIOU`, rune(s[0]))
}

// CleanLine prepares a player's free line: control characters and markup
// brackets are dropped, whitespace is collapsed, and the length is capped.
// It rejects a line any word of which matches a banned name pattern, the
// same filter names go through. banned reports a match and may be nil.
func CleanLine(line string, max int, banned func(string) (string, bool)) (string, error) {
	var b strings.Builder
	for _, r := range line {
		switch {
		case r == '<' || r == '>' || r == '{' || r == '}':
			continue
		case r < 0x20 || r == 0x7f:
			b.WriteRune(' ')
		default:
			b.WriteRune(r)
		}
	}
	cleaned := strings.Join(strings.Fields(b.String()), ` `)
	if max > 0 && len([]rune(cleaned)) > max {
		return ``, fmt.Errorf("keep it to %d characters or fewer", max)
	}
	if banned != nil && cleaned != `` {
		if pattern, ok := banned(cleaned); ok {
			return ``, fmt.Errorf(`that matched the prohibited pattern "%s"`, pattern)
		}
		for _, w := range strings.Fields(cleaned) {
			if pattern, ok := banned(w); ok {
				return ``, fmt.Errorf(`that matched the prohibited pattern "%s"`, pattern)
			}
		}
	}
	return cleaned, nil
}
