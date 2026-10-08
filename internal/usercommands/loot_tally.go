package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// lootTally collects everything one looting action takes so it can be told
// in a single line per kind of corpse ("You take 12 gold, 2 wooden shields
// and the leather cap from the skeleton corpses.") instead of a line per item.
type lootTally struct {
	groups []*lootGroup
}

type lootGroup struct {
	color   string
	name    string
	corpses map[*rooms.Corpse]struct{}
	gold    int
	order   []string
	counts  map[string]int
}

func (t *lootTally) group(c *rooms.Corpse) *lootGroup {
	color := `mob-corpse`
	if c.UserId > 0 {
		color = `user-corpse`
	}
	for _, g := range t.groups {
		if g.name == c.Character.Name && g.color == color {
			g.corpses[c] = struct{}{}
			return g
		}
	}
	g := &lootGroup{color: color, name: c.Character.Name, corpses: map[*rooms.Corpse]struct{}{c: {}}, counts: map[string]int{}}
	t.groups = append(t.groups, g)
	return g
}

func (t *lootTally) addGold(c *rooms.Corpse, amt int) {
	t.group(c).gold += amt
}

func (t *lootTally) addItem(c *rooms.Corpse, name string) {
	g := t.group(c)
	if g.counts[name] == 0 {
		g.order = append(g.order, name)
	}
	g.counts[name]++
}

// lootItemPhrase is "the leather cap" for one, "2 wooden shields" for more.
func lootItemPhrase(name string, n int) string {
	if n <= 1 {
		return fmt.Sprintf(`the <ansi fg="itemname">%s</ansi>`, name)
	}
	plural := name
	if !strings.HasSuffix(strings.ToLower(name), "s") { // "tattered pants" stays
		plural = mobparty.Plural(name)
	}
	return fmt.Sprintf(`%d <ansi fg="itemname">%s</ansi>`, n, plural)
}

func joinLoot(parts []string) string {
	switch len(parts) {
	case 0:
		return ``
	case 1:
		return parts[0]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + ` and ` + parts[len(parts)-1]
}

func (g *lootGroup) corpseName() string {
	noun := `corpse`
	if len(g.corpses) > 1 {
		noun = `corpses`
	}
	return fmt.Sprintf(`<ansi fg="%s">%s %s</ansi>`, g.color, g.name, noun)
}

// flush tells the user, and everyone else in the room, what was taken.
func (t *lootTally) flush(user *users.UserRecord, room *rooms.Room) {
	for _, g := range t.groups {
		var mine, theirs []string
		if g.gold > 0 {
			mine = append(mine, fmt.Sprintf(`<ansi fg="gold">%d gold</ansi>`, g.gold))
			theirs = append(theirs, `some <ansi fg="gold">gold</ansi>`)
		}
		for _, name := range g.order {
			p := lootItemPhrase(name, g.counts[name])
			mine = append(mine, p)
			theirs = append(theirs, p)
		}
		if len(mine) == 0 {
			continue
		}
		user.SendText(fmt.Sprintf(`You take %s from the %s.`, joinLoot(mine), g.corpseName()))
		room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> takes %s from the %s.`, user.Character.Name, joinLoot(theirs), g.corpseName()), user.UserId)
	}
	t.groups = nil
}
