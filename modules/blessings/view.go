package blessings

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/blessings"
	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// panelBlessing is one blessing as the web client draws it.
type panelBlessing struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Text      string `json:"text"`
	Condition string `json:"condition"`
	Perk      string `json:"perk"`
	Iron      bool   `json:"iron,omitempty"`
	Have      int    `json:"have,omitempty"` // progress, for blessings still to earn
	Need      int    `json:"need,omitempty"`
}

// panel is the `Char.Blessings` GMCP message: what this character was
// given, what the account has earned for its next character, and what is
// still to earn.
type panel struct {
	Iron     bool            `json:"iron"`
	Discount int             `json:"discount,omitempty"` // percent off recruits this character has
	Carried  []panelBlessing `json:"carried"`
	Waiting  []panelBlessing `json:"waiting"`
	Next     []panelBlessing `json:"next"`
}

func described(b blessings.Blessing) panelBlessing {
	return panelBlessing{ID: b.ID, Name: b.Name, Text: b.Text, Condition: b.Condition(), Perk: b.PerkText(), Iron: b.Iron, Need: b.Count}
}

// panelFor reads the user's blessings. char may be nil (an unknown
// character has nothing carried).
func (m *Module) panelFor(userID int, u *users.UserRecord) panel {
	p := panel{Carried: []panelBlessing{}, Waiting: []panelBlessing{}, Next: []panelBlessing{}}
	carried := map[string]bool{}
	if u != nil && u.Character != nil {
		p.Iron = u.Character.IsIron()
		p.Discount = blessings.DiscountPercent(u.Character)
		for _, id := range u.Character.Blessings {
			if b, ok := blessings.Get(id); ok {
				p.Carried = append(p.Carried, described(b))
				carried[id] = true
			}
		}
	}
	earned := map[string]bool{}
	for _, id := range m.Earned(userID) {
		earned[id] = true
		if carried[id] {
			continue
		}
		if b, ok := blessings.Get(id); ok {
			p.Waiting = append(p.Waiting, described(b))
		}
	}
	if d := blessings.Current(); d != nil {
		for _, b := range d.Blessings {
			if earned[b.ID] {
				continue
			}
			row := described(b)
			if !b.Iron || p.Iron {
				row.Have = min(chronicle.Total(userID, chronicle.Kind(b.Deed)), b.Count-1)
			}
			p.Next = append(p.Next, row)
		}
	}
	return p
}

func (m *Module) push(userID int) {
	u, online := m.w.User(userID)
	if !online {
		return
	}
	m.w.Push(userID, `Char.Blessings`, m.panelFor(userID, u))
}

func (m *Module) render(userID int, u *users.UserRecord) string {
	p := m.panelFor(userID, u)
	var b strings.Builder
	b.WriteString(`<ansi fg="white-bold">Blessings</ansi> <ansi fg="black-bold">(small perks for your later characters; see</ansi> <ansi fg="command">help blessings</ansi><ansi fg="black-bold">)</ansi>` + "\n")
	if p.Iron {
		b.WriteString(`<ansi fg="yellow-bold">You are an Iron character</ansi> (<ansi fg="command">help hardcore</ansi>): the Iron blessings below can be earned by you.` + "\n")
	}
	section := func(title string, rows []panelBlessing, line func(panelBlessing) string) {
		if len(rows) == 0 {
			return
		}
		b.WriteString("\n" + `<ansi fg="yellow">` + title + `</ansi>` + "\n")
		for _, r := range rows {
			b.WriteString(line(r) + "\n")
		}
	}
	section(`You carry:`, p.Carried, func(r panelBlessing) string {
		return fmt.Sprintf(`  <ansi fg="itemname">%s</ansi>: %s.`, r.Name, r.Perk)
	})
	section(`Earned, and waiting for your next character:`, p.Waiting, func(r panelBlessing) string {
		return fmt.Sprintf(`  <ansi fg="itemname">%s</ansi>: %s.`, r.Name, r.Perk)
	})
	section(`Still to earn:`, p.Next, func(r panelBlessing) string {
		line := fmt.Sprintf(`  <ansi fg="itemname">%s</ansi>: %s`, r.Name, r.Condition)
		if r.Iron && !p.Iron {
			line += ` <ansi fg="black-bold">(Iron characters only)</ansi>`
		} else if r.Need > 1 {
			line += fmt.Sprintf(` <ansi fg="black-bold">(%d of %d)</ansi>`, r.Have, r.Need)
		}
		return line + fmt.Sprintf(`; a later character will %s.`, r.Perk)
	})
	if len(p.Carried)+len(p.Waiting)+len(p.Next) == 0 {
		b.WriteString("\nThis world has no blessings.\n")
	}
	if p.Discount > 0 {
		b.WriteString(fmt.Sprintf("\nYour blessings take %d%% off every recruit's price.\n", p.Discount))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *Module) blessingsCommand(_ string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	user.SendText(m.render(user.UserId, user))
	m.push(user.UserId)
	return true, nil
}
