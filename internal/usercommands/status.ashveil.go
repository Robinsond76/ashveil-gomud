package usercommands

// Ashveil's additions to the status sheet (Phase 26a). The engine's panel
// code calls these when the layout defines the Ashveil panels; the values
// come from internal/companyview, the same read model as the prompt.

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/util"
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
	archetype := s.Leader.RankName()
	if archetype == `` {
		archetype = `<ansi fg="black-bold">none chosen</ansi>`
	}
	addRow(p, `Path:   `, `Pth:`, archetype)
}

func addAshveilAlignment(p *templates.Panel, s companyview.Summary) {
	addRow(p, `Align:  `, `Aln:`, fmt.Sprintf(`%d (%s)`, s.Alignment, company.AlignmentBand(s.Alignment)))
}

// addAshveilBurden shows how burdened the character's own load leaves
// them in a fight (Phase 30g3): a word, never a ratio.
func addAshveilBurden(p *templates.Panel, c *characters.Character) {
	addRow(p, `Burden: `, `Bdn:`, burdenValue(c.BurdenWord()))
}

// burdenValue colours a burden word: green when unburdened, yellow when
// light, red otherwise.
func burdenValue(word string) string {
	colour := `red`
	switch word {
	case characters.BurdenNone:
		colour = `green`
	case characters.BurdenLight:
		colour = `yellow`
	}
	return fmt.Sprintf(`<ansi fg="%s">%s</ansi>`, colour, util.CapitalizeFirst(word))
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
	// Phase 45: each companion's class, so the sheet shows who is who.
	for _, m := range s.Companions {
		if line := companionLine(m); line != `` {
			addRow(p, `  `+companionLabel(m), companionLabel(m), line)
		}
	}
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

// companionLabel is a short row label for a companion: its name cut to fit
// the panel's label column.
func companionLabel(m companyview.Member) string {
	name := m.Name
	if name == `` {
		name = fmt.Sprintf(`#%d`, m.ID)
	}
	if r := []rune(name); len(r) > 7 {
		name = string(r[:7])
	}
	return name + `:`
}

// companionLine is a companion's class and level, or "" when nothing is
// known about it (Phase 45).
func companionLine(m companyview.Member) string {
	rank := m.RankName()
	if rank == `` && m.Level == 0 {
		return ``
	}
	line := rank
	if m.Level > 0 {
		if line != `` {
			line += `, `
		}
		line += fmt.Sprintf(`Lv %d`, m.Level)
	}
	if m.Status == company.MemberDead {
		line += ` <ansi fg="red">(fallen)</ansi>`
	}
	return line
}
