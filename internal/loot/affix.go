package loot

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

// Phase 36a affix data. Affixes live in _datafiles/world/<world>/lootaffixes
// (beside, not inside, the Phase 18a category tables, which walk loot/).
// Each affix maps onto a mechanic that already exists (items.ValidMechanic).

// Slot classes an affix can be restricted to.
const (
	SlotWeapon  = "weapon"  // any weapon
	SlotShield  = "shield"  // an offhand with protection
	SlotArmor   = "armor"   // head, body, belt, gloves, legs, feet
	SlotJewelry = "jewelry" // neck and ring
)

// AffixTier is one value range, available from an item level.
type AffixTier struct {
	MinILvl int `yaml:"minilvl"`
	Min     int `yaml:"min"`
	Max     int `yaml:"max"`
}

// Affix is one rollable property.
type Affix struct {
	ID       string      `yaml:"id"`
	Group    string      `yaml:"group"` // one affix per group per item
	Prefix   string      `yaml:"prefix,omitempty"`
	Suffix   string      `yaml:"suffix,omitempty"`
	Mechanic string      `yaml:"mechanic"`
	Slots    []string    `yaml:"slots"`
	Tiers    []AffixTier `yaml:"tiers"`
	Weight   int         `yaml:"weight"`
	Major    bool        `yaml:"major,omitempty"` // Epic's one major affix
}

// Names are the word lists a Rare or better's generated name draws from.
type Names struct {
	Prefixes []string `yaml:"prefixes"`
	Suffixes []string `yaml:"suffixes"`
}

// AffixFile is one affix data file.
type AffixFile struct {
	Affixes []Affix `yaml:"affixes,omitempty"`
	Names   *Names  `yaml:"names,omitempty"`
}

// AffixSet is the loaded, validated affix data.
type AffixSet struct {
	Affixes []Affix
	Names   Names
}

func validSlot(s string) bool {
	switch s {
	case SlotWeapon, SlotShield, SlotArmor, SlotJewelry:
		return true
	}
	return false
}

// Validate checks one affix.
func (a Affix) Validate() error {
	if strings.TrimSpace(a.ID) == "" || a.ID != strings.ToLower(a.ID) {
		return fmt.Errorf("affix id %q must be a nonempty lowercase name", a.ID)
	}
	if a.Group == "" {
		return fmt.Errorf("affix %q has no group", a.ID)
	}
	if a.Prefix == "" && a.Suffix == "" {
		return fmt.Errorf("affix %q has neither a prefix nor a suffix", a.ID)
	}
	if !items.ValidMechanic(a.Mechanic) {
		return fmt.Errorf("affix %q has unknown mechanic %q", a.ID, a.Mechanic)
	}
	if a.Weight <= 0 {
		return fmt.Errorf("affix %q needs a positive weight", a.ID)
	}
	if len(a.Slots) == 0 {
		return fmt.Errorf("affix %q lists no slots", a.ID)
	}
	for _, s := range a.Slots {
		if !validSlot(s) {
			return fmt.Errorf("affix %q has unknown slot %q", a.ID, s)
		}
	}
	if len(a.Tiers) == 0 {
		return fmt.Errorf("affix %q has no tiers", a.ID)
	}
	prev := 0
	for i, t := range a.Tiers {
		if t.MinILvl < 1 || t.Min < 1 || t.Max < t.Min {
			return fmt.Errorf("affix %q tier %d is invalid", a.ID, i+1)
		}
		if i > 0 && t.MinILvl <= prev {
			return fmt.Errorf("affix %q tiers must rise in minilvl", a.ID)
		}
		prev = t.MinILvl
	}
	return nil
}

// NewAffixSet validates and indexes affixes and names.
func NewAffixSet(affixes []Affix, names Names) (AffixSet, error) {
	seen := map[string]bool{}
	for _, a := range affixes {
		if err := a.Validate(); err != nil {
			return AffixSet{}, err
		}
		if seen[a.ID] {
			return AffixSet{}, fmt.Errorf("affix id %q is defined twice", a.ID)
		}
		seen[a.ID] = true
	}
	sorted := append([]Affix(nil), affixes...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	return AffixSet{Affixes: sorted, Names: names}, nil
}

// LoadAffixSet reads every .yaml file under dir. A missing directory is an
// empty set, so a world with no loot affixes still starts.
func LoadAffixSet(dir string) (AffixSet, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return AffixSet{}, nil
	} else if err != nil {
		return AffixSet{}, err
	}
	var affixes []Affix
	var names Names
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return AffixSet{}, err
		}
		var f AffixFile
		if err := yaml.UnmarshalStrict(raw, &f); err != nil {
			return AffixSet{}, fmt.Errorf("%s: %w", e.Name(), err)
		}
		affixes = append(affixes, f.Affixes...)
		if f.Names != nil {
			names.Prefixes = append(names.Prefixes, f.Names.Prefixes...)
			names.Suffixes = append(names.Suffixes, f.Names.Suffixes...)
		}
	}
	return NewAffixSet(affixes, names)
}

var (
	affixMu  sync.RWMutex
	affixSet AffixSet
)

// LoadAffixDataFiles loads the world's affix data at startup.
func LoadAffixDataFiles() {
	set, err := LoadAffixSet(configs.GetFilePathsConfig().DataFiles.String() + "/lootaffixes")
	if err != nil {
		panic(err)
	}
	SetAffixes(set)
	mudlog.Info("loot.LoadAffixDataFiles()", "affixes", len(set.Affixes))
}

// SetAffixes installs an affix set (startup, and tests).
func SetAffixes(set AffixSet) {
	affixMu.Lock()
	defer affixMu.Unlock()
	affixSet = set
}

// Affixes is the loaded affix set.
func Affixes() AffixSet {
	affixMu.RLock()
	defer affixMu.RUnlock()
	return affixSet
}
