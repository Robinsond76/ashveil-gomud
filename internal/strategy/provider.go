package strategy

import "sync"

// Provider is the durable store of strategies (modules/strategy). Keys are
// formation member keys: "leader" for the player, "companion:<id>" for a
// companion.
type Provider interface {
	// Stored is what the player has set for a member (blank fields mean
	// the default).
	Stored(userID int, key string) Strategy
	// AutoSpells is the configured list of automatic spells, in order,
	// without costs.
	AutoSpells() []Spell
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

func current() Provider {
	providerMu.RLock()
	defer providerMu.RUnlock()
	return provider
}

// For is a member's strategy, resolved against its archetype's default.
func For(userID int, key, archetype string) Strategy {
	var s Strategy
	if p := current(); p != nil {
		s = p.Stored(userID, key)
	}
	return s.Resolve(archetype)
}

// AutoSpells is the configured list of automatic spells, or the shipped
// list when no store is registered.
func AutoSpells() []Spell {
	if p := current(); p != nil {
		if spells := p.AutoSpells(); len(spells) > 0 {
			return spells
		}
	}
	return DefaultAutoSpells()
}
