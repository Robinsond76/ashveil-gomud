// Package tutorial is the seam between the engine's character creation and
// the Ashveil tutorial module (Phase 27a). It holds no state.
package tutorial

import (
	"sync"

	"github.com/GoMudEngine/GoMud/internal/util"
)

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

// Active reports whether a provider is installed (the tutorial module is
// loaded).
func Active() bool {
	mu.RLock()
	defer mu.RUnlock()
	return provider != nil
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

// View is a player's place in the course, for surfaces other than the
// terminal (Phase 27d: the browser panel). Its text is plain, without
// colour markup.
type View struct {
	// Stage is 1-based, of Stages.
	Stage     int
	Stages    int
	ID        string
	Title     string
	Goal      string
	Checklist []Check
	Hints     []string
}

// Check is one checklist line.
type Check struct {
	Label string
	Done  bool
}

// Viewer is optionally implemented by the Provider. TutorialView reports
// false for a player not in the course. It reads character state, so call
// it on the game loop.
type Viewer interface {
	TutorialView(userID int) (View, bool)
}

// ViewOf is a player's view of the course. ok is false without a provider
// that can show one, or when the player isn't in the course.
func ViewOf(userID int) (View, bool) {
	mu.RLock()
	p := provider
	mu.RUnlock()
	v, ok := p.(Viewer)
	if !ok {
		return View{}, false
	}
	return v.TutorialView(userID)
}

// OnChanged fires, on the game loop, when a player's place in the course
// changes (a stage passed or waived, placement, leaving): Phase 27d's
// panel resends at once rather than at the next refresh. Handlers must not
// change the course.
var OnChanged util.Hook[int]

// Changed reports a change to a player's place in the course.
func Changed(userID int) { OnChanged.Fire(userID) }
