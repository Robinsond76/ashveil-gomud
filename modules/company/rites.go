package company

// Phase 74: rites for the dead. A companion lost for good, or one who leaves
// after long service, leaves an occasion on the leader's record (queued in
// the same save as the departure). The next camp or inn stay announces it;
// the leader holds the rites there or lets them pass. Holding gives a
// chronicle line, steadies the companions who trusted the one gone and
// draws those close together; skipping costs the company's loyalty. Every
// loyalty move stays inside the rules' floor and ceiling and nothing here
// gives gold, experience or power (internal/rites).

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/bonds"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/chronicle"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rites"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

var (
	_ domain.RitesProvider = (*CompanyModule)(nil)
	_ domain.RiteViewer    = (*CompanyModule)(nil)
)

const ritesUsage = "Usage: rites | rite hold [member] | rite skip [member] (a number #N or a name; `all` for every one). See help rites."

// riteBusy is why rites cannot be held now, or "". Rites are held at a camp
// or an inn stay, never in a fight. Tests replace it.
func (m *CompanyModule) riteBusy(user *users.UserRecord) string {
	if m.riteSeam != nil {
		return m.riteSeam(user)
	}
	if _, busy := battle.Current(user.UserId); busy {
		return "Not in the middle of a battle."
	}
	if _, here := camping.LeaderRest(user.UserId); !here {
		return "Rites are held at a camp or an inn: make camp (camp) or take a room (inn) first. See help rites."
	}
	return ""
}

// mournerSets splits the living company into those who trusted the
// departed (a bond at "trusts" or better) and everyone else.
func mournerSets(record domain.Record, departed int) (closeIDs, band []int) {
	for _, c := range record.Companions {
		if c.ID == departed || !bondMember(c) {
			continue
		}
		if bond, ok := record.BondOf(departed, c.ID); ok && bonds.IsFriend(bond.Value) {
			closeIDs = append(closeIDs, c.ID)
		} else {
			band = append(band, c.ID)
		}
	}
	return closeIDs, band
}

// queueRite puts a departing companion's occasion on the record, in memory:
// the departure's own save writes it, so the two are one write. It reports
// the occasion's name, or "" when the loss is not mourned. Call it before
// the companion is dropped (their service and bonds end with them).
func (m *CompanyModule) queueRite(leaderUserID int, c domain.Companion, cause rites.Cause) string {
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return ""
	}
	rounds := 0
	if s, ok := record.FindService(domain.CompanionMemberKey(c.ID)); ok {
		rounds = s.Rounds
	}
	if !rites.Mourns(cause, rounds) {
		return ""
	}
	closeIDs, band := mournerSets(record, c.ID)
	rite := domain.Rite{
		Op: domain.RiteOp(c.ID, string(cause)), Companion: c.ID, MobTemplateID: c.MobTemplateID,
		Name: companionName(c), Cause: string(cause), Close: closeIDs, Band: band, At: m.now().Unix(),
	}
	if c.State != nil {
		rite.Level = c.State.Level
	}
	if !record.QueueRite(rite) {
		return rite.Op
	}
	m.registry.Put(record)
	return rite.Op
}

// unqueueRite takes back a queued occasion whose departure failed.
func (m *CompanyModule) unqueueRite(leaderUserID int, op string) {
	if op == "" {
		return
	}
	if record, ok := m.registry.Get(leaderUserID); ok && record.RemoveRite(op) {
		m.registry.Put(record)
	}
}

// riteHint is the sentence that points a departure at the rites it left, or
// "" when none are waiting.
func (m *CompanyModule) riteHint(leaderUserID, companionID int) string {
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return ""
	}
	if _, waiting := record.RiteOf(companionID); !waiting {
		return ""
	}
	return " You can hold rites for them at your next camp or inn (rites)."
}

// settleRite applies one rite's outcome to the record: loyalty through the
// real disposition, within the rules' floor and ceiling, and the occasion
// removed. It returns the lines to show.
func (m *CompanyModule) settleRite(record *domain.Record, rite domain.Rite, held bool) []string {
	var lines []string
	touch := func(ids []int, close bool) {
		for _, id := range ids {
			for i, c := range record.Companions {
				if c.ID != id || c.Dead() || c.MoraleDesert || c.Disposition == nil {
					continue
				}
				d := *c.Disposition
				var next int
				if held {
					next = rites.Steady(d.Loyalty, close)
				} else {
					next = rites.Grieve(d.Loyalty, close)
				}
				if next == d.Loyalty {
					continue
				}
				delta := next - d.Loyalty
				d.Loyalty = next
				record.Companions[i].Disposition = &d
				if held {
					lines = append(lines, fmt.Sprintf("%s is steadied by it. (loyalty %+d)", companionName(c), delta))
				} else {
					lines = append(lines, fmt.Sprintf("%s takes it hard. (loyalty %+d)", companionName(c), delta))
				}
			}
		}
	}
	touch(rite.Close, true)
	touch(rite.Band, false)
	record.RemoveRite(rite.Op)
	return lines
}

// riteDeed is the chronicle's line for a settled rite.
func riteDeed(rite domain.Rite, held bool) chronicle.Entry {
	ref := "rite:skipped"
	if held {
		ref = "rite:held"
	}
	return chronicle.Entry{Kind: chronicle.Rites, Subject: rite.Name, Detail: rites.Phrase(rites.Cause(rite.Cause)), Ref: ref}
}

// riteIntro is the leader's own line for a held or skipped rite.
func riteIntro(rite domain.Rite, held bool) string {
	what := rites.Verb(rites.Cause(rite.Cause))
	if held {
		return fmt.Sprintf("You gather the company at the fire and hold rites for %s, who %s. Names are said; no one speaks long.", rite.Name, what)
	}
	return fmt.Sprintf("You let %s go without a word. The company notices.", rite.Name)
}

// settleSaved applies the given rites as held or skipped in one save with a
// rollback, then records their deeds and, for a held rite, draws the close
// companions together. It returns the text to show.
func (m *CompanyModule) settleSaved(leaderUserID int, record domain.Record, chosen []domain.Rite, held bool) (string, error) {
	var lines []string
	for _, rite := range chosen {
		lines = append(lines, riteIntro(rite, held))
		lines = append(lines, m.settleRite(&record, rite, held)...)
	}
	before := m.registry.Clone()
	m.registry.Put(record)
	if err := m.save(); err != nil {
		m.registry = before
		return "", err
	}
	for _, rite := range chosen {
		chronicle.Record(leaderUserID, riteDeed(rite, held))
		if held {
			m.drawTogether(leaderUserID, rite)
		}
	}
	return strings.Join(lines, "\n"), nil
}

// drawTogether is a held rite's gift to the companions who trusted the one
// gone: each pair of them a little closer (bonds.Rite, with its own cooldown
// and the usual limit at "close").
func (m *CompanyModule) drawTogether(leaderUserID int, rite domain.Rite) {
	var changes []bondChange
	for i, a := range rite.Close {
		for _, b := range rite.Close[i+1:] {
			changes = append(changes, bondChange{a, b, bonds.Rite, rites.BondGain})
		}
	}
	if err := m.applyBonds(leaderUserID, changes); err != nil {
		mudlog.Warn("company: rite bonds", "leader", leaderUserID, "error", err)
	}
}

// OfferRites implements company.RitesProvider: a camp or an inn stay has
// begun. Rites a earlier stay announced and the leader never answered pass
// (with their cost), and the rest are announced. It returns text for the
// stay's opening, "" when no rite waits.
func (m *CompanyModule) OfferRites(leaderUserID int) string {
	if m.persistenceAvailable() != nil {
		return ""
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok || len(record.Rites) == 0 {
		return ""
	}
	var passed []domain.Rite
	var fresh []string
	for i := range record.Rites {
		if record.Rites[i].Offered {
			passed = append(passed, record.Rites[i])
		}
	}
	var text []string
	if len(passed) > 0 {
		// The earlier unanswered rites pass first, in one save.
		said, err := m.settleSaved(leaderUserID, record, passed, false)
		if err != nil {
			mudlog.Warn("company: rites passed", "leader", leaderUserID, "error", err)
			return ""
		}
		text = append(text, said)
		record, _ = m.registry.Get(leaderUserID)
	}
	for i := range record.Rites {
		record.Rites[i].Offered = true
		fresh = append(fresh, fmt.Sprintf("  %s (%s)", record.Rites[i].Name, rites.Phrase(rites.Cause(record.Rites[i].Cause))))
	}
	if len(fresh) > 0 {
		before := m.registry.Clone()
		m.registry.Put(record)
		if err := m.save(); err != nil {
			m.registry = before
			mudlog.Warn("company: rites offered", "leader", leaderUserID, "error", err)
			return strings.Join(text, "\n")
		}
		text = append(text, "The company has not yet mourned:\n"+strings.Join(fresh, "\n")+"\nHold the rites at this fire (rite hold) or let them pass (rite skip). If they pass unspoken at the next camp, the company will feel it. See help rites.")
	}
	return strings.Join(text, "\n")
}

// riteMatches picks the rites a selector names: "#N" or a number by the
// departed's companion number, a name (any letter case), or `all`. An empty
// selector names the only one waiting.
func riteMatches(record domain.Record, selector string) ([]domain.Rite, string) {
	selector = strings.ToLower(strings.TrimSpace(selector))
	switch {
	case len(record.Rites) == 0:
		return nil, "No one is waiting to be mourned."
	case selector == "all":
		return append([]domain.Rite(nil), record.Rites...), ""
	case selector == "":
		if len(record.Rites) > 1 {
			return nil, "More than one is waiting; name one (rites) or say `all`."
		}
		return []domain.Rite{record.Rites[0]}, ""
	}
	if n, err := strconv.Atoi(strings.TrimPrefix(selector, "#")); err == nil {
		if rite, ok := record.RiteOf(n); ok {
			return []domain.Rite{rite}, ""
		}
		return nil, "No one by that number is waiting to be mourned."
	}
	var found []domain.Rite
	for _, rite := range record.Rites {
		if strings.EqualFold(rite.Name, selector) {
			found = append(found, rite)
		}
	}
	if len(found) == 0 {
		for _, rite := range record.Rites {
			if strings.HasPrefix(strings.ToLower(rite.Name), selector) {
				found = append(found, rite)
			}
		}
	}
	switch len(found) {
	case 0:
		return nil, "No one by that name is waiting to be mourned."
	case 1:
		return found, ""
	}
	return nil, "More than one answers to that; use their number (#N)."
}

// ritesView lists what waits.
func (m *CompanyModule) ritesView(leaderUserID int) string {
	record, ok := m.registry.Get(leaderUserID)
	if !ok || len(record.Rites) == 0 {
		return "No one is waiting to be mourned."
	}
	lines := []string{"Waiting to be mourned (rite hold [member] at a camp or an inn, or rite skip):"}
	for _, rite := range record.Rites {
		line := fmt.Sprintf("  #%d %s, level %d: %s", rite.Companion, rite.Name, max(rite.Level, 1), rites.Phrase(rites.Cause(rite.Cause)))
		if n := len(rite.Close); n > 0 {
			line += fmt.Sprintf("; %d trusted them", n)
		}
		if rite.Offered {
			line += " (offered: it passes at the next camp or inn if you say nothing)"
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// ritesCommand is `rites`, `rite hold` and `rite skip`.
func (m *CompanyModule) ritesCommand(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	args := strings.Fields(strings.TrimSpace(rest))
	if len(args) == 0 || strings.EqualFold(args[0], "list") || strings.EqualFold(args[0], "status") {
		user.SendText(m.ritesView(user.UserId))
		return true, nil
	}
	var held bool
	switch strings.ToLower(args[0]) {
	case "hold", "mourn", "honor", "honour":
		held = true
	case "skip", "pass", "ignore":
	default:
		user.SendText(ritesUsage)
		return true, nil
	}
	user.SendText(m.ritesAct(user, strings.Join(args[1:], " "), held))
	return true, nil
}

// ritesAct holds or skips the rites a selector names.
func (m *CompanyModule) ritesAct(user *users.UserRecord, selector string, held bool) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	record, ok := m.registry.Get(user.UserId)
	if !ok {
		return "No one is waiting to be mourned."
	}
	chosen, refusal := riteMatches(record, selector)
	if refusal != "" {
		return refusal
	}
	if busy := m.riteBusy(user); busy != "" {
		return busy
	}
	text, err := m.settleSaved(user.UserId, record, chosen, held)
	if err != nil {
		mudlog.Warn("company: rite", "leader", user.UserId, "error", err)
		return "The rite could not be kept: " + err.Error()
	}
	return text
}

// RitePanel implements company.RiteViewer: the web client's Rites tab.
func (m *CompanyModule) RitePanel(leaderUserID int) (domain.RitePanel, bool) {
	if m.persistenceAvailable() != nil {
		return domain.RitePanel{}, false
	}
	panel := domain.RitePanel{Rows: []domain.RiteRow{}}
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return panel, true
	}
	if user := users.GetByUserId(leaderUserID); user != nil {
		if busy := m.riteBusy(user); busy != "" {
			panel.Where = busy
		} else {
			panel.Here = true
			panel.Where = "You can hold rites here."
		}
	}
	for _, rite := range record.Rites {
		names := []string{}
		for _, id := range rite.Close {
			if c, ok := findCompanion(record, id); ok {
				names = append(names, companionName(c))
			}
		}
		panel.Rows = append(panel.Rows, domain.RiteRow{
			ID: rite.Companion, Name: rite.Name, Level: max(rite.Level, 1), Cause: rites.Phrase(rites.Cause(rite.Cause)),
			Close: names, Offered: rite.Offered,
		})
	}
	return panel, true
}
