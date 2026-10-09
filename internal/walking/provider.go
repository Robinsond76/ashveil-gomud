package walking

import (
	"sort"
	"sync"
	"testing"
)

// StepProvider is implemented by modules/walking. The user Go command calls
// Stepped after a successful ordinary move so the module can charge the
// mover's company its walking strain.
type StepProvider interface {
	Stepped(userID, fromRoomID, toRoomID int)
}

// EffortProvider is optionally implemented by the step provider (Phase
// 40a2): a company working in place (gathering) pays the strain of one step
// on the room's terrain, scaled by a percentage, through the same carry
// and multipliers a step uses.
type EffortProvider interface {
	Effort(userID, roomID, pct int)
}

// Effort charges the mover's company pct% of one step's strain in roomID. It
// does nothing without a provider that charges effort.
func Effort(userID, roomID, pct int) {
	providerMu.RLock()
	p := stepProvider
	providerMu.RUnlock()
	if ep, ok := p.(EffortProvider); ok && pct > 0 {
		ep.Effort(userID, roomID, pct)
	}
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

// SuspendListeners sets aside every registered step and arrival listener,
// including the ones module init() wires, until tb's test ends. Tests use
// it so only the listener under test hears a step: a module's production
// listener would otherwise roll real dice alongside the test's fixed ones.
// It takes a test handle so game code cannot reach it, and it restores the
// listeners through tb.Cleanup, after cleanups registered later (such as
// removing the test's own listener). Not for parallel tests: the listener
// set is global.
func SuspendListeners(tb testing.TB) {
	tb.Helper()
	providerMu.Lock()
	defer providerMu.Unlock()
	steps, arrivals := listeners, arrivalListeners
	listeners, arrivalListeners = map[int]StepListener{}, map[int]ArrivalListener{}
	tb.Cleanup(func() {
		providerMu.Lock()
		defer providerMu.Unlock()
		listeners, arrivalListeners = steps, arrivals
	})
}
