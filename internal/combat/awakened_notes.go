package combat

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// awakenedNotes names the awakened relic powers that raised a landed blow
// (Phase 67), for the strike's explained line (`why`). A power is named only
// when its condition held for this blow: a bonus on every blow always, a
// wounded-foe bonus when the foe is at or below half health, a smite on the
// dead and the damned, a shooting bonus with a shooting weapon. Other
// awakened powers (armor, evasion, health, auras) act on the numbers they
// change and are read on the item.
func awakenedNotes(src, tgt *characters.Character) []string {
	var notes []string
	notes = append(notes, trophyNotes(src, tgt)...)
	for _, slot := range characters.AllSlots() {
		if slot == items.Pack {
			continue
		}
		itm := src.Equipment.Get(slot)
		if itm == nil || itm.ItemId < 1 {
			continue
		}
		for _, a := range itm.AwakenedPowers() {
			var acting []string
			for _, key := range classes.GearEffectKeys() {
				v := a.Effects[key]
				if v < 1 {
					continue
				}
				switch key {
				case classes.Damage:
				case classes.Wounded:
					if tgt.HealthMax.Value <= 0 || tgt.Health*100/tgt.HealthMax.Value > 50 {
						continue
					}
				case classes.Smite:
					if !unholyRaces[strings.ToLower(tgt.Race())] {
						continue
					}
				case classes.RangedPct:
					if !src.Shooting() {
						continue
					}
				default:
					continue
				}
				acting = append(acting, classes.DescribeGearEffects(map[string]int{key: v})...)
			}
			if len(acting) > 0 {
				notes = append(notes, fmt.Sprintf("%s woke %s: %s", itm.Name(), a.Name, strings.Join(acting, "; ")))
			}
		}
	}
	return notes
}

// trophyNotes names the trophy enchants (Phase 71) that raised a landed
// blow, by the same conditions as awakened powers. The figure is the
// wearer's whole enchant total for the effect (held to its cap), and the
// trophies named are the ones worn that give it.
func trophyNotes(src, tgt *characters.Character) []string {
	var worn []items.Item
	for _, slot := range characters.AllSlots() {
		if slot != items.Pack {
			worn = append(worn, *src.Equipment.Get(slot))
		}
	}
	total := items.TrophyGear(worn)
	if len(total) == 0 {
		return nil
	}
	var acting []string
	var names []string
	seen := map[string]bool{}
	for _, key := range classes.GearEffectKeys() {
		v := total[key]
		if v < 1 {
			continue
		}
		switch key {
		case classes.Damage:
		case classes.Wounded:
			if tgt.HealthMax.Value <= 0 || tgt.Health*100/tgt.HealthMax.Value > 50 {
				continue
			}
		case classes.Smite:
			if !unholyRaces[strings.ToLower(tgt.Race())] {
				continue
			}
		case classes.RangedPct:
			if !src.Shooting() {
				continue
			}
		default:
			continue
		}
		acting = append(acting, classes.DescribeGearEffects(map[string]int{key: v})...)
		for idx := range worn {
			if _, gives := worn[idx].TrophyEffects()[key]; gives {
				if spec := worn[idx].TrophySpecOf(); spec != nil && !seen[spec.Name] {
					seen[spec.Name] = true
					names = append(names, spec.Name)
				}
			}
		}
	}
	if len(acting) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("Enchanted with %s: %s", strings.Join(names, ", "), strings.Join(acting, "; "))}
}
