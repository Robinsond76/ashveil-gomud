package encumbrance

import "sync"

// Provider is implemented by modules/encumbrance. It is a read-only query
// seam, the same shape as weather.Provider and survival.CompanyService.
type Provider interface {
	CurrentLoad(leaderUserID int) (Load, bool)
}

var (
	providerMu sync.RWMutex
	provider   Provider
)

// SetProvider registers the active encumbrance provider. Passing nil clears
// it.
func SetProvider(p Provider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	provider = p
}

// CurrentLoad consults the registered provider for a leader's current party
// load. Without a provider, ok is false.
func CurrentLoad(leaderUserID int) (Load, bool) {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	if p == nil {
		return Load{}, false
	}
	return p.CurrentLoad(leaderUserID)
}
