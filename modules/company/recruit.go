package company

// Phase 22c: settlement recruiters. A recruiter is a configured room that
// offers authored companion candidates, free once (tutorial) or for gold.
// See modules/company/AGENTS.md and the Phase 22c design.

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
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

// listing shows every candidate here with what a player needs to choose.
func (m *CompanyModule) listing(leaderUserID int, rec recruiter) string {
	if len(rec.Candidates) == 0 {
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
	c, ok := matchCandidate(rec, selector)
	if !ok {
		return fmt.Sprintf(`No one called "%s" is hiring here. Type "company recruit" to see who is.`, selector), nil
	}
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
	companion, err := m.enlist(user.UserId, roomID, c.MobTemplateID, map[int]struct{}{c.MobTemplateID: {}}, c.Tutorial)
	if err != nil {
		return "", err
	}
	text := fmt.Sprintf("%s joins your company (#%d).", name, companion.ID)
	if price > 0 {
		user.Character.Gold -= price
		events.AddToQueue(events.EquipmentChange{UserId: user.UserId, GoldChange: -price})
		if m.saveUser != nil {
			// The gold is in memory and goes out with the next autosave,
			// logout, or copyover save if this one fails.
			if err := m.saveUser(user); err != nil {
				mudlog.Error("company: save after recruit", "user", user.UserId, "error", err)
			}
		}
		text = fmt.Sprintf("You pay %d gold. %s", price, text)
	}
	return text + ` Place them with "formation move".`, nil
}

func nativeSaveUser(user *users.UserRecord) error { return users.SaveUser(*user) }
