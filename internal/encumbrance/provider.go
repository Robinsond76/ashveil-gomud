package encumbrance

import (
	"fmt"
	"sync"
)

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

// WouldExceed reports whether adding grams to a leader's company would put
// its load over capacity (Phase 32f), with the current load. An untracked
// load (no provider, no capacity configured) never refuses. Call it on the
// game loop: the load reads company state.
func WouldExceed(leaderUserID, addGrams int) (Load, bool) {
	if addGrams <= 0 {
		return Load{}, false
	}
	load, ok := CurrentLoad(leaderUserID)
	if !ok {
		return Load{}, false
	}
	return load, load.WouldExceed(addGrams)
}

// TooMuchToCarry is the refusal a player sees when adding grams would put
// their company over capacity (Phase 32f); ok is false when it fits.
func TooMuchToCarry(leaderUserID, addGrams int) (string, bool) {
	load, refuse := WouldExceed(leaderUserID, addGrams)
	if !refuse {
		return "", false
	}
	return fmt.Sprintf(`That would be too much for your company to carry: %.1f kg of %.1f kg already (<ansi fg="command">help cargo</ansi>).`,
		float64(load.TotalGrams())/1000, float64(load.CapacityGrams)/1000), true
}
