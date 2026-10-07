package items

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/classes"
)

// Phase 71: trophy enchanting. A trophy is a creature part (a heart, a hide
// or ash) that falls to a hunted foe by its race. An enchanter works one
// into a weapon or piece of armor for a fee, and the item then grants the
// trophy's gear effects while it is worn, through the same class effects a
// relic's signature uses. The enchant lives on the item instance
// (Item.Trophy), so it follows the item into the pack, cargo or a
// companion's gear; it never changes the item's spec, so it never changes
// what a merchant will pay (the fee and the trophy are spent for nothing a
// shop values).

// Trophy parts.
const (
	TrophyHeart = "heart" // offence
	TrophyHide  = "hide"  // defence
	TrophyAsh   = "ash"   // spirit and craft
)

// EnchantFeePerTier is the gold an enchanter asks for each tier of the item
// worked on (a Common item is tier 1).
const EnchantFeePerTier = 25

// TrophySpec is the `trophy:` block of a trophy item.
type TrophySpec struct {
	Part    string         `yaml:"part"`    // heart, hide or ash
	Races   []string       `yaml:"races"`   // lowercase mob races that drop it
	Chance  int            `yaml:"chance"`  // percent an ordinary kill by one company drops it (an elite doubles it, a boss always drops one)
	Effects map[string]int `yaml:"effects"` // classes.GearEffect keys the enchanted item grants while worn
}

// maxTrophyEffect is the most one trophy may add to a gear effect: what all
// of a wearer's enchants together may give (Phase 71 review), so one
// trophy can fill a wearer's share of an effect and a second adds nothing.
// One-shot effects cannot be enchanted in.
func maxTrophyEffect(e classes.GearEffect) int {
	if e.Max <= 1 {
		return 0
	}
	return TrophyAggregateCap(e)
}

func (t *TrophySpec) validate(spec *ItemSpec) error {
	switch t.Part {
	case TrophyHeart, TrophyHide, TrophyAsh:
	default:
		return fmt.Errorf("part %q is not heart, hide or ash", t.Part)
	}
	if spec.Type != Commodity {
		return fmt.Errorf("a trophy is a commodity")
	}
	if len(t.Races) == 0 {
		return fmt.Errorf("a trophy names the races that drop it")
	}
	for _, r := range t.Races {
		if r == "" || r != strings.ToLower(r) {
			return fmt.Errorf("race %q must be lowercase", r)
		}
	}
	if t.Chance < 1 || t.Chance > 100 {
		return fmt.Errorf("drop chance %d is outside 1 to 100", t.Chance)
	}
	if len(t.Effects) == 0 {
		return fmt.Errorf("a trophy grants nothing")
	}
	if err := classes.ValidateGearEffects(t.Effects); err != nil {
		return err
	}
	for k, v := range t.Effects {
		e, _ := classes.GearEffectFor(k)
		if limit := maxTrophyEffect(e); v > limit {
			return fmt.Errorf("%q may add at most %d, not %d", k, limit, v)
		}
	}
	return nil
}

// DropsFrom reports whether a creature of the race may drop the trophy.
func (t *TrophySpec) DropsFrom(race string) bool {
	race = strings.ToLower(race)
	for _, r := range t.Races {
		if r == race {
			return true
		}
	}
	return false
}

// Enchantable reports whether an item may be enchanted: a weapon or worn
// armor or jewellery, never a pack, a quest item or a trophy.
func Enchantable(spec *ItemSpec) bool {
	if spec == nil || spec.Trophy != nil || spec.QuestToken != `` || spec.Type == Pack {
		return false
	}
	return spec.Type == Weapon || spec.Subtype == Wearable
}

// EnchantFee is the gold an enchanter asks to work a trophy into the item.
func (i *Item) EnchantFee() int {
	tier := max(1, i.GetSpec().Tier)
	if i.IsRolled() && i.Loot.Tier > 0 {
		tier = i.Loot.Tier
	}
	return tier * EnchantFeePerTier
}

// TrophySpecOf is the spec of the trophy worked into the item, or nil.
func (i *Item) TrophySpecOf() *ItemSpec {
	if i.Trophy < 1 {
		return nil
	}
	spec := GetItemSpec(i.Trophy)
	if spec == nil || spec.Trophy == nil {
		return nil
	}
	return spec
}

// IsTrophyEnchanted reports whether a trophy has been worked into the item.
func (i *Item) IsTrophyEnchanted() bool { return i.TrophySpecOf() != nil }

// TrophyEffects is what the item's enchant grants while it is worn, held so
// that a relic's own effects, awakenings and the enchant together stay
// within each effect's cap. Nil for an item with no enchant.
func (i *Item) TrophyEffects() map[string]int {
	spec := i.TrophySpecOf()
	if spec == nil {
		return nil
	}
	var own map[string]int
	if rs := i.GetSpec().Relic; rs != nil {
		own = map[string]int{}
		for k, v := range rs.Effects {
			own[k] += v
		}
		for k, v := range i.AwakenedEffects() {
			own[k] += v
		}
	}
	out := map[string]int{}
	for k, v := range spec.Trophy.Effects {
		e, ok := classes.GearEffectFor(k)
		if !ok {
			continue
		}
		if room := e.Max - own[k]; v > room {
			v = room
		}
		if v > 0 {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// TrophyLines says what the item's enchant does, or nil for none.
func (i *Item) TrophyLines() []string {
	spec := i.TrophySpecOf()
	if spec == nil {
		return nil
	}
	fx := i.TrophyEffects()
	if len(fx) == 0 {
		return []string{fmt.Sprintf("Enchanted with %s: it adds nothing more to this item.", spec.Name)}
	}
	return []string{fmt.Sprintf("Enchanted with %s (while worn): %s.", spec.Name, strings.Join(classes.DescribeGearEffects(fx), "; "))}
}

// Enchant works the trophy in. The caller has checked the item and the
// trophy and taken the fee; it refuses an item that already carries one.
func (i *Item) EnchantWithTrophy(trophyId int) error {
	spec := i.GetSpec()
	if !Enchantable(&spec) {
		return fmt.Errorf("that cannot be enchanted")
	}
	if spec := GetItemSpec(trophyId); spec == nil || spec.Trophy == nil {
		return fmt.Errorf("that is not a trophy")
	}
	if i.IsTrophyEnchanted() {
		return fmt.Errorf("already enchanted")
	}
	i.Trophy = trophyId
	return nil
}

// TrophyAggregateCap is the most all of a wearer's enchants together may add
// to one gear effect: a sixth of its cap, at least 1 (+2 Attack, +1 damage).
// Phase 71 review: at half the cap a fully enchanted company won every hard
// even fight (75% -> 100%, health lost 180 -> 15), worth two or three levels
// in the balance mirror; at a sixth a full set is worth about one level.
func TrophyAggregateCap(e classes.GearEffect) int { return max(1, e.Max/6) }

// TrophyGear sums the enchants of the worn items, held to the aggregate cap
// for each effect.
func TrophyGear(worn []Item) map[string]int {
	sum := map[string]int{}
	for idx := range worn {
		it := &worn[idx]
		if it.ItemId < 1 || it.Trophy < 1 {
			continue
		}
		for k, v := range it.TrophyEffects() {
			sum[k] += v
		}
	}
	for k, v := range sum {
		if e, ok := classes.GearEffectFor(k); ok {
			sum[k] = min(v, TrophyAggregateCap(e))
		}
	}
	return sum
}
