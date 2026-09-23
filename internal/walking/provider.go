package walking

import "sync"

// StepProvider is implemented by modules/walking. The user Go command calls
// Stepped after a successful ordinary move so the module can charge the
// mover's company its walking strain.
type StepProvider interface {
	Stepped(userID, fromRoomID, toRoomID int)
}

var (
	providerMu   sync.RWMutex
	stepProvider StepProvider
)

// SetStepProvider registers the active step provider. nil clears it.
func SetStepProvider(p StepProvider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	stepProvider = p
}

// Stepped reports a completed ordinary step. Without a provider it does
// nothing.
func Stepped(userID, fromRoomID, toRoomID int) {
	providerMu.RLock()
	p := stepProvider
	providerMu.RUnlock()
	if p == nil {
		return
	}
	p.Stepped(userID, fromRoomID, toRoomID)
}
