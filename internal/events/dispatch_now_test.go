package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type dispatchProbe struct{ id int }

func (d dispatchProbe) Type() string { return `dispatchProbe` }

func TestDispatchNowTakesOnlyTheMatchesInOrder(t *testing.T) {
	drainForTest()
	t.Cleanup(drainForTest)
	var got, slots []int
	handle := func(e Event) ListenerReturn {
		got = append(got, e.(dispatchProbe).id)
		slots = append(slots, Slot())
		return Continue
	}
	match := func(e Event) bool { p, ok := e.(dispatchProbe); return ok && p.id > 0 }

	SetSlot(3)
	AddToQueue(dispatchProbe{id: 1})
	SetSlot(0)
	AddToQueue(causeProbe{name: "kept"})
	AddToQueue(dispatchProbe{id: 2})
	AddToQueue(dispatchProbe{id: -9})

	// Called from inside a listener, as the combat round is: the listener
	// lock is held, so DispatchNow must not go back through DoListeners.
	var n int
	id := RegisterListener(causeFollow{}, func(Event) ListenerReturn { n = DispatchNow(match, handle); return Continue })
	t.Cleanup(func() { UnregisterListener(causeFollow{}, id) })
	DoListeners(causeFollow{})
	assert.Equal(t, 2, n)
	assert.Equal(t, []int{1, 2}, got, "dispatched in queued order")
	assert.Equal(t, []int{3, 0}, slots, "each keeps the slot it was queued in")
	assert.Equal(t, 0, Slot(), "the caller's slot is restored")
	assert.Equal(t, 2, globalQueue.Len(), "the rest stay queued")
	assert.Zero(t, DispatchNow(match, handle))
}
