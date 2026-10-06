package walkto

import (
	"sync"

	"github.com/GoMudEngine/GoMud/internal/util"
)

// Active is one leader's walk in progress. It lives in memory only: after a
// restart or copyover the player simply is not walking (Phase 40d design).
type Active struct {
	Route
	Idx  int    // index into Path of the room the walker is expected to stand in
	Gen  uint64 // changes with every start, so a stale timer finds it gone
	Warn [3]bool
}

// View is what the web map shows: the target and the rooms still to walk.
type View struct {
	Target int
	Path   []int // the rooms ahead, ending at the target
}

// OnChanged fires with a user id whenever that user's walk starts, advances
// or stops, so the GMCP feed sends the new path at once.
var OnChanged util.Hook[int]

var (
	mu     sync.Mutex
	active = map[int]*Active{}
	nextGn uint64
)

// Begin records a new walk for a leader, replacing any earlier one, and
// returns a copy of it.
func Begin(userID int, route Route, warn [3]bool) Active {
	mu.Lock()
	nextGn++
	a := &Active{Route: route, Gen: nextGn, Warn: warn}
	active[userID] = a
	out := *a
	mu.Unlock()
	OnChanged.Fire(userID)
	return out
}

// Current returns a copy of the user's walk.
func Current(userID int) (Active, bool) {
	mu.Lock()
	defer mu.Unlock()
	a, ok := active[userID]
	if !ok {
		return Active{}, false
	}
	return *a, true
}

// Advance moves the walker's expected position along the path, if gen still
// matches. It reports whether it did.
func Advance(userID int, gen uint64, idx int) bool {
	mu.Lock()
	a, ok := active[userID]
	if !ok || a.Gen != gen || idx < 0 || idx >= len(a.Path) {
		mu.Unlock()
		return false
	}
	changed := a.Idx != idx
	a.Idx = idx
	mu.Unlock()
	if changed {
		OnChanged.Fire(userID)
	}
	return true
}

// Clear ends the user's walk; it reports whether there was one.
func Clear(userID int) bool {
	mu.Lock()
	_, ok := active[userID]
	delete(active, userID)
	mu.Unlock()
	if ok {
		OnChanged.Fire(userID)
	}
	return ok
}

// ViewOf is the user's walk as the map shows it; false when not walking.
func ViewOf(userID int) (View, bool) {
	mu.Lock()
	defer mu.Unlock()
	a, ok := active[userID]
	if !ok {
		return View{}, false
	}
	ahead := append([]int(nil), a.Path[a.Idx+1:]...)
	return View{Target: a.Target, Path: ahead}, true
}

// ResetForTest clears every walk.
func ResetForTest() {
	mu.Lock()
	active = map[int]*Active{}
	mu.Unlock()
}
