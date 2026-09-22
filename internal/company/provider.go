package company

import "sync"

// FormationProvider is implemented by modules/company. It is a read-only
// query seam — the same shape as survival.CompanyService and
// weather.Provider — so internal/ packages (in particular
// internal/hooks's combat loop) can read a leader's current Formation
// without importing modules/company.
type FormationProvider interface {
	// FormationFor returns leaderUserID's current company Formation. ok is
	// false when the leader has no company record at all (a solo player,
	// or one who has never summoned a companion) — callers should treat
	// that as "no formation concept applies," not as an error.
	FormationFor(leaderUserID int) (Formation, bool)
}

var (
	formationProviderMu sync.RWMutex
	formationProvider   FormationProvider
)

// SetFormationProvider registers the active formation provider. Passing
// nil clears it.
func SetFormationProvider(p FormationProvider) {
	formationProviderMu.Lock()
	defer formationProviderMu.Unlock()
	formationProvider = p
}

// FormationFor calls through to the registered FormationProvider. It
// returns ok=false if no provider is registered (e.g. a test binary that
// never loaded modules/company) or the leader has no company record.
func FormationFor(leaderUserID int) (Formation, bool) {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	if p == nil {
		return Formation{}, false
	}
	return p.FormationFor(leaderUserID)
}
