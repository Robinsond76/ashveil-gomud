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
func (m *CampingModule) planTheftLocked(room *rooms.Room) *camping.Theft {
	chance := m.campSettings().Thefts[room.Zone]
	if chance <= 0 || m.rollPct() >= chance {
		return nil
	}
	return &camping.Theft{}
}

// campGearItem is the camp gear thieves leave alone.
func campGearItem(itemID int) bool {
	return itemID >= bedrollItemID && itemID <= surgeonKitItemID
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
		leader := m.userByID(leaderUserID)
		if leader == nil || leader.Character == nil {
			continue // noticed when the leader is back online
		}
		if m.inBattle != nil && m.inBattle(leaderUserID) {
			continue
		}
		m.mu.Lock()
		camp, ok := m.camps[leaderUserID]
		m.mu.Unlock()
		if !ok || camp.Rest == nil || !camp.Rest.TheftPending() {
			continue
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
			continue
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
			continue
		}
		m.mu.Unlock()

		if watched {
			leader.SendText("Your watch hears something at the edge of the camp and shouts. Footsteps run off into the dark; nothing is missing.")
			continue
		}
		losses := m.campTheft(leaderUserID, cfg.TheftSharePct, cfg.TheftMaxItems)
		if len(losses) == 0 {
			continue // nothing worth taking: the thieves left empty-handed
		}
		leader.SendText(theftText(losses))
	}
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
