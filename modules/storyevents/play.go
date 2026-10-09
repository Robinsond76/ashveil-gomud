package storyevents

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/storyevents"
)

// outcomeOrder runs the quiet outcomes first and the ones that change the
// scene last: a move before a battle, so the fight is in the new room.
var outcomeOrder = map[string]int{
	storyevents.OutcomeFlag: 0, storyevents.OutcomeGold: 1, storyevents.OutcomeItem: 2,
	storyevents.OutcomeLoseItem: 3, storyevents.OutcomeLoyalty: 4, storyevents.OutcomeNeed: 5,
	storyevents.OutcomeAilment: 6, storyevents.OutcomeWound: 7, storyevents.OutcomeMove: 8,
	storyevents.OutcomeBattle: 9,
}

// choose answers the waiting page with the n-th choice (1-based). page,
// when set, is the page token the answer was given on (the web client sends
// it); an answer meant for a page that has already turned is refused, so a
// double click never answers the next page too.
func (m *Module) choose(userID, n int, page string) {
	ev, p, ok := m.waiting(userID)
	if !ok {
		m.w.Send(userID, "No scene is waiting on your company.")
		return
	}
	if page != "" && page != pageToken(ev.ID, p.Page) {
		return
	}
	user := m.w.User(userID)
	if user == nil || user.Character == nil {
		return
	}
	// camp = false: an answer can move the company or start a fight, so
	// none is taken during a rest either.
	if m.w.Busy(userID, p.Room, false) {
		m.w.Send(userID, "Not now: finish the fight or the rest first.")
		return
	}
	pg := ev.Pages[p.Page]
	if n < 1 || n > len(pg.Choices) {
		m.w.Send(userID, fmt.Sprintf("Choose a number from 1 to %d.", len(pg.Choices)))
		return
	}
	c := pg.Choices[n-1]
	members := m.w.Members(userID)
	m.mu.Lock()
	flags := m.state[userID].flagSet()
	m.mu.Unlock()
	co := m.w.Company(userID, flags)
	actor, open := actorFor(c, members, co)
	if !open {
		needs := c.Hint
		if needs == "" {
			needs = c.Require.Describe()
		}
		m.w.Send(userID, fmt.Sprintf("That is closed to your company: it needs %s.", needs))
		return
	}

	outs, text, next := c.Do, c.Text, c.Next
	if c.Risk != nil && m.rng(100) < c.Risk.RiskPct(c.Require.Rank(actor)) {
		outs, text, next = c.Fail, c.FailText, c.FailNext
	}
	moveRoom := 0
	for _, o := range outs {
		if o.Kind == storyevents.OutcomeMove {
			moveRoom = o.Room
		}
	}

	op, committed := m.commit(userID, p, ev, next, moveRoom, outs)
	if !committed {
		m.w.Send(userID, "Something went wrong, and the moment holds. Try again.")
		return
	}

	if next == "" {
		// Phase 63: a scene's end is a deed, with the answer that ended it
		// and whoever took it. Written before the outcomes run, so a move
		// outcome does not file it under the room the company was sent to.
		chronicle.Record(userID, chronicle.Entry{Kind: chronicle.Story, Members: []string{actor.Name}, Keys: []string{actor.Key}, Subject: ev.Title, Detail: c.Label, Ref: "event:" + ev.ID})
	}
	var lines []string
	if text != "" {
		lines = append(lines, storyevents.Fill(text, actor.Name))
	}
	if c.Stance != "" {
		// Phase 64: a stance is a choice the companions have a view on.
		lines = append(lines, m.w.Opinion(userID, fmt.Sprintf("story:%s:%d", ev.ID, op), c.Stance, ev.Title, members)...)
	}
	sorted := append([]storyevents.Outcome(nil), outs...)
	sort.SliceStable(sorted, func(i, j int) bool { return outcomeOrder[sorted[i].Kind] < outcomeOrder[sorted[j].Kind] })
	room := p.Room
	for i, o := range sorted {
		lines = append(lines, m.apply(userID, ev.ID, op, i, &room, actor, members, o)...)
	}
	if next != "" {
		m.show(userID, ev, next, lines)
		return
	}
	m.finish(userID, ev, lines)
}

// commit saves the answer before it is applied: the page moves on (or the
// event ends and is marked done) and its flags are set. It returns the
// operation number the outcomes' ids are built from.
func (m *Module) commit(userID int, p Pending, ev storyevents.Event, next string, moveRoom int, outs []storyevents.Outcome) (int, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, ok := m.state[userID]
	if !ok || cur.Pending == nil || *cur.Pending != p {
		return 0, false
	}
	st := cur.clone()
	op := st.Ops
	st.Ops++
	if next != "" {
		np := p
		np.Page = next
		if moveRoom > 0 {
			np.Room = moveRoom
		}
		st.Pending = &np
	} else {
		st.Pending = nil
		if st.Done == nil {
			st.Done = map[string]int64{}
		}
		st.Done[ev.ID] = m.clock().Unix()
	}
	for _, o := range outs {
		if o.Kind == storyevents.OutcomeFlag {
			st.addFlag(o.Flag)
		}
	}
	m.state[userID] = st
	if err := m.saveLocked(); err != nil {
		m.state[userID] = cur
		mudlog.Warn("storyevents: save an answer", "user", userID, "event", ev.ID, "error", err)
		return 0, false
	}
	return op, true
}

// targets is who an outcome lands on.
func (m *Module) targets(o storyevents.Outcome, actor storyevents.Facts, members []storyevents.Facts) []storyevents.Facts {
	switch o.Who {
	case storyevents.WhoLeader:
		if len(members) > 0 {
			return members[:1]
		}
	case storyevents.WhoAll:
		return members
	case storyevents.WhoRandom:
		if len(members) > 0 {
			return []storyevents.Facts{members[m.rng(len(members))]}
		}
	}
	return []storyevents.Facts{actor}
}

// apply runs one outcome against the world and returns the lines it earns.
// room is where the company stands; a move updates it for what follows.
func (m *Module) apply(userID int, eventID string, op, index int, room *int, actor storyevents.Facts, members []storyevents.Facts, o storyevents.Outcome) []string {
	opID := fmt.Sprintf("story:%s:%d:%d", eventID, op, index)
	var lines []string
	say := func(generic string, who storyevents.Facts) {
		if o.Line != "" {
			lines = append(lines, storyevents.Fill(o.Line, who.Name))
		} else if generic != "" {
			lines = append(lines, generic)
		}
	}
	switch o.Kind {
	case storyevents.OutcomeWound:
		for _, f := range m.targets(o, actor, members) {
			say(m.w.Wound(userID, f, o.Pct), f)
		}
	case storyevents.OutcomeAilment:
		for _, f := range m.targets(o, actor, members) {
			say(m.w.Ailment(userID, f, o.Ailment), f)
		}
	case storyevents.OutcomeNeed:
		for _, f := range m.targets(o, actor, members) {
			say(m.w.Need(userID, f, o.Stat, o.Amount), f)
		}
	case storyevents.OutcomeItem:
		say(m.w.AddItems(userID, opID, o.Item, max(o.Count, 1)), actor)
	case storyevents.OutcomeLoseItem:
		say(m.w.TakeItems(userID, o.Item, max(o.Count, 1)), actor)
	case storyevents.OutcomeGold:
		say(m.w.Gold(userID, o.Amount), actor)
	case storyevents.OutcomeLoyalty:
		who := m.targets(o, actor, members)
		// A custom line only when someone's loyalty actually moved.
		if line := m.w.Loyalty(userID, opID, who, o.Amount); line != "" {
			say(line, actor)
		}
	case storyevents.OutcomeFlag:
		say("", actor)
	case storyevents.OutcomeMove:
		say(m.w.Move(userID, o.Room), actor)
		*room = o.Room
	case storyevents.OutcomeBattle:
		say(m.w.Battle(userID, *room, o.Foes), actor)
	}
	return lines
}

// describe lists what a finished scene gave, for tests and logs.
func describe(lines []string) string { return strings.Join(lines, " | ") }
