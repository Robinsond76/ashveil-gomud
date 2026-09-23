package mount

import "sync"

// Provider is implemented by modules/mount. It is a read-only query seam,
// the same shape as weather.Provider and encumbrance.Provider, so
// modules/encumbrance can consult a leader's mount capacity bonus without
// importing modules/mount.
type Provider interface {
	CapacityBonusGrams(leaderUserID int) int
}

var (
	providerMu sync.RWMutex
	provider   Provider
)

// SetProvider registers the active mount provider. Passing nil clears it.
func SetProvider(p Provider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	provider = p
}

// CapacityBonus consults the registered provider for a leader's current
// mount cargo-capacity bonus. Without a provider, or without a tracked
// mount, it is 0 — never a guessed default.
func CapacityBonus(leaderUserID int) int {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	if p == nil {
		return 0
	}
	return p.CapacityBonusGrams(leaderUserID)
}

// ReliefProvider is optionally implemented by the registered Provider
// (Phase 16): a leader's mount fatigue relief and travel-duration
// multiplier.
type ReliefProvider interface {
	// Relief is the walking fatigue multiplier and how many company members
	// ride. Without a mount it is (100, 0).
	Relief(leaderUserID int) (fatiguePct, riders int)
	// TravelDurationPct is the route travel-duration multiplier; 100 without
	// a mount.
	TravelDurationPct(leaderUserID int) int
}

func reliefProvider() (ReliefProvider, bool) {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	rp, ok := p.(ReliefProvider)
	return rp, ok
}

// Relief consults the registered provider. Without one it is (100, 0).
func Relief(leaderUserID int) (fatiguePct, riders int) {
	rp, ok := reliefProvider()
	if !ok {
		return 100, 0
	}
	return rp.Relief(leaderUserID)
}

// TravelDurationPct consults the registered provider. Without one it is 100.
func TravelDurationPct(leaderUserID int) int {
	rp, ok := reliefProvider()
	if !ok {
		return 100
	}
	return rp.TravelDurationPct(leaderUserID)
}
