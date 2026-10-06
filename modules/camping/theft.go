package camping

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Phase 40a4 camp theft: thieves slip into a camp rest that has no bells
// and trip lines, unseen, and take a small share of the company's loose
// goods. The rest records whether thieves are coming when it starts; they
// are resolved on the game loop once the rest is done and the leader is
// online and out of battle, and the leader is told on waking.
//
// Never taken: equipped gear, the leader's own pack, the treasury's gold,
// quest tokens, keys, and the camp gear itself (items 45-50). A posted
// watch gets the same chance to catch them as it has to spot raiders.
// Stolen goods are gone; they are not tracked.

// planTheftLocked rolls whether thieves come to a rest in this zone.
func (m *CampingModule) planTheftLocked(room *rooms.Room, tentPct int) *camping.Theft {
	chance := m.campSettings().Thefts[room.Zone]
	if chance <= 0 || m.rollPct() >= camping.ScaleChance(chance, tentPct) {
		return nil
	}
	return &camping.Theft{}
}

// campGearItem is the camp gear thieves leave alone.
func campGearItem(itemID int) bool {
	if itemID >= bedrollItemID && itemID <= surgeonKitItemID {
		return true
	}
	for _, t := range camping.Tents { // Phase 52
		if itemID == t.ItemID {
			return true
		}
	}
	return false
}

func (m *CampingModule) campTheft(leaderUserID, sharePct, maxUnits int) []company.TheftLoss {
	pick := func(n int) int {
		if m.roll == nil || n <= 1 {
			return 0
		}
		return m.roll(n)
	}
	if m.theft != nil {
		return m.theft(leaderUserID, sharePct, maxUnits, pick, campGearItem)
	}
	return company.CampTheft(leaderUserID, sharePct, maxUnits, pick, campGearItem)
}

// resolveCampTheft settles each finished rest's theft (game loop) for a
// leader who is online and out of battle. The rest is saved as done before
// anything is taken, so it never happens twice. A watch posted at the camp
// may catch the thieves first.
func (m *CampingModule) resolveCampTheft() {
	cfg := m.campSettings()
	m.mu.Lock()
	var due []int
	for leaderUserID, camp := range m.camps {
		if camp.Rest != nil && camp.Rest.TheftPending() {
			due = append(due, leaderUserID)
		}
	}
	m.mu.Unlock()
	sort.Ints(due)
	for _, leaderUserID := range due {
		m.resolveTheftFor(leaderUserID, cfg)
	}
}

// settleTheft completes a due rest and resolves its theft before the
// leader breaks camp or rests again (40a4 review), so neither can slip
// between a rest's end and the next round to dodge thieves. Game loop;
// call without m.mu.
func (m *CampingModule) settleTheft(leaderUserID int) {
	m.mu.Lock()
	if err := m.syncLocked(leaderUserID); err != nil {
		mudlog.Warn("camping: theft settle sync", "leader", leaderUserID, "error", err)
	}
	camp, ok := m.camps[leaderUserID]
	pending := ok && camp.Rest != nil && camp.Rest.TheftPending()
	m.mu.Unlock()
	if pending {
		m.resolveTheftFor(leaderUserID, m.campSettings())
	}
}

// resolveTheftFor settles one leader's pending theft, if the leader is
// online and out of battle.
func (m *CampingModule) resolveTheftFor(leaderUserID int, cfg campSettings) {
	leader := m.userByID(leaderUserID)
	if leader == nil || leader.Character == nil {
		return // noticed when the leader is back online
	}
	if m.inBattle != nil && m.inBattle(leaderUserID) {
		return
	}
	m.mu.Lock()
	camp, ok := m.camps[leaderUserID]
	m.mu.Unlock()
	if !ok || camp.Rest == nil || !camp.Rest.TheftPending() {
		return
	}
	watched := false
	if leader.Character.RoomId == camp.RoomID && m.specialist != nil {
		if watch, ok := m.specialist(leaderUserID, archetypes.UtilityWatch, camp.RoomID); ok {
			pct := archetypes.PctByLevel(watch.Level, cfg.WatchPctPerLevel, 100)
			watched = pct > 0 && m.rollPct() < pct
		}
	}

	m.mu.Lock()
	camp, ok = m.camps[leaderUserID]
	if !ok || camp.Rest == nil || !camp.Rest.TheftPending() {
		m.mu.Unlock()
		return
	}
	rest := *camp.Rest
	rest.Theft = &camping.Theft{Done: true}
	updated := camp
	updated.Rest = &rest
	m.camps[leaderUserID] = updated
	if err := m.saveLocked(); err != nil {
		m.camps[leaderUserID] = camp
		m.mu.Unlock()
		mudlog.Error("camping: theft save", "leader", leaderUserID, "error", err)
		return
	}
	m.mu.Unlock()

	if watched {
		leader.SendText("Your watch hears something at the edge of the camp and shouts. Footsteps run off into the dark; nothing is missing.")
		return
	}
	losses := m.campTheft(leaderUserID, cfg.TheftSharePct, cfg.TheftMaxItems)
	if len(losses) == 0 {
		return // nothing worth taking: the thieves left empty-handed
	}
	leader.SendText(theftText(losses))
}

// theftRiskIn reports whether thieves work this room's zone (40a4 review):
// a rest there without bells and trip lines may be robbed.
func (m *CampingModule) theftRiskIn(room *rooms.Room) bool {
	return room != nil && m.campSettings().Thefts[room.Zone] > 0
}

// theftText is the wake-up report of what thieves took.
func theftText(losses []company.TheftLoss) string {
	parts := make([]string, 0, len(losses))
	for _, l := range losses {
		if l.Count > 1 {
			parts = append(parts, fmt.Sprintf("%d %s", l.Count, l.Name))
		} else {
			parts = append(parts, l.Name)
		}
	}
	return "When you wake, the packs have been rifled. Missing: " + strings.Join(parts, ", ") + ".\n" +
		`Camp bells and trip lines would have kept thieves out (<ansi fg="command">help camp gear</ansi>).`
}
