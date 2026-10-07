// Package blessings is Ashveil Phase 77's account blessings: small perks a
// player's later characters start with, unlocked by milestones in an
// earlier character's chronicle (internal/chronicle). A blessing is data
// (blessings.yaml in the world files): a deed kind, how many, and a perk.
//
// A perk is a starting item or a recruit discount. Blessings never add
// combat power beyond a starting item, and the discount is capped
// (MaxDiscount). The package is GoMud-engine-free apart from characters and
// items: the module (modules/blessings) owns the saved account list, and
// callers read through the package-level seam (EarnedFor).
package blessings

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// MaxDiscount is the most a character's blessings take off a recruit's
// price, in percent.
const MaxDiscount = 10

// MaxItemsEach is the most of one item a single blessing may give.
const MaxItemsEach = 3

// Perk is what a blessing gives a new character.
type Perk struct {
	Item     int `yaml:"item,omitempty"`     // a starting item id
	Qty      int `yaml:"qty,omitempty"`      // how many (1 when 0)
	Discount int `yaml:"discount,omitempty"` // percent off a recruit's price
}

// Blessing is one milestone and its perk.
type Blessing struct {
	ID    string `yaml:"id"`
	Name  string `yaml:"name"`
	Text  string `yaml:"text"`
	Deed  string `yaml:"deed"`           // a chronicle kind word, such as boss
	Count int    `yaml:"count"`          // how many of that deed (lifetime)
	Iron  bool   `yaml:"iron,omitempty"` // only an Iron character's deeds count
	Perk  Perk   `yaml:"perk"`
}

// Data is the whole blessings file.
type Data struct {
	Blessings []Blessing `yaml:"blessings"`
}

var idPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Validate rejects a file whose blessings could not be read or applied.
func (d *Data) Validate() error {
	seen := map[string]bool{}
	for _, b := range d.Blessings {
		if !idPattern.MatchString(b.ID) {
			return fmt.Errorf("blessings: bad id %q", b.ID)
		}
		if seen[b.ID] {
			return fmt.Errorf("blessings: duplicate id %q", b.ID)
		}
		seen[b.ID] = true
		if strings.TrimSpace(b.Name) == `` || strings.TrimSpace(b.Text) == `` {
			return fmt.Errorf("blessings: %s needs a name and text", b.ID)
		}
		if k := chronicle.Kind(b.Deed); !k.Valid() {
			return fmt.Errorf("blessings: %s names an unknown deed %q", b.ID, b.Deed)
		}
		if b.Count < 1 {
			return fmt.Errorf("blessings: %s needs a count of at least 1", b.ID)
		}
		if b.Perk.Item == 0 && b.Perk.Discount == 0 {
			return fmt.Errorf("blessings: %s gives nothing", b.ID)
		}
		if b.Perk.Item != 0 && (b.Perk.Qty < 0 || b.Perk.Qty > MaxItemsEach) {
			return fmt.Errorf("blessings: %s gives %d of an item (at most %d)", b.ID, b.Perk.Qty, MaxItemsEach)
		}
		if b.Perk.Discount < 0 || b.Perk.Discount > MaxDiscount {
			return fmt.Errorf("blessings: %s discounts %d%% (at most %d)", b.ID, b.Perk.Discount, MaxDiscount)
		}
	}
	return nil
}

var (
	mu     sync.RWMutex
	loaded *Data
)

// Load reads blessings.yaml from the world data files. A missing file
// leaves the package empty (no blessings are earned).
func Load() error {
	path := string(configs.GetFilePathsConfig().DataFiles) + `/blessings.yaml`
	if _, err := os.Stat(path); err != nil {
		SetData(nil)
		return nil
	}
	raw, err := util.ReadFile(path)
	if err != nil {
		return fmt.Errorf("blessings: %w", err)
	}
	return LoadBytes(raw)
}

// LoadBytes parses and validates a blessings file and makes it current.
func LoadBytes(raw []byte) error {
	var d Data
	if err := yaml.Unmarshal(raw, &d); err != nil {
		return fmt.Errorf("blessings: %w", err)
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

// Current is the loaded data, or nil when the world has none.
func Current() *Data {
	mu.RLock()
	defer mu.RUnlock()
	return loaded
}

// Get is the blessing with the id.
func Get(id string) (Blessing, bool) {
	d := Current()
	if d == nil {
		return Blessing{}, false
	}
	for _, b := range d.Blessings {
		if b.ID == id {
			return b, true
		}
	}
	return Blessing{}, false
}

// Earned lists the blessings a character's chronicle has earned: total is
// the lifetime count of a deed kind, iron whether the character is Iron.
func Earned(total func(chronicle.Kind) int, iron bool) []Blessing {
	d := Current()
	if d == nil {
		return nil
	}
	var out []Blessing
	for _, b := range d.Blessings {
		if b.Iron && !iron {
			continue
		}
		if total(chronicle.Kind(b.Deed)) >= b.Count {
			out = append(out, b)
		}
	}
	return out
}

// Condition is a blessing's milestone in words ("slay 3 bosses").
func (b Blessing) Condition() string {
	what := deedPhrase(chronicle.Kind(b.Deed), b.Count)
	if b.Iron {
		return what + ` as an Iron character`
	}
	return what
}

// deedPhrase is a milestone's deed in words: "slay 3 bosses".
func deedPhrase(k chronicle.Kind, n int) string {
	times := func(one, many string) string {
		if n == 1 {
			return fmt.Sprintf("%s once", one)
		}
		return fmt.Sprintf("%s %d times", many, n)
	}
	count := func(verb, one, many string) string {
		if n == 1 {
			return fmt.Sprintf("%s a %s", verb, one)
		}
		return fmt.Sprintf("%s %d %s", verb, n, many)
	}
	switch k {
	case chronicle.Boss:
		return count("slay", "boss", "bosses")
	case chronicle.Relic:
		return count("find", "relic", "relics")
	case chronicle.Joined:
		return count("take on", "companion", "companions")
	case chronicle.Spared:
		return times("show mercy", "show mercy")
	case chronicle.Story:
		return count("see through", "story scene", "story scenes")
	case chronicle.Promoted:
		return count("take", "class promotion", "class promotions")
	case chronicle.Awakened:
		return count("wake", "relic's power", "relic powers")
	case chronicle.Errand:
		return count("see", "errand through", "errands through")
	case chronicle.Rites:
		return count("hold", "rite", "rites")
	}
	for _, info := range chronicle.Kinds {
		if info.Kind == k {
			return fmt.Sprintf("record %d %s deeds", n, strings.ToLower(info.Label))
		}
	}
	return fmt.Sprintf("record %d deeds", n)
}

// PerkText is what the blessing gives, in words.
func (b Blessing) PerkText() string {
	var parts []string
	if b.Perk.Item != 0 {
		name := fmt.Sprintf(`item %d`, b.Perk.Item)
		if spec := items.GetItemSpec(b.Perk.Item); spec != nil {
			name = spec.Name
		}
		qty := max(b.Perk.Qty, 1)
		if qty > 1 {
			name = fmt.Sprintf(`%d x %s`, qty, name)
		}
		parts = append(parts, `start with `+name)
	}
	if b.Perk.Discount > 0 {
		parts = append(parts, fmt.Sprintf(`recruits cost %d%% less`, b.Perk.Discount))
	}
	return strings.Join(parts, `; `)
}

// Provider is the module's saved list of the blessings an account has
// earned.
type Provider interface {
	// Earned is the ids of the blessings the account has earned, in the
	// order they were earned.
	Earned(userID int) []string
}

var (
	provMu   sync.RWMutex
	provider Provider
)

// SetProvider installs the provider (the module, at init; tests).
func SetProvider(p Provider) {
	provMu.Lock()
	provider = p
	provMu.Unlock()
}

// EarnedFor is the account's earned blessing ids ("" provider: none).
func EarnedFor(userID int) []string {
	provMu.RLock()
	p := provider
	provMu.RUnlock()
	if p == nil {
		return nil
	}
	return p.Earned(userID)
}

// Watcher is a provider that wants to hear when a character is given its
// blessings (the module refreshes the web client's panel).
type Watcher interface {
	Given(userID int)
}

// NotifyGiven tells the provider, when it watches, that the user's new
// character was given blessings.
func NotifyGiven(userID int) {
	provMu.RLock()
	p := provider
	provMu.RUnlock()
	if w, ok := p.(Watcher); ok {
		w.Given(userID)
	}
}

// Given is what Apply did.
type Given struct {
	Blessing Blessing
	Item     string // the starting item's name, when one was given
}

// Apply gives a new character the blessings: each one recorded on the
// character (so its perks, such as a recruit discount, are read from the
// character itself), and starting items put in the pack. A blessing the
// character already has is skipped, so a repeat gives nothing twice. Ids
// the world no longer defines are skipped.
func Apply(c *characters.Character, ids []string) []Given {
	var out []Given
	for _, id := range ids {
		b, ok := Get(id)
		if !ok || c.HasBlessing(id) {
			continue
		}
		g := Given{Blessing: b}
		if b.Perk.Item != 0 {
			for i := 0; i < max(b.Perk.Qty, 1); i++ {
				itm := items.New(b.Perk.Item)
				if itm.ItemId == 0 {
					break
				}
				if !c.StoreItem(itm) {
					break
				}
				g.Item = itm.Name()
			}
		}
		c.Blessings = append(c.Blessings, id)
		out = append(out, g)
	}
	return out
}

// DiscountPercent is the share a character's blessings take off a
// recruit's price, never more than MaxDiscount.
func DiscountPercent(c *characters.Character) int {
	if c == nil || len(c.Blessings) == 0 {
		return 0
	}
	total := 0
	for _, id := range c.Blessings {
		if b, ok := Get(id); ok {
			total += b.Perk.Discount
		}
	}
	return min(total, MaxDiscount)
}

// RecruitPrice is a recruit's price after the character's blessings.
// A price of 0 stays 0, and a discounted price is never below 1.
func RecruitPrice(c *characters.Character, price int) int {
	pct := DiscountPercent(c)
	if price <= 0 || pct == 0 {
		return price
	}
	return max(price*(100-pct)/100, 1)
}
