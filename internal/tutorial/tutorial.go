// Package tutorial is the seam between the engine's character creation and
// the Ashveil tutorial module (Phase 27a). It holds no state.
package tutorial

import "sync"

// Provider is implemented by modules/tutorial.
type Provider interface {
	// Begin starts the course for a newly created character and places
	// them in it. It reports false when it couldn't, and the caller then
	// uses its own path.
	Begin(userID int) bool
}

var (
	mu       sync.RWMutex
	provider Provider
)

// SetProvider installs the provider. Passing nil clears it.
func SetProvider(p Provider) {
	mu.Lock()
	defer mu.Unlock()
	provider = p
}

// Begin hands a new character to the tutorial. It reports false without a
// provider, or when the provider couldn't place them.
func Begin(userID int) bool {
	mu.RLock()
	p := provider
	mu.RUnlock()
	if p == nil {
		return false
	}
	return p.Begin(userID)
}
