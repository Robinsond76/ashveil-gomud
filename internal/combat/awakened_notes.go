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
