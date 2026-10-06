package weather

import "sync"

// Provider is implemented by modules/weather. It is a read-only query seam,
// the same shape as survival.CompanyService and camping.ViewProvider.
type Provider interface {
	CurrentCondition(zone string) (Condition, bool)
}

var (
	providerMu sync.RWMutex
	provider   Provider
)

// SetProvider registers the active weather provider. Passing nil clears it.
func SetProvider(p Provider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	provider = p
}

// CurrentCondition consults the registered provider for a zone's current
// weather condition. Without a provider, or for an untracked zone, ok is
// false.
func CurrentCondition(zone string) (Condition, bool) {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	if p == nil {
		return Condition{}, false
	}
	return p.CurrentCondition(zone)
}
