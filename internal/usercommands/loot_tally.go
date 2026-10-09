package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// lootTally collects everything one looting action takes so it can be told
// in a single line per kind of corpse ("You take 12 gold, 2 wooden shields
// and the leather cap from the skeleton corpses.") instead of a line per item.
type lootTally struct {
	groups []*lootGroup
	left   int // things the company couldn't carry, told once after the take
}

type lootGroup struct {
	color   string
	name    string
	corpses map[*rooms.Corpse]struct{}
	gold    int
	order   []string
	counts  map[string]int
	bases   map[string]string // display name -> spec name, for plurals
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
	g := &lootGroup{color: color, name: c.Character.Name, corpses: map[*rooms.Corpse]struct{}{c: {}}, counts: map[string]int{}, bases: map[string]string{}}
	t.groups = append(t.groups, g)
	return g
}

func (t *lootTally) addGold(c *rooms.Corpse, amt int) {
	t.group(c).gold += amt
}

func (t *lootTally) addItem(c *rooms.Corpse, item items.Item) {
	g := t.group(c)
	name := item.DisplayName()
	if g.counts[name] == 0 {
		g.order = append(g.order, name)
		g.bases[name] = item.Name()
	}
	g.counts[name]++
}

// lootItemPhrase is "the leather cap" for one, "2 wooden shields" for more.
// base is the item's plain name inside its display name (which may carry
// colour, a quest star or a "(cursed)" tag); only base is pluralised.
func lootItemPhrase(name, base string, n int) string {
	if n <= 1 {
		return fmt.Sprintf(`the <ansi fg="itemname">%s</ansi>`, name)
	}
	if i := strings.LastIndex(name, base); base != `` && i >= 0 {
		name = name[:i] + lootPlural(base) + name[i+len(base):]
	} else {
		return fmt.Sprintf(`the <ansi fg="itemname">%s</ansi> (x%d)`, name, n)
	}
	return fmt.Sprintf(`%d <ansi fg="itemname">%s</ansi>`, n, name)
}

// lootPlural pluralises an item name: "bolt of silk" -> "bolts of silk",
// "plate cuirass" -> "plate cuirasses", while names already plural
// ("tattered pants", "iron greaves") stay as they are.
func lootPlural(name string) string {
	if i := strings.Index(name, " of "); i > 0 {
		return lootPlural(name[:i]) + name[i:]
	}
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, "s") && !strings.HasSuffix(lower, "ss") && !strings.HasSuffix(lower, "us") {
		return name
	}
	return mobparty.Plural(name)
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
			p := lootItemPhrase(name, g.bases[name], g.counts[name])
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
	leftBehind(user, t.left)
	t.left = 0
}
