package camping

import (
	"sync"
	"time"
)

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

// RestActivity is a leader's camp or inn stay as the information surfaces
// show it (Phase 26a).
type RestActivity struct {
	// Inn is true for an inn stay, false for a camp.
	Inn bool
	// Resting is true while a rest runs; a pitched camp that isn't resting
	// is false.
	Resting   bool
	Remaining time.Duration
}

// RestProvider is optionally implemented by the registered movement
// provider (Phase 26a). Both read state only.
type RestProvider interface {
	// LeaderRest reports the leader's camp or inn stay; ok is false with
	// neither.
	LeaderRest(leaderUserID int) (RestActivity, bool)
	// RestTierOf reports a character's rest tier buff (Rested, Well
	// Rested, or TierNone) and its time left; ok is false when it can't
	// tell.
	RestTierOf(userID int) (tier Tier, remaining time.Duration, ok bool)
}

func restProvider() (RestProvider, bool) {
	providerMu.RLock()
	p := movementProvider
	providerMu.RUnlock()
	rp, ok := p.(RestProvider)
	return rp, ok
}

// RestReporting reports whether a registered provider can report camps and
// inn stays (Phase 26a), so "none" can be told from "can't tell".
func RestReporting() bool {
	_, ok := restProvider()
	return ok
}

// LeaderRest reports a leader's camp or inn stay. ok is false without a
// provider or either.
func LeaderRest(leaderUserID int) (RestActivity, bool) {
	rp, ok := restProvider()
	if !ok {
		return RestActivity{}, false
	}
	return rp.LeaderRest(leaderUserID)
}

// RestTierOf reports a character's rest tier. ok is false without a
// provider or a tier.
func RestTierOf(userID int) (Tier, time.Duration, bool) {
	rp, ok := restProvider()
	if !ok {
		return TierNone, 0, false
	}
	return rp.RestTierOf(userID)
}

// CampAbandoner is implemented by modules/camping (Phase 27b). AbandonCamp
// removes a leader's camp, resting or not, and saves at once; a finished
// rest keeps its recovery and any rest tier owed. Inn stays are untouched.
// An error means the camp is still there.
type CampAbandoner interface {
	AbandonCamp(leaderUserID int) error
}

var campAbandoner CampAbandoner

// SetCampAbandoner registers the active camp abandoner. Passing nil clears
// it.
func SetCampAbandoner(p CampAbandoner) {
	providerMu.Lock()
	defer providerMu.Unlock()
	campAbandoner = p
}

// AbandonCamp removes a leader's camp. Without a provider there is nothing
// to remove.
func AbandonCamp(leaderUserID int) error {
	providerMu.RLock()
	p := campAbandoner
	providerMu.RUnlock()
	if p == nil {
		return nil
	}
	return p.AbandonCamp(leaderUserID)
}
