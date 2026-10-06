package items

import (
	"fmt"
	"math"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/statmods"
)

// Phase 36a: rolled gear. A generated item keeps its roll on the instance
// (Item.Loot) and the resulting numbers in the instance's Spec override, so
// a later change to the shipped base type or to the affix data never
// silently rewrites an item a player owns. An item with no roll is a
// "legacy" common item of its template.

// RollVersion is the generator version saved with a roll. Bump it when the
// affix data or the quality table is rebalanced, so old items can be
// migrated explicitly instead of drifting.
const RollVersion = 1

// Quality is workmanship: it scales the base type's damage and protection
// and the item's value.
type Quality string

const (
	QualityCrude     Quality = "crude"
	QualityWorn      Quality = "worn"
	QualityStandard  Quality = "standard"
	QualityFine      Quality = "fine"
	QualitySuperior  Quality = "superior"
	QualityExquisite Quality = "exquisite"
)

// Qualities lists every quality, worst first.
func Qualities() []Quality {
	return []Quality{QualityCrude, QualityWorn, QualityStandard, QualityFine, QualitySuperior, QualityExquisite}
}

// qualityTable: base stat percent and value multiplier (per cent).
var qualityTable = map[Quality]struct{ statPct, valuePct int }{
	QualityCrude:     {-20, 40},
	QualityWorn:      {-10, 70},
	QualityStandard:  {0, 100},
	QualityFine:      {10, 160},
	QualitySuperior:  {20, 250},
	QualityExquisite: {30, 400},
}

// StatPct is the percent a quality adds to base damage and protection.
func (q Quality) StatPct() int { return qualityTable[q].statPct }

// ValuePct is the percent of its base value an item of the quality is worth.
func (q Quality) ValuePct() int {
	if e, ok := qualityTable[q]; ok {
		return e.valuePct
	}
	return 100
}

// Valid reports whether q is a known quality.
func (q Quality) Valid() bool { _, ok := qualityTable[q]; return ok }

// Rarity is the number and kind of special properties.
type Rarity string

const (
	RarityCommon    Rarity = "common"
	RarityUncommon  Rarity = "uncommon"
	RarityRare      Rarity = "rare"
	RarityEpic      Rarity = "epic"
	RarityLegendary Rarity = "legendary"
	RaritySet       Rarity = "set"
)

// Rarities lists every rarity, plainest first.
func Rarities() []Rarity {
	return []Rarity{RarityCommon, RarityUncommon, RarityRare, RarityEpic, RarityLegendary, RaritySet}
}

// Valid reports whether r is a known rarity.
func (r Rarity) Valid() bool {
	for _, k := range Rarities() {
		if k == r {
			return true
		}
	}
	return false
}

// Rank orders the rarities: common 0 through set 5.
func (r Rarity) Rank() int {
	for i, k := range Rarities() {
		if k == r {
			return i
		}
	}
	return 0
}

// DropsUnidentified is true for Rare and above (owner, 2026-10-05).
func (r Rarity) DropsUnidentified() bool { return r.Rank() >= RarityRare.Rank() }

// Colour is the ANSI alias that colours the rarity's name.
func (r Rarity) Colour() string { return "rarity-" + string(r) }

// Label is the rarity as a capitalised word.
func (r Rarity) Label() string {
	s := string(r)
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// RolledAffix is one affix an item rolled. Mechanic names the effect it has
// on the item's numbers, one of:
//
//	statmod:<stat>  adds Value to a statmods stat (strength, healthmax, damage, healing ...)
//	protection      adds Value to the item's damage reduction
//	parry           adds Value percent to the item's parry
//	warmth          adds Value to the item's insulation
//	weightpct       cuts the item's weight by Value percent
type RolledAffix struct {
	ID       string `yaml:"id"`
	Label    string `yaml:"label"`            // "Keen" or "of the Bear"
	Suffix   bool   `yaml:"suffix,omitempty"` // a suffix follows the base name, else a prefix precedes the quality
	Major    bool   `yaml:"major,omitempty"`
	Mechanic string `yaml:"mechanic"`
	Value    int    `yaml:"value"`
	Tier     int    `yaml:"tier,omitempty"`     // the affix tier its value came from, 1 is lowest
	MinValue int    `yaml:"minvalue,omitempty"` // the tier's range, shown at Scribe rank 4
	MaxValue int    `yaml:"maxvalue,omitempty"`
}

// Rolled is the saved result of generating an item.
type Rolled struct {
	Version    int           `yaml:"version"`
	Tier       int           `yaml:"tier,omitempty"` // material and power budget, from the base type
	ILvl       int           `yaml:"ilvl"`
	Quality    Quality       `yaml:"quality"`
	Rarity     Rarity        `yaml:"rarity"`
	Identified bool          `yaml:"identified"`
	LevelReq   int           `yaml:"levelreq,omitempty"`
	Name       string        `yaml:"name,omitempty"` // a Rare's generated name, "Gloomfang"
	BaseValue  int           `yaml:"basevalue,omitempty"`
	Source     string        `yaml:"source,omitempty"` // where it came from, shown at Scribe rank 4
	Affixes    []RolledAffix `yaml:"affixes,omitempty"`
}

// IsRolled is true for generated gear.
func (r Rolled) IsRolled() bool { return r.Version > 0 }

func (r Rolled) clone() Rolled {
	r.Affixes = append([]RolledAffix(nil), r.Affixes...)
	return r
}

// weight is a base weight with the roll's identified weight affixes applied,
// the same cut applyAffix makes to the Spec override.
func (r Rolled) weight(base int) int {
	if !r.IsRolled() || !r.Identified || base <= 0 {
		return base
	}
	for _, a := range r.Affixes {
		if a.Mechanic == "weightpct" {
			base = max(1, base*(100-a.Value)/100)
		}
	}
	return base
}

// IsRolled reports whether the item is generated gear with a saved roll.
func (i *Item) IsRolled() bool { return i.Loot.IsRolled() }

// RollRarity is the item's rarity; common for an item with no roll.
func (i *Item) RollRarity() Rarity {
	if !i.IsRolled() || i.Loot.Rarity == "" {
		return RarityCommon
	}
	return i.Loot.Rarity
}

// IsIdentified is true for every item without hidden affixes.
func (i *Item) IsIdentified() bool { return !i.IsRolled() || i.Loot.Identified }

// LevelRequirement is the level needed to wear the item; 0 for none.
func (i *Item) LevelRequirement() int {
	if !i.IsRolled() {
		return 0
	}
	return max(0, i.Loot.LevelReq)
}

// WearRefusal says why a wearer of the given level may not equip the item,
// or "" when they may.
func (i *Item) WearRefusal(level int) string {
	if req := i.LevelRequirement(); req > level {
		return fmt.Sprintf("The %s requires level %d to use (the wearer is level %d).", i.Name(), req, level)
	}
	return ""
}

// ApplyRoll sets the item's roll and rebuilds its Spec override from the
// base type: quality always, affixes only once identified.
func (i *Item) ApplyRoll(r Rolled) {
	i.Loot = r.clone()
	i.rebuildRolledSpec()
}

// Identify reveals an unidentified item's affixes and applies them to its
// numbers. It reports whether anything changed.
func (i *Item) Identify() bool {
	if !i.IsRolled() || i.Loot.Identified {
		return false
	}
	loot := i.Loot.clone()
	loot.Identified = true
	i.Loot = loot
	if i.Spec == nil {
		i.rebuildRolledSpec()
		return true
	}
	// Add the affixes to the item as it now is, so an enchantment, a worn
	// buff or a new name it has gained since the roll stay.
	spec := *i.Spec
	spec.StatMods = cloneStatMods(spec.StatMods)
	for _, a := range loot.Affixes {
		applyAffix(&spec, a)
		spec.Value += a.worth()
	}
	i.Spec = &spec
	return true
}

// rebuildRolledSpec derives the item's Spec override from the base type and
// the roll. Enchantments are replayed by the caller when they matter.
func (i *Item) rebuildRolledSpec() {
	base := GetItemSpec(i.ItemId)
	if base == nil {
		return
	}
	spec := RolledSpec(*base, i.Loot)
	i.Spec = &spec
}

// RolledSpec applies a roll to a base spec and returns a spec that shares
// no map or slice with it.
func RolledSpec(base ItemSpec, r Rolled) ItemSpec {
	spec := base
	spec.StatMods = cloneStatMods(base.StatMods)
	spec.WornBuffIds = append([]int(nil), base.WornBuffIds...)
	spec.BuffIds = append([]int(nil), base.BuffIds...)
	spec.Damage.CritBuffIds = append([]int(nil), base.Damage.CritBuffIds...)

	pct := r.Quality.StatPct()
	if pct != 0 {
		if spec.Damage.DiceCount > 0 && spec.Damage.SideCount > 0 {
			// BonusDamage is added to every hit, so scale the per-hit average.
			expected := float64(spec.Damage.DiceCount*(spec.Damage.SideCount+1))/2 + float64(spec.Damage.BonusDamage)
			spec.Damage.BonusDamage += scaled(expected, pct)
			spec.Damage.FormatDiceRoll()
		}
		if spec.DamageReduction > 0 {
			spec.DamageReduction = max(0, spec.DamageReduction+scaled(float64(spec.DamageReduction), pct))
		}
	}

	spec.Value = base.Value * r.Quality.ValuePct() / 100
	if r.BaseValue > 0 {
		spec.Value = r.BaseValue * r.Quality.ValuePct() / 100
	}
	if r.Identified {
		for _, a := range r.Affixes {
			applyAffix(&spec, a)
			spec.Value += a.worth()
		}
	}
	return spec
}

// scaled is value*pct/100 rounded away from zero, at least 1 in magnitude
// when pct is not zero, so a Fine dagger is still better than a Standard one.
func scaled(value float64, pct int) int {
	if pct == 0 {
		return 0
	}
	d := int(math.Round(value * float64(pct) / 100))
	if d == 0 {
		if pct > 0 {
			return 1
		}
		return -1
	}
	return d
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func cloneStatMods(in statmods.StatMods) statmods.StatMods {
	out := make(statmods.StatMods, len(in)+2)
	for k, v := range in {
		out[k] = v
	}
	return out
}

// worth is what an affix adds to an item's value, in the same terms as
// ItemSpec.AutoCalculateValue.
func (a RolledAffix) worth() int {
	switch {
	case strings.HasPrefix(a.Mechanic, "statmod:"):
		return a.Value * 11
	case a.Mechanic == "protection":
		return a.Value * 17
	case a.Mechanic == "parry":
		return a.Value * 8
	case a.Mechanic == "warmth":
		return a.Value * 5
	case a.Mechanic == "weightpct":
		return a.Value * 2
	}
	return 0
}

func applyAffix(spec *ItemSpec, a RolledAffix) {
	switch {
	case strings.HasPrefix(a.Mechanic, "statmod:"):
		spec.StatMods[strings.TrimPrefix(a.Mechanic, "statmod:")] += a.Value
	case a.Mechanic == "protection":
		spec.DamageReduction += a.Value
	case a.Mechanic == "parry":
		spec.Parry += a.Value
	case a.Mechanic == "warmth":
		// Added on top of the item's resolved warmth: Warmth 0 means the
		// slot default and negative means none, so it can't simply grow.
		spec.WarmthBonus += a.Value
	case a.Mechanic == "weightpct":
		if spec.Weight > 0 {
			spec.Weight = max(1, spec.Weight*(100-a.Value)/100)
		}
	}
}

// ValidMechanic reports whether a mechanic string is one ApplyRoll knows.
func ValidMechanic(m string) bool {
	switch {
	case strings.HasPrefix(m, "statmod:"):
		return len(m) > len("statmod:")
	case m == "protection", m == "parry", m == "warmth", m == "weightpct":
		return true
	}
	return false
}

// RollName composes the item's displayed name from its roll:
// "fine short sword" (Standard drops the quality word), "keen fine short
// sword of the Bear" for an identified Uncommon, and a Rare or better's
// generated name once identified: "Gloomfang, a fine short sword". A name
// set by Rename always wins (DisplayName handles that).
func (i *Item) RollName(base string) string {
	r := i.Loot
	words := []string{}
	if r.Quality != QualityStandard && r.Quality != "" {
		words = append(words, string(r.Quality))
	}
	words = append(words, base)
	plain := strings.Join(words, " ")

	if !r.Identified {
		return plain
	}
	if r.Rarity.Rank() >= RarityRare.Rank() && r.Name != "" {
		article := "a"
		if startsWithVowel(plain) {
			article = "an"
		}
		return r.Name + ", " + article + " " + plain
	}
	var pre, suf []string
	for _, a := range r.Affixes {
		if a.Suffix {
			suf = append(suf, a.Label)
		} else {
			pre = append(pre, strings.ToLower(a.Label))
		}
	}
	out := plain
	if len(pre) > 0 {
		out = strings.Join(pre, " ") + " " + out
		out = strings.ToUpper(out[:1]) + out[1:]
	}
	if len(suf) > 0 {
		out += " " + strings.Join(suf, " ")
	}
	return out
}

// RollLabel is the short tag shown after a rolled item's name: its rarity
// (when above common) and whether it is unidentified.
func (i *Item) RollLabel() string {
	if !i.IsRolled() {
		return ""
	}
	parts := []string{}
	if i.Loot.Rarity != RarityCommon && i.Loot.Rarity != "" {
		parts = append(parts, string(i.Loot.Rarity))
	}
	if !i.Loot.Identified {
		parts = append(parts, "unidentified")
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf(`<ansi fg="item-flags">(%s)</ansi>`, strings.Join(parts, ", "))
}

// RolledDescription lists a rolled item's layers for look and inspect:
// tier, item level, quality, rarity, level requirement, and (once
// identified) each affix. rank is the viewer's Scribe rank: at 4 each
// affix also shows its tier and range, and where the item came from.
func (i *Item) RolledDescription(rank int) string {
	if !i.IsRolled() {
		return ""
	}
	r := i.Loot
	var b strings.Builder
	fmt.Fprintf(&b, "Rarity: <ansi fg=%q>%s</ansi>   Quality: %s   Tier: %d   Item level: %d",
		r.Rarity.Colour(), r.Rarity.Label(), r.Quality, max(1, r.Tier), r.ILvl)
	if req := i.LevelRequirement(); req > 0 {
		fmt.Fprintf(&b, "\nRequires level %d to use.", req)
	}
	if !r.Identified {
		b.WriteString("\nIts properties are not yet known. A Scribe can read them (help identify).")
		return b.String()
	}
	for _, a := range r.Affixes {
		line := "  " + AffixText(a)
		if rank >= 4 && a.MaxValue > 0 {
			line += fmt.Sprintf(" (tier %d, %d to %d)", max(1, a.Tier), a.MinValue, a.MaxValue)
		}
		if a.Major {
			line += " [major]"
		}
		b.WriteString("\n" + line)
	}
	if rank >= 4 && r.Source != "" {
		fmt.Fprintf(&b, "\nFound from: %s.", r.Source)
	}
	return b.String()
}

// AffixText says what an affix does: "+3 strength", "-10% weight".
func AffixText(a RolledAffix) string {
	switch {
	case strings.HasPrefix(a.Mechanic, "statmod:"):
		stat := strings.TrimPrefix(a.Mechanic, "statmod:")
		switch stat {
		case "healthmax":
			stat = "maximum health"
		case "manamax":
			stat = "maximum mana"
		case "healing":
			return fmt.Sprintf("+%d%% healing", a.Value)
		case "damage":
			stat = "damage per hit"
		}
		return fmt.Sprintf("+%d %s", a.Value, stat)
	case a.Mechanic == "protection":
		return fmt.Sprintf("+%d protection", a.Value)
	case a.Mechanic == "parry":
		return fmt.Sprintf("+%d%% parry", a.Value)
	case a.Mechanic == "warmth":
		return fmt.Sprintf("+%d warmth", a.Value)
	case a.Mechanic == "weightpct":
		return fmt.Sprintf("-%d%% weight", a.Value)
	}
	return a.ID
}
