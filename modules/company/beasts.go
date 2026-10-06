package company

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/beasts"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 39e: the beast command. A Beast Tamer's bonded beast is a record on
// its character (a player's, or a live companion's); this shows the
// company's beasts and names the player's own. A beast mends with rest, so
// there is nothing to feed or buy here. Never in the middle of a battle.

const beastUsage = `Use <ansi fg="command">beast</ansi> to see your beast, <ansi fg="command">beast [member]</ansi> for a companion's and <ansi fg="command">beast name [name]</ansi> to name yours. See <ansi fg="command">help beast</ansi>.`

func (m *CompanyModule) beastCommand(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	args := util.SplitButRespectQuotes(rest)
	user.SendText(m.runBeast(user, args))
	return true, nil
}

// beastTamer is a Beast Tamer found in the leader's company: the leader, or
// a live companion.
type beastTamer struct {
	name string
	char *characters.Character
	own  bool
}

// beastTamers lists the company's Beast Tamers, the leader first.
func (m *CompanyModule) beastTamers(user *users.UserRecord) []beastTamer {
	var out []beastTamer
	if beasts.IsTamer(user.Character) {
		out = append(out, beastTamer{name: "You", char: user.Character, own: true})
	}
	record, ok := m.registry.Get(user.UserId)
	if !ok {
		return out
	}
	for _, c := range record.Companions {
		id, live := m.instance(user.UserId, c.ID)
		if !live {
			continue
		}
		if mob := mobs.GetInstance(id); mob != nil && beasts.IsTamer(&mob.Character) {
			out = append(out, beastTamer{name: nameOf(c, "#"+strconv.Itoa(c.ID)), char: &mob.Character})
		}
	}
	return out
}

func (m *CompanyModule) runBeast(user *users.UserRecord, args []string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	tamers := m.beastTamers(user)
	if len(tamers) == 0 {
		return "No one in your company keeps a bonded beast. Beast Tamers do: see help beast."
	}
	if len(args) > 0 && strings.ToLower(args[0]) == "name" {
		return m.beastName(user, args[1:])
	}
	if len(args) > 0 {
		sel := strings.ToLower(strings.Join(args, " "))
		for _, bt := range tamers {
			if !bt.own && strings.Contains(strings.ToLower(bt.name), sel) {
				return beastView(bt)
			}
		}
		return fmt.Sprintf("No Beast Tamer answers to %q in your company.\n%s", sel, beastUsage)
	}
	var lines []string
	for _, bt := range tamers {
		lines = append(lines, beastView(bt))
	}
	return strings.Join(lines, "\n")
}

// beastView describes a Tamer's beast: its name, kind, condition and health.
func beastView(bt beastTamer) string {
	g := beasts.GiftsFor(bt.char.ClassEffects())
	rec := bt.char.EnsureBeast(beasts.DefaultName(g.Kind))
	limit := beasts.HealthLimit(bt.char)
	health := max(0, limit-rec.Damage)
	if rec.Wounded {
		health = 0
	}
	who := bt.name + " keep"
	if !bt.own {
		who += "s"
	}
	lines := []string{fmt.Sprintf("%s %s, a %s: %s (%d/%d health).", who, rec.Name, beasts.KindName(g.Kind), beasts.Condition(*rec, limit), health, limit)}
	if rec.Wounded {
		lines = append(lines, "  It is wounded and sits out battles until the company rests.")
	} else if rec.Damage > 0 {
		lines = append(lines, "  A rest brings it back to full health.")
	}
	return strings.Join(lines, "\n")
}

// beastName names the player's own beast.
func (m *CompanyModule) beastName(user *users.UserRecord, args []string) string {
	if _, fighting := battle.Current(user.UserId); fighting {
		return "Not in the middle of a battle."
	}
	c := user.Character
	if !beasts.IsTamer(c) {
		return "Only a Beast Tamer has a bonded beast to name."
	}
	name := strings.TrimSpace(strings.Join(args, " "))
	if name == "" || len(name) > 20 {
		return `Give it a name of one to twenty characters: <ansi fg="command">beast name Ash</ansi>.`
	}
	for _, r := range name {
		if !(r == ' ' || r == '\'' || r == '-' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
			return "A name has letters, spaces, apostrophes and hyphens only."
		}
	}
	g := beasts.GiftsFor(c.ClassEffects())
	c.EnsureBeast(beasts.DefaultName(g.Kind)).Name = name
	return fmt.Sprintf("Your %s answers to %s now.", beasts.KindName(g.Kind), name)
}
