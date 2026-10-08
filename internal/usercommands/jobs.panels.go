package usercommands

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const (
	jobsBarLength       = 39  // number of characters in the progress bar
	jobsExperienceWidth = 11  // fixed column width for the experience title
	jobsBarFull         = `█` // character used for the filled portion of the bar
	jobsBarEmpty        = `░` // character used for the empty portion of the bar
)

func buildJobsPanel(user *users.UserRecord) string {
	allRanks := user.Character.GetAllSkillRanks()

	type jobRow struct {
		name       string
		experience string
		barFull    string
		barEmpty   string
		completion string
	}

	rows := make([]jobRow, 0)
	for _, rank := range skills.GetProfessionRanks(allRanks) {
		barFull, barEmpty := util.ProgressBar(rank.Completion, jobsBarLength, jobsBarFull, jobsBarEmpty)
		rows = append(rows, jobRow{
			name:       rank.Profession,
			experience: rank.ExperienceTitle,
			barFull:    barFull,
			barEmpty:   barEmpty,
			completion: fmt.Sprintf(`%d%%`, int(math.Floor(rank.Completion*100))),
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.Compare(rows[i].name, rows[j].name) == -1
	})

	layout, err := templates.LoadPanelLayout("character/jobs")
	if err != nil {
		layout = templates.NewPanelLayout("open", "single", 1, 1)
		layout.AddPanelsToSlot(layout.AddSlot(), "jobs")
		layout.Panel("jobs").SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="69">Jobs</ansi> `).SetWidth(78)
	}

	if len(rows) == 0 {
		layout.Panel("jobs").Add(``, ``, `No jobs. Train skills to unlock professions.`)
		return layout.Render() + term.CRLFStr
	}

	maxNameWidth := 0
	for _, r := range rows {
		if w := len(r.name); w > maxNameWidth {
			maxNameWidth = w
		}
	}

	panel := layout.Panel("jobs").SetLabelWidth(maxNameWidth)
	for _, r := range rows {
		value := fmt.Sprintf(
			`<ansi fg="white-bold">%s</ansi> <ansi fg="green">%s</ansi><ansi fg="black-bold">%s</ansi> <ansi fg="cyan-bold">%s</ansi>`,
			fmt.Sprintf(`%-*s`, jobsExperienceWidth, r.experience), r.barFull, r.barEmpty, r.completion,
		)
		panel.AddWithWrapWidth(
			fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi>`, r.name),
			fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi>`, r.name),
			value,
			-1,
		)
	}

	return layout.Render() + term.CRLFStr
}
