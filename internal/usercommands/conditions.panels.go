package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
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

	// Ashveil (Phase 26a): conditions are grouped (conditions.ashveil.go).
	groups := conditionGroups(user, summaryFor(user))
	if len(groups) == 0 {
		layout.Panel("conditions").Add(``, ``, `<ansi fg="black-bold">None</ansi>`)
		return layout.Render() + term.CRLFStr
	}

	maxNameWidth := 0
	for _, g := range groups {
		for _, row := range g.rows {
			if w := len(row.name); w > maxNameWidth {
				maxNameWidth = w
			}
		}
	}
	roundSecs := int(configs.GetTimingConfig().RoundSeconds)

	panel := layout.Panel("conditions").SetLabelWidth(maxNameWidth)
	for i, g := range groups {
		if i > 0 {
			panel.AddBlank()
		}
		panel.Add(``, ``, fmt.Sprintf(`<ansi fg="20">%s</ansi>`, g.title))
		for _, row := range g.rows {
			var value string
			switch {
			case row.left != ``:
				value = fmt.Sprintf(`<ansi fg="yellow">%s</ansi>  <ansi fg="red">(%s)</ansi>`, row.description, row.left)
			case row.permaBuff || row.roundsLeft >= buffs.TriggersLeftUnlimited || row.roundsLeft <= 0:
				value = fmt.Sprintf(`<ansi fg="yellow">%s</ansi>`, row.description)
			default:
				timeStr := formatDurationFromRounds(row.roundsLeft, roundSecs)
				value = fmt.Sprintf(`<ansi fg="yellow">%s</ansi>  <ansi fg="red">(%s left)</ansi>`, row.description, timeStr)
			}
			panel.Add(
				fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi>`, row.name),
				fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi>`, row.name),
				value,
			)
		}
	}

	return layout.Render() + term.CRLFStr
}
