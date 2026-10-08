package company

// Phase 64: companion opinions. A companion's personality likes and
// dislikes a few kinds of choice the leader makes (internal/opinions). When
// one is made, the companions who saw it say one line each and their
// loyalty moves a little, through the real loyalty record, at most once per
// choice per companion. It reuses the banter personalities, the loyalty
// disposition and the chronicle's deeds; nothing here keeps a new meter.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/banter"
	"github.com/GoMudEngine/GoMud/internal/chronicle"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/creatures"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/morale"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

var (
	_ domain.OpinionProvider = (*CompanyModule)(nil)
	_ domain.OpinionViewer   = (*CompanyModule)(nil)
)

// opinionOp is the saved name of one choice, so a retry does nothing twice.
func opinionOp(c opinions.Choice) string {
	if c.Op == "" {
		return ""
	}
	return "opinion:" + string(c.Kind) + ":" + c.Op
}

// opinionWitnesses are the companions who may react: living, with the
// leader, not a creature or construct (they have no opinions). Callers may
// name who saw it; otherwise it is everyone walking with the leader.
func (m *CompanyModule) opinionWitnesses(leaderUserID int, record domain.Record, named []int) map[int]bool {
	ids := named
	if ids == nil {
		ids = m.CompanionsWithLeader(leaderUserID)
	}
	out := map[int]bool{}
	for _, id := range ids {
		out[id] = true
	}
	for _, c := range record.Companions {
		if c.Dead() || c.PendingReturn || c.MoraleDesert || bound(c) || creatures.Is(c.Archetype) {
			delete(out, c.ID)
		}
	}
	return out
}

// Opinion implements company.OpinionProvider.
func (m *CompanyModule) Opinion(leaderUserID int, c opinions.Choice) ([]string, error) {
	info, known := opinions.InfoOf(c.Kind)
	if !known {
		return nil, nil
	}
	if err := m.persistenceAvailable(); err != nil {
		return nil, err
	}
	record, ok := m.registry.Get(leaderUserID)
	op := opinionOp(c)
	if !ok || (op != "" && record.HasApplied(op)) {
		return nil, nil
	}
	seen := m.opinionWitnesses(leaderUserID, record, c.Witnesses)
	now := m.now().Unix()
	var reactions []opinions.Reaction
	var narrator banter.Narrator
	for i, comp := range record.Companions {
		if !seen[comp.ID] {
			continue
		}
		personality := personalityOf(comp)
		verdict := opinions.Verdict(personality, c.Kind)
		if verdict == opinions.Neutral {
			continue
		}
		// Alignment already answers a mercy choice (Phase 27); one reaction
		// per choice per companion.
		if (c.Kind == opinions.Spare || c.Kind == opinions.Execute) && morale.Reaction(m.companionAlignment(comp), c.Kind == opinions.Spare) != 0 {
			continue
		}
		var mem *domain.OpinionMemory
		if comp.Opinions != nil {
			mem = comp.Opinions.Clone()
		} else {
			mem = &domain.OpinionMemory{}
		}
		if last, spoke := mem.Spoke[string(c.Kind)]; spoke && now-last < int64(info.Cooldown.Seconds()) {
			continue
		}
		delta := 0
		if d := comp.Disposition; d != nil {
			next := d.Loyalty
			switch {
			case verdict == opinions.Likes && d.Loyalty < opinions.Ceiling:
				next = min(opinions.Ceiling, d.Loyalty+opinions.Nudge)
			case verdict == opinions.Dislikes && d.Loyalty > opinions.Floor:
				next = max(opinions.Floor, d.Loyalty-opinions.Nudge)
			}
			delta = next - d.Loyalty
			if delta != 0 {
				nd := *d
				nd.Loyalty = next
				record.Companions[i].Disposition = &nd
			}
		}
		mem.Remember(domain.OpinionNote{Kind: string(c.Kind), Verdict: verdict, At: now, Subject: c.Subject})
		record.Companions[i].Opinions = mem
		name := companionName(comp)
		line := opinions.LineFor(&narrator, name, personality, c.Kind)
		if delta != 0 {
			line += banter.Aside(delta)
		}
		reactions = append(reactions, opinions.Reaction{CompanionID: comp.ID, Name: name, Personality: personality, Kind: c.Kind, Verdict: verdict, Line: line, Delta: delta})
	}
	if len(reactions) == 0 {
		return nil, nil
	}
	if op != "" {
		record.MarkApplied(op)
	}
	before := m.registry.Clone()
	m.registry.Put(record)
	if err := m.save(); err != nil {
		m.registry = before
		return nil, err
	}
	lines := make([]string, 0, len(reactions))
	for _, r := range reactions {
		lines = append(lines, r.Line)
	}
	// Observers (Phase 65 bonds) hear who actually saw it, silent ones too.
	c.Witnesses = make([]int, 0, len(seen))
	for _, comp := range record.Companions {
		if seen[comp.ID] {
			c.Witnesses = append(c.Witnesses, comp.ID)
		}
	}
	opinions.Announce(leaderUserID, c, reactions)
	return lines, nil
}

// moodOf is loyalty in a word.
func moodOf(loyalty int) string {
	switch {
	case loyalty >= 80:
		return "devoted"
	case loyalty >= 60:
		return "loyal"
	case loyalty >= 40:
		return "steady"
	case loyalty >= 20:
		return "wavering"
	}
	return "close to leaving"
}

// OpinionPanel implements company.OpinionViewer: each living companion's
// likes, dislikes and recent words, and the chronicle's lifetime counts of
// mercy and execution.
func (m *CompanyModule) OpinionPanel(leaderUserID int) (domain.OpinionPanel, bool) {
	if m.persistenceAvailable() != nil {
		return domain.OpinionPanel{}, false
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return domain.OpinionPanel{}, false
	}
	now := m.now().Unix()
	panel := domain.OpinionPanel{
		Spared:   chronicle.Total(leaderUserID, chronicle.Spared),
		Executed: chronicle.Total(leaderUserID, chronicle.Executed),
		Members:  []domain.OpinionRow{},
	}
	for _, c := range record.Companions {
		if c.Dead() || bound(c) || creatures.Is(c.Archetype) {
			continue
		}
		personality := personalityOf(c)
		likes, dislikes := opinions.Leanings(personality)
		row := domain.OpinionRow{
			Key: string(domain.CompanionMemberKey(c.ID)), ID: c.ID, Name: companionName(c), Personality: personality,
			Loyalty: domain.MaxLoyalty, Likes: kindLabels(likes), Dislikes: kindLabels(dislikes),
			Notes: []domain.OpinionNoteView{}, Deeds: []string{},
		}
		if c.Disposition != nil {
			row.Loyalty = c.Disposition.Loyalty
		}
		row.Mood = moodOf(row.Loyalty)
		if c.Opinions != nil {
			for i := len(c.Opinions.Notes) - 1; i >= 0 && len(row.Notes) < 3; i-- {
				n := c.Opinions.Notes[i]
				row.Notes = append(row.Notes, domain.OpinionNoteView{Kind: n.Kind, Label: opinions.Label(opinions.Kind(n.Kind)), Verdict: n.Verdict, Ago: chronicle.Ago(n.At, now), Subject: n.Subject})
			}
		}
		for _, e := range chronicle.Query(leaderUserID, chronicle.Filter{Key: row.Key, Limit: 2}) {
			row.Deeds = append(row.Deeds, chronicle.Prose(e))
		}
		panel.Members = append(panel.Members, row)
	}
	return panel, true
}

func kindLabels(kinds []opinions.Kind) []string {
	out := make([]string, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, opinions.Label(k))
	}
	return out
}

// opinionsView is the text of `opinions [member]`.
func (m *CompanyModule) opinionsView(leaderUserID int, selector string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	panel, ok := m.OpinionPanel(leaderUserID)
	if !ok || len(panel.Members) == 0 {
		return "You have no companions to hold opinions. Recruit some first (help company)."
	}
	rows := panel.Members
	if sel := strings.ToLower(strings.TrimSpace(selector)); sel != "" {
		record, _ := m.registry.Get(leaderUserID)
		c, found := resolveCompanion(record, sel)
		if !found {
			return fmt.Sprintf("No companion called %q. Try: opinions", selector)
		}
		rows = nil
		for _, r := range panel.Members {
			if r.ID == c.ID {
				rows = append(rows, r)
			}
		}
		if len(rows) == 0 {
			return fmt.Sprintf("%s holds no opinions.", companionName(c))
		}
	}
	lines := []string{"What your companions think of your choices:"}
	for _, r := range rows {
		lines = append(lines, fmt.Sprintf("#%d %s, %s: %s (loyalty %d).", r.ID, r.Name, r.Personality, r.Mood, r.Loyalty))
		lines = append(lines, "  Likes: "+strings.Join(r.Likes, ", ")+".")
		lines = append(lines, "  Dislikes: "+strings.Join(r.Dislikes, ", ")+".")
		for _, n := range r.Notes {
			verb := "approved of"
			if n.Verdict < 0 {
				verb = "disliked"
			}
			text := fmt.Sprintf("  %s %s, %s", strings.ToUpper(verb[:1])+verb[1:], n.Label, n.Ago)
			if n.Subject != "" {
				text += " (" + n.Subject + ")"
			}
			lines = append(lines, text+".")
		}
		for _, d := range r.Deeds {
			lines = append(lines, "  Remembers: "+d)
		}
	}
	lines = append(lines, fmt.Sprintf("The company has seen you spare %s and execute %s.", times(panel.Spared), times(panel.Executed)))
	lines = append(lines, "A companion says its piece once per choice, and again on the same kind only after a while. Approval lifts loyalty no higher than 80 and disapproval never takes it below 30 (help opinions).")
	return strings.Join(lines, "\n")
}

func times(n int) string {
	if n == 1 {
		return "1 foe"
	}
	return strconv.Itoa(n) + " foes"
}

// opinionsCommand is `opinions [member]`.
func (m *CompanyModule) opinionsCommand(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	user.SendText(m.opinionsView(user.UserId, rest))
	return true, nil
}

// opinionLine is one companion's view of the leader's choices on a line,
// for `company inspect`; blank for one that holds none.
func (m *CompanyModule) opinionLine(leaderUserID, companionID int) string {
	panel, ok := m.OpinionPanel(leaderUserID)
	if !ok {
		return ""
	}
	for _, r := range panel.Members {
		if r.ID != companionID {
			continue
		}
		return fmt.Sprintf("Temperament: %s, %s (loyalty %d). Likes %s; dislikes %s. See opinions.",
			r.Personality, r.Mood, r.Loyalty, strings.Join(r.Likes, ", "), strings.Join(r.Dislikes, ", "))
	}
	return ""
}
