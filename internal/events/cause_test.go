package events

import "testing"

type causeProbe struct{ name string }

func (c causeProbe) Type() string { return `causeProbe` }

type causeFollow struct{ name string }

func (c causeFollow) Type() string { return `causeFollow` }

// drainForTest empties anything a previous test left queued.
func drainForTest() {
	qLock.Lock()
	globalQueue = globalQueue[:0]
	requeues = requeues[:0]
	uniqueMap = map[string]struct{}{}
	qLock.Unlock()
}

func TestCauseIsInheritedThroughTheQueue(t *testing.T) {
	drainForTest()
	t.Cleanup(drainForTest)

	seen := map[string]uint64{}
	probe := RegisterListener(causeProbe{}, func(e Event) ListenerReturn {
		p := e.(causeProbe)
		seen[p.name] = Cause()
		// A listener's own follow-on event inherits the cause.
		AddToQueue(causeFollow{name: p.name + "-follow"})
		return Continue
	})
	follow := RegisterListener(causeFollow{}, func(e Event) ListenerReturn {
		seen[e.(causeFollow).name] = Cause()
		return Continue
	})
	t.Cleanup(func() {
		UnregisterListener(causeProbe{}, probe)
		UnregisterListener(causeFollow{}, follow)
	})

	WithCause(42, func() {
		if Cause() != 42 {
			t.Fatalf("Cause inside WithCause = %d", Cause())
		}
		AddToQueue(causeProbe{name: "combat"})
		// What a player types never belongs to the round, even when the
		// input worker queues it mid-dispatch.
		AddTyped(Input{UserId: 3, InputText: "say hi"})
	})
	if Cause() != 0 {
		t.Fatalf("Cause after WithCause = %d", Cause())
	}
	AddToQueue(causeProbe{name: "plain"})

	var inputCause uint64 = 99
	var inputTyped bool
	in := RegisterListener(Input{}, func(e Event) ListenerReturn {
		inputCause = Cause()
		inputTyped = Typed()
		return Continue
	})
	t.Cleanup(func() { UnregisterListener(Input{}, in) })

	ProcessEvents()

	want := map[string]uint64{"combat": 42, "combat-follow": 42, "plain": 0, "plain-follow": 0}
	for k, v := range want {
		if got, ok := seen[k]; !ok || got != v {
			t.Errorf("%s dispatched with cause %d (seen %v), want %d", k, got, ok, v)
		}
	}
	if inputCause != 0 || !inputTyped {
		t.Errorf("typed Input dispatched with cause %d typed %v, want 0 and typed", inputCause, inputTyped)
	}
	if Cause() != 0 {
		t.Errorf("Cause after ProcessEvents = %d", Cause())
	}
}

func TestMobInputInheritsCause(t *testing.T) {
	drainForTest()
	t.Cleanup(drainForTest)
	var got uint64
	in := RegisterListener(Input{}, func(e Event) ListenerReturn {
		got = Cause()
		return Continue
	})
	t.Cleanup(func() { UnregisterListener(Input{}, in) })
	WithCause(8, func() { AddToQueue(Input{MobInstanceId: 5, InputText: "emote snarls"}) })
	ProcessEvents()
	if got != 8 {
		t.Fatalf("mob Input cause = %d, want 8", got)
	}
}

func TestRequeuedEventKeepsItsCause(t *testing.T) {
	drainForTest()
	t.Cleanup(drainForTest)
	var causes []uint64
	probe := RegisterListener(causeProbe{}, func(e Event) ListenerReturn {
		causes = append(causes, Cause())
		if len(causes) == 1 {
			return CancelAndRequeue
		}
		return Continue
	})
	t.Cleanup(func() { UnregisterListener(causeProbe{}, probe) })
	WithCause(6, func() { AddToQueue(causeProbe{name: "requeue"}) })
	ProcessEvents()
	ProcessEvents()
	if len(causes) != 2 || causes[0] != 6 || causes[1] != 6 {
		t.Fatalf("requeued causes = %v, want [6 6]", causes)
	}
}

// TestGameCommandsInheritTheRound: a command the game issues for a player
// mid-round (a slain player's "suicide") is part of the round, and what a
// typed command causes is typed.
func TestGameCommandsInheritTheRoundAndTypedSpreads(t *testing.T) {
	drainForTest()
	t.Cleanup(drainForTest)
	type seen struct {
		cause uint64
		typed bool
	}
	got := map[string]seen{}
	in := RegisterListener(Input{}, func(e Event) ListenerReturn {
		i := e.(Input)
		got[i.InputText] = seen{Cause(), Typed()}
		if i.InputText == "look" {
			AddToQueue(causeProbe{name: "look-output"})
		}
		return Continue
	})
	probe := RegisterListener(causeProbe{}, func(e Event) ListenerReturn {
		got[e.(causeProbe).name] = seen{Cause(), Typed()}
		return Continue
	})
	t.Cleanup(func() {
		UnregisterListener(Input{}, in)
		UnregisterListener(causeProbe{}, probe)
	})
	WithCause(10, func() { AddToQueue(Input{UserId: 3, InputText: "suicide"}) })
	AddTyped(Input{UserId: 3, InputText: "look"})
	ProcessEvents()
	want := map[string]seen{
		"suicide":     {10, false},
		"look":        {0, true},
		"look-output": {0, true},
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s dispatched with %+v, want %+v", k, got[k], v)
		}
	}
	if Typed() {
		t.Error("typed leaked past dispatch")
	}
}
