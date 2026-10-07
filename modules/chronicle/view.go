package chronicle

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

const (
	pageSize = 12
	// panelEntries is how many of the newest deeds the web tab holds.
	panelEntries = 60
)

// panelEntry is one deed as the web client draws it.
type panelEntry struct {
	Seq   int    `json:"seq"`
	At    int64  `json:"at"`
	Ago   string `json:"ago"`
	Kind  string `json:"kind"`
	Label string `json:"label"`
	Text  string `json:"text"`
}

// panel is the `Company.Chronicle` GMCP message: the newest deeds, the
// lifetime count of each kind, and the kinds' names for the filter.
type panel struct {
	Total   int            `json:"total"`
	Tally   map[string]int `json:"tally"`
	Entries []panelEntry   `json:"entries"`
}

func labelOf(k chronicle.Kind) string {
	for _, i := range chronicle.Kinds {
		if i.Kind == k {
			return i.Label
		}
	}
	return string(k)
}

func (m *Module) panelFor(userID int) panel {
	l := m.Log(userID)
	now := m.clock().Unix()
	p := panel{Tally: map[string]int{}, Entries: []panelEntry{}}
	for k, n := range l.Tally {
		p.Tally[string(k)] = n
		p.Total += n
	}
	for _, e := range l.Query(chronicle.Filter{Limit: panelEntries}) {
		p.Entries = append(p.Entries, panelEntry{
			Seq: e.Seq, At: e.At, Ago: chronicle.Ago(e.At, now),
			Kind: string(e.Kind), Label: labelOf(e.Kind), Text: chronicle.Prose(e),
		})
	}
	return p
}

// push sends a signed-in leader's web client the current chronicle.
func (m *Module) push(userID int) {
	if !m.w.Online(userID) {
		return
	}
	m.w.Push(userID, "Company.Chronicle", m.panelFor(userID))
}

// request is what `chronicle` was asked for.
type request struct {
	kind  chronicle.Kind // "" for every kind
	page  int
	all   bool
	error string
}

func parseRequest(rest string) request {
	r := request{page: 1}
	for _, w := range strings.Fields(strings.ToLower(rest)) {
		if w == "all" {
			r.all = true
			continue
		}
		if n, err := strconv.Atoi(w); err == nil && n >= 1 {
			r.page = n
			continue
		}
		if k, ok := chronicle.KindByWord(w); ok {
			r.kind = k
			continue
		}
		r.error = fmt.Sprintf(`The chronicle has no deeds called "%s".`, w)
	}
	return r
}

func kindWords() string {
	var words []string
	for _, k := range chronicle.Kinds {
		words = append(words, k.Words[0])
	}
	return strings.Join(words, ", ")
}

// render is the `chronicle` command's text for a leader, plain apart from
// the colour tags the game uses.
func (m *Module) render(userID int, rest string) string {
	r := parseRequest(rest)
	if r.error != "" {
		return r.error + ` Try: ` + kindWords() + ".\nUsage: chronicle [kind] [page] | chronicle all"
	}
	l := m.Log(userID)
	f := chronicle.Filter{}
	if r.kind != "" {
		f.Kinds = []chronicle.Kind{r.kind}
	}
	all := l.Query(f)
	if len(all) == 0 {
		if len(l.Entries) == 0 {
			return `Your chronicle is empty. Deeds are written down as the company does them: a recruit, a fallen friend, a boss slain, a relic found. See <ansi fg="command">help chronicle</ansi>.`
		}
		return fmt.Sprintf("Nothing of that kind is in your chronicle yet. Type <ansi fg=\"command\">chronicle</ansi> to read all %d.", len(l.Entries))
	}
	pages := (len(all) + pageSize - 1) / pageSize
	shown := all
	if !r.all {
		if r.page > pages {
			r.page = pages
		}
		lo := (r.page - 1) * pageSize
		hi := min(lo+pageSize, len(all))
		shown = all[lo:hi]
	}
	now := m.clock().Unix()
	var b strings.Builder
	title := "The company's chronicle"
	if r.kind != "" {
		title += " (" + strings.ToLower(labelOf(r.kind)) + ")"
	}
	b.WriteString(`<ansi fg="yellow-bold">` + title + "</ansi>\n")
	for _, e := range shown {
		fmt.Fprintf(&b, `  <ansi fg="black-bold">%-14s</ansi> %s`+"\n", chronicle.Ago(e.At, now), chronicle.Prose(e))
	}
	if r.all || pages == 1 {
		fmt.Fprintf(&b, `<ansi fg="black-bold">%d deed(s) kept, newest first.</ansi>`, len(all))
	} else {
		more := ""
		if r.kind != "" {
			more = " " + firstWord(r.kind)
		}
		fmt.Fprintf(&b, `<ansi fg="black-bold">Page %d of %d, newest first. </ansi><ansi fg="command">chronicle%s %d</ansi><ansi fg="black-bold"> reads the next; </ansi><ansi fg="command">chronicle all</ansi><ansi fg="black-bold"> reads every deed.</ansi>`,
			r.page, pages, more, min(r.page+1, pages))
	}
	if r.kind == "" {
		if t := m.tallyLine(l); t != "" {
			b.WriteString("\n" + t)
		}
	}
	return b.String()
}

// tallyLine counts every deed ever recorded (the log forgets old ones, the
// count does not).
func (m *Module) tallyLine(l chronicle.Log) string {
	var parts []string
	for _, k := range chronicle.Kinds {
		if n := l.Tally[k.Kind]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, strings.ToLower(k.Label)))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return `<ansi fg="black-bold">All told: ` + strings.Join(parts, ", ") + ".</ansi>"
}

func (m *Module) chronicleCommand(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	user.SendText(m.render(user.UserId, rest))
	m.push(user.UserId)
	return true, nil
}

func firstWord(k chronicle.Kind) string {
	for _, i := range chronicle.Kinds {
		if i.Kind == k {
			return i.Words[0]
		}
	}
	return string(k)
}
