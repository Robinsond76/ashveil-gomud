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

// BandProvider is optionally implemented by the registered Provider. It
// resolves the configured load band for a leader's current load (Phase 16).
type BandProvider interface {
	CurrentBand(leaderUserID int) (LoadBand, bool)
}

// CurrentBand resolves a leader's current load band through the registered
// provider. Without one, or for an untracked load, ok is false and the band
// is neutral (100%/100%).
func CurrentBand(leaderUserID int) (LoadBand, bool) {
	neutral := LoadBand{TravelDurationPct: 100, FatiguePct: 100}
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	bp, ok := p.(BandProvider)
	if !ok {
		return neutral, false
	}
	band, ok := bp.CurrentBand(leaderUserID)
	if !ok {
		return neutral, false
	}
	return band, true
}
