// Package lifestory is Ashveil Phase 72a's life story: three stages a
// player answers at creation (homeland, upbringing, trade), each with a
// short tale and a small effect, loaded from the world's lifestory.yaml.
// The trade is the character's background. The package knows nothing of
// characters: callers pass the picks and apply the effects.
//
// Picks are a flat map: a stage id maps to its option id, and "<stage>-stat"
// to the stat that option's +1 went to. Stat bonuses are derived from the
// picks every time they are read, never stored as a counter, so they cannot
// apply twice.
package lifestory

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// The three stages, in the order a player answers them.
const (
	StageHomeland   = `homeland`
	StageUpbringing = `upbringing`
	StageTrade      = `trade`

	// MaxStatBonus is the most the life story may add to one stat.
	MaxStatBonus = 2
)

// Stages are the stage ids in order.
var Stages = []string{StageHomeland, StageUpbringing, StageTrade}

// StatNames are the stats an option may name.
var StatNames = []string{`strength`, `speed`, `smarts`, `vitality`, `mysticism`, `perception`}

// Option is one answer to a stage.
type Option struct {
	ID       string   `yaml:"id"`
	Name     string   `yaml:"name"`
	Text     string   `yaml:"text"`
	Stats    []string `yaml:"stats"`
	Keepsake int      `yaml:"keepsake,omitempty"`
	Skill    string   `yaml:"skill,omitempty"`
}

// Stage is one question with its answers.
type Stage struct {
	ID      string   `yaml:"id"`
	Title   string   `yaml:"title"`
	Options []Option `yaml:"options"`
}

// Data is the whole life story file.
type Data struct {
	Stages []Stage `yaml:"stages"`
}

// Pronouns are the forms the backstory text is filled with.
type Pronouns struct {
	Subject    string
	Object     string
	Possessive string
}

// Picks are a character's answers.
type Picks map[string]string

var (
	mu     sync.RWMutex
	loaded *Data
)

var idPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Load reads lifestory.yaml from the world data files. A missing file
// leaves the package empty (Available is false).
func Load() error {
	path := string(configs.GetFilePathsConfig().DataFiles) + `/lifestory.yaml`
	if _, err := os.Stat(path); err != nil {
		mu.Lock()
		loaded = nil
		mu.Unlock()
		return nil
	}
	raw, err := util.ReadFile(path)
	if err != nil {
		return fmt.Errorf("lifestory: %w", err)
	}
	return LoadBytes(raw)
}

// LoadBytes parses and validates a life story file and makes it current.
func LoadBytes(raw []byte) error {
	var d Data
	if err := yaml.Unmarshal(raw, &d); err != nil {
		return fmt.Errorf("lifestory: %w", err)
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

// Current is the loaded data, or nil when the world has no lifestory.yaml.
// It loads the file on first use.
func Current() *Data {
	mu.RLock()
	d := loaded
	mu.RUnlock()
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

// Available reports whether the world defines a life story.
func Available() bool { return Current() != nil }

// Validate rejects a malformed file rather than guessing at it.
func (d *Data) Validate() error {
	for _, id := range Stages {
		if d.Stage(id) == nil {
			return fmt.Errorf("lifestory: missing stage %q", id)
		}
	}
	for _, s := range d.Stages {
		if strings.TrimSpace(s.Title) == `` || len(s.Options) == 0 {
			return fmt.Errorf("lifestory: stage %q needs a title and options", s.ID)
		}
		seen := map[string]bool{}
		for _, o := range s.Options {
			if !idPattern.MatchString(o.ID) || strings.TrimSpace(o.Name) == `` {
				return fmt.Errorf("lifestory: stage %q has an option without a valid id and name", s.ID)
			}
			if seen[o.ID] {
				return fmt.Errorf("lifestory: stage %q repeats option %q", s.ID, o.ID)
			}
			seen[o.ID] = true
			if strings.TrimSpace(o.Text) == `` {
				return fmt.Errorf("lifestory: %s/%s has no text", s.ID, o.ID)
			}
			if len(o.Stats) < 1 || len(o.Stats) > 2 {
				return fmt.Errorf("lifestory: %s/%s must name one or two stats", s.ID, o.ID)
			}
			for _, st := range o.Stats {
				if !isStat(st) {
					return fmt.Errorf("lifestory: %s/%s names unknown stat %q", s.ID, o.ID, st)
				}
			}
			if s.ID != StageTrade && (o.Keepsake != 0 || o.Skill != ``) {
				return fmt.Errorf("lifestory: %s/%s: only a trade gives a keepsake or skill", s.ID, o.ID)
			}
			if o.Keepsake < 0 {
				return fmt.Errorf("lifestory: %s/%s has a bad keepsake id", s.ID, o.ID)
			}
			if bad := badPlaceholder(o.Text); bad != `` {
				return fmt.Errorf("lifestory: %s/%s uses unknown placeholder %s", s.ID, o.ID, bad)
			}
		}
	}
	return nil
}

var placeholderPattern = regexp.MustCompile(`\{[A-Za-z]+\}`)

var knownPlaceholders = map[string]bool{
	`{name}`: true, `{their}`: true, `{they}`: true, `{them}`: true,
	`{Their}`: true, `{They}`: true, `{Them}`: true,
}

func badPlaceholder(text string) string {
	for _, m := range placeholderPattern.FindAllString(text, -1) {
		if !knownPlaceholders[m] {
			return m
		}
	}
	return ``
}

func isStat(s string) bool {
	for _, n := range StatNames {
		if n == s {
			return true
		}
	}
	return false
}

// Stage finds a stage by id.
func (d *Data) Stage(id string) *Stage {
	for i := range d.Stages {
		if d.Stages[i].ID == id {
			return &d.Stages[i]
		}
	}
	return nil
}

// Option finds an answer by stage and option id.
func (d *Data) Option(stageID, optionID string) (Option, bool) {
	if s := d.Stage(stageID); s != nil {
		for _, o := range s.Options {
			if o.ID == optionID {
				return o, true
			}
		}
	}
	return Option{}, false
}

// StatKey is the picks key holding the stat a stage's +1 went to.
func StatKey(stageID string) string { return stageID + `-stat` }

// Offered are the stats of an option a +1 may still go to: those not
// already at MaxStatBonus from the picks so far.
func Offered(o Option, picks Picks) []string {
	have := StatBonus(picks)
	out := make([]string, 0, len(o.Stats))
	for _, st := range o.Stats {
		if have[st] < MaxStatBonus {
			out = append(out, st)
		}
	}
	return out
}

// StatBonus is what the picks add to each stat: +1 per stage to the stat
// that stage's pick went to, in stage order, never above MaxStatBonus. It
// needs no data: the picks hold the chosen stat.
func StatBonus(picks Picks) map[string]int {
	out := map[string]int{}
	for _, id := range Stages {
		st := picks[StatKey(id)]
		if st == `` || !isStat(st) || out[st] >= MaxStatBonus {
			continue
		}
		out[st]++
	}
	return out
}

// Check reports the first problem with a full set of picks: a stage left
// out, an answer that doesn't exist, or a stat the answer doesn't name.
func (d *Data) Check(picks Picks) error {
	for _, id := range Stages {
		o, ok := d.Option(id, picks[id])
		if !ok {
			return fmt.Errorf("lifestory: %s is not chosen", id)
		}
		st := picks[StatKey(id)]
		if st == `` {
			// Allowed only when every stat the answer names was already full.
			prior := Picks{}
			for _, earlier := range Stages {
				if earlier == id {
					break
				}
				prior[StatKey(earlier)] = picks[StatKey(earlier)]
			}
			if len(Offered(o, prior)) != 0 {
				return fmt.Errorf("lifestory: %s has no stat chosen", id)
			}
			continue
		}
		named := false
		for _, s := range o.Stats {
			if s == st {
				named = true
			}
		}
		if !named {
			return fmt.Errorf("lifestory: %s does not give %s", o.ID, st)
		}
	}
	return nil
}

// Backstory writes the three tales into one short passage for a named
// character.
func (d *Data) Backstory(picks Picks, name string, p Pronouns) string {
	parts := []string{}
	for _, id := range Stages {
		o, ok := d.Option(id, picks[id])
		if !ok {
			continue
		}
		parts = append(parts, fillText(o.Text, name, p))
	}
	return strings.Join(parts, ` `)
}

func fillText(text, name string, p Pronouns) string {
	return strings.NewReplacer(
		`{name}`, name,
		`{their}`, p.Possessive, `{Their}`, upperFirst(p.Possessive),
		`{they}`, p.Subject, `{They}`, upperFirst(p.Subject),
		`{them}`, p.Object, `{Them}`, upperFirst(p.Object),
	).Replace(text)
}

func upperFirst(s string) string {
	if s == `` {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// Trade is the trade option a set of picks holds: the character's
// background. ok is false without data or a pick.
func (d *Data) Trade(picks Picks) (Option, bool) {
	return d.Option(StageTrade, picks[StageTrade])
}

// Background is the background id (the trade option id) of a set of picks,
// or "" when there is none. Phase 72 reads it; it needs no loaded data.
func Background(picks Picks) string { return picks[StageTrade] }

// PicksWithBackground is a test helper for later phases: a picks set whose
// only content is the background id.
func PicksWithBackground(id string) Picks { return Picks{StageTrade: id} }
