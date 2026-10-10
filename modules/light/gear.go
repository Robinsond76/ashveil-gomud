package light

// Light gear (living map design, phase LG; owner decisions 2026-10-10):
//
//   - A torch is a one-time consumable. `light torch` uses one up and gives
//     its bearer light (the Torchlight buff) for about 15 real minutes, six
//     game hours. It can't be put out.
//   - A lantern (item 20036) burns oil. While lit and held in the off hand it
//     gives light (the Lantern light buff) and burns an hour of oil each game
//     hour; `light lantern` and `douse lantern` turn it on and off. Stowed or
//     dry, it goes out. A flask of lamp oil refills it: `fill lantern`.
//
// A lantern's oil and lit state live on the item, so they persist with the
// character and survive a restart:
//
//   - Uses is its oil plus one: a full lantern's spec uses (25) is 24 hours
//     of oil, and 1 is empty. Nothing about a lantern is ever 0 or less,
//     which items.Validate would refill and cargo would stack as full.
//   - Lit says it is lit, and LastUsedRound is the round its current hour of
//     burning began, counted from the shared round counter, which this
//     never advances.

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const (
	lanternItemID = 20036
	torchItemID   = 303
	lampOilItemID = 304
	lanternBuffID = 1001
	torchBuffID   = 1002
	// lanternFullUses is a full lantern's uses when its spec says none:
	// 24 hours of oil, plus one.
	lanternFullUses = 25
)

// fullUses is a full lantern's uses: its spec's, else 25.
func fullUses() int {
	if spec := items.GetItemSpec(lanternItemID); spec != nil && spec.Uses > 1 {
		return spec.Uses
	}
	return lanternFullUses
}

// oil is a lantern's hours of oil.
func oil(l items.Item) int {
	if l.Uses <= 1 {
		return 0
	}
	return l.Uses - 1
}

// perHour is the rounds in a game hour.
func perHour(roundsPerDay int) uint64 {
	n := uint64(roundsPerDay / 24)
	if n < 1 {
		n = 1
	}
	return n
}

// burn is how many hours of oil a lit lantern burns between lastRound and
// round, and the round its next hour counts from. The round tick runs every
// round its holder is online, so a gap of more than two hours means the
// lantern was not burning in between (its holder was away, or it lay
// somewhere out of hand): that time is not charged, and it starts afresh.
func burn(lastRound, round uint64, roundsPerDay int) (hours int, next uint64) {
	h := perHour(roundsPerDay)
	if lastRound == 0 || lastRound > round || round-lastRound > 2*h {
		return 0, round
	}
	hours = int((round - lastRound) / h)
	return hours, lastRound + uint64(hours)*h
}

// tickLantern burns a user's held lantern and keeps their Lantern light
// buff in step with it. It returns the line to tell them, if any.
func tickLantern(c lanternHolder, round uint64, roundsPerDay int) string {
	msg := ""
	lantern := c.offhand()
	if lantern.ItemId == lanternItemID && lantern.Lit {
		hours, next := burn(lantern.LastUsedRound, round, roundsPerDay)
		lantern.LastUsedRound = next
		lantern.Uses -= hours
		if lantern.Uses <= 1 {
			lantern.Uses = 1
			lantern.Lit = false
			msg = `Your <ansi fg="itemname">lantern</ansi> gutters and goes out: it needs oil (<ansi fg="command">fill lantern</ansi>).`
		}
		c.setOffhand(lantern)
	}
	// A lit lantern stowed in the pack goes out.
	c.dousePacked()
	lit := lantern.ItemId == lanternItemID && lantern.Lit && oil(lantern) > 0
	switch has := c.hasBuff(lanternBuffID); {
	case lit && !has:
		c.addBuff(lanternBuffID, buffs.TriggersLeftUnlimited)
	case !lit && has:
		c.removeBuff(lanternBuffID)
	}
	return msg
}

// lanternHolder is what the lantern rules need of a character, so they can
// be tested without a game.
type lanternHolder interface {
	offhand() items.Item
	setOffhand(items.Item)
	dousePacked()
	hasBuff(int) bool
	addBuff(id, triggers int)
	removeBuff(int)
}

type userHolder struct{ u *users.UserRecord }

func (h userHolder) offhand() items.Item      { return h.u.Character.Equipment.Offhand }
func (h userHolder) setOffhand(i items.Item)  { h.u.Character.Equipment.Offhand = i }
func (h userHolder) hasBuff(id int) bool      { return liveBuff(h.u.Character.GetBuffs(id)) }
func (h userHolder) addBuff(id, triggers int) { _ = h.u.Character.AddBuff(id, false, triggers) }
func (h userHolder) removeBuff(id int)        { h.u.Character.RemoveBuff(id) }
func (h userHolder) dousePacked() {
	for i := range h.u.Character.Items {
		if h.u.Character.Items[i].ItemId == lanternItemID && h.u.Character.Items[i].Lit {
			h.u.Character.Items[i].Lit = false
		}
	}
}

// liveBuff reports a buff that is on and not expired: RemoveBuff only marks
// a buff expired until the next prune.
func liveBuff(list []*buffs.Buff) bool {
	for _, b := range list {
		if b != nil && !b.Expired() {
			return true
		}
	}
	return false
}

// onNewRound burns every online player's lit lantern.
func (m *LightModule) onNewRound(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.NewRound)
	if !ok {
		return events.Continue
	}
	perDay := gametime.GetDate().RoundsPerDay
	for _, u := range users.GetAllActiveUsers() {
		if u == nil || u.Character == nil {
			continue
		}
		if msg := tickLantern(userHolder{u}, evt.RoundNumber, perDay); msg != "" {
			u.SendText(msg)
		}
	}
	return events.Continue
}

func isLantern(what string) bool { return strings.HasPrefix("lantern", what) || what == "lamp" }

// lightThing is `light lantern` and `light torch`.
func lightThing(what string, user *users.UserRecord, room *rooms.Room) {
	c := user.Character
	switch {
	case isLantern(what):
		lantern := c.Equipment.Offhand
		if lantern.ItemId != lanternItemID {
			if _, ok := findItem(c.Items, lanternItemID); ok {
				user.SendText(`Hold your lantern first (<ansi fg="command">wear lantern</ansi>): it lights only in your off hand.`)
			} else {
				user.SendText(`You have no lantern.`)
			}
			return
		}
		if lantern.Lit {
			user.SendText(`Your lantern is already lit.`)
			return
		}
		if oil(lantern) <= 0 {
			user.SendText(`Your lantern is out of oil. Fill it with a flask of lamp oil (<ansi fg="command">fill lantern</ansi>).`)
			return
		}
		lantern.Lit = true
		lantern.LastUsedRound = util.GetRoundCount()
		c.Equipment.Offhand = lantern
		_ = c.AddBuff(lanternBuffID, false, buffs.TriggersLeftUnlimited)
		user.SendText(fmt.Sprintf(`You light your <ansi fg="itemname">lantern</ansi>. It has oil for about %s.`, hoursText(oil(lantern))))
		room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> lights a lantern.`, c.Name), user.UserId)
	case strings.HasPrefix("torch", what):
		torch, ok := findItem(c.Items, torchItemID)
		if !ok {
			user.SendText(`You have no torch.`)
			return
		}
		if liveBuff(c.GetBuffs(torchBuffID)) {
			user.SendText(`Your torch is still burning; light another when it is spent.`)
			return
		}
		c.RemoveItem(torch)
		events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: torch, Gained: false})
		_ = c.AddBuff(torchBuffID, false)
		user.SendText(`You light a <ansi fg="itemname">torch</ansi>. It will burn for about six hours, and can't be put out.`)
		room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> lights a torch.`, c.Name), user.UserId)
	default:
		user.SendText(`Light what? <ansi fg="command">light lantern</ansi> or <ansi fg="command">light torch</ansi>; <ansi fg="command">light</ansi> alone shows how well you can see.`)
	}
}

// douse is `douse` / `douse lantern`. The hour it was burning in is spent,
// so dousing and relighting never saves oil.
func (m *LightModule) douse(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	what := strings.ToLower(strings.TrimSpace(rest))
	c := user.Character
	if what != "" && strings.HasPrefix("torch", what) {
		user.SendText(`A torch burns until it's spent; it can't be put out.`)
		return true, nil
	}
	if what != "" && !isLantern(what) {
		user.SendText(`Douse what? <ansi fg="command">douse lantern</ansi>.`)
		return true, nil
	}
	lantern := c.Equipment.Offhand
	if lantern.ItemId != lanternItemID || !lantern.Lit {
		user.SendText(`You have no lit lantern to douse.`)
		return true, nil
	}
	lantern = douseCharge(lantern, util.GetRoundCount(), gametime.GetDate().RoundsPerDay)
	c.Equipment.Offhand = lantern
	c.RemoveBuff(lanternBuffID)
	user.SendText(fmt.Sprintf(`You douse your <ansi fg="itemname">lantern</ansi>. It has oil left for about %s.`, hoursText(oil(lantern))))
	room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> douses a lantern.`, c.Name), user.UserId)
	return true, nil
}

// douseCharge puts a lantern out, charging the whole hours burned and the
// hour it had started.
func douseCharge(l items.Item, round uint64, roundsPerDay int) items.Item {
	hours, next := burn(l.LastUsedRound, round, roundsPerDay)
	if l.LastUsedRound != 0 && round > next && round-next <= 2*perHour(roundsPerDay) {
		hours++ // the hour under way
	}
	l.Uses -= hours
	if l.Uses < 1 {
		l.Uses = 1
	}
	l.Lit = false
	l.LastUsedRound = 0
	return l
}

// fillLantern handles `fill lantern` for the core fill command; anything
// else is left to water. It fills the held lantern, else the emptiest one
// in the pack.
func fillLantern(rest string, user *users.UserRecord) bool {
	what := strings.ToLower(rest)
	if what == "" || !isLantern(what) {
		return false
	}
	c := user.Character
	full := fullUses()
	held := c.Equipment.Offhand.ItemId == lanternItemID
	var lantern items.Item
	found := held
	if held {
		lantern = c.Equipment.Offhand
	} else {
		for _, itm := range c.Items {
			if itm.ItemId == lanternItemID && (!found || itm.Uses < lantern.Uses) {
				lantern, found = itm, true
			}
		}
	}
	if !found {
		user.SendText(`You have no lantern.`)
		return true
	}
	if lantern.Uses >= full {
		user.SendText(`Your lantern is already full of oil.`)
		return true
	}
	flask, ok := findItem(c.Items, lampOilItemID)
	if !ok {
		user.SendText(`You have no lamp oil. The markets that sell lanterns sell flasks of it.`)
		return true
	}
	if c.UseItem(flask) < 1 {
		events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: flask, Gained: false})
	}
	lantern.Uses = full
	if held {
		c.Equipment.Offhand = lantern
	} else {
		for i := range c.Items {
			if c.Items[i].Equals(lantern) {
				c.Items[i] = lantern
			}
		}
	}
	user.SendText(fmt.Sprintf(`You fill your <ansi fg="itemname">lantern</ansi> with lamp oil: enough for about %s.`, hoursText(oil(lantern))))
	return true
}

func findItem(list []items.Item, itemID int) (items.Item, bool) {
	for _, itm := range list {
		if itm.ItemId == itemID {
			return itm, true
		}
	}
	return items.Item{}, false
}

// hoursText says game hours of oil in words.
func hoursText(hours int) string {
	if hours == 1 {
		return "one hour"
	}
	return fmt.Sprintf("%d hours", hours)
}

// lanternLine is what `light` says of the player's lantern.
func lanternLine(c lanternHolder) string {
	l := c.offhand()
	if l.ItemId != lanternItemID {
		return ""
	}
	switch {
	case l.Lit:
		return fmt.Sprintf("Your lantern is lit, with oil for about %s.", hoursText(oil(l)))
	case oil(l) <= 0:
		return "Your lantern is out of oil."
	default:
		return fmt.Sprintf("Your lantern is dark, with oil for about %s (light lantern).", hoursText(oil(l)))
	}
}
