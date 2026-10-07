package strategy

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/orders"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	domain "github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 61: battle orders. `orders` shows every member's, `orders [who]`
// one's, and `orders [who] add|remove|up|clear|preset` changes them, outside
// a battle only.

const ordersUsage = `Add one with <ansi fg="command">orders [who] add [condition] then [action]</ansi>, for example <ansi fg="command">orders [who] add ally 50 then heal</ansi>; take one off with <ansi fg="command">orders [who] remove [number]</ansi>, reorder with <ansi fg="command">orders [who] up [number]</ansi>, or load a class's starting set with <ansi fg="command">orders [who] preset</ansi>. See <ansi fg="command">help orders</ansi>.`

func (m *StrategyModule) ordersCommand(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	user.SendText(m.runOrders(user, strings.Fields(strings.ToLower(rest))))
	return true, nil
}

func (m *StrategyModule) runOrders(user *users.UserRecord, args []string) string {
	members, ok := m.env.members(user)
	if ok {
		keep := map[string]bool{}
		for _, mb := range members {
			keep[mb.key] = true
		}
		m.prune(user.UserId, keep)
	}
	if len(args) == 0 {
		return m.ordersList(user.UserId, members)
	}
	mb, found := resolve(members, args[0])
	if !found {
		return fmt.Sprintf(`No one in your company answers to "%s". Type <ansi fg="command">orders</ansi> to see them.`, args[0])
	}
	if len(args) == 1 {
		return m.ordersOf(user.UserId, mb)
	}
	if m.env.inBattle(user) {
		return usercommands.BattleUnderWay
	}
	list := m.StoredOrders(user.UserId, mb.key)
	switch args[1] {
	case "add", "new":
		if len(list) >= orders.MaxOrders {
			return fmt.Sprintf(`%s already %s %d orders, the most there are. Take one off first with <ansi fg="command">orders [who] remove [number]</ansi>.`, mb.name, verb(mb, "carry", "carries"), orders.MaxOrders)
		}
		o, err := orders.Parse(args[2:])
		if err != nil {
			return fmt.Sprintf(`That is not an order: %s. %s`, err.Error(), ordersMenu())
		}
		for i, had := range list {
			if had == o {
				return fmt.Sprintf("%s already %s that order (number %d).", mb.name, verb(mb, "carry", "carries"), i+1)
			}
		}
		next := append(list, o)
		if err := m.setOrders(user.UserId, mb.key, next); err != nil {
			return err.Error()
		}
		out := fmt.Sprintf("Order %d for %s: %s", len(next), mb.name, o.Describe())
		if warn := m.cantCarryOut(mb, o); warn != "" {
			out += "\n" + warn
		}
		return out
	case "remove", "delete", "drop":
		n, ok := orderNumber(args[2:], len(list))
		if !ok {
			return fmt.Sprintf(`Take off which order? A number from 1 to %d, as <ansi fg="command">orders [who]</ansi> lists them.`, max(len(list), 1))
		}
		gone := list[n-1]
		next := append(append([]orders.Order(nil), list[:n-1]...), list[n:]...)
		if err := m.setOrders(user.UserId, mb.key, next); err != nil {
			return err.Error()
		}
		return fmt.Sprintf("Took off %s's order: %s", mb.name, gone.Describe())
	case "up":
		n, ok := orderNumber(args[2:], len(list))
		if !ok || n < 2 {
			return `Move which order up? A number from 2 up, as <ansi fg="command">orders [who]</ansi> lists them: the first one is already read first.`
		}
		next := append([]orders.Order(nil), list...)
		next[n-1], next[n-2] = next[n-2], next[n-1]
		if err := m.setOrders(user.UserId, mb.key, next); err != nil {
			return err.Error()
		}
		return fmt.Sprintf("%s's order %d is now read before order %d.", mb.name, n, n-1)
	case "clear", "none", "reset":
		if len(list) == 0 {
			return fmt.Sprintf("%s %s no orders.", mb.name, verb(mb, "have", "has"))
		}
		if err := m.setOrders(user.UserId, mb.key, nil); err != nil {
			return err.Error()
		}
		return fmt.Sprintf("%s %s no orders now: only the strategy.", mb.name, verb(mb, "have", "has"))
	case "preset", "default":
		next := orders.Preset(mb.archetype)
		if err := m.setOrders(user.UserId, mb.key, next); err != nil {
			return err.Error()
		}
		return fmt.Sprintf("%s's orders are now the starting set for %s:\n%s", mb.name, archLabel(mb), numbered(next))
	}
	return fmt.Sprintf(`"%s" is not something to do with orders. %s`, args[1], ordersUsage)
}

func archLabel(mb member) string {
	if mb.archetype == "" {
		return "anyone"
	}
	return mb.archetype
}

func orderNumber(args []string, count int) (int, bool) {
	if len(args) != 1 {
		return 0, false
	}
	n, err := strconv.Atoi(args[0])
	return n, err == nil && n >= 1 && n <= count
}

func numbered(list []orders.Order) string {
	var b strings.Builder
	for i, o := range list {
		fmt.Fprintf(&b, "  %d. %s\n", i+1, o.Describe())
	}
	return strings.TrimRight(b.String(), "\n")
}

func ordersMenu() string {
	var dos []string
	for _, d := range orders.Dos {
		dos = append(dos, string(d))
	}
	return fmt.Sprintf(`Conditions: ally [n] (an ally below n%% health), self [n], chanting, boss, first, foe [%s]. Actions: %s. A health share is a multiple of 5 from %d to %d. Heal and guard need an ally or self condition, break a foe condition (chanting, boss or foe).`,
		strings.Join(orders.Kinds, "|"), strings.Join(dos, ", "), orders.MinPct, orders.MaxPct)
}

// cantCarryOut warns when a member has nothing to carry an order out with: a
// heal with no healing spell, an attack with no attack spell.
func (m *StrategyModule) cantCarryOut(mb member, o orders.Order) string {
	has := func(uses ...domain.Use) bool {
		for _, sp := range m.AutoSpells() {
			for _, u := range uses {
				if sp.Use == u && mb.knows != nil && mb.knows(sp.ID) {
					return true
				}
			}
		}
		return false
	}
	switch o.Do {
	case orders.Heal:
		if !has(domain.UseHeal, domain.UseBigHeal, domain.UseRejuv) {
			return fmt.Sprintf(`%s %s no healing spell yet, so this order waits until %s learns one.`, mb.name, verb(mb, "know", "knows"), verb(mb, "you", "they"))
		}
	case orders.Strongest, orders.Hold:
		if !has(domain.UseAttack, domain.UseAttackAll, domain.UseBurst, domain.UseStorm) {
			return fmt.Sprintf(`%s %s no attack spell yet, so this order waits until %s learns one.`, mb.name, verb(mb, "know", "knows"), verb(mb, "you", "they"))
		}
	}
	return ""
}

func (m *StrategyModule) ordersList(userID int, members []member) string {
	var b strings.Builder
	b.WriteString("Battle orders (read each round, in order, before a member's strategy):\n")
	any := false
	for _, mb := range members {
		list := m.StoredOrders(userID, mb.key)
		if len(list) == 0 {
			continue
		}
		any = true
		fmt.Fprintf(&b, "%s:\n%s\n", mb.name, numbered(list))
	}
	if !any {
		b.WriteString("  None set: everyone fights by their strategy alone.\n")
	}
	b.WriteString(ordersUsage)
	return b.String()
}

func (m *StrategyModule) ordersOf(userID int, mb member) string {
	list := m.StoredOrders(userID, mb.key)
	var b strings.Builder
	if len(list) == 0 {
		fmt.Fprintf(&b, "%s %s no orders: fights by strategy alone.\n", mb.name, verb(mb, "have", "has"))
	} else {
		fmt.Fprintf(&b, "%s's orders, read in this order each round:\n%s\n", mb.name, numbered(list))
	}
	b.WriteString(ordersMenu() + "\n")
	b.WriteString(ordersUsage)
	return b.String()
}
