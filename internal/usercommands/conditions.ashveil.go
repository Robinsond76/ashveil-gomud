package usercommands

// Ashveil's grouping of the conditions panel (Phase 26a): rest, weapon
// edges, survival, company chemistry, then every other effect. Each row
// says what is left: real time, strikes, or nothing for a lasting state.

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
)

type condRow struct {
	name        string
	description string
	permaBuff   bool
	roundsLeft  int
	// left, when set, is shown as is in place of the rounds.
	left string
}

type condGroup struct {
	title string
	rows  []condRow
}

// conditionGroups sorts the character's visible conditions into groups,
// leaving out empty ones.
func conditionGroups(user *users.UserRecord, s companyview.Summary) []condGroup {
	rest := condGroup{title: `Rest`}
	edges := condGroup{title: `Weapon edges`}
	survival := condGroup{title: `Survival`}
	chemistry := condGroup{title: `Company`}
	other := condGroup{title: `Other effects`}

	for _, itm := range []items.Item{user.Character.Equipment.Weapon, user.Character.Equipment.Offhand} {
		if itm.ItemId > 0 && itm.Sharpened() {
			edges.rows = append(edges.rows, condRow{name: `Sharpened`, description: fmt.Sprintf(`%s: +%d damage`, itm.Name(), itm.SharpBonus),
				left: fmt.Sprintf(`%d strikes left`, itm.SharpStrikes)})
		}
	}

	for _, n := range []struct {
		name string
		need companyview.Need
	}{{`Hunger`, s.Leader.Hunger}, {`Thirst`, s.Leader.Thirst}, {`Fatigue`, s.Leader.Fatigue}} {
		if n.need.Warns() {
			survival.rows = append(survival.rows, condRow{name: n.name, description: fmt.Sprintf(`%s (%d)`, n.need.Label, n.need.Value), permaBuff: true})
		}
	}

	for _, buff := range user.Character.GetBuffs() {
		spec := buffs.GetBuffSpec(buff.BuffId)
		if spec == nil {
			continue
		}
		roundsLeft, _ := buffs.GetDurations(buff, spec)
		name, description := spec.VisibleNameDesc()
		row := condRow{name: name, description: description, permaBuff: buff.PermaBuff, roundsLeft: roundsLeft}
		group, _ := companyview.GroupOf(buff.BuffId)
		switch {
		case spec.Secret:
			other.rows = append(other.rows, row)
		case group == companyview.GroupRest:
			rest.rows = append(rest.rows, row)
		case group == companyview.GroupSurvival:
			survival.rows = append(survival.rows, row)
		default:
			other.rows = append(other.rows, row)
		}
	}

	// Chemistry is lasting, not an expiring buff: its tier and the band.
	if standing, ok := company.ChemistryStanding(user.UserId, company.LeaderMemberKey); ok && standing.Together >= 2 {
		description := fmt.Sprintf(`%s band, %d together`, company.TierName(standing.Tier), standing.Together)
		if standing.Bonus > 0 {
			description += fmt.Sprintf(`: +%d%% to hit`, standing.Bonus)
		}
		chemistry.rows = append(chemistry.rows, condRow{name: `Chemistry`, description: description, permaBuff: true})
	}

	var out []condGroup
	for _, g := range []condGroup{rest, edges, survival, chemistry, other} {
		if len(g.rows) > 0 {
			out = append(out, g)
		}
	}
	return out
}
