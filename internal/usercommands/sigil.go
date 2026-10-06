package usercommands

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/sigils"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// sigilNow is the clock sigils are laid and faded by; tests replace it.
var sigilNow = time.Now

// UseSigilClockForTest replaces the clock until the returned restore is called.
func UseSigilClockForTest(now func() time.Time) (restore func()) {
	prev := sigilNow
	sigilNow = now
	return func() { sigilNow = prev }
}

// sigilsList is `cast sigil` with no or an unknown kind: what can be laid and
// what this company's sigil is doing now.
func sigilsList(user *users.UserRecord) string {
	var b strings.Builder
	b.WriteString(`Lay a sigil with <ansi fg="command">cast sigil of [kind]</ansi> (<ansi fg="command">help sigils</ansi>). Each costs mana and one sigil chalk, and lasts ` + fmt.Sprint(sigils.Minutes) + ` minutes:` + "\n")
	for _, k := range sigils.Kinds {
		fmt.Fprintf(&b, "  %-10s %2d mana: %s\n", string(k), k.ManaCost(), k.Effect())
	}
	if l := user.Character.Sigil; l.Live(sigilNow()) {
		fmt.Fprintf(&b, "Your company's %s lasts %d more minutes, in room %d.", l.Kind.Name(), l.MinutesLeft(sigilNow()), l.RoomId)
	} else {
		b.WriteString("Your company has no sigil laid.")
	}
	return b.String()
}

// castSigil lays a sigil (Phase 54, help sigils). words is what followed
// `cast sigil`. A sigil is laid before a fight: nothing is cast in one.
func castSigil(words string, user *users.UserRecord, room *rooms.Room) {
	words = strings.TrimSpace(words)
	if _, inBattle := battle.Current(user.UserId); inBattle || fightingMob(user) {
		user.SendText(BattleUnderWay)
		return
	}
	kind, ok := sigils.Parse(words)
	if !ok {
		user.SendText(sigilsList(user))
		return
	}
	now := sigilNow()
	if l := user.Character.Sigil; l.In(room.RoomId, now) {
		user.SendText(fmt.Sprintf(`Your company's %s already lies here, with %d minutes left. A company keeps one sigil to a room.`, l.Kind.Name(), l.MinutesLeft(now)))
		return
	}
	cost := kind.ManaCost()
	if user.Character.Mana < cost {
		user.SendText(fmt.Sprintf(`You don't have enough mana to lay the %s (%d needed).`, kind.Name(), cost))
		return
	}
	var chalk items.Item
	for _, it := range user.Character.GetAllBackpackItems() {
		if it.ItemId == sigils.ChalkItemID {
			chalk = it
			break
		}
	}
	if chalk.ItemId == 0 {
		user.SendText(`You have no sigil chalk to draw with. A market sells it (<ansi fg="command">help sigils</ansi>).`)
		return
	}
	if !user.Character.RemoveItem(chalk) {
		return
	}
	user.Character.Mana -= cost
	events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})
	elsewhere := user.Character.Sigil.Live(now)
	user.Character.Sigil = sigils.Lay(kind, room.RoomId, now)
	user.SendText(fmt.Sprintf(`You kneel and draw a %s in chalk, whispering over it. It will hold for %d minutes. (%s)`, kind.Name(), sigils.Minutes, kind.Effect()))
	if elsewhere {
		user.SendText(`The sigil you laid before fades.`)
	}
	events.AddToQueue(events.RoomResourcesChanged{RoomId: room.RoomId}) // resends Room.Info with the sigil
	room.SendText(fmt.Sprintf(`%s kneels and draws a %s in chalk on the ground.`, user.Character.Name, kind.Name()), user.UserId)
}

// sigilLines is what look shows for the sigils lit in a room: one line each.
// The viewer's own reads "Your", with its minutes left.
func sigilLines(roomId, viewerUserId int) []string {
	var lines []string
	for _, sg := range users.SigilsIn(roomId, sigilNow()) {
		whose := sg.Owner + "'s"
		if sg.UserId == viewerUserId {
			whose = "your company's"
		}
		lines = append(lines, fmt.Sprintf("%s: %s %s, %d min left.", util.CapitalizeFirst(sg.Kind.Glow()), whose, sg.Kind.Name(), sg.Minutes))
	}
	return lines
}
