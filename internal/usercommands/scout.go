package usercommands

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/encounters"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/assessment"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Ashveil Phase 32c: scout. A player sees how an enemy group stands before
// a fight (or during one), free and instant: no skill, no roll, no round.
// It sees what look sees: hidden members are left out, and in the dark it
// sees nothing.

func Scout(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if room.VisibilityForUser(user) < 1 && !user.Character.HasBuffFlag("nightvision") {
		user.SendText(`It's too dark to make them out.`)
		return true, nil
	}

	rest = strings.TrimSpace(rest)
	if rest == `` {
		user.SendText(scoutList(room, user))
		return true, nil
	}

	g, ok := enemyparty.FindGroup(room, rest)
	if !ok || len(g.Visible()) == 0 {
		user.SendText(fmt.Sprintf(`You see no group called "%s" here. Type <ansi fg="command">scout</ansi> to see who's here.`, rest))
		return true, nil
	}
	user.SendText(scoutGroup(room, g, user))
	return true, nil
}

// scoutList lists the room's enemy groups, as the room's lines do.
func scoutList(room *rooms.Room, user *users.UserRecord) string {
	var lines []string
	for _, g := range enemyparty.Groups(room) {
		vis := g.Visible()
		if len(vis) == 0 {
			continue
		}
		if g.Solo() && !vis[0].Hostile {
			continue // a shopkeeper or a bystander is no group to scout
		}
		names := make([]string, len(vis))
		for i, m := range vis {
			names[i] = m.Character.Name
		}
		line := fmt.Sprintf(`  <ansi fg="mobname">%s</ansi> (%d)`, mobparty.Capitalize(g.Name), len(vis))
		if !g.Solo() {
			line += ": " + mobparty.ListKinds(names)
		}
		if doing := rooms.GroupDoing(user.UserId, g.Party.Members); doing != `` {
			line += ` (` + doing + `)`
		}
		lines = append(lines, line+`. Type <ansi fg="command">scout `+GroupKeyword(room, g)+`</ansi>.`)
	}
	if len(lines) == 0 {
		if room.Encounter != nil && room.Encounter.Enabled {
			out := `You see no enemies here, but the place feels dangerous: a fight could find you.`
			if note := zoneBandNote(user, room); note != `` {
				out += ` ` + note
			}
			return out
		}
		return `You see no enemies here.`
	}
	return "Enemies here:\n" + strings.Join(lines, "\n")
}

// scoutGroup draws the group's formation, front row nearest the viewer,
// each member with how hurt it looks, and marks those the viewer can reach
// from their place in their company's formation.
func scoutGroup(room *rooms.Room, g enemyparty.Group, user *users.UserRecord) string {
	visible := map[int]*mobs.Mob{}
	for _, m := range g.Visible() {
		visible[m.InstanceId] = m
	}

	col, placed, hasCompany := 0, false, false
	if f, ok := company.FormationFor(user.UserId); ok {
		hasCompany = true
		_, col, placed = f.Find(company.LeaderMemberKey)
	}
	reach := combat.ResolveReach(user.Character, false)
	alive := enemyparty.Alive(g.Party)

	lines := []string{fmt.Sprintf(`<ansi fg="mobname">%s</ansi>, as they stand (front row nearest you):`, mobparty.Capitalize(g.Name))}
	lines = append(lines, `        `+fmt.Sprintf(`%-24s%-24s%s`, `col 1`, `col 2`, `col 3`))
	rowLabels := [company.FormationRows]string{"front", "mid  ", "back "}
	marked := false
	for r := 0; r < company.FormationRows; r++ {
		cells := make([]string, 0, company.FormationCols)
		for c := 0; c < company.FormationCols; c++ {
			cell := `------`
			key := g.Party.Formation.At(r, c)
			if id, ok := mobparty.InstanceIdFromMemberKey(key); ok {
				if m := visible[id]; m != nil {
					mark := ` `
					if placed && formationcombat.Legal(col, g.Party.Formation, key, alive, reach) {
						mark, marked = `*`, true
					}
					cell = fmt.Sprintf(`%s%s (%s)`, mark, m.Character.Name, enemyparty.HealthWord(m.Character.Health, m.Character.HealthMax.Value))
				}
			}
			cells = append(cells, fmt.Sprintf(`%-21s`, cell))
		}
		lines = append(lines, fmt.Sprintf(`%s [ %s ]`, rowLabels[r], strings.Join(cells, ` | `)))
	}
	if line := burdenedLine(g.Visible()); line != `` {
		lines = append(lines, line)
	}
	switch {
	case marked:
		lines = append(lines, `* you can reach them from your place in the formation.`)
	case placed:
		lines = append(lines, `You can reach none of them from your place in the formation.`)
	case hasCompany:
		lines = append(lines, `You aren't placed in your company's formation: you can strike any of them, and any of them you.`)
	}
	lines = append(lines, `The front of each column takes the first blows: plain melee reaches only the front, a reach weapon one rank deeper, a bow anyone.`)
	// Phase 33i1: the company's assessment of the group.
	if rep, ok := assessment.Gather(user, room, g); ok {
		lines = append(lines, rep.Lines()...)
	}
	return strings.Join(lines, "\n")
}

// describeGroup is look at a group: its description, how many and what it
// is doing, each member with how hurt it looks, and what to type.
func describeGroup(room *rooms.Room, g enemyparty.Group, user *users.UserRecord) string {
	vis := g.Visible()
	var lines []string
	if g.Naming.Desc != `` {
		lines = append(lines, g.Naming.Desc)
	}
	lines = append(lines, fmt.Sprintf(`<ansi fg="mobname">%s</ansi>, %s strong, %s.`,
		mobparty.Capitalize(g.Name), mobparty.CountWord(len(vis)), rooms.DoingPhrase(rooms.GroupDoing(user.UserId, g.Party.Members))))
	parts := make([]string, len(vis))
	for i, m := range vis {
		state := enemyparty.HealthWord(m.Character.Health, m.Character.HealthMax.Value)
		if word := m.Character.BurdenWord(); word != characters.BurdenNone {
			state += `, ` + word
		}
		parts[i] = fmt.Sprintf(`%s %s (%s)`, mobparty.Article(m.Character.Name), m.Character.Name, state)
	}
	lines = append(lines, `  `+strings.Join(parts, `, `))
	kw := GroupKeyword(room, g)
	lines = append(lines, fmt.Sprintf(`  Type <ansi fg="command">scout %s</ansi> to see how they stand, or <ansi fg="command">attack %s</ansi> to fight them.`, kw, kw))
	return strings.Join(lines, "\n")
}

// burdenedLine names the group's members whose own load burdens them in a
// fight (Phase 30g3), each with its word; "" when none is. The grid stays
// narrow: the unburdened are left out.
func burdenedLine(members []*mobs.Mob) string {
	var parts []string
	for _, m := range members {
		if word := m.Character.BurdenWord(); word != characters.BurdenNone {
			parts = append(parts, fmt.Sprintf(`%s (%s)`, m.Character.Name, word))
		}
	}
	if len(parts) == 0 {
		return ``
	}
	return `Burdened: ` + strings.Join(parts, `, `) + `. A burdened fighter dodges less.`
}

// ratingPhrases say how a zone's band weighs on the viewer's level
// (encounters.Rating): the owner's rule that difficulty comes only from
// entering a zone above your level.
var ratingPhrases = map[string]string{
	encounters.RatingEasy:      `Your company should manage.`,
	encounters.RatingFair:      `A fair test at your level.`,
	encounters.RatingRisky:     `Risky at your level: expect losses.`,
	encounters.RatingDangerous: `Dangerous at your level: prepare carefully.`,
}

// zoneBandNote is the zone's level band for look and scout: "Foes in the
// Dark Forest are of levels 5 to 7." and how that weighs on the viewer's
// level. Empty for a zone with no band (towns, unfinished zones).
func zoneBandNote(user *users.UserRecord, room *rooms.Room) string {
	cfg := rooms.GetZoneConfig(room.Zone)
	if cfg == nil || !cfg.Encounters.Band.Valid() {
		return ``
	}
	b := cfg.Encounters.Band
	out := fmt.Sprintf(`Foes in %s are of levels %d to %d.`, cfg.Name, b.Low, b.High)
	if user != nil && user.Character != nil {
		if phrase := ratingPhrases[encounters.Rating(user.Character.Level, b)]; phrase != `` {
			out += ` ` + phrase
		}
	}
	return out
}
