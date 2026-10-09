package storyevents

import (
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/storyevents"
)

// entered is called after an ordinary step or a journey's arrival. A room
// or tag trigger fires on both; an arrival trigger only when a journey
// ends. It never rolls on look, scout, login, spawn or a failed move,
// because only those two producers call it.
func (m *Module) entered(userID, roomID int, arrival bool) {
	kinds := []string{storyevents.TriggerRoom, storyevents.TriggerTag}
	if arrival {
		kinds = append(kinds, storyevents.TriggerArrival)
	}
	m.open(userID, roomID, false, kinds...)
}

// campRestEnded is called when a camp rest finishes at the camp's room.
func (m *Module) campRestEnded(userID, roomID int) {
	m.open(userID, roomID, true, storyevents.TriggerCamp)
}

// open decides whether a trigger of these kinds opens an event for the
// leader's company, and opens the first that qualifies. It reports whether
// one opened.
func (m *Module) open(userID, roomID int, camp bool, kinds ...string) bool {
	zone, tags, ok := m.w.Zone(roomID)
	if !ok || m.w.Busy(userID, roomID, camp) {
		return false
	}
	// A follower in a player party does not roll: the leader's move did.
	if p := parties.Get(userID); p != nil && !p.IsLeader(userID) {
		return false
	}
	if _, _, waiting := m.waiting(userID); waiting {
		return false
	}
	cat := m.events()
	var candidates []storyevents.Triggered
	for _, kind := range kinds {
		candidates = append(candidates, cat.Triggered(kind, roomID, zone, tags)...)
	}
	if len(candidates) == 0 {
		return false
	}
	user := m.w.User(userID)
	if user == nil || user.Character == nil {
		return false
	}
	level := user.Character.Level
	m.mu.Lock()
	st := m.state[userID]
	m.mu.Unlock()
	company := m.w.Company(userID, st.flagSet())
	now := m.clock().Unix()
	for _, c := range candidates {
		ev := c.Event
		if done, seen := st.Done[ev.ID]; seen && (ev.CooldownMinutes == 0 || now < done+int64(ev.CooldownMinutes)*60) {
			continue
		}
		if (ev.MinLevel > 0 && level < ev.MinLevel) || (ev.MaxLevel > 0 && level > ev.MaxLevel) {
			continue
		}
		if !ev.Require.Meets(company) {
			continue
		}
		if chance := c.Trigger.Chance; chance > 0 && chance < 100 && m.rng(100) >= chance {
			continue
		}
		if !m.begin(userID, ev, roomID, now) {
			return false
		}
		m.show(userID, ev, ev.StartPage(), nil)
		return true
	}
	return false
}

// begin records the page the company is now looking at, saved at once.
func (m *Module) begin(userID int, ev storyevents.Event, roomID int, now int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.state[userID]
	if st.Pending != nil {
		return false
	}
	prev, had := m.state[userID]
	st = st.clone()
	st.Pending = &Pending{Event: ev.ID, Page: ev.StartPage(), Room: roomID, Since: now}
	m.state[userID] = st
	if err := m.saveLocked(); err != nil {
		if had {
			m.state[userID] = prev
		} else {
			delete(m.state, userID)
		}
		mudlog.Warn("storyevents: save a new scene", "user", userID, "event", ev.ID, "error", err)
		return false
	}
	return true
}
