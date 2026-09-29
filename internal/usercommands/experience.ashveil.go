package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/death"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// lastLossLine is the experience line for the most recent level lost to
// death (Phase 26a), or "" when there is none.
func lastLossLine(user *users.UserRecord) string {
	from, to, ok := death.LastLoss(user.Character)
	if !ok {
		return ``
	}
	if from > to {
		return fmt.Sprintf(`<ansi fg="black-bold">Your last death cost you a level: %d to %d.</ansi>`, from, to) + "\n"
	}
	return fmt.Sprintf(`<ansi fg="black-bold">Your last death cost you your progress toward level %d.</ansi>`, from+1) + "\n"
}

// companyExperienceLines is Phase 32e: each companion's level and progress,
// below the leader's own block. Empty for a solo player or while the
// company can't be read. A companion that isn't out shows its level
// alone, since only a live mob knows its progress.
func companyExperienceLines(user *users.UserRecord) string {
	s := companyview.For(user)
	if !s.CompanyKnown || len(s.Companions) == 0 {
		return ``
	}
	var b strings.Builder
	b.WriteString(`<ansi fg="yellow">Your company:</ansi>` + "\n")
	for _, m := range s.Companions {
		switch {
		case m.Status == company.MemberDead:
			fmt.Fprintf(&b, `  <ansi fg="mobname">%s</ansi> <ansi fg="black-bold">level %d, fallen</ansi>`+"\n", m.Name, m.Level)
		case m.ExpKnown && m.ExpTNL > 0:
			fmt.Fprintf(&b, `  <ansi fg="mobname">%s</ansi> <ansi fg="yellow">Lvl:</ansi> <ansi fg="white">%d</ansi> <ansi fg="yellow">XP:</ansi> <ansi fg="white">%d/%d</ansi> <ansi fg="black-bold">(%d%%)</ansi>`+"\n",
				m.Name, m.Level, m.ExpInto, m.ExpTNL, m.ExpInto*100/m.ExpTNL)
		default:
			fmt.Fprintf(&b, `  <ansi fg="mobname">%s</ansi> <ansi fg="yellow">Lvl:</ansi> <ansi fg="white">%d</ansi>`+"\n", m.Name, m.Level)
		}
	}
	return b.String()
}
