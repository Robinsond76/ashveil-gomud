package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/assessment"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
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
		if playerId, _ := room.FindByName(rest); playerId > 0 {
			if playerId == user.UserId {
				user.SendText(`You know yourself well enough. <ansi fg="command">consider</ansi> weighs your company against an enemy.`)
				return true, nil
			}
			if u := users.GetByUserId(playerId); u != nil {
				user.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is no enemy of yours. <ansi fg="command">consider</ansi> weighs your company against an enemy.`, u.Character.Name))
				return true, nil
			}
		}
		user.SendText(fmt.Sprintf(`You see no enemy called "%s" here. Type <ansi fg="command">scout</ansi> to see who's here.`, rest))
		return true, nil
	}

	rep, ok := assessment.Gather(user, room, g)
	if !ok {
		user.SendText(fmt.Sprintf(`You see no enemy called "%s" here. Type <ansi fg="command">scout</ansi> to see who's here.`, rest))
		return true, nil
	}
	lines := []string{fmt.Sprintf(`You consider <ansi fg="mobname">%s</ansi>...`, g.Name)}
	lines = append(lines, rep.Lines()...)
	lines = append(lines, fmt.Sprintf(`Type <ansi fg="command">scout %s</ansi> to see how they stand.`, GroupKeyword(room, g)))
	user.SendText(strings.Join(lines, "\n"))
	return true, nil
}

// considerGroup is the group the search names: a group or a lone creature
// by name, as scout finds it, else the group of a visible member named.
func considerGroup(room *rooms.Room, search string) (enemyparty.Group, bool) {
	if g, ok := enemyparty.FindGroup(room, search); ok && len(g.Visible()) > 0 {
		return g, true
	}
	playerId, mobId := room.FindByName(search)
	if playerId > 0 || mobId == 0 {
		return enemyparty.Group{}, false
	}
	g, ok := enemyparty.GroupOf(room, mobId)
	if !ok {
		return enemyparty.Group{}, false
	}
	for _, m := range g.Visible() {
		if m.InstanceId == mobId {
			return g, true
		}
	}
	return enemyparty.Group{}, false
}
