package stance

import "sync"

// Provider is the durable store of stances (modules/strategy). Keys are
// formation member keys: "leader" for the player, "companion:<id>" for a
// companion.
type Provider interface {
	// StoredStance is the member's chosen stance, none when unset.
	StoredStance(userID int, key string) Stance
}

var (
	providerMu sync.RWMutex
	provider   Provider
)

// SetProvider registers the store. The module calls it once, at init.
func SetProvider(p Provider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	provider = p
}

// For is a member's chosen stance, or none when no store is registered.
func For(userID int, key string) Stance {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	if p == nil {
		return None
	}
	return p.StoredStance(userID, key)
}
