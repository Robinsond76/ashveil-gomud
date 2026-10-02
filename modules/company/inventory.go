package company

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mount"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// inventoryView is `company inventory` (Phase 32f): the load, every
// member's worn and carried items with their weight and pack, the horses,
// and the cargo. Game loop only: it reads the leader, live companions, and
// the other modules' providers, none under this module's state.
func (m *CompanyModule) inventoryView(user *users.UserRecord) string {
	leaderUserID := user.UserId
	lines := []string{}
	if load, ok := encumbrance.CurrentLoad(leaderUserID); ok {
		line := fmt.Sprintf("Company load: %s / %s (%.0f%%)", kg(load.TotalGrams()), kg(load.CapacityGrams), load.Ratio()*100)
		if load.MountCapacityGrams > 0 {
			line += fmt.Sprintf(", of which horses %s", kg(load.MountCapacityGrams))
		}
		lines = append(lines, line+".", "")
	}

	leader := domain.MemberState{Items: user.Character.Items, Equipment: user.Character.Equipment}
	if user.Character.CompanyCargo {
		leader.Items = nil
		lines = append(lines, fmt.Sprintf("Company treasury: %d gold.", user.Character.Gold))
	}
	if !user.Character.CompanyCargo {
		lines = append(lines, memberBlock(user.Character.Name+" (you)", leader)...)
	}

	if !user.Character.CompanyCargo && m.persistenceAvailable() == nil {
		if record, ok := m.registry.Get(leaderUserID); ok {
			for _, c := range record.Companions {
				name := fmt.Sprintf("#%d %s", c.ID, nameOf(c, strconv.Itoa(c.MobTemplateID)))
				if c.Dead() {
					lines = append(lines, name+"  fallen; their gear is with the body")
					continue
				}
				// Show the live mob's gear when it is out, as `company gear`
				// does.
				if m.refreshSnapshot(leaderUserID, c.ID) {
					if current, ok := m.registry.Get(leaderUserID); ok {
						if updated, ok := resolveCompanion(current, "#"+strconv.Itoa(c.ID)); ok {
							c = updated
						}
					}
				}
				state := c.State
				if state == nil {
					if template, ok := m.runtime.TemplateState(c.MobTemplateID); ok {
						state = &template
					}
				}
				if state == nil {
					lines = append(lines, name+"  gear not yet recorded")
					continue
				}
				lines = append(lines, memberBlock(name, *state)...)
			}
		}
	}

	lines = append(lines, "")
	if herd := mount.HerdOf(leaderUserID); len(herd) > 0 {
		horses := make([]string, 0, len(herd))
		for _, h := range herd {
			horses = append(horses, horseLabel(h))
		}
		lines = append(lines, "Horses: "+strings.Join(horses, "; "))
	} else {
		lines = append(lines, "Horses: none")
	}
	if user.Character.CompanyCargo {
		lines = append(lines, "Assigned containers:")
		lines = append(lines, packLine(user.Character.Name, user.Character.Equipment.Pack, user.Character.Health > 0))
		if members, ok := m.CompanyInventory(leaderUserID); ok {
			for _, member := range members {
				if member.Pack != "" {
					lines = append(lines, fmt.Sprintf("  %s (%s): %.1f kg — %s", member.Pack, member.Name, float64(member.PackBonusGrams)/1000, availableWord(member.Available && !member.Fallen)))
				}
			}
		}
		lines = append(lines, "Company cargo:")
		for _, itm := range user.Character.Items {
			lines = append(lines, fmt.Sprintf("  %s  %s", itm.ShorthandId(), itemName(itm)))
		}
	} else {
		lines = append(lines, cargoLine(encumbrance.CargoContents(leaderUserID)))
	}
	return strings.Join(lines, "\n")
}

// memberBlock is one member's header (weight, pack) and gear lines.
func memberBlock(name string, s domain.MemberState) []string {
	packLabel := "no pack"
	if pack, grams := domain.BestPack(s.Items); grams > 0 {
		packLabel = fmt.Sprintf("pack: %s (+%s)", itemName(pack), kg(grams))
	}
	worn := []string{}
	for _, slot := range characters.AllSlots() {
		if itm := s.Equipment.Get(slot); itm != nil && itm.ItemId > 0 {
			worn = append(worn, itemName(*itm))
		}
	}
	return []string{
		fmt.Sprintf("%-24s %8s   %s", name, kg(gearGrams(s)), packLabel),
		"  Wearing: " + listOrNothing(worn),
		"  Carrying: " + listOrNothing(groupItems(s.Items)),
	}
}

// groupItems names carried items, a partly used one with its uses left,
// and counts repeats: "waterskin (3 of 5), seared game meat x2".
func groupItems(carried []items.Item) []string {
	order := []string{}
	counts := map[string]int{}
	for _, itm := range carried {
		if itm.ItemId <= 0 {
			continue
		}
		label := itemName(itm)
		if spec := items.GetItemSpec(itm.ItemId); spec != nil && spec.Uses > 1 && itm.Uses > 0 && itm.Uses < spec.Uses {
			label += fmt.Sprintf(" (%d of %d)", itm.Uses, spec.Uses)
		}
		if counts[label] == 0 {
			order = append(order, label)
		}
		counts[label]++
	}
	out := make([]string, 0, len(order))
	for _, label := range order {
		if counts[label] > 1 {
			label += " x" + strconv.Itoa(counts[label])
		}
		out = append(out, label)
	}
	return out
}

func horseLabel(h mount.HorseView) string {
	label := fmt.Sprintf("#%d %s", h.ID, h.Name)
	saddle := "no saddle"
	if h.Saddle != "" {
		saddle = h.Saddle
	}
	switch {
	case h.Kind == mount.KindRiding && h.Saddle != "":
		return fmt.Sprintf("%s (%s, carries a rider)", label, saddle)
	case h.CapacityGrams > 0:
		return fmt.Sprintf("%s (%s, +%s)", label, saddle, kg(h.CapacityGrams))
	}
	return fmt.Sprintf("%s (%s)", label, saddle)
}

func cargoLine(stacks []encumbrance.CargoStack) string {
	if len(stacks) == 0 {
		return "Cargo: empty"
	}
	grams := 0
	names := make([]string, 0, len(stacks))
	for _, s := range stacks {
		name := fmt.Sprintf("item %d", s.ItemId)
		if spec := items.GetItemSpec(s.ItemId); spec != nil {
			name = itemName(items.Item{ItemId: s.ItemId})
			grams += spec.Weight * s.Count
			if s.Uses > 0 && spec.Uses > 1 {
				name += fmt.Sprintf(" (%d of %d)", s.Uses, spec.Uses)
			}
		}
		if s.Count > 1 {
			name += " x" + strconv.Itoa(s.Count)
		}
		names = append(names, name)
	}
	return fmt.Sprintf("Cargo (%s): %s", kg(grams), strings.Join(names, ", "))
}

func listOrNothing(names []string) string {
	if len(names) == 0 {
		return "nothing"
	}
	return strings.Join(names, ", ")
}

func kg(grams int) string {
	return fmt.Sprintf("%.1f kg", float64(grams)/1000)
}

func availableWord(ok bool) string {
	if ok {
		return "available"
	}
	return "unavailable"
}
func packLine(name string, pack items.Item, available bool) string {
	if pack.ItemId < 1 {
		return "  " + name + ": no assigned pack"
	}
	return fmt.Sprintf("  %s (%s): %.1f kg — %s", domain.PlainLabel(pack), name, float64(pack.CarryBonusGrams())/1000, availableWord(available))
}
