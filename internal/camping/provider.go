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
