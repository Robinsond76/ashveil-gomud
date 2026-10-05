package usercommands

import (
	"fmt"
	"math"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/assessment"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Ashveil Phase 33i1: consider weighs the player's company against an
// enemy's whole group (a lone creature is a group of one), as scout's
// assessment does, without the grid. The engine's one-on-one odds are
// retired: a company fights together. Free and instant, like scout.

func Consider(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	rest = strings.TrimSpace(rest)
	if rest == `` {
		user.SendText(`Consider whom? Type <ansi fg="command">consider [enemy]</ansi>, or <ansi fg="command">scout</ansi> to see the groups here.`)
		return true, nil
	}

	if room.VisibilityForUser(user) < 1 && !user.Character.HasBuffFlag("nightvision") {
		user.SendText(`It's too dark to make them out.`)
		return true, nil
	}

	g, ok := considerGroup(room, rest)
	if !ok {
		playerId, mobId := room.FindByName(rest)
		if playerId > 0 {
			if playerId == user.UserId {
				user.SendText(`You know yourself well enough. <ansi fg="command">consider</ansi> weighs your company against an enemy.`)
				return true, nil
			}
			if u := users.GetByUserId(playerId); u != nil {
				user.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is no enemy of yours. <ansi fg="command">consider</ansi> weighs your company against an enemy.`, u.Character.Name))
				return true, nil
			}
		}
		if leaderId, _, own := company.LeaderAndKeyForInstance(mobId); own && leaderId == user.UserId {
			if m := mobs.GetInstance(mobId); m != nil && !m.Character.HasBuffFlag("hidden") {
				user.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> travels with you. <ansi fg="command">consider</ansi> weighs your company against an enemy.`, m.Character.Name))
				return true, nil
			}
		}
		user.SendText(fmt.Sprintf(`You see no enemy called "%s" here. Type <ansi fg="command">scout</ansi> to see who's here.`, rest))
		return true, nil
	}

	// A lone creature that means no harm (a shopkeeper, a bystander) is no
	// group to weigh, as scout doesn't list it (33i1 review finding 5).
	if vis := g.Visible(); g.Solo() && !vis[0].Hostile {
		user.SendText(fmt.Sprintf(`The <ansi fg="mobname">%s</ansi> is no enemy of yours. <ansi fg="command">consider</ansi> weighs your company against an enemy.`, vis[0].Character.Name))
		return true, nil
	}

	rep, ok := assessment.Gather(user, room, g)
	if !ok {
		user.SendText(fmt.Sprintf(`You see no enemy called "%s" here. Type <ansi fg="command">scout</ansi> to see who's here.`, rest))
		return true, nil
	}
	lines := []string{fmt.Sprintf(`You consider <ansi fg="mobname">%s</ansi>...`, g.Name)}
	lines = append(lines, rep.Lines()...)
	if line := skillGapLine(user, g.Visible()); line != `` {
		lines = append(lines, line)
	}
	lines = append(lines, fmt.Sprintf(`Type <ansi fg="command">scout %s</ansi> to see how they stand.`, GroupKeyword(room, g)))
	user.SendText(strings.Join(lines, "\n"))
	return true, nil
}

// considerGroup is the group the search names: a group by one of its
// words, else the group of a visible member named as the room names mobs
// ("ruffian", "ruffian#2", "#<id>"). Only members the viewer can see are
// matched, so a hidden namesake never shadows a visible one (33i1 review
// finding 3).
func considerGroup(room *rooms.Room, search string) (enemyparty.Group, bool) {
	if g, ok := enemyparty.FindGroupNamed(room, search); ok {
		return g, true
	}
	groupOf := map[int]enemyparty.Group{}
	for _, g := range enemyparty.Groups(room) {
		for _, m := range g.Visible() {
			groupOf[m.InstanceId] = g
		}
	}
	var names []string
	byName := map[string]int{}
	for _, id := range room.GetMobs() {
		if _, ok := groupOf[id]; !ok {
			continue
		}
		if search == fmt.Sprintf("#%d", id) {
			return groupOf[id], true
		}
		name := fmt.Sprintf("%s#%d", mobs.GetInstance(id).Character.Name, len(names)+1)
		byName[name] = id
		names = append(names, name)
	}
	closeMatch, fullMatch := util.FindMatchIn(search, names...)
	if fullMatch == "" {
		fullMatch = closeMatch
	}
	if fullMatch == "" {
		return enemyparty.Group{}, false
	}
	return groupOf[byName[fullMatch]], true
}

// skillGapLine describes, in words, how the enemies' Attack and Evasion
// compare with the player's (Phase 35a2): the gap between the group's
// average rating and the player's.
func skillGapLine(user *users.UserRecord, foes []*mobs.Mob) string {
	if len(foes) == 0 {
		return ``
	}
	sum := 0
	for _, m := range foes {
		sum += m.Character.AttackSkill() + m.Character.Evasion()
	}
	theirs := float64(sum) / float64(2*len(foes))
	mine := float64(user.Character.AttackSkill()+user.Character.Evasion()) / 2
	return `  They are ` + SkillGapWords(int(math.Round(theirs-mine))) + `.`
}

// SkillGapWords names a rating gap (theirs less yours) without numbers:
// 10 or more far more skilled, 4-9 more skilled, within 3 evenly matched,
// and the mirror phrases behind.
func SkillGapWords(gap int) string {
	switch {
	case gap >= 10:
		return `far more skilled than you`
	case gap >= 4:
		return `more skilled than you`
	case gap > -4:
		return `evenly matched with you`
	case gap > -10:
		return `less skilled than you`
	}
	return `novices next to you`
}
