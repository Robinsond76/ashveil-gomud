// Package creation is Ashveil Phase 72a's seam between the creation steps
// (the looks and life story questions in internal/usercommands) and the web
// client's creation panel: the step a player is on, as data. The steps
// publish a View whenever a question is asked and clear it when they end;
// the GMCP module subscribes and sends it as Char.Creation. The panel
// answers by sending the same input a telnet player types, so the server
// stays the one source of truth.
package creation

import "sync"

// Option is one answer a player may pick.
type Option struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Text  string   `json:"text,omitempty"`
	Color string   `json:"color,omitempty"`
	Stats []string `json:"stats,omitempty"`
}

// Kinds of question.
const (
	KindChoice  = `choice`
	KindText    = `text`
	KindConfirm = `confirm`
)

// View is what the panel shows: the question, its answers, and the picks
// so far with a live preview.
type View struct {
	Active    bool              `json:"active"`
	Mode      string            `json:"mode,omitempty"` // new, edit, story or legacy
	Step      string            `json:"step,omitempty"` // looks, story or summary
	Key       string            `json:"key,omitempty"`  // the trait or stage being asked
	Kind      string            `json:"kind,omitempty"`
	Title     string            `json:"title,omitempty"`
	Options   []Option          `json:"options,omitempty"`
	Number    int               `json:"number,omitempty"` // this question's place
	Total     int               `json:"total,omitempty"`  // of how many
	Picks     map[string]string `json:"picks,omitempty"`  // looks and story picks so far
	Preview   string            `json:"preview,omitempty"`
	Backstory string            `json:"backstory,omitempty"`
	// Lineage, Skin and Hair let the panel draw the figure the player is
	// becoming: their archetype's sprite key and the chosen colours.
	Lineage string `json:"lineage,omitempty"`
	Skin    string `json:"skin,omitempty"`
	Hair    string `json:"hair,omitempty"`
	CanBack bool   `json:"canback,omitempty"`
	CanSkip bool   `json:"canskip,omitempty"`
}

var (
	mu          sync.RWMutex
	current     = map[int]View{}
	subscribers []func(userID int, v View)
)

// Subscribe registers fn to hear every published or cleared view. Call it
// during init, before the game loop runs.
func Subscribe(fn func(userID int, v View)) {
	mu.Lock()
	subscribers = append(subscribers, fn)
	mu.Unlock()
}

// Publish makes v the user's current view and tells the subscribers.
func Publish(userID int, v View) {
	v.Active = true
	mu.Lock()
	current[userID] = v
	subs := append([]func(int, View){}, subscribers...)
	mu.Unlock()
	for _, fn := range subs {
		fn(userID, v)
	}
}

// Clear ends the user's view (once, so subscribers hear it only when there
// was one).
func Clear(userID int) {
	mu.Lock()
	_, had := current[userID]
	delete(current, userID)
	subs := append([]func(int, View){}, subscribers...)
	mu.Unlock()
	if !had {
		return
	}
	for _, fn := range subs {
		fn(userID, View{Active: false})
	}
}

// Forget drops the user's view without telling anyone (logout, purge).
func Forget(userID int) {
	mu.Lock()
	delete(current, userID)
	mu.Unlock()
}

// Get is the user's current view.
func Get(userID int) (View, bool) {
	mu.RLock()
	defer mu.RUnlock()
	v, ok := current[userID]
	return v, ok
}
