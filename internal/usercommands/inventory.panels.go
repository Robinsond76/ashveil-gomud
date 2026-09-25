package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// buildInventoryPanel renders the inventory using the panel system.
// When not searching: Equipment panel followed by a wrapped "Carrying:" line.
// When searching: Equipment panel is omitted; results appear as "Found in your bag:".
// itemList is the filtered set of carried items. searching is true when a filter was given.
func buildInventoryPanel(user *users.UserRecord, itemList []items.Item, searching bool) string {
	c := user.Character

	var sb strings.Builder

	if !searching {
		layout, err := templates.LoadPanelLayout("character/inventory")
		if err != nil {
			return ""
		}

		equipPanel := layout.Panel("equipment")
		equipPanel.SetLabelWidth(9)

		for _, slot := range characters.AllSlots() {
			itm := c.Equipment.Get(slot)
			if itm.IsDisabled() {
				continue
			}
			label := characters.SlotLabel(slot)
			equipPanel.Add(
				fmt.Sprintf(`<ansi fg="yellow">%s</ansi>`, label),
				fmt.Sprintf(`<ansi fg="yellow">%s</ansi>`, label),
				fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, itm.NameComplex()),
			)
		}

		// Ashveil (Phase 26a): the company's load and supplies lead.
		sb.WriteString(companyLoadLines(user, summaryFor(user)))
		sb.WriteString(layout.Render())
		sb.WriteString(term.CRLFStr)

		// Build the wrapped "Carrying:" line, matching the original template behaviour.
		// Items are comma-separated and wrapped at 68 visible chars.
		// The count "(n/cap)" is right-aligned on the second line prefix.
		count := fmt.Sprintf(`(%d/%d)`, len(itemList), c.CarryCapacity())
		sb.WriteString(` Carrying: `)
		lineLen := 0
		lineNum := 1
		for i, item := range itemList {
			name := formatInventoryItemName(item)
			plainLen := len(item.Name())
			if showsUses(item.GetSpec()) {
				plainLen += 2 + len(fmt.Sprintf(`%d`, item.Uses)) + 1 // " (N)"
			}
			if item.Sharpened() {
				plainLen += len(fmt.Sprintf(` (sharp: %d)`, item.SharpStrikes))
			}
			proposed := lineLen + plainLen + 2 // +2 for ", "
			if lineLen > 0 && proposed > 68 {
				lineLen = 0
				lineNum++
				if lineNum == 2 {
					sb.WriteString(term.CRLFStr)
					sb.WriteString(fmt.Sprintf(` %-8s  `, count))
				} else {
					sb.WriteString(term.CRLFStr)
					sb.WriteString(`           `)
				}
			}
			sb.WriteString(name)
			if i < len(itemList)-1 {
				sb.WriteString(`, `)
				lineLen += plainLen + 2
			}
		}
		sb.WriteString(term.CRLFStr)

	} else {
		// Searching: show "Found in your bag:" with each match on its own line.
		sb.WriteString(term.CRLFStr)
		sb.WriteString(` Found in your bag: `)
		for _, item := range itemList {
			sb.WriteString(formatInventoryItemName(item))
			sb.WriteString(term.CRLFStr)
			sb.WriteString(`                   `)
		}
		sb.WriteString(term.CRLFStr)
	}

	return sb.String()
}

// formatInventoryItemName returns the display string for one carried item,
// including a uses count for consumables, lockpicks, and objects with uses
// (a whetstone), and a sharpened weapon's edge (Phase 23b).
func formatInventoryItemName(item items.Item) string {
	name := fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, item.DisplayName())
	if showsUses(item.GetSpec()) {
		name = fmt.Sprintf(`%s <ansi fg="uses-left">(%d)</ansi>`, name, item.Uses)
	}
	if edge := item.EdgeLabel(); edge != `` {
		name = fmt.Sprintf(`%s %s`, name, edge)
	}
	return name
}

// showsUses reports whether an item's uses left are shown in the pack.
func showsUses(iSpec items.ItemSpec) bool {
	return iSpec.Uses > 0 &&
		(iSpec.Subtype == items.Drinkable || iSpec.Subtype == items.Edible ||
			iSpec.Subtype == items.Usable || iSpec.Type == items.Lockpicks ||
			iSpec.Type == items.Object)
}
