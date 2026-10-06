package main

import (
	"sync"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/stretchr/testify/assert"
)

// TestTypedInputNeverTakesACombatCause (Phase 29f): what a player types
// reaches the queue through the real input worker, off the game loop. Even
// while a combat round's cause is current on the loop, it is queued typed
// and without the cause, so pacing never holds its output.
func TestTypedInputNeverTakesACombatCause(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	events.ProcessEvents()
	w := &World{worldInput: make(chan WorldInput)}
	shutdown := make(chan bool)
	var wg sync.WaitGroup
	go w.InputWorker(shutdown, &wg)
	t.Cleanup(func() {
		close(shutdown)
		wg.Wait() // the worker logs as it stops; let it finish before the next test
	})

	type seen struct {
		cause uint64
		typed bool
	}
	got := map[string]seen{}
	id := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
		got[e.(events.Input).InputText] = seen{events.Cause(), events.Typed()}
		return events.Cancel
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(events.Input{}, id) })

	events.WithCause(12, func() {
		w.SendInput(WorldInput{FromId: 3, InputText: "look"})
		// The second send returns only once the worker has queued the first.
		w.SendInput(WorldInput{FromId: 3, InputText: "inventory"})
	})
	events.ProcessEvents()

	assert.Equal(t, seen{0, true}, got["look"], "typed input queued during a combat round")
}
