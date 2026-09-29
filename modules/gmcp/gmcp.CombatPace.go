package gmcp

import (
	"sort"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/combatpace"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/events"
)

// Ashveil Phase 29f: while a player's combat round is being paced out
// (combatpace Busy), the web client's views that would give the round away
// wait: Char.Vitals (alone or in a full Char payload) and the Company
// payload, the battle view included. When the lines drain
// (events.CombatPaceDrained), they catch up. The Party payload of a party of
// players is not held.

var (
	heldCharMu sync.Mutex
	heldChar   = map[int]map[string]bool{}
)

// holdCharNode reports whether a Char node must wait for a player's paced
// lines, remembering to send it when they drain.
func holdCharNode(userId int, node string) bool {
	if node != `Char` && node != `Char.Vitals` {
		return false
	}
	if !combatpace.Default().Busy(userId) {
		return false
	}
	heldCharMu.Lock()
	defer heldCharMu.Unlock()
	if heldChar[userId] == nil {
		heldChar[userId] = map[string]bool{}
	}
	heldChar[userId][node] = true
	return true
}

// holdCompany reports whether a player's Company payload must wait.
func holdCompany(userId int) bool {
	return combatpace.Default().Busy(userId)
}

// takeHeldChar returns and forgets the Char nodes held for a player.
func takeHeldChar(userId int) []string {
	heldCharMu.Lock()
	defer heldCharMu.Unlock()
	var nodes []string
	for node := range heldChar[userId] {
		nodes = append(nodes, node)
	}
	delete(heldChar, userId)
	sort.Strings(nodes)
	return nodes
}

func onCombatPaceDrained(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.CombatPaceDrained)
	if !ok {
		return events.Continue
	}
	if nodes := takeHeldChar(evt.UserId); len(nodes) > 0 {
		events.AddToQueue(GMCPCharUpdate{UserId: evt.UserId, Identifier: strings.Join(nodes, `, `)})
	}
	// Sends only what changed while the lines were held.
	companyview.RefreshUser(evt.UserId)
	return events.Continue
}

func init() {
	events.RegisterListener(events.CombatPaceDrained{}, onCombatPaceDrained)
	events.RegisterListener(events.PlayerDespawn{}, func(e events.Event) events.ListenerReturn {
		if evt, ok := e.(events.PlayerDespawn); ok {
			takeHeldChar(evt.UserId)
		}
		return events.Continue
	})
}
