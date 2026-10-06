package company

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/flasks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 39g: the brew command. An Alchemist's flasks are a count on its
// character (a player's, or a live companion's); brew refills every
// Alchemist's satchel in the company from the leader's reagents, one reagent
// a flask, and shows the satchels. A camp rest does the same at its end
// (modules/camping). Never in the middle of a battle.

const brewUsage = `Use <ansi fg="command">brew</ansi> to refill your company's flask satchels from the reagents you carry, one reagent a flask. See <ansi fg="command">help brew</ansi>.`

func (m *CompanyModule) brewCommand(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	user.SendText(m.runBrew(user, strings.ToLower(strings.TrimSpace(rest))))
	return true, nil
}

// alchemist is an Alchemist found in the leader's company: the leader, or a
// live companion.
type alchemist struct {
	name string
	char *characters.Character
	own  bool
}

// alchemists lists the company's Alchemists, the leader first.
func (m *CompanyModule) alchemists(user *users.UserRecord) []alchemist {
	var out []alchemist
	if flasks.IsAlchemist(user.Character) {
		out = append(out, alchemist{name: "You", char: user.Character, own: true})
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
		if mob := mobs.GetInstance(id); mob != nil && flasks.IsAlchemist(&mob.Character) {
			out = append(out, alchemist{name: nameOf(c, "#"+strconv.Itoa(c.ID)), char: &mob.Character})
		}
	}
	return out
}

func (m *CompanyModule) runBrew(user *users.UserRecord, arg string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	if arg != "" && arg != "status" {
		return brewUsage
	}
	as := m.alchemists(user)
	if len(as) == 0 {
		return "No one in your company throws flasks. Alchemists do: see help alchemist."
	}
	var lines []string
	if arg == "" {
		if inBattle(user) {
			return "Not in the middle of a battle. Once the fighting is done."
		}
		lines = m.brewFlasks(user, as)
	}
	return strings.Join(append(lines, satchelLines(user.Character, as)...), "\n")
}

// brewFlasks refills the Alchemists' satchels from the leader's reagents and
// says what it did.
func (m *CompanyModule) brewFlasks(user *users.UserRecord, as []alchemist) []string {
	chars := make([]*characters.Character, len(as))
	for i, a := range as {
		chars[i] = a.char
	}
	if !flasks.NeedsBrewing(chars...) {
		return []string{"Every satchel is full."}
	}
	if flasks.Reagents(user.Character) == 0 {
		return []string{"You have no reagents to brew from."}
	}
	taken, brewed := flasks.Brew(user.Character, chars)
	for _, itm := range taken {
		events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: itm, Gained: false})
	}
	var lines []string
	for i, b := range brewed {
		if b.Brewed > 0 {
			verb := "brews"
			if as[i].own {
				verb = "brew"
			}
			lines = append(lines, fmt.Sprintf("%s %s %s from %s.", as[i].name, verb, countOf(b.Brewed, "flask"), countOf(b.Brewed, "reagent")))
		}
	}
	if user.UserId > 0 {
		events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})
	}
	return lines
}

// satchelLines show each Alchemist's flasks and the reagents the leader
// carries.
func satchelLines(leader *characters.Character, as []alchemist) []string {
	var lines []string
	for _, a := range as {
		have, size := flasks.Satchel(a.char)
		who := a.name + " carry"
		if !a.own {
			who += "s"
		}
		lines = append(lines, fmt.Sprintf("%s %d of %d flasks.", who, have, size))
	}
	return append(lines, fmt.Sprintf("Reagents carried: %d.", flasks.Reagents(leader)))
}
