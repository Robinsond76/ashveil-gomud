package walking

import (
	"sort"
	"sync"
)

// StepProvider is implemented by modules/walking. The user Go command calls
// Stepped after a successful ordinary move so the module can charge the
// mover's company its walking strain.
type StepProvider interface {
	Stepped(userID, fromRoomID, toRoomID int)
}

// StepListener hears every ordinary step after the provider has charged it
// (Phase 17: archetype auto-skills). It runs on the game loop.
type StepListener func(userID, fromRoomID, toRoomID int)

var (
	providerMu     sync.RWMutex
	stepProvider   StepProvider
	listeners      = map[int]StepListener{}
	nextListenerID int
)

// SetStepProvider registers the active step provider. nil clears it.
func SetStepProvider(p StepProvider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	stepProvider = p
}

// AddStepListener registers a listener for ordinary steps and returns a
// function that removes it. Listeners run in registration order.
func AddStepListener(fn StepListener) (remove func()) {
	providerMu.Lock()
	defer providerMu.Unlock()
	nextListenerID++
	id := nextListenerID
	listeners[id] = fn
	return func() {
		providerMu.Lock()
		defer providerMu.Unlock()
		delete(listeners, id)
	}
}

// Stepped reports a completed ordinary step: the provider (if any) charges
// it, then every listener hears it. Without either it does nothing.
func Stepped(userID, fromRoomID, toRoomID int) {
	providerMu.RLock()
	p := stepProvider
	ids := make([]int, 0, len(listeners))
	for id := range listeners {
		ids = append(ids, id)
	}
	fns := make([]StepListener, 0, len(ids))
	sort.Ints(ids)
	for _, id := range ids {
		fns = append(fns, listeners[id])
	}
	providerMu.RUnlock()
	if p != nil {
		p.Stepped(userID, fromRoomID, toRoomID)
	}
	for _, fn := range fns {
		fn(userID, fromRoomID, toRoomID)
	}
}

// ArrivalListener hears a completed journey's arrival (Phase 37: random
// room encounters roll on it). It runs on the game loop after the company
// has relocated and the traveller was told, and never charges walking
// strain: the journey already paid its own.
type ArrivalListener func(userID, fromRoomID, toRoomID int)

var arrivalListeners = map[int]ArrivalListener{}

// AddArrivalListener registers a listener for journey arrivals and returns
// a function that removes it. Listeners run in registration order.
func AddArrivalListener(fn ArrivalListener) (remove func()) {
	providerMu.Lock()
	defer providerMu.Unlock()
	nextListenerID++
	id := nextListenerID
	arrivalListeners[id] = fn
	return func() {
		providerMu.Lock()
		defer providerMu.Unlock()
		delete(arrivalListeners, id)
	}
}

// Arrived reports a completed journey to the arrival listeners.
func Arrived(userID, fromRoomID, toRoomID int) {
	providerMu.RLock()
	ids := make([]int, 0, len(arrivalListeners))
	for id := range arrivalListeners {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	fns := make([]ArrivalListener, 0, len(ids))
	for _, id := range ids {
		fns = append(fns, arrivalListeners[id])
	}
	providerMu.RUnlock()
	for _, fn := range fns {
		fn(userID, fromRoomID, toRoomID)
	}
}
