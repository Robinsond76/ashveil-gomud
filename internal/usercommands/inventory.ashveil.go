package usercommands

// Ashveil's lead lines for the inventory (Phase 26a): the company's load
// against its capacity, the cargo part, and food and water on hand. The
// engine's item-count limit on the Carrying line is a separate thing.

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf(`%d %s`, n, one)
	}
	return fmt.Sprintf(`%d %s`, n, many)
}

// companyLoadLines are the inventory's lead lines, or "" when the load
// can't be read.
func companyLoadLines(user *users.UserRecord, s companyview.Summary) string {
	if !s.LoadKnown {
		return ``
	}
	food, water := 0, 0
	for _, itm := range user.Character.Items {
		spec := itm.GetSpec()
		switch {
		case spec.Nutrition > 0:
			food++
		case spec.Hydration > 0:
			water++
		}
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf(` <ansi fg="yellow">Company load:</ansi> %s, %s of %s kg`, s.LoadLabel, kg(s.Load.TotalGrams()), kg(s.Load.CapacityGrams)))
	if s.Load.CargoGrams > 0 {
		b.WriteString(fmt.Sprintf(` <ansi fg="black-bold">(%s kg in cargo)</ansi>`, kg(s.Load.CargoGrams)))
	}
	b.WriteString(term.CRLFStr)
	b.WriteString(fmt.Sprintf(` <ansi fg="yellow">Supplies:</ansi>     %s, %s`, plural(food, `food`, `food`), plural(water, `drink`, `drinks`)))
	b.WriteString(term.CRLFStr)
	b.WriteString(` <ansi fg="black-bold">Weight is the company's burden (see</ansi> <ansi fg="command">cargo</ansi><ansi fg="black-bold">); the count on the Carrying line is how many items you can hold.</ansi>`)
	b.WriteString(term.CRLFStr)
	return b.String()
}
