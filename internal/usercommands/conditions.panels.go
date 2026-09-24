package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func buildConditionsPanel(user *users.UserRecord) string {
	layout, err := templates.LoadPanelLayout("character/conditions")
	if err != nil {
		layout = templates.NewPanelLayout("open", "single", 1, 1)
		layout.AddPanelsToSlot(layout.AddSlot(), "conditions")
		layout.Panel("conditions").SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Conditions</ansi> `).SetWidth(78)
	}

	charBuffs := user.Character.GetBuffs()
	edges := sharpenedSummary(user)
	if len(charBuffs) == 0 && edges == `` {
		layout.Panel("conditions").Add(``, ``, `<ansi fg="black-bold">None</ansi>`)
		return layout.Render() + term.CRLFStr
	}

	// Collect rows and measure the longest name for label alignment.
	type condRow struct {
		name        string
		description string
		permaBuff   bool
		roundsLeft  int
	}
	rows := make([]condRow, 0, len(charBuffs)+1)
	maxNameWidth := 0
	roundSecs := int(configs.GetTimingConfig().RoundSeconds)

	// Phase 23b: a whetstone's edge is on the weapon, not a buff, but it
	// is still the wielder's condition. It counts strikes, not time, so it
	// is shown like a permanent row.
	if edges != `` {
		rows = append(rows, condRow{name: `Sharpened`, description: edges, permaBuff: true})
		maxNameWidth = len(`Sharpened`)
	}

	for _, buff := range charBuffs {
		spec := buffs.GetBuffSpec(buff.BuffId)
		roundsLeft, _ := buffs.GetDurations(buff, spec)
		name, description := spec.VisibleNameDesc()
		rows = append(rows, condRow{
			name:        name,
			description: description,
			permaBuff:   buff.PermaBuff,
			roundsLeft:  roundsLeft,
		})
		if w := len(name); w > maxNameWidth {
			maxNameWidth = w
		}
	}

	panel := layout.Panel("conditions").SetLabelWidth(maxNameWidth)
	for _, row := range rows {
		var value string
		if row.permaBuff || row.roundsLeft >= buffs.TriggersLeftUnlimited {
			value = fmt.Sprintf(`<ansi fg="yellow">%s</ansi>`, row.description)
		} else {
			timeStr := formatDurationFromRounds(row.roundsLeft, roundSecs)
			value = fmt.Sprintf(`<ansi fg="yellow">%s</ansi>  <ansi fg="red">(%s left)</ansi>`, row.description, timeStr)
		}
		panel.Add(
			fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi>`, row.name),
			fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi>`, row.name),
			value,
		)
	}

	return layout.Render() + term.CRLFStr
}

// sharpenedSummary describes the edges on the character's equipped
// weapons, or "" when none has one.
func sharpenedSummary(user *users.UserRecord) string {
	parts := []string{}
	for _, itm := range []items.Item{user.Character.Equipment.Weapon, user.Character.Equipment.Offhand} {
		if itm.ItemId > 0 && itm.Sharpened() {
			parts = append(parts, fmt.Sprintf(`%s: +%d damage for %d more strikes`, itm.Name(), itm.SharpBonus, itm.SharpStrikes))
		}
	}
	return strings.Join(parts, `; `)
}
