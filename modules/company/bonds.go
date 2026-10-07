package company

// Phase 65: bonds between companions. Each pair of companions has one saved
// value (internal/bonds) that rises with time spent and dangers faced
// together, with agreeing over the leader's choices and with one saving the
// other, and falls with clashing temperaments, opinions and a refused guard.
// Friends step in for each other in battle; a rival won't. A rivalry pushed
// to its end warns the leader, then one of the pair leaves. Nothing here
// gives gold, experience or power, and every source has a cooldown and a
// limit, so a bond cannot be farmed.

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/banter"
	"github.com/GoMudEngine/GoMud/internal/bonds"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/creatures"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

var (
	_ domain.BondProvider = (*CompanyModule)(nil)
	_ domain.BondViewer   = (*CompanyModule)(nil)
)

// bondMember reports whether a companion takes part in bonds: a living
// person with the company, not a creature or construct.
func bondMember(c domain.Companion) bool {
	return !(c.Dead() || c.PendingReturn || c.MoraleDesert || bound(c) || creatures.Is(c.Archetype))
}

// bondChange is one change to one pair's bond.
type bondChange struct {
	a, b   int
	source bonds.Source
	delta  int
}

// BondValue implements company.BondProvider. Combat calls it on every
// guard decision, so it reads the registry in place.
func (m *CompanyModule) BondValue(leaderUserID int, a, b domain.MemberKey) int {
	ia, okA := domain.CompanionIDFromMemberKey(a)
	ib, okB := domain.CompanionIDFromMemberKey(b)
	if !okA || !okB || ia == ib {
		return 0
	}
	ia, ib = domain.BondPair(ia, ib)
	for _, bond := range m.registry.Companies[leaderUserID].Bonds {
		if bond.A == ia && bond.B == ib {
			return bond.Value
		}
	}
	return 0
}

// BondEvent implements company.BondProvider: one member stepped in for
// another, or a rival let the other take the blow.
func (m *CompanyModule) BondEvent(leaderUserID int, a, b domain.MemberKey, source bonds.Source) {
	ia, okA := domain.CompanionIDFromMemberKey(a)
	ib, okB := domain.CompanionIDFromMemberKey(b)
	if !okA || !okB || ia == ib {
		return
	}
	delta := 0
	switch source {
	case bonds.Rescue:
		delta = bonds.RescueGain
	case bonds.Refusal:
		delta = -bonds.RefusalLoss
	default:
		return
	}
	if err := m.applyBonds(leaderUserID, []bondChange{{ia, ib, source, delta}}); err != nil {
		mudlog.Warn("company: bond event", "leader", leaderUserID, "error", err)
	}
}

// bondSign reports the sign of a pair's bond for banter: 1 for friends,
// -1 for rivals.
func bondSign(value int) int {
	switch {
	case bonds.IsFriend(value):
		return 1
	case bonds.IsRival(value):
		return -1
	}
	return 0
}

// applyBonds applies changes to the leader's pairs and saves once. A change
// that is inside its source's cooldown for that pair, or between two who are
// not both company people, does nothing. Deepening feelings are announced,
// a rivalry at its end is warned of, and a warned rivalry at its end sends
// one of the pair away (the one with less loyalty).
func (m *CompanyModule) applyBonds(leaderUserID int, changes []bondChange) error {
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok || len(changes) == 0 {
		return nil
	}
	now := m.now().Unix()
	var lines []string
	var leavers []int
	touched := false
	for _, ch := range changes {
		a, b := domain.BondPair(ch.a, ch.b)
		ca, okA := findCompanion(record, a)
		cb, okB := findCompanion(record, b)
		if a == b || !okA || !okB || !bondMember(ca) || !bondMember(cb) || ch.delta == 0 {
			continue
		}
		bond, _ := record.BondOf(a, b)
		src := string(ch.source)
		if last, ok := bond.At[src]; ok && now-last < int64(bonds.CooldownOf(ch.source).Seconds()) {
			continue
		}
		old := bond.Value
		bond.Value = bonds.Apply(old, ch.delta, ch.source)
		changed := bond.Value != old
		if changed {
			if bond.At == nil {
				bond.At = map[string]int64{}
			}
			bond.At[src] = now
		}
		nameA, nameB := companionName(ca), companionName(cb)
		oldTier, newTier := bonds.TierOf(old), bonds.TierOf(bond.Value)
		if changed && deepens(oldTier, newTier) {
			lines = append(lines, fmt.Sprintf("%s and %s %s. (bond %+d)", nameA, nameB, bonds.Together(bond.Value), bond.Value))
		}
		warn, leave, mended := bonds.Warning(bond.Value, bond.Warned)
		if leave && !changed && ch.source != bonds.Opinion && ch.source != bonds.Refusal {
			// Phase 65 review: a warned pair held at the bottom leaves on a
			// real clash, never on a camp or a battle that could not move it.
			leave = false
		}
		switch {
		case warn:
			bond.Warned = true
			lines = append(lines, fmt.Sprintf("%s and %s can hardly stand to share a company. If it goes on, one of them will leave. (help bonds)", nameA, nameB))
		case mended:
			bond.Warned = false
		case leave:
			leaver, stayer := ca, cb
			if lessLoyal(cb, ca) {
				leaver, stayer = cb, ca
			}
			leavers = append(leavers, leaver.ID)
			lines = append(lines, fmt.Sprintf("%s has had enough of %s.", companionName(leaver), companionName(stayer)))
			for i, c := range record.Companions {
				if c.ID == leaver.ID {
					d := domain.Disposition{Alignment: m.companionAlignment(c), Loyalty: 0}
					if c.Disposition != nil {
						d = *c.Disposition
						d.Loyalty = 0
					}
					record.Companions[i].Disposition = &d
					record.Companions[i].MoraleDesert = true
				}
			}
			touched = true
		}
		if changed || warn || mended {
			record.SetBond(bond)
			touched = true
		}
	}
	if !touched {
		return nil
	}
	// A failed save leaves the registry exactly as it was: no cooldown,
	// warning or leaving that was never saved (Phase 65 review).
	before := m.registry.Clone()
	m.registry.Put(record)
	if err := m.save(); err != nil {
		m.registry = before
		return err
	}
	for _, line := range lines {
		m.alignmentWorld().Tell(leaderUserID, line)
	}
	if len(leavers) > 0 {
		return m.moraleDepartures(leaderUserID)
	}
	return nil
}

// deepens reports whether a bond moved to a deeper tier of the same feeling
// (or across to the other sign): worth saying. Easing back is silent.
func deepens(from, to int) bool {
	if to == 0 || from == to {
		return false
	}
	if from == 0 || (from > 0) != (to > 0) {
		return true
	}
	if to > 0 {
		return to > from
	}
	return to < from
}

// lessLoyal reports whether a is the likelier of the two to leave: lower
// loyalty, and for a tie the newer companion.
func lessLoyal(a, b domain.Companion) bool {
	la, lb := domain.MaxLoyalty, domain.MaxLoyalty
	if a.Disposition != nil {
		la = a.Disposition.Loyalty
	}
	if b.Disposition != nil {
		lb = b.Disposition.Loyalty
	}
	if la != lb {
		return la < lb
	}
	return a.ID > b.ID
}

// bondPairs lists every pair among the given companions, once.
func bondPairs(members []domain.Companion, f func(a, b domain.Companion)) {
	for i := range members {
		for j := i + 1; j < len(members); j++ {
			f(members[i], members[j])
		}
	}
}

// pairAffinity is a camp's worth to two companions.
func (m *CompanyModule) pairAffinity(a, b domain.Companion) int {
	return bonds.Affinity(personalityOf(a), personalityOf(b), m.companionAlignment(a), m.companionAlignment(b))
}

// bondsCampRest is a camp rest that has ended: every pair of companions who
// camped together is a little closer or a little colder, by how well their
// temperaments and alignments suit. It runs whether or not the leader has
// banter on.
func (m *CompanyModule) bondsCampRest(leaderUserID int) {
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return
	}
	var camped []domain.Companion
	for _, c := range record.Companions {
		if bondMember(c) && !c.Separated() {
			camped = append(camped, c)
		}
	}
	var changes []bondChange
	bondPairs(camped, func(a, b domain.Companion) {
		changes = append(changes, bondChange{a.ID, b.ID, bonds.Camp, m.pairAffinity(a, b)})
	})
	if err := m.applyBonds(leaderUserID, changes); err != nil {
		mudlog.Warn("company: camp bonds", "leader", leaderUserID, "error", err)
	}
}

// bondsBattleWon is a won battle: every pair of companions who stood to the
// end gains a point, unless their temperaments rub.
func (m *CompanyModule) bondsBattleWon(leaderUserID int) {
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return
	}
	views, ok := domain.CompanyMembers(leaderUserID)
	if !ok {
		return
	}
	standing := map[int]bool{}
	for _, v := range views {
		if v.Status == domain.MemberPresent && v.HP >= 1 {
			standing[v.ID] = true
		}
	}
	var fought []domain.Companion
	for _, c := range record.Companions {
		if standing[c.ID] && bondMember(c) {
			fought = append(fought, c)
		}
	}
	var changes []bondChange
	bondPairs(fought, func(a, b domain.Companion) {
		changes = append(changes, bondChange{a.ID, b.ID, bonds.Battle, bonds.BattleGain(m.pairAffinity(a, b))})
	})
	if err := m.applyBonds(leaderUserID, changes); err != nil {
		mudlog.Warn("company: battle bonds", "leader", leaderUserID, "error", err)
	}
}

// onOpinionBonds is the Phase 64 seam: two companions who both liked (or
// both disliked) the leader's choice grow closer, and two who split over it
// grow apart.
func (m *CompanyModule) onOpinionBonds(leaderUserID int, _ opinions.Choice, reactions []opinions.Reaction) {
	var changes []bondChange
	for i := range reactions {
		for j := i + 1; j < len(reactions); j++ {
			a, b := reactions[i], reactions[j]
			changes = append(changes, bondChange{a.CompanionID, b.CompanionID, bonds.Opinion, bonds.OpinionDelta(a.Verdict, b.Verdict)})
		}
	}
	if err := m.applyBonds(leaderUserID, changes); err != nil {
		mudlog.Warn("company: opinion bonds", "leader", leaderUserID, "error", err)
	}
}

// bondSigns is banter's view of the leader's bonds: 1 for friends, -1 for
// rivals, by companion number.
func (m *CompanyModule) bondSigns(leaderUserID int) func(a, b int) int {
	record, ok := m.registry.Get(leaderUserID)
	if !ok || len(record.Bonds) == 0 {
		return nil
	}
	return func(a, b int) int {
		bond, _ := record.BondOf(a, b)
		return bondSign(bond.Value)
	}
}

// bondsFromTalk moves a pair a point when they talked about each other: a
// friend line warms them, a rival line sours them.
func (m *CompanyModule) bondsFromTalk(leaderUserID int, said []banter.Said) {
	var ids []int
	delta := 0
	for _, s := range said {
		switch s.Ctx {
		case banter.CtxFriend:
			delta = 1
		case banter.CtxRival:
			delta = -1
		default:
			continue
		}
		if !slices.Contains(ids, s.Member) {
			ids = append(ids, s.Member)
		}
	}
	if delta == 0 || len(ids) < 2 {
		return
	}
	if err := m.applyBonds(leaderUserID, []bondChange{{ids[0], ids[1], bonds.Talk, delta}}); err != nil {
		mudlog.Warn("company: talk bonds", "leader", leaderUserID, "error", err)
	}
}

// bondEffect is what a pair's bond does in a battle, in words.
func bondEffect(value int, warned bool) string {
	switch {
	case value >= bonds.KinAt:
		return fmt.Sprintf("Each steps in twice a battle for the other when hurt (at %d%% health or less).", bonds.GuardBelowPct)
	case value >= bonds.FriendAt:
		return fmt.Sprintf("Each steps in once a battle for the other when hurt (at %d%% health or less).", bonds.GuardBelowPct)
	case value <= bonds.RivalAt && warned:
		return "Won't guard each other: set a guardian's ward to someone else (help guardian). One of them will leave if this goes on."
	case value <= bonds.RivalAt:
		return "Won't guard each other: set a guardian's ward to someone else (help guardian)."
	}
	return ""
}

// BondPanel implements company.BondViewer: every pair of the leader's
// companions, strongest feelings first, and each companion's view.
func (m *CompanyModule) BondPanel(leaderUserID int) (domain.BondPanel, bool) {
	if m.persistenceAvailable() != nil {
		return domain.BondPanel{}, false
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return domain.BondPanel{}, false
	}
	var people []domain.Companion
	for _, c := range record.Companions {
		if !c.Dead() && !bound(c) && !creatures.Is(c.Archetype) {
			people = append(people, c)
		}
	}
	panel := domain.BondPanel{Pairs: []domain.BondRow{}, Members: []domain.BondMember{}}
	feel := map[int][]domain.BondFeeling{}
	bondPairs(people, func(a, b domain.Companion) {
		bond, _ := record.BondOf(a.ID, b.ID)
		na, nb := companionName(a), companionName(b)
		panel.Pairs = append(panel.Pairs, domain.BondRow{
			A: a.ID, B: b.ID, AName: na, BName: nb, Value: bond.Value, Tier: bonds.TierOf(bond.Value),
			Phrase: fmt.Sprintf("%s and %s %s", na, nb, bonds.Together(bond.Value)),
			Effect: bondEffect(bond.Value, bond.Warned), Warned: bond.Warned,
		})
		tier := bonds.TierOf(bond.Value)
		feel[a.ID] = append(feel[a.ID], domain.BondFeeling{ID: b.ID, Name: nb, Words: bonds.Feeling(bond.Value) + " " + nb, Tier: tier})
		feel[b.ID] = append(feel[b.ID], domain.BondFeeling{ID: a.ID, Name: na, Words: bonds.Feeling(bond.Value) + " " + na, Tier: tier})
	})
	sort.SliceStable(panel.Pairs, func(i, j int) bool {
		return absInt(panel.Pairs[i].Value) > absInt(panel.Pairs[j].Value)
	})
	for _, c := range people {
		fs := feel[c.ID]
		if fs == nil {
			fs = []domain.BondFeeling{}
		}
		panel.Members = append(panel.Members, domain.BondMember{ID: c.ID, Name: companionName(c), Feelings: fs})
	}
	return panel, true
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// bondsView is the text of `bonds [member]`.
func (m *CompanyModule) bondsView(leaderUserID int, selector string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	panel, ok := m.BondPanel(leaderUserID)
	if !ok || len(panel.Members) < 2 {
		return "Bonds form between companions. You need at least two to have any (help bonds)."
	}
	pairs := panel.Pairs
	if sel := strings.TrimSpace(selector); sel != "" {
		record, _ := m.registry.Get(leaderUserID)
		c, found := resolveCompanion(record, sel)
		if !found {
			return fmt.Sprintf("No companion called %q. Try: bonds", selector)
		}
		pairs = nil
		for _, p := range panel.Pairs {
			if p.A == c.ID || p.B == c.ID {
				pairs = append(pairs, p)
			}
		}
		if len(pairs) == 0 {
			return fmt.Sprintf("%s has no one to bond with.", companionName(c))
		}
	}
	lines := []string{"How your companions feel about each other:"}
	for _, p := range pairs {
		lines = append(lines, fmt.Sprintf("%s. (bond %+d)", p.Phrase, p.Value))
		if p.Effect != "" {
			lines = append(lines, "  "+p.Effect)
		}
	}
	lines = append(lines, "Time together raises a bond to 50 at most and lowers it to -25 at most; stepping in for each other takes it higher, and only real clashes (splitting over your choices, a refused guard) make rivals (help bonds).")
	return strings.Join(lines, "\n")
}

// bondsCommand is `bonds [member]`.
func (m *CompanyModule) bondsCommand(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	user.SendText(m.bondsView(user.UserId, rest))
	return true, nil
}

// bondLine is one companion's bonds on a line, for `company inspect`;
// blank for one with no one to bond with.
func (m *CompanyModule) bondLine(leaderUserID, companionID int) string {
	panel, ok := m.BondPanel(leaderUserID)
	if !ok {
		return ""
	}
	for _, mem := range panel.Members {
		if mem.ID != companionID || len(mem.Feelings) == 0 {
			continue
		}
		words := make([]string, 0, len(mem.Feelings))
		for _, f := range mem.Feelings {
			words = append(words, f.Words)
		}
		return "Bonds: " + strings.Join(words, ", ") + ". See bonds."
	}
	return ""
}
