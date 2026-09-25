package usercommands

// Ashveil's additions to the status sheet (Phase 26a). The engine's panel
// code calls these when the layout defines the Ashveil panels; the values
// come from internal/companyview, the same read model as the prompt.

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/templates"
)

// summaryFor builds the read model; tests swap it.
var summaryFor = companyview.For

const ashveilStatusFooter = ` <ansi fg="black-bold">More:</ansi> <ansi fg="command">company status</ansi>, <ansi fg="command">formation</ansi>, <ansi fg="command">conditions</ansi>, <ansi fg="command">survival</ansi>, <ansi fg="command">status bonuses</ansi>`

func yellowLabel(full, short string) (string, string) {
	return `<ansi fg="yellow">` + full + `</ansi>`, `<ansi fg="yellow">` + short + `</ansi>`
}

func addRow(p *templates.Panel, full, short, value string) {
	f, s := yellowLabel(full, short)
	p.Add(f, s, value)
}

func addAshveilIdentity(p *templates.Panel, s companyview.Summary) {
	if !s.Leader.ArchetypeKnown {
		return
	}
	archetype := s.Leader.Archetype
	if archetype == `` {
		archetype = `<ansi fg="black-bold">none chosen</ansi>`
	}
	addRow(p, `Path:   `, `Pth:`, archetype)
}

func addAshveilAlignment(p *templates.Panel, s companyview.Summary) {
	addRow(p, `Align:  `, `Aln:`, fmt.Sprintf(`%d (%s)`, s.Alignment, company.AlignmentBand(s.Alignment)))
}

// needValue renders a need, in warning colour when Low or worse.
func needValue(n companyview.Need) string {
	colour := `green`
	if n.Warns() {
		colour = `red`
	}
	return fmt.Sprintf(`<ansi fg="%s">%s</ansi> <ansi fg="black-bold">(%d)</ansi>`, colour, n.Label, n.Value)
}

func addAshveilVitals(p *templates.Panel, s companyview.Summary) {
	for _, row := range []struct {
		full, short string
		need        companyview.Need
	}{
		{`Hunger: `, `Hun:`, s.Leader.Hunger},
		{`Thirst: `, `Thr:`, s.Leader.Thirst},
		{`Fatigue:`, `Ftg:`, s.Leader.Fatigue},
	} {
		if row.need.Known {
			addRow(p, row.full, row.short, needValue(row.need))
		}
	}
	if s.Leader.WarmthKnown {
		warmth := `<ansi fg="green">Comfortable</ansi>`
		if s.Leader.Warmth != `` {
			warmth = `<ansi fg="red">` + s.Leader.Warmth + `</ansi>`
		}
		addRow(p, `Warmth: `, `Wrm:`, warmth)
	}
	if s.LightKnown {
		light := `<ansi fg="green">Lit</ansi>`
		if label := companyview.LightLabel(s.Light); label != `` {
			light = `<ansi fg="yellow">` + label + `</ansi>`
		}
		addRow(p, `Light:  `, `Lgt:`, light)
	}
}

// companyLine is "You alone", or "3 alive, 1 fallen".
func companyLine(s companyview.Summary) string {
	if !s.CompanyKnown {
		return `<ansi fg="black-bold">unknown</ansi>`
	}
	if len(s.Companions) == 0 {
		return `You alone`
	}
	line := fmt.Sprintf(`%d alive`, s.Alive)
	if s.Dead > 0 {
		line += fmt.Sprintf(`, <ansi fg="red">%d fallen</ansi>`, s.Dead)
	}
	return line
}

func kg(grams int) string { return fmt.Sprintf(`%.1f`, float64(grams)/1000) }

func addAshveilCompany(p *templates.Panel, s companyview.Summary) {
	addRow(p, `Members: `, `Mem:`, companyLine(s))
	if s.LoadKnown {
		addRow(p, `Load:    `, `Lod:`, fmt.Sprintf(`%s <ansi fg="black-bold">(%s/%s kg)</ansi>`, s.LoadLabel, kg(s.Load.TotalGrams()), kg(s.Load.CapacityGrams)))
	}
	if s.ActivityKnown {
		doing := s.Activity.Label()
		if doing == `` {
			doing = `<ansi fg="black-bold">Nothing in particular</ansi>`
		}
		addRow(p, `Doing:   `, `Dng:`, doing)
	}
	if s.RestKnown {
		addRow(p, `Rest:    `, `Rst:`, restLine(s))
	}
	if s.Checkpoint != `` {
		addRow(p, `Wake at: `, `Wke:`, s.Checkpoint)
	}
}

func restLine(s companyview.Summary) string {
	if s.RestTier == camping.TierNone {
		return `<ansi fg="black-bold">Not rested</ansi>`
	}
	return fmt.Sprintf(`<ansi fg="green">%s</ansi> <ansi fg="black-bold">(%s left)</ansi>`, s.RestTier, companyview.FormatRemaining(s.RestLeft))
}
