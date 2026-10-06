package items

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// Phase 36d: authored relics. A Legendary is a hand-made named item with a
// fixed signature effect; a Set piece belongs to a small set whose bonuses
// grow with the pieces worn. Both are ordinary item files with a `relic:`
// block, so a relic is its own base type (tier 5 or 6, outside the plain
// catalog the drop tables draw from) and its numbers live in data. Gear
// effects are class effects (classes.GearEffect) that the wearer's
// ClassEffects adds on top of its own, so a signature uses the same
// mechanics the combat code already reads.

// RelicSpec is the `relic:` block of an authored item.
type RelicSpec struct {
	Signature string         `yaml:"signature,omitempty"` // a Legendary's named effect; set pieces carry none (their set does)
	Effects   map[string]int `yaml:"effects,omitempty"`   // classes.GearEffect keys the signature grants while worn
	Lore      string         `yaml:"lore,omitempty"`
	Set       string         `yaml:"set,omitempty"` // the SetSpec this piece belongs to; empty for a Legendary
	ILvl      int            `yaml:"ilvl"`          // its fixed item level (the level requirement is five below)
	Mob       int            `yaml:"mob"`           // the boss that drops it
	Chance    int            `yaml:"chance"`        // percent a boss kill by one company drops it
}

// IsSet reports whether the relic is a set piece.
func (r RelicSpec) IsSet() bool { return r.Set != "" }

// Rarity is the rarity a relic generates at.
func (r RelicSpec) Rarity() Rarity {
	if r.IsSet() {
		return RaritySet
	}
	return RarityLegendary
}

// Validate checks a relic on its item.
func (r RelicSpec) validate(spec *ItemSpec) error {
	if spec.Type == Pack || (spec.Type != Weapon && spec.Subtype != Wearable) {
		return fmt.Errorf("a relic must be a weapon or wearable armor")
	}
	if spec.Tier < 4 {
		return fmt.Errorf("a relic is tier 4 to 6, not %d", spec.Tier)
	}
	if r.ILvl < 1 || r.ILvl > 70 {
		return fmt.Errorf("relic item level %d is outside 1 to 70", r.ILvl)
	}
	if r.Mob < 1 || r.Chance < 1 || r.Chance > 100 {
		return fmt.Errorf("a relic needs a boss (mob) and a drop chance of 1 to 100")
	}
	if err := classes.ValidateGearEffects(r.Effects); err != nil {
		return err
	}
	if r.IsSet() {
		if r.Signature != "" || len(r.Effects) > 0 {
			return fmt.Errorf("a set piece carries no signature of its own, its set gives the bonuses")
		}
		return nil
	}
	if strings.TrimSpace(r.Signature) == "" || len(r.Effects) == 0 {
		return fmt.Errorf("a legendary needs a signature name and at least one effect")
	}
	return nil
}

// SetBonus is what a set grants once Pieces of it are worn.
type SetBonus struct {
	Pieces  int            `yaml:"pieces"`
	Effects map[string]int `yaml:"effects"`
}

// SetSpec is an authored set: small (3 or 4 pieces), with a bonus at 2
// pieces worn and a stronger one at 3 or 4.
type SetSpec struct {
	SetId   string     `yaml:"setid"`
	Name    string     `yaml:"name"`
	Lore    string     `yaml:"lore,omitempty"`
	Bonuses []SetBonus `yaml:"bonuses"`
}

func (s *SetSpec) Id() string       { return s.SetId }
func (s *SetSpec) Filepath() string { return s.SetId + ".yaml" }

func (s *SetSpec) Validate() error {
	if s.SetId == "" || s.SetId != strings.ToLower(strings.TrimSpace(s.SetId)) || strings.ContainsAny(s.SetId, " /\\") {
		return fmt.Errorf("set id must be a nonempty lowercase word")
	}
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("set %q has no name", s.SetId)
	}
	if len(s.Bonuses) < 2 {
		return fmt.Errorf("set %q needs a bonus at 2 pieces and a stronger one at 3 or 4", s.SetId)
	}
	prev := 1
	for _, b := range s.Bonuses {
		if b.Pieces <= prev || b.Pieces > 4 {
			return fmt.Errorf("set %q bonuses must rise from 2 pieces to at most 4", s.SetId)
		}
		prev = b.Pieces
		if len(b.Effects) == 0 {
			return fmt.Errorf("set %q has an empty bonus at %d pieces", s.SetId, b.Pieces)
		}
		if err := classes.ValidateGearEffects(b.Effects); err != nil {
			return fmt.Errorf("set %q: %w", s.SetId, err)
		}
	}
	if s.Bonuses[0].Pieces != 2 {
		return fmt.Errorf("set %q first bonus must be at 2 pieces", s.SetId)
	}
	return nil
}

var (
	setsMu   sync.RWMutex
	itemSets = map[string]*SetSpec{}
)

// LoadSetDataFiles loads the world's sets (items must be loaded first).
func LoadSetDataFiles() {
	path := configs.GetFilePathsConfig().DataFiles.String() + "/sets"
	loaded := map[string]*SetSpec{}
	if _, err := os.Stat(path); err == nil {
		got, err := fileloader.LoadAllFlatFiles[string, *SetSpec](path)
		if err != nil {
			panic(err)
		}
		loaded = got
	}
	setsMu.Lock()
	itemSets = loaded
	setsMu.Unlock()
	mudlog.Info("items.LoadSetDataFiles()", "sets", len(loaded))
}

// GetSet returns a set by id.
func GetSet(id string) (*SetSpec, bool) {
	setsMu.RLock()
	defer setsMu.RUnlock()
	s, ok := itemSets[id]
	return s, ok
}

// SetPieces lists the item ids that belong to a set, ascending.
func SetPieces(id string) []int {
	var out []int
	for _, spec := range GetAllItemSpecs() {
		if spec.Relic != nil && spec.Relic.Set == id {
			out = append(out, spec.ItemId)
		}
	}
	sort.Ints(out)
	return out
}

// RelicSpecs lists every authored relic, ascending by item id.
func RelicSpecs() []ItemSpec {
	var out []ItemSpec
	for _, spec := range GetAllItemSpecs() {
		if spec.Relic != nil {
			out = append(out, spec)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ItemId < out[b].ItemId })
	return out
}

// SetTestSet registers a set for a test.
func SetTestSet(s *SetSpec) {
	setsMu.Lock()
	itemSets[s.SetId] = s
	setsMu.Unlock()
}

// RemoveTestSet removes a set registered with SetTestSet.
func RemoveTestSet(id string) {
	setsMu.Lock()
	delete(itemSets, id)
	setsMu.Unlock()
}

// ActiveSet is one set's progress on a wearer.
type ActiveSet struct {
	Set    *SetSpec
	Worn   int        // distinct pieces worn
	Total  int        // pieces the set has
	Active []SetBonus // the bonuses in force
}

// GearEffects is what a wearer's worn relics grant: every distinct
// Legendary's signature and every active set bonus, added together, and
// each set's progress. A relic worn twice counts once. Gear with no relic
// costs only a spec check.
func GearEffects(worn []Item) (map[string]int, []ActiveSet) {
	var fx map[string]int
	var counts map[string]map[int]bool
	seen := map[int]bool{}
	for _, it := range worn {
		if it.ItemId < 1 || seen[it.ItemId] {
			continue
		}
		spec := it.GetSpec()
		if spec.Relic == nil {
			continue
		}
		seen[it.ItemId] = true
		r := spec.Relic
		if r.IsSet() {
			if counts == nil {
				counts = map[string]map[int]bool{}
			}
			if counts[r.Set] == nil {
				counts[r.Set] = map[int]bool{}
			}
			counts[r.Set][it.ItemId] = true
			continue
		}
		for k, v := range r.Effects {
			if fx == nil {
				fx = map[string]int{}
			}
			fx[k] += v
		}
	}
	var active []ActiveSet
	ids := make([]string, 0, len(counts))
	for id := range counts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		set, ok := GetSet(id)
		if !ok {
			continue
		}
		a := ActiveSet{Set: set, Worn: len(counts[id]), Total: len(SetPieces(id))}
		for _, b := range set.Bonuses {
			if a.Worn >= b.Pieces {
				a.Active = append(a.Active, b)
				for k, v := range b.Effects {
					if fx == nil {
						fx = map[string]int{}
					}
					fx[k] += v
				}
			}
		}
		active = append(active, a)
	}
	return fx, active
}

// RelicLines says what a relic does, in plain text: a Legendary's signature
// and effects, or a set piece's set and each bonus. Empty for no relic.
func (i *Item) RelicLines() []string {
	spec := i.GetSpec()
	if spec.Relic == nil {
		return nil
	}
	r := spec.Relic
	if !r.IsSet() {
		return []string{fmt.Sprintf("%s (while worn): %s.", r.Signature, strings.Join(classes.DescribeGearEffects(r.Effects), "; "))}
	}
	set, ok := GetSet(r.Set)
	if !ok {
		return []string{"Piece of the " + r.Set + " set."}
	}
	lines := []string{fmt.Sprintf("Piece of the %s set (%d pieces).", set.Name, len(SetPieces(r.Set)))}
	for _, bonus := range set.Bonuses {
		lines = append(lines, fmt.Sprintf("  %d worn: %s.", bonus.Pieces, strings.Join(classes.DescribeGearEffects(bonus.Effects), "; ")))
	}
	return lines
}

// RelicDescription lists a relic's signature or set for look and inspect:
// its effect in words, and its lore.
func (i *Item) RelicDescription() string {
	spec := i.GetSpec()
	if spec.Relic == nil {
		return ""
	}
	lines := i.RelicLines()
	colour := RarityLegendary.Colour()
	if spec.Relic.IsSet() {
		colour = RaritySet.Colour()
	}
	if len(lines) > 0 {
		lines[0] = fmt.Sprintf(`<ansi fg="%s">%s</ansi>`, colour, lines[0])
	}
	if spec.Relic.Lore != "" {
		lines = append(lines, spec.Relic.Lore)
	}
	return strings.Join(lines, "\n")
}

// SetProgressText lists the sets a wearer has pieces of, with the bonuses
// in force: for the `equipment` command.
func SetProgressText(active []ActiveSet) []string {
	var out []string
	for _, a := range active {
		line := fmt.Sprintf("%s: %d of %d pieces", a.Set.Name, a.Worn, a.Total)
		for _, b := range a.Active {
			line += "\n    " + strings.Join(classes.DescribeGearEffects(b.Effects), "; ")
		}
		out = append(out, line)
	}
	return out
}

// IsRelicItem reports whether an item id is an authored relic.
func IsRelicItem(itemId int) bool {
	spec := GetItemSpec(itemId)
	return spec != nil && spec.Relic != nil
}
