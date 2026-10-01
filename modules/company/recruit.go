package company

// Phase 22c: settlement recruiters. A recruiter is a configured room that
// offers authored companion candidates, free once (tutorial) or for gold.
// See modules/company/AGENTS.md and the Phase 22c design.

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

type candidate struct {
	ID            string
	MobTemplateID int
	Price         int
	// Tutorial candidates are free and claimable once per character.
	Tutorial bool
}

// cost is what the candidate asks: nothing for a tutorial candidate.
func (c candidate) cost() int {
	if c.Tutorial {
		return 0
	}
	return c.Price
}

type recruiter struct {
	RoomID     int
	Name       string
	Candidates []candidate
	// Generated recruiters also post each player's own roster (Phase
	// 32a2); Weights, when set, is their archetype mix.
	Generated bool
	Weights   map[string]int
}

// parseRecruiters reads Recruiters ([{RoomId, Name, Candidates: [{Id,
// MobTemplateId, Price, Tutorial}]}]), skipping malformed entries with a
// warning. Templates are checked when used, not here.
func parseRecruiters(raw any) map[int]recruiter {
	out := map[int]recruiter{}
	list, ok := raw.([]any)
	if !ok {
		return out
	}
	for _, entry := range list {
		fields := lowerKeys(entry)
		if fields == nil {
			mudlog.Warn("company: recruiter entry is not a map; skipped")
			continue
		}
		roomID, ok := configInt(fields["roomid"])
		if !ok || roomID <= 0 {
			mudlog.Warn("company: recruiter without a room; skipped", "roomid", fields["roomid"])
			continue
		}
		if _, dup := out[roomID]; dup {
			mudlog.Warn("company: recruiter room listed twice; later entry skipped", "roomid", roomID)
			continue
		}
		name, _ := fields["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			name = "the recruiter"
		}
		rec := recruiter{RoomID: roomID, Name: name}
		rec.Generated, _ = fields["generated"].(bool)
		rec.Weights = parseArchetypeWeights(fields["archetypeweights"], roomID)
		seen := map[string]bool{}
		candidates, _ := fields["candidates"].([]any)
		for _, rawCandidate := range candidates {
			cf := lowerKeys(rawCandidate)
			if cf == nil {
				mudlog.Warn("company: recruit candidate is not a map; skipped", "roomid", roomID)
				continue
			}
			id, _ := cf["id"].(string)
			id = strings.ToLower(strings.TrimSpace(id))
			templateID, templateOK := configInt(cf["mobtemplateid"])
			price := 0
			priceOK := true
			if rawPrice, set := cf["price"]; set {
				price, priceOK = configInt(rawPrice)
			}
			tutorial, _ := cf["tutorial"].(bool)
			switch {
			case id == "" || seen[id]:
				mudlog.Warn("company: recruit candidate id blank or repeated; skipped", "roomid", roomID, "id", id)
				continue
			case !templateOK || templateID <= 0:
				mudlog.Warn("company: recruit candidate without a mob template; skipped", "roomid", roomID, "id", id)
				continue
			case !priceOK || price < 0:
				mudlog.Warn("company: recruit candidate with a bad price; skipped", "roomid", roomID, "id", id)
				continue
			}
			seen[id] = true
			rec.Candidates = append(rec.Candidates, candidate{ID: id, MobTemplateID: templateID, Price: price, Tutorial: tutorial})
		}
		out[roomID] = rec
	}
	return out
}

func (m *CompanyModule) recruiters() map[int]recruiter {
	if m.recruitersForTest != nil {
		return m.recruitersForTest
	}
	if m.plug != nil {
		return parseRecruiters(m.plug.Config.Get("Recruiters"))
	}
	return nil
}

// matchCandidate finds a candidate by id, then exact name, then a unique
// name substring.
func matchCandidate(rec recruiter, selector string) (candidate, bool) {
	selector = strings.ToLower(strings.TrimSpace(selector))
	if selector == "" {
		return candidate{}, false
	}
	for _, c := range rec.Candidates {
		if c.ID == selector {
			return c, true
		}
	}
	var exact, partial []candidate
	for _, c := range rec.Candidates {
		name := strings.ToLower(templateName(c.MobTemplateID, ""))
		if name == "" {
			continue
		}
		if name == selector {
			exact = append(exact, c)
		} else if strings.Contains(name, selector) {
			partial = append(partial, c)
		}
	}
	if len(exact) == 1 {
		return exact[0], true
	}
	if len(exact) == 0 && len(partial) == 1 {
		return partial[0], true
	}
	return candidate{}, false
}

// resolveCandidate is who a selector means at a recruiter (Phase 32a2): a
// generated candidate by key or exact name, then a regular by id or exact
// name, then part of a name across both lists together, refused when it
// fits more than one. At most one result is set.
func resolveCandidate(rec recruiter, roster domain.Roster, selector string) (*candidate, *domain.Candidate) {
	sel := strings.ToLower(strings.TrimSpace(selector))
	if g, ok := findGenerated(roster, sel, false); ok {
		return nil, &g
	}
	c, ok := matchCandidate(rec, sel)
	if ok && (c.ID == sel || strings.ToLower(templateName(c.MobTemplateID, "")) == sel) {
		return &c, nil
	}
	matches := 0
	for _, r := range rec.Candidates {
		if strings.Contains(strings.ToLower(templateName(r.MobTemplateID, "")), sel) {
			matches++
		}
	}
	for _, g := range roster.Candidates {
		if strings.Contains(strings.ToLower(g.Name), sel) {
			matches++
		}
	}
	if matches != 1 {
		return nil, nil
	}
	if ok {
		return &c, nil
	}
	if g, ok := findGenerated(roster, sel, true); ok {
		return nil, &g
	}
	return nil, nil
}

// listing shows every candidate here with what a player needs to choose.
func (m *CompanyModule) listing(leaderUserID int, rec recruiter) string {
	roster, _ := m.rosterFor(leaderUserID, rec)
	if len(rec.Candidates) == 0 && len(roster.Candidates) == 0 {
		return fmt.Sprintf("No one on %s is looking for work right now.", rec.Name)
	}
	record, _ := m.registry.Get(leaderUserID)
	lines := []string{fmt.Sprintf("%s lists:", capitalize(rec.Name))}
	for _, c := range rec.Candidates {
		name := templateName(c.MobTemplateID, c.ID)
		state, ok := m.runtime.TemplateState(c.MobTemplateID)
		if !ok {
			lines = append(lines, fmt.Sprintf("  %s (company recruit %s): not available right now.", name, c.ID))
			continue
		}
		lines = append(lines, fmt.Sprintf("  %s (company recruit %s): %s, level %d, alignment %s",
			name, c.ID, archetypeLabel(m.companionArchetypes()[c.MobTemplateID]), state.Level,
			alignmentLabel(m.alignmentWorld().TemplateAlignment(c.MobTemplateID))))
		lines = append(lines, "    Gear: "+gearSummary(state))
		switch {
		case c.Tutorial && record.HasClaimed(c.MobTemplateID):
			lines = append(lines, "    Price: already claimed (once only)")
		case c.Tutorial:
			lines = append(lines, "    Price: free, once only")
		default:
			lines = append(lines, fmt.Sprintf("    Price: %d gold", c.Price))
		}
	}
	// Phase 32a2: the viewer's own generated candidates, after the regulars.
	for _, c := range roster.Candidates {
		lines = append(lines, m.generatedListing(c)...)
	}
	lines = append(lines, fmt.Sprintf("Your company: %d/%d companions.", len(record.Companions), m.maxCompanions()))
	return strings.Join(lines, "\n")
}

func gearSummary(state domain.MemberState) string {
	names := []string{}
	for _, slot := range characters.AllSlots() {
		if itm := state.Equipment.Get(slot); itm != nil && itm.ItemId > 0 {
			names = append(names, itemName(*itm))
		}
	}
	for _, itm := range state.Items {
		names = append(names, itemName(itm))
	}
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// recruit lists the candidates here, or takes one on. Every refusal leaves
// gold, the record, and survival unchanged. Gold is taken only after the
// company save, so a failed save costs nothing; the user is saved at once
// after so the company and user files move together.
func (m *CompanyModule) recruit(user *users.UserRecord, roomID int, selector string) (string, error) {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error(), nil
	}
	// Phase 27a: an ephemeral copy (the tutorial's) uses its template
	// room's recruiter.
	rec, ok := m.recruiters()[rooms.GetOriginalRoom(roomID)]
	if !ok {
		return "No one here is hiring. Look for a recruiter in a settlement.", nil
	}
	selector = strings.TrimSpace(selector)
	if selector == "" || selector == "list" {
		return m.listing(user.UserId, rec), nil
	}
	roster, _ := m.rosterFor(user.UserId, rec)
	regular, g := resolveCandidate(rec, roster, selector)
	if g != nil {
		return m.hireGenerated(user, roomID, roster, *g)
	}
	if regular == nil {
		return fmt.Sprintf(`No one called "%s" is hiring here. Type "company recruit" to see who is.`, selector), nil
	}
	c := *regular
	name := templateName(c.MobTemplateID, c.ID)
	if _, ok := m.runtime.TemplateState(c.MobTemplateID); !ok {
		return fmt.Sprintf("%s isn't available right now.", name), nil
	}
	record, _ := m.registry.Get(user.UserId)
	if c.Tutorial && record.HasClaimed(c.MobTemplateID) {
		return fmt.Sprintf("You've already taken %s on once; that offer doesn't come twice.", name), nil
	}
	if len(record.Companions) >= m.maxCompanions() {
		return fmt.Sprintf("Your company is full (%d/%d companions). Dismiss someone first.", len(record.Companions), m.maxCompanions()), nil
	}
	if refusal := m.recruitRefusal(user.UserId, c.MobTemplateID, name); refusal != "" {
		return refusal, nil
	}
	price := c.cost()
	if user.Character.Gold < price {
		return fmt.Sprintf("%s asks %d gold, and you have %d.", name, price, user.Character.Gold), nil
	}
	companion, err := m.enlist(user.UserId, roomID, c.MobTemplateID, map[int]struct{}{c.MobTemplateID: {}}, c.Tutorial, nil)
	if err != nil {
		return "", err
	}
	text := fmt.Sprintf("%s joins your company (#%d).", name, companion.ID)
	if price > 0 {
		m.chargeGold(user, price)
		text = fmt.Sprintf("You pay %d gold. %s", price, text)
	}
	return text + ` Place them with "formation move".`, nil
}

func nativeSaveUser(user *users.UserRecord) error { return users.SaveUserAtomic(*user) }

var _ domain.RecruiterViewProvider = (*CompanyModule)(nil)

// noticeEntry is one candidate as a recruiter room's notice shows it to one
// viewer (Phase 32a).
type noticeEntry struct {
	Name    string
	Price   int
	Free    bool
	Refused bool
}

// noticeLines renders a recruiter's notice: the candidates on it, or that
// no one is left for this viewer.
// full is the viewer's companion count when their company is full (0
// otherwise), with its limit.
func noticeLines(noticeName string, entries []noticeEntry, full, limit int) []string {
	if len(entries) == 0 {
		return []string{fmt.Sprintf("No one on %s is looking for work with you now.", noticeName)}
	}
	parts := make([]string, 0, len(entries))
	for _, e := range entries {
		label := fmt.Sprintf("%d gold", e.Price)
		switch {
		case e.Refused:
			label = "won't join you"
		case e.Free:
			label = "free"
		}
		parts = append(parts, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> (%s)`, e.Name, label))
	}
	lines := []string{
		fmt.Sprintf("On %s: %s.", noticeName, strings.Join(parts, ", ")),
		`  Type <ansi fg="command">company recruit</ansi> to see them, or <ansi fg="command">company inspect [name]</ansi>.`,
	}
	if full > 0 {
		lines = append(lines, fmt.Sprintf("  Your company is full (%d/%d companions); dismiss someone to take another on.", full, limit))
	}
	return lines
}

// RecruiterLines implements domain.RecruiterViewProvider: the notice in a
// recruiter room, less the candidates already in the viewer's company and
// the once-only offers they've already taken. One the company would refuse
// is listed as such.
func (m *CompanyModule) RecruiterLines(viewerUserID, roomID int) []string {
	rec, ok := m.recruiters()[rooms.GetOriginalRoom(roomID)]
	if !ok {
		return nil
	}
	record, _ := m.registry.Get(viewerUserID)
	inCompany := map[int]bool{}
	for _, c := range record.Companions {
		inCompany[c.MobTemplateID] = true
	}
	entries := []noticeEntry{}
	for _, c := range rec.Candidates {
		if inCompany[c.MobTemplateID] || (c.Tutorial && record.HasClaimed(c.MobTemplateID)) {
			continue
		}
		if _, ok := m.runtime.TemplateState(c.MobTemplateID); !ok {
			continue
		}
		name := templateName(c.MobTemplateID, c.ID)
		entries = append(entries, noticeEntry{
			Name:    name,
			Price:   c.cost(),
			Free:    c.Tutorial,
			Refused: m.recruitRefusal(viewerUserID, c.MobTemplateID, name) != "",
		})
	}
	// Phase 32a2: then the viewer's own generated candidates.
	roster, _ := m.rosterFor(viewerUserID, rec)
	for _, c := range roster.Candidates {
		entries = append(entries, noticeEntry{
			Name:    c.Name,
			Price:   c.Price,
			Refused: m.alignmentRefusal(viewerUserID, c.Alignment, c.Name) != "",
		})
	}
	full := 0
	if len(record.Companions) >= m.maxCompanions() {
		full = len(record.Companions)
	}
	return noticeLines(rec.Name, entries, full, m.maxCompanions())
}

// LookCandidate implements domain.RecruiterViewProvider: "look <name>" at
// a candidate on the notice.
func (m *CompanyModule) LookCandidate(viewerUserID, roomID int, selector string) (string, bool) {
	rec, ok := m.recruiters()[rooms.GetOriginalRoom(roomID)]
	if !ok {
		return "", false
	}
	// "look post", "look hiring slate": the notice itself.
	if sel := strings.ToLower(strings.TrimSpace(selector)); len(sel) >= 3 && strings.Contains(strings.ToLower(rec.Name), sel) {
		return strings.Join(m.RecruiterLines(viewerUserID, roomID), "\n"), true
	}
	roster, _ := m.rosterFor(viewerUserID, rec)
	regular, g := resolveCandidate(rec, roster, selector)
	if g != nil {
		return lookGenerated(rec.Name, *g), true
	}
	if regular == nil {
		return "", false
	}
	c := *regular
	name := templateName(c.MobTemplateID, c.ID)
	lines := []string{fmt.Sprintf(`You read about <ansi fg="mobname">%s</ansi> on %s.`, name, rec.Name)}
	if spec := mobs.GetMobSpec(mobs.MobId(c.MobTemplateID)); spec != nil && spec.Character.Description != "" {
		lines = append(lines, spec.Character.Description)
	}
	lines = append(lines, fmt.Sprintf(`Type <ansi fg="command">company inspect %s</ansi> to weigh them against your company.`, c.ID))
	return strings.Join(lines, "\n"), true
}
