package company

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/dolls"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 39d: the doll command. A Doll Master's dolls are records on its
// character (a player's, or a live companion's); this shows them, dresses
// and names the player's own, and mends them with doll parts. Never in the
// middle of a battle.

const dollUsage = `Use <ansi fg="command">doll</ansi> to see your dolls, <ansi fg="command">doll [member]</ansi> for a companion's, <ansi fg="command">doll wield|wear [number] [item]</ansi> and <ansi fg="command">doll remove [number] [slot]</ansi> to dress one, <ansi fg="command">doll name [number] [name]</ansi> to name one, and <ansi fg="command">doll mend</ansi> to mend them with doll parts. See <ansi fg="command">help doll</ansi>.`

func (m *CompanyModule) dollCommand(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	args := util.SplitButRespectQuotes(strings.ToLower(rest))
	user.SendText(m.runDoll(user, args))
	return true, nil
}

// dollMaster is a Doll Master found in the leader's company: the leader, or
// a live companion.
type dollMaster struct {
	name string
	char *characters.Character
	own  bool
}

// dollMasters lists the company's Doll Masters, the leader first.
func (m *CompanyModule) dollMasters(user *users.UserRecord) []dollMaster {
	var out []dollMaster
	if dolls.IsMaster(user.Character) {
		out = append(out, dollMaster{name: "You", char: user.Character, own: true})
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
		if mob := mobs.GetInstance(id); mob != nil && dolls.IsMaster(&mob.Character) {
			out = append(out, dollMaster{name: nameOf(c, "#"+strconv.Itoa(c.ID)), char: &mob.Character})
		}
	}
	return out
}

func (m *CompanyModule) runDoll(user *users.UserRecord, args []string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	masters := m.dollMasters(user)
	if len(masters) == 0 {
		return "No one in your company drives a doll. Doll Masters do: see help doll."
	}
	if len(args) > 0 {
		switch args[0] {
		case "wield", "wear", "remove", "name", "mend":
			return m.dollAction(user, args)
		}
		// A companion's dolls.
		sel := strings.Join(args, " ")
		for _, dm := range masters {
			if !dm.own && strings.Contains(strings.ToLower(dm.name), sel) {
				return dollView(dm)
			}
		}
		return fmt.Sprintf("No Doll Master answers to %q in your company.\n%s", sel, dollUsage)
	}
	var lines []string
	for _, dm := range masters {
		lines = append(lines, dollView(dm))
	}
	lines = append(lines, fmt.Sprintf("Doll parts carried: %d.", dolls.Parts(user.Character)))
	return strings.Join(lines, "\n")
}

// dollView lists a Master's dolls.
func dollView(dm dollMaster) string {
	g := dolls.GiftsFor(dm.char.ClassEffects())
	dm.char.EnsureDolls(g.Count)
	limit := dolls.HealthLimit(dm.char) // a whole doll's health
	who := dm.name + " drive"
	if !dm.own {
		who += "s"
	}
	lines := []string{fmt.Sprintf("%s %s:", who, countOf(g.Count, "doll"))}
	for i := 0; i < g.Count && i < len(dm.char.Dolls); i++ {
		d := dm.char.Dolls[i]
		health := limit - d.Damage
		if d.Broken {
			health = 0
		}
		lines = append(lines, fmt.Sprintf("  %d. %s, %s (%d/%d health)", i+1, d.Name, dolls.Condition(d, limit), max(0, health), limit))
		var worn []string
		for _, slot := range characters.AllSlots() {
			if itm := d.Equipment.Get(slot); itm != nil && itm.ItemId > 0 {
				worn = append(worn, fmt.Sprintf("%s %s", strings.TrimSuffix(characters.SlotLabel(slot), ":"), itemName(*itm)))
			}
		}
		if g.NoWear {
			worn = worn[:0]
			if d.Equipment.Weapon.ItemId > 0 {
				worn = append(worn, "Weapon "+itemName(d.Equipment.Weapon))
			}
			worn = append(worn, "its own body is its armor")
		}
		if len(worn) == 0 {
			worn = []string{"nothing"}
		}
		lines = append(lines, "     Wearing: "+strings.Join(worn, ", "))
	}
	return strings.Join(lines, "\n")
}

func countOf(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// dollAction handles the commands that change a doll; they act on the
// player's own dolls.
func (m *CompanyModule) dollAction(user *users.UserRecord, args []string) string {
	c := user.Character
	if inBattle(user) {
		return "Not in the middle of a battle. Once the fighting is done."
	}
	cmd, rest := args[0], args[1:]
	if cmd == "mend" {
		return m.mendDolls(user)
	}
	if !dolls.IsMaster(c) {
		return "Only a Doll Master has a doll to dress or name."
	}
	g := dolls.GiftsFor(c.ClassEffects())
	c.EnsureDolls(g.Count)
	idx := 0
	if len(rest) > 0 {
		if n, err := strconv.Atoi(rest[0]); err == nil {
			if n < 1 || n > g.Count {
				return fmt.Sprintf("You have %s; pick 1 to %d.", countOf(g.Count, "doll"), g.Count)
			}
			idx, rest = n-1, rest[1:]
		}
	}
	d := &c.Dolls[idx]
	if len(rest) == 0 {
		return dollUsage
	}
	switch cmd {
	case "name":
		name := strings.TrimSpace(strings.Join(rest, " "))
		if len(name) < 2 || len(name) > 20 {
			return "A doll's name is 2 to 20 letters."
		}
		d.Name = util.CapitalizeFirst(name)
		return fmt.Sprintf("You name the doll %s.", d.Name)
	case "wield", "wear":
		return dollWear(user, d, g, strings.Join(rest, " "))
	case "remove":
		return dollRemove(user, d, strings.Join(rest, " "))
	}
	return dollUsage
}

func inBattle(user *users.UserRecord) bool {
	_, ok := battle.Current(user.UserId)
	return ok
}

// dollWear puts a carried item on a doll: a weapon in its hand, armor on its
// body. What it wore comes back to the pack.
func dollWear(user *users.UserRecord, d *characters.DollState, g dolls.Gifts, name string) string {
	itm, ok := user.Character.FindInBackpack(name)
	if !ok {
		return fmt.Sprintf("You carry no %q.", name)
	}
	spec := itm.GetSpec()
	slot := spec.Type
	valid := false
	for _, s := range characters.AllSlots() {
		if s == slot && s != items.Pack {
			valid = true
		}
	}
	if !valid {
		return fmt.Sprintf("A doll can't wear %s.", itm.DisplayName())
	}
	if g.NoWear && slot != items.Weapon {
		return "A golem's body is its own armor; it can only hold a weapon."
	}
	if slot == items.Weapon && spec.Hands >= 2 && d.Equipment.Offhand.ItemId > 0 {
		return fmt.Sprintf("%s needs both hands; take the doll's offhand item off first.", util.CapitalizeFirst(itm.DisplayName()))
	}
	if slot == items.Offhand && d.Equipment.Weapon.ItemId > 0 && d.Equipment.Weapon.GetSpec().Hands >= 2 {
		return "Its weapon needs both hands."
	}
	user.Character.RemoveItem(itm)
	events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: itm, Gained: false})
	var back items.Item
	if old := d.Equipment.Get(slot); old != nil && old.ItemId > 0 {
		back = *old
	}
	d.Equipment.Set(slot, itm)
	line := fmt.Sprintf("%s takes %s.", d.Name, itm.DisplayName())
	if back.ItemId > 0 {
		user.Character.StoreItem(back)
		events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: back, Gained: true})
		line += fmt.Sprintf(" You put %s back in your pack.", back.DisplayName())
	}
	return line
}

// dollRemove takes an item off a doll, by slot name or item name.
func dollRemove(user *users.UserRecord, d *characters.DollState, what string) string {
	for _, slot := range characters.AllSlots() {
		itm := d.Equipment.Get(slot)
		if itm == nil || itm.ItemId < 1 {
			continue
		}
		label := strings.ToLower(strings.TrimSuffix(characters.SlotLabel(slot), ":"))
		if label != what && !strings.Contains(strings.ToLower(itm.DisplayName()), what) {
			continue
		}
		taken := *itm
		d.Equipment.Set(slot, items.Item{})
		user.Character.StoreItem(taken)
		events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: taken, Gained: true})
		return fmt.Sprintf("You take %s off %s.", taken.DisplayName(), d.Name)
	}
	return fmt.Sprintf("%s wears nothing like that.", d.Name)
}

// mendDolls mends the company's dolls with the leader's doll parts.
func (m *CompanyModule) mendDolls(user *users.UserRecord) string {
	if inBattle(user) {
		return "Not in the middle of a battle. Once the fighting is done."
	}
	masters := m.dollMasters(user)
	chars := make([]*characters.Character, len(masters))
	for i, dm := range masters {
		chars[i] = dm.char
	}
	if dolls.Parts(user.Character) == 0 {
		for _, c := range chars {
			if dolls.NeedsMending(c) {
				return "You have no doll parts."
			}
		}
		return "Your dolls need no mending."
	}
	taken, used := dolls.MendCompany(user.Character, chars)
	for _, itm := range taken {
		events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: itm, Gained: false})
	}
	var lines []string
	for i, n := range used {
		if n > 0 {
			lines = append(lines, fmt.Sprintf("%s mend%s the dolls with %s (each mends %d health).", masters[i].name, verbS(masters[i].own), countOf(n, "doll part"), dolls.MendPerPart(chars[i].Level)))
		}
	}
	if len(lines) == 0 {
		return "Your dolls need no mending."
	}
	return strings.Join(lines, "\n")
}

func verbS(own bool) string {
	if own {
		return ""
	}
	return "s"
}
