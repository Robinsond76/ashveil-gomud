package company

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func equipmentEdge(itm items.Item) (int, int) {
	if itm.GetSpec().Type == items.Weapon && itm.Sharpened() {
		return itm.SharpBonus, itm.SharpStrikes
	}
	return 0, 0
}

func equipmentStats(c *characters.Character, load encumbrance.Load) domain.EquipmentStats {
	weapon, offhand := c.Equipment.Weapon.GetDamage(), c.Equipment.Offhand.GetDamage()
	edgeBonus, edgeStrikes := equipmentEdge(c.Equipment.Weapon)
	offhandEdgeBonus, offhandEdgeStrikes := equipmentEdge(c.Equipment.Offhand)
	offhandDamage := ""
	if c.Equipment.Offhand.GetSpec().Type == items.Weapon {
		offhandDamage = offhand.DiceRoll
	}
	return domain.EquipmentStats{EdgeBonus: edgeBonus, EdgeStrikes: edgeStrikes, OffhandEdgeBonus: offhandEdgeBonus, OffhandEdgeStrikes: offhandEdgeStrikes, Hands: c.HandsRequired(c.Equipment.Weapon), Reach: c.Equipment.Weapon.GetSpec().Reach, Shield: c.HasShield(), HealthMax: c.HealthMax.Value, ManaMax: c.ManaMax.Value, Damage: weapon.DiceRoll, OffhandDamage: offhandDamage, Defense: c.GetDefense(), WornG: c.PersonalGrams(), Burden: c.BurdenWord(), DodgePct: combat.DodgeRetentionPct(c), PackCapacityG: c.Equipment.Pack.CarryBonusGrams(), CapacityG: load.CapacityGrams, CargoG: load.TotalGrams(), Stats: map[string]int{
		"strength": c.Stats.Strength.ValueAdj, "speed": c.Stats.Speed.ValueAdj, "smarts": c.Stats.Smarts.ValueAdj, "vitality": c.Stats.Vitality.ValueAdj, "mysticism": c.Stats.Mysticism.ValueAdj, "perception": c.Stats.Perception.ValueAdj}}
}

func equipmentFits(loadBefore, loadAfter encumbrance.Load) bool {
	return max(0, loadAfter.TotalGrams()-loadAfter.CapacityGrams) <= max(0, loadBefore.TotalGrams()-loadBefore.CapacityGrams)
}

// EquipmentView never prepares assets, saves or mutates an item. Catalogue
// choices are exact instances owned by this leader; commands re-resolve them.
func (m *CompanyModule) EquipmentView(id int) domain.EquipmentView {
	out := domain.EquipmentView{Slots: []domain.EquipmentSlot{}}
	u := users.GetByUserId(id)
	if u == nil || u.Character == nil {
		out.Reason = "Character unavailable."
		return out
	}
	if !u.Character.CompanyCargo {
		out.Reason = "Shared cargo is not ready."
		return out
	}
	load, known := encumbrance.CurrentLoad(id)
	out.Current = equipmentStats(u.Character, load)
	_, actor, err := m.equipmentActor(u, "me")
	if err != nil {
		out.Reason = err.Error()
	} else if err = m.persistenceAvailable(); err != nil {
		out.Reason = err.Error()
	} else {
		if rec, ok := m.registry.Get(id); ok && rec.AssetOperation != nil {
			out.Reason = "Equipment operation awaits recovery."
		} else if !known {
			out.Reason = "Cargo capacity unavailable."
		} else {
			out.Available = true
		}
	}
	if actor == nil {
		actor = u.Character
	}
	preview := func(verb, slot string, itm items.Item) domain.EquipmentChoice {
		choice := domain.EquipmentChoice{Ref: itm.ShorthandId(), Label: domain.PlainLabel(itm)}
		args := []string{verb, "me", itm.ShorthandId(), slot}
		if verb == "remove" {
			args = []string{verb, "me", slot, itm.ShorthandId()}
		}
		choice.Command = "company " + strings.Join(args, " ")
		proposed, cargo, _, displaced, err := equipmentProposal(u, actor, args)
		if err != nil {
			choice.Reason = err.Error()
			return choice
		}
		before, after, ok := equipmentLoad(id, actor, proposed, u.Character.Items, cargo)
		for _, old := range displaced {
			if old.ItemId > 0 {
				choice.Returned = append(choice.Returned, domain.PlainLabel(old))
			}
		}
		if verb == "remove" {
			choice.Returned = append(choice.Returned, domain.PlainLabel(itm))
		}
		stats := equipmentStats(proposed, after)
		choice.After = &stats
		if !out.Available {
			choice.Reason = out.Reason
		} else if !ok {
			choice.Reason = "Cargo capacity unavailable."
		} else if !(verb == "remove" && slot == string(items.Pack)) && !equipmentFits(before, after) {
			choice.Reason = "That change would exceed company cargo capacity. Drop or sell cargo, or assign a larger pack first."
		} else {
			choice.Allowed = true
		}
		return choice
	}
	for _, slot := range characters.AllSlots() {
		entry := domain.EquipmentSlot{Slot: string(slot), Label: strings.TrimSuffix(characters.SlotLabel(slot), ":"), Choices: []domain.EquipmentChoice{}}
		if itm := actor.Equipment.Get(slot); itm != nil && itm.ItemId > 0 {
			equipped := domain.EquipmentChoice{Ref: itm.ShorthandId(), Label: domain.PlainLabel(*itm)}
			removal := preview("remove", string(slot), *itm)
			entry.Equipped, entry.Remove = &equipped, &removal
		}
		for _, itm := range u.Character.Items {
			spec := itm.GetSpec()
			if spec.Type == slot || (slot == items.Offhand && spec.Type == items.Weapon) {
				entry.Choices = append(entry.Choices, preview("equip", string(slot), itm))
			}
		}
		out.Slots = append(out.Slots, entry)
	}
	return out
}
