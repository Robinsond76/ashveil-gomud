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
