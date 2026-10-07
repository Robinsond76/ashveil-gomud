package storyevents

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/lifestory"
	"github.com/GoMudEngine/GoMud/internal/storyevents"
)

// payload is the `Event` GMCP message the web client's modal draws. Active
// is false when no page is waiting (Result then carries the last outcome,
// once, so the modal can show how the scene ended).
type payload struct {
	Active bool   `json:"active"`
	ID     string `json:"id,omitempty"`
	// Page is the page token the web client sends back with its answer.
	Page    string       `json:"page,omitempty"`
	Title   string       `json:"title,omitempty"`
	Picture string       `json:"picture,omitempty"`
	Text    string       `json:"text,omitempty"`
	Result  []string     `json:"result,omitempty"`
	Choices []choiceView `json:"choices,omitempty"`
}

type choiceView struct {
	N     int    `json:"n"`
	Label string `json:"label"`
	Open  bool   `json:"open"`
	// Who names the member who would take the choice, when it asks for one.
	Who string `json:"who,omitempty"`
	// Needs says what a closed choice asks for.
	Needs string `json:"needs,omitempty"`
	// Because says what opened a choice that asks for a life story
	// (Phase 72), so a player sees their past mattered: "life story: a
	// soldier".
	Because string `json:"because,omitempty"`
	// Risk is a word for how likely it is to go wrong; empty for a sure
	// thing.
	Risk string `json:"risk,omitempty"`
}

// pageToken names one page of one event, for answers from the web client.
func pageToken(eventID, pageID string) string { return eventID + "/" + pageID }

// riskWord puts a failure chance in words; numbers stay out of the page.
func riskWord(pct int) string {
	switch {
	case pct >= 60:
		return "a long shot"
	case pct >= 35:
		return "chancy"
	case pct > 0:
		return "a small risk"
	}
	return ""
}

// actorFor is who takes a choice: the best qualified member when it asks
// for one, else the leader. ok is false when the choice is closed.
func actorFor(c storyevents.Choice, members []storyevents.Facts, co storyevents.Company) (storyevents.Facts, bool) {
	if !c.Require.MemberSet() {
		if len(members) == 0 || !c.Require.CompanyRequirement.Meets(co) {
			return storyevents.Facts{}, false
		}
		return members[0], true
	}
	return storyevents.Best(c.Require, members, co)
}

func (m *Module) choiceViews(page storyevents.Page, members []storyevents.Facts, co storyevents.Company) []choiceView {
	out := make([]choiceView, 0, len(page.Choices))
	for i, c := range page.Choices {
		v := choiceView{N: i + 1, Label: c.Label}
		who, open := actorFor(c, members, co)
		v.Open = open
		switch {
		case !open:
			v.Needs = c.Hint
			if v.Needs == "" {
				v.Needs = c.Require.Describe()
			}
		case c.Require.MemberSet():
			v.Who = who.Name
			if name, ok := lifestory.TagName(c.Require.Tag); ok {
				v.Because = "life story: " + name
			}
		}
		if open && c.Risk != nil {
			v.Risk = riskWord(c.Risk.RiskPct(c.Require.Rank(who)))
		}
		out = append(out, v)
	}
	return out
}

// pageView is what the company sees of one page.
func (m *Module) pageView(userID int, ev storyevents.Event, pageID string, result []string) payload {
	page := ev.Pages[pageID]
	members := m.w.Members(userID)
	m.mu.Lock()
	flags := m.state[userID].flagSet()
	m.mu.Unlock()
	co := m.w.Company(userID, flags)
	picture := page.Picture
	if picture == "" {
		picture = ev.Picture
	}
	return payload{
		Active: true, ID: ev.ID, Page: pageToken(ev.ID, pageID), Title: ev.Title, Picture: picture, Text: page.Text,
		Result: result, Choices: m.choiceViews(page, members, co),
	}
}

// render is a page as the terminal shows it.
func render(p payload) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<ansi fg=\"yellow-bold\">%s</ansi>\n", p.Title)
	for _, line := range p.Result {
		fmt.Fprintf(&b, "<ansi fg=\"black-bold\">%s</ansi>\n", line)
	}
	if len(p.Result) > 0 {
		b.WriteString("\n")
	}
	b.WriteString(p.Text)
	b.WriteString("\n\n")
	for _, c := range p.Choices {
		fmt.Fprintf(&b, "  <ansi fg=\"command\">%d</ansi>. %s", c.N, c.Label)
		switch {
		case !c.Open:
			fmt.Fprintf(&b, " <ansi fg=\"black-bold\">(closed: needs %s)</ansi>", c.Needs)
		case c.Who != "" || c.Risk != "":
			notes := []string{}
			if c.Who != "" {
				notes = append(notes, c.Who)
			}
			if c.Because != "" {
				notes = append(notes, c.Because)
			}
			if c.Risk != "" {
				notes = append(notes, c.Risk)
			}
			fmt.Fprintf(&b, " <ansi fg=\"black-bold\">(%s)</ansi>", strings.Join(notes, ", "))
		}
		b.WriteString("\n")
	}
	b.WriteString("\nAnswer with <ansi fg=\"command\">choose</ansi> and a number.")
	return b.String()
}

// show tells the leader a page, on the terminal and in the web modal.
func (m *Module) show(userID int, ev storyevents.Event, pageID string, result []string) {
	p := m.pageView(userID, ev, pageID, result)
	m.w.Send(userID, render(p))
	m.w.Push(userID, "Event", p)
}

// reshow shows a waiting page again; header, when set, goes first.
func (m *Module) reshow(userID int, header string) bool {
	ev, p, ok := m.waiting(userID)
	if !ok {
		return false
	}
	var result []string
	if header != "" {
		result = []string{header}
	}
	m.show(userID, ev, p.Page, result)
	return true
}

// finish tells the leader how a scene ended.
func (m *Module) finish(userID int, ev storyevents.Event, result []string) {
	if len(result) > 0 {
		m.w.Send(userID, strings.Join(result, "\n"))
	}
	m.w.Push(userID, "Event", payload{Active: false, ID: ev.ID, Title: ev.Title, Result: result})
}
