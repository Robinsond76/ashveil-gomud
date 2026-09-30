package battle

import (
	"errors"
	"sort"
)

// Phase 30c: a company focus changed mid-battle holds for that battle
// only. An order waits for the next round's upkeep, which turns the
// company at once (TakeRefocus); until then another order is refused.

var (
	// ErrNoBattle: the player is in no battle.
	ErrNoBattle = errors.New("not in a battle")
	// ErrFocusPending: an order is already waiting for the next round.
	ErrFocusPending = errors.New("an order is already waiting for the next round")
)

// SetFocus orders a focus for the player's battle.
func SetFocus(userId int, rule string) error {
	return orderFocus(userId, rule, true)
}

// ClearFocus returns the player's battle to their saved focus.
func ClearFocus(userId int) error {
	return orderFocus(userId, "", false)
}

func orderFocus(userId int, rule string, set bool) error {
	mu.Lock()
	defer mu.Unlock()
	b, ok := battles[userId]
	if !ok {
		return ErrNoBattle
	}
	if b.FocusPending {
		return ErrFocusPending
	}
	b.Focus, b.FocusSet, b.FocusPending = rule, set, true
	return nil
}

// Focus is the focus ordered for the player's battle; set is false when
// none was (the saved focus holds).
func Focus(userId int) (rule string, set bool) {
	mu.Lock()
	defer mu.Unlock()
	if b, ok := battles[userId]; ok && b.FocusSet {
		return b.Focus, true
	}
	return "", false
}

// FocusReady reports whether the player's battle takes an order now: no
// order is waiting for the next round.
func FocusReady(userId int) bool {
	mu.Lock()
	defer mu.Unlock()
	b, ok := battles[userId]
	return ok && !b.FocusPending
}

// TakeRefocus lists, ascending, the players whose order waits for this
// round's upkeep, and marks those orders applied.
func TakeRefocus() []int {
	mu.Lock()
	defer mu.Unlock()
	var out []int
	for id, b := range battles {
		if b.FocusPending {
			b.FocusPending = false
			out = append(out, id)
		}
	}
	sort.Ints(out)
	return out
}
