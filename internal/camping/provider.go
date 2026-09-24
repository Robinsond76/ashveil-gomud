package camping

import "sync"

// ViewProvider is implemented by modules/camping. RenderCampView returns
// handled=true when it rendered an active resting session's view in place of
// ordinary room rendering.
type ViewProvider interface {
	RenderCampView(leaderUserID int) (handled bool, err error)
}

// MovementProvider is implemented by modules/camping. MovementBlocked reports
// an active rest and the refusal text for ordinary movement.
type MovementProvider interface {
	MovementBlocked(leaderUserID int) (blocked bool, message string)
}

var (
	providerMu       sync.RWMutex
	viewProvider     ViewProvider
	movementProvider MovementProvider
)

// SetViewProvider registers the active view provider. Passing nil clears it.
func SetViewProvider(p ViewProvider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	viewProvider = p
}

// SetMovementProvider registers the active movement-block provider. Passing
// nil clears it.
func SetMovementProvider(p MovementProvider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	movementProvider = p
}

// CampView consults the registered view provider. Without one, native look
// behavior is unchanged.
func CampView(leaderUserID int) (bool, error) {
	providerMu.RLock()
	p := viewProvider
	providerMu.RUnlock()
	if p == nil {
		return false, nil
	}
	return p.RenderCampView(leaderUserID)
}

// MovementBlocked reports whether an active rest must refuse ordinary
// movement, and the refusal text to show. Without a provider, movement is
// unchanged.
func MovementBlocked(leaderUserID int) (bool, string) {
	providerMu.RLock()
	p := movementProvider
	providerMu.RUnlock()
	if p == nil {
		return false, ""
	}
	return p.MovementBlocked(leaderUserID)
}

// AbandonProvider is implemented by modules/camping (Phase 25a).
// AbandonForDeath removes a dead leader's camp and any inn stay, resting or
// not, and saves that at once. An error means they are still there.
type AbandonProvider interface {
	AbandonForDeath(leaderUserID int) error
}

var abandonProvider AbandonProvider

// SetAbandonProvider registers the active abandon provider. Passing nil
// clears it.
func SetAbandonProvider(p AbandonProvider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	abandonProvider = p
}

// AbandonForDeath removes a dead leader's camp and inn stay. Without a
// provider there is nothing to remove.
func AbandonForDeath(leaderUserID int) error {
	providerMu.RLock()
	p := abandonProvider
	providerMu.RUnlock()
	if p == nil {
		return nil
	}
	return p.AbandonForDeath(leaderUserID)
}
