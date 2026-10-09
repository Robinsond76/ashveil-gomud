package bounties

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/bounty"
	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/encounters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// postingRow is one posting as the web client draws it.
type postingRow struct {
	N      int    `json:"n"`
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Zone   string `json:"zone"`
	Band   string `json:"band"`
	Rating string `json:"rating,omitempty"`
	Count  int    `json:"count"`
	Reward int    `json:"reward"`
	Taken  bool   `json:"taken"`
	Done   bool   `json:"done"`
}

// heldRow is one bounty the company holds.
type heldRow struct {
	N      int    `json:"n"`
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Zone   string `json:"zone"`
	Have   int    `json:"have"`
	Count  int    `json:"count"`
	Reward int    `json:"reward"`
	Left   int64  `json:"left"` // seconds until it lapses
	Ready  bool   `json:"ready"`
}

// panel is the `Company.Bounties` GMCP message.
type panel struct {
	Now      int64        `json:"now"`
	AtBoard  bool         `json:"at_board"`
	Board    string       `json:"board,omitempty"`
	Band     string       `json:"band,omitempty"`
	Rotates  int64        `json:"rotates,omitempty"` // seconds until the board posts a new list
	Max      int          `json:"max"`
	Postings []postingRow `json:"postings"`
	Held     []heldRow    `json:"held"`
}

func (m *Module) panelFor(userID int) panel {
	now := m.clock().Unix()
	st, _ := m.stateOf(userID)
	p := panel{Now: now, Max: bounty.MaxHeld, Postings: []postingRow{}, Held: []heldRow{}}
	level := m.w.Level(userID)
	for i, h := range st.Held {
		have := progress(userID, h)
		p.Held = append(p.Held, heldRow{N: i + 1, Kind: h.Target.Kind, Name: h.Target.Name, Zone: h.Target.Zone, Have: have,
			Count: max(1, h.Target.Count), Reward: h.Reward, Left: max(0, h.Due-now), Ready: have >= max(1, h.Target.Count)})
	}
	if b, ok := m.w.Board(userID); ok {
		p.AtBoard, p.Board = true, b.Title
		if b.Band.Valid() {
			p.Band = b.Band.String()
		}
		p.Rotates = bounty.Window(now) + bounty.WindowSeconds - now
		for i, po := range m.postings(b) {
			_, done := st.Done[po.ID]
			p.Postings = append(p.Postings, postingRow{N: i + 1, Kind: po.Target.Kind, Name: po.Target.Name, Zone: po.Target.Zone,
				Band: po.Target.Band().String(), Rating: encounters.Rating(level, encounters.Band{Low: po.Target.Low, High: po.Target.High}),
				Count: po.Target.Count, Reward: po.Reward, Taken: st.Holds(po.Target), Done: done})
		}
	}
	return p
}

// push sends a signed-in leader's web client the current board and holdings.
func (m *Module) push(userID int) {
	if !m.w.Online(userID) {
		return
	}
	m.w.Push(userID, "Company.Bounties", m.panelFor(userID))
}

func duration(seconds int64) string {
	switch {
	case seconds >= 3600:
		return fmt.Sprintf("%dh %02dm", seconds/3600, seconds%3600/60)
	case seconds >= 60:
		return fmt.Sprintf("%d min", seconds/60)
	}
	return "under a minute"
}

func what(t bounty.Target) string {
	if t.Kind == bounty.Boss {
		return fmt.Sprintf(`slay <ansi fg="mobname">%s</ansi>, master of its lair in %s`, t.Name, t.Zone)
	}
	return fmt.Sprintf(`break %d groups of <ansi fg="mobname">%s</ansi> in %s`, max(1, t.Count), t.Name, t.Zone)
}

// renderBoard is the board's list for a leader who stands at it.
func (m *Module) renderBoard(userID int, b board) string {
	now := m.clock().Unix()
	st, _ := m.stateOf(userID)
	level := m.w.Level(userID)
	var out strings.Builder
	fmt.Fprintf(&out, `<ansi fg="yellow-bold">The bounty board at %s</ansi>`+"\n", b.Title)
	posts := m.postings(b)
	if len(posts) == 0 {
		out.WriteString("Nothing is posted today. The board lists lairs and bands from the zones near here, and none are known.\n")
	}
	for i, po := range posts {
		note := ""
		switch {
		case st.Holds(po.Target):
			note = ` <ansi fg="cyan">(yours)</ansi>`
		case func() bool { _, d := st.Done[po.ID]; return d }():
			note = ` <ansi fg="black-bold">(settled)</ansi>`
		}
		rate := ""
		if r := encounters.Rating(level, encounters.Band{Low: po.Target.Low, High: po.Target.High}); r != "" {
			rate = ", " + r + " for you"
		}
		fmt.Fprintf(&out, "  <ansi fg=\"command\">%d</ansi>. %s: %d gold (zone level %s%s)%s\n", i+1, what(po.Target), po.Reward, po.Target.Band(), rate, note)
	}
	fmt.Fprintf(&out, `<ansi fg="black-bold">A new list is posted in %s. </ansi><ansi fg="command">bounty take [number]</ansi><ansi fg="black-bold"> takes one; </ansi><ansi fg="command">bounty claim</ansi><ansi fg="black-bold"> collects.</ansi>`, duration(bounty.Window(now)+bounty.WindowSeconds-now))
	return out.String()
}

// renderHeld lists the bounties a company holds.
func (m *Module) renderHeld(userID int) string {
	now := m.clock().Unix()
	st, lapsed := m.stateOf(userID)
	var out strings.Builder
	if lapsed > 0 {
		fmt.Fprintf(&out, `<ansi fg="red">%d of your bounties lapsed unclaimed.</ansi>`+"\n", lapsed)
	}
	if len(st.Held) == 0 {
		out.WriteString("Your company holds no bounties. Take one at a bounty board (help bounties).")
		return out.String()
	}
	out.WriteString(`<ansi fg="yellow-bold">Your company's bounties</ansi>` + "\n")
	for i, h := range st.Held {
		have := progress(userID, h)
		need := max(1, h.Target.Count)
		state := fmt.Sprintf("%d of %d", have, need)
		if have >= need {
			state = `<ansi fg="green">ready to claim</ansi>`
		}
		fmt.Fprintf(&out, "  <ansi fg=\"command\">%d</ansi>. %s: %d gold, %s, %s left\n", i+1, what(h.Target), h.Reward, state, duration(max(0, h.Due-now)))
	}
	out.WriteString(`<ansi fg="black-bold">Claim a finished bounty at any board with </ansi><ansi fg="command">bounty claim</ansi><ansi fg="black-bold">.</ansi>`)
	return out.String()
}

// numberArg reads "2" from the rest of a command.
func numberArg(rest string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(rest))
	return n, err == nil && n >= 1
}

func (m *Module) command(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	fields := strings.Fields(strings.ToLower(rest))
	sub, arg := "", ""
	if len(fields) > 0 {
		sub = fields[0]
		arg = strings.Join(fields[1:], " ")
	}
	b, atBoard := m.w.Board(user.UserId)
	switch sub {
	case "":
		if atBoard {
			user.SendText(m.renderBoard(user.UserId, b) + "\n" + m.renderHeld(user.UserId))
		} else {
			user.SendText(m.renderHeld(user.UserId))
		}
	case "list", "held", "mine":
		user.SendText(m.renderHeld(user.UserId))
	case "board", "look":
		if !atBoard {
			user.SendText("There is no bounty board here. Look for one in a town (help bounties).")
		} else {
			user.SendText(m.renderBoard(user.UserId, b))
		}
	case "take", "accept":
		user.SendText(m.take(user.UserId, b, atBoard, arg))
	case "claim", "collect":
		user.SendText(m.claim(user.UserId, atBoard, arg))
	case "drop", "abandon":
		user.SendText(m.drop(user.UserId, arg))
	default:
		user.SendText(`Usage: bounty | bounty take [number] | bounty claim [number] | bounty drop [number]. See <ansi fg="command">help bounties</ansi>.`)
	}
	m.push(user.UserId)
	return true, nil
}

func (m *Module) take(userID int, b board, atBoard bool, arg string) string {
	if !atBoard {
		return "There is no bounty board here. Bounties are taken at a board in a town."
	}
	if m.w.Busy(userID) {
		return "Not in the middle of a battle."
	}
	n, ok := numberArg(arg)
	if !ok {
		return `Take which bounty? Usage: bounty take [number] (the numbers are on the board).`
	}
	posts := m.postings(b)
	if n > len(posts) {
		return fmt.Sprintf("The board has no posting %d.", n)
	}
	po := posts[n-1]
	st, _ := m.stateOf(userID)
	st = st.Clone()
	if err := st.Take(po, m.clock().Unix(), newestSeq(userID)); err != nil {
		return capital(err.Error()) + "."
	}
	if err := m.commit(userID, st); err != nil {
		return "The clerk cannot write that down right now. Try again in a moment."
	}
	return fmt.Sprintf(`You take the bounty: %s. It pays %d gold on proof and holds for %s.`, what(po.Target), po.Reward, duration(bounty.TermSeconds))
}

func capital(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (m *Module) drop(userID int, arg string) string {
	n, ok := numberArg(arg)
	st, _ := m.stateOf(userID)
	if !ok || n > len(st.Held) {
		return `Drop which bounty? Usage: bounty drop [number] (the numbers are in "bounty").`
	}
	st = st.Clone()
	h := st.Held[n-1]
	st.Held = append(st.Held[:n-1:n-1], st.Held[n:]...)
	if err := m.commit(userID, st); err != nil {
		return "The clerk cannot write that down right now. Try again in a moment."
	}
	return fmt.Sprintf("You drop the bounty on %s. It can be taken again while the board still lists it.", h.Target.Name)
}

func (m *Module) claim(userID int, atBoard bool, arg string) string {
	if !atBoard {
		return "Bounties are claimed at a board. Walk to the town's bounty board."
	}
	if m.w.Busy(userID) {
		return "Not in the middle of a battle."
	}
	st, _ := m.stateOf(userID)
	if len(st.Held) == 0 {
		return "Your company holds no bounties."
	}
	var ready []int
	if n, ok := numberArg(arg); ok {
		if n > len(st.Held) {
			return fmt.Sprintf("You hold no bounty %d.", n)
		}
		h := st.Held[n-1]
		if have := progress(userID, h); have < max(1, h.Target.Count) {
			return fmt.Sprintf("The clerk finds no proof yet: %d of %d. The chronicle must show the kill after you took the bounty (help bounties).", have, max(1, h.Target.Count))
		}
		ready = []int{n - 1}
	} else {
		for i, h := range st.Held {
			if progress(userID, h) >= max(1, h.Target.Count) {
				ready = append(ready, i)
			}
		}
		if len(ready) == 0 {
			return "The clerk finds no proof for any of your bounties yet. The chronicle must show the kill after you took the bounty (help bounties)."
		}
	}
	now := m.clock().Unix()
	next := st.Clone()
	var paid []bounty.Held
	for k := len(ready) - 1; k >= 0; k-- { // from the end, so the indexes stay true
		paid = append([]bounty.Held{next.Settle(ready[k], now)}, paid...)
	}
	if err := m.commit(userID, next); err != nil {
		return "The clerk cannot write that down right now. Try again in a moment."
	}
	total := 0
	var lines []string
	for _, h := range paid {
		total += h.Reward
		lines = append(lines, fmt.Sprintf("%s (%d gold)", h.Target.Name, h.Reward))
	}
	if !m.w.Pay(userID, total) {
		return "The clerk counts out your gold, but you are not there to take it."
	}
	for _, h := range paid {
		chronicle.Record(userID, chronicle.Entry{Kind: chronicle.Bounty, Subject: h.Target.Name, Detail: fmt.Sprintf("%d gold", h.Reward), Ref: "bounty:" + h.Target.Ref, Zone: h.Target.Zone})
	}
	return fmt.Sprintf(`<ansi fg="green">The clerk reads the chronicle, nods, and counts out %d gold for %s.</ansi>`, total, strings.Join(lines, ", "))
}
