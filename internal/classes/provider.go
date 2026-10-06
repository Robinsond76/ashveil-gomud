package classes

import (
	"errors"
	"sync"
)

// ErrUnavailable is returned when no module keeps class state.
var ErrUnavailable = errors.New("class records are unavailable right now")

// State is a character's durable class choices: the advanced or elite class
// it promoted into and the talents it has picked, in the order picked.
type State struct {
	Class   string
	Talents []string
}

// PlayerProvider is implemented by modules/archetype, which keeps a player's
// class state in its registry. Companions carry theirs on the mob instance
// (Character.SetClassState), written from the company record.
type PlayerProvider interface {
	PlayerClass(userID int) State
}

var (
	providerMu sync.RWMutex
	provider   PlayerProvider
)

// SetProvider registers the active provider. nil clears it.
func SetProvider(p PlayerProvider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	provider = p
}

// PlayerClass is a player's class state; the zero State without a provider
// or for a player who has not promoted.
func PlayerClass(userID int) State {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	if p == nil {
		return State{}
	}
	return p.PlayerClass(userID)
}

// PlayerWriter is implemented by the same module: it commits a promotion or
// a talent for a player, saving with rollback. The caller has checked the
// rules (Check, CanPick); the writer holds the durable state only.
type PlayerWriter interface {
	PromotePlayer(userID int, class string) error
	PickPlayerTalent(userID int, talent string) error
}

// PromotePlayer commits a player's promotion.
func PromotePlayer(userID int, class string) error {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	if w, ok := p.(PlayerWriter); ok {
		return w.PromotePlayer(userID, class)
	}
	return ErrUnavailable
}

// PickPlayerTalent commits a player's talent pick.
func PickPlayerTalent(userID int, talent string) error {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	if w, ok := p.(PlayerWriter); ok {
		return w.PickPlayerTalent(userID, talent)
	}
	return ErrUnavailable
}
