package gmcp

import (
	"sync"

	"github.com/GoMudEngine/GoMud/internal/combatpace"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/events"
)

// Ashveil Phase 29f: while a player's combat lines are paced out
// (combatpace Busy), the web client's views that would give the round away
// wait: Char.Vitals and the Company payload (the battle view included).
// When the lines drain (events.CombatPaceDrained), they catch up.

var (
	heldVitalsMu sync.Mutex
	heldVitals   = map[int]bool{}
)

// holdVitals reports whether a player's vitals must wait, remembering to
// send them when the lines drain.
func holdVitals(userId int) bool {
	if !combatpace.Default().Busy(userId) {
		return false
	}
	heldVitalsMu.Lock()
	heldVitals[userId] = true
	heldVitalsMu.Unlock()
	return true
}

// holdCompany reports whether a player's Company payload must wait.
func holdCompany(userId int) bool {
	return combatpace.Default().Busy(userId)
}

func onCombatPaceDrained(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.CombatPaceDrained)
	if !ok {
		return events.Continue
	}
	heldVitalsMu.Lock()
	vitals := heldVitals[evt.UserId]
	delete(heldVitals, evt.UserId)
	heldVitalsMu.Unlock()
	if vitals {
		events.AddToQueue(GMCPCharUpdate{UserId: evt.UserId, Identifier: `Char.Vitals`})
	}
	// Sends only what changed while the lines were held.
	companyview.RefreshUser(evt.UserId)
	return events.Continue
}

func init() {
	events.RegisterListener(events.CombatPaceDrained{}, onCombatPaceDrained)
}
