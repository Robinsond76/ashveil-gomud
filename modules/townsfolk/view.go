package townsfolk

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/townsfolk"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// panelTold is one telling as the web client draws it.
type panelTold struct {
	Ago  string `json:"ago"`
	Text string `json:"text"`
}

// panelFresh is a deed the towns have yet to speak of.
type panelFresh struct {
	Ago  string `json:"ago"`
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// panel is the `Company.Townsfolk` GMCP message: what talkers have told
// the player, the deeds they may yet speak of, and the marks the company
// carries from them.
type panel struct {
	Total int          `json:"total"`
	Told  []panelTold  `json:"told"`
	Fresh []panelFresh `json:"fresh"`
	Marks []string     `json:"marks"`
}

// freshDeeds are the deeds in the chronicle that some deed line could still
// tell (any talker; the NPC's tags decide who actually does): in their
// window and not yet told, newest first.
func (m *Module) freshDeeds(uid int, st UserState, cat townsfolk.Catalog, now int64) []chronicle.Entry {
	var out []chronicle.Entry
	lines := cat.Lines()
	for _, e := range chronicle.Query(uid, chronicle.Filter{Since: now - maxWindow(cat), Limit: queryLimit}) {
		if st.heard(e.Seq) {
			continue
		}
		for _, l := range lines {
			if l.IsDeed() && l.Kind == e.Kind && now-e.At <= l.Window() && (l.Ref == "" || l.Ref == e.Ref) {
				out = append(out, e)
				break
			}
		}
	}
	return out
}

// marks are the flags the catalog's lines leave that the company carries.
func (m *Module) marks(uid int, cat townsfolk.Catalog) []string {
	seen := map[string]bool{}
	var out []string
	for _, l := range cat.Lines() {
		if l.Sets != "" && !seen[l.Sets] && m.w.Flag(uid, l.Sets) {
			seen[l.Sets] = true
			out = append(out, l.Sets)
		}
	}
	return out
}

func (m *Module) panelFor(uid int) panel {
	cat := m.lines()
	m.mu.Lock()
	st := m.state[uid]
	m.mu.Unlock()
	now := m.clock().Unix()
	p := panel{Total: st.Total, Told: []panelTold{}, Fresh: []panelFresh{}, Marks: []string{}}
	for i := len(st.Told) - 1; i >= 0; i-- {
		t := st.Told[i]
		p.Told = append(p.Told, panelTold{Ago: chronicle.Ago(t.At, now), Text: t.Text})
	}
	for _, e := range m.freshDeeds(uid, st, cat, now) {
		p.Fresh = append(p.Fresh, panelFresh{Ago: chronicle.Ago(e.At, now), Kind: string(e.Kind), Text: chronicle.Prose(e)})
	}
	p.Marks = append(p.Marks, m.marks(uid, cat)...)
	return p
}

// push sends a signed-in player's web client the current view.
func (m *Module) push(uid int) {
	if m.w.Name(uid) == "" {
		return
	}
	m.w.Push(uid, "Company.Townsfolk", m.panelFor(uid))
}

// render is the `townsfolk` command's text.
func (m *Module) render(uid int) string {
	p := m.panelFor(uid)
	var b strings.Builder
	b.WriteString(`<ansi fg="yellow-bold">What the towns say of you</ansi>` + "\n")
	if len(p.Told) == 0 && len(p.Fresh) == 0 {
		b.WriteString(`Nobody has spoken of your company yet. Townsfolk mention the company's deeds (bosses slain, relics found, mercy shown) once each, a little while after, when you pass them idle in a town. See <ansi fg="command">help townsfolk</ansi>.`)
		return b.String()
	}
	if len(p.Told) > 0 {
		b.WriteString(`<ansi fg="black-bold">Spoken of:</ansi>` + "\n")
		for _, t := range p.Told {
			fmt.Fprintf(&b, `  <ansi fg="black-bold">%-14s</ansi> "%s"`+"\n", t.Ago, t.Text)
		}
	}
	if len(p.Fresh) > 0 {
		b.WriteString(`<ansi fg="black-bold">Folk may yet speak of:</ansi>` + "\n")
		for _, f := range p.Fresh {
			fmt.Fprintf(&b, `  <ansi fg="black-bold">%-14s</ansi> %s`+"\n", f.Ago, f.Text)
		}
	}
	if len(p.Marks) > 0 {
		b.WriteString(`<ansi fg="black-bold">Marks the towns carry of you: ` + strings.Join(p.Marks, ", ") + ".</ansi>\n")
	}
	fmt.Fprintf(&b, `<ansi fg="black-bold">%d telling(s) so far. A deed is told once; talk fades after a couple of weeks.</ansi>`, p.Total)
	return b.String()
}

func (m *Module) command(_ string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	user.SendText(m.render(user.UserId))
	m.push(user.UserId)
	return true, nil
}
