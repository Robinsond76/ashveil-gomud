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

// HorseView is one horse as a player sees it (Phase 32f).
type HorseView struct {
	ID            int
	Name          string
	Kind          Kind
	Saddle        string // the fitted saddle's name, or "" when bare
	CapacityGrams int
}

// HerdProvider is optionally implemented by the registered Provider (Phase
// 32f): a leader's horses, by ascending id.
type HerdProvider interface {
	Herd(leaderUserID int) []HorseView
}

// HerdOf consults the registered provider; nil without one.
func HerdOf(leaderUserID int) []HorseView {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	hp, ok := p.(HerdProvider)
	if !ok {
		return nil
	}
	return hp.Herd(leaderUserID)
}
