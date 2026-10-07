// Package modtimer holds the one-shot timer scaffolding that the camping and
// expedition modules share: the injectable Scheduler seam, and helpers that
// keep a per-leader timer map and generation counter consistent.
//
// Callers keep their own maps and mutex; every helper must be called with
// that mutex held. A generation counter makes a callback whose timer was
// replaced or stopped after it fired a no-op.
package modtimer

import "time"

// Timer is a cancellable scheduled callback.
type Timer interface {
	Stop() bool
}

// Scheduler schedules a one-shot callback after a delay. Tests inject a
// deterministic implementation.
type Scheduler interface {
	AfterFunc(d time.Duration, f func()) Timer
}

// Direct runs the callback on the timer's own goroutine.
type Direct struct{}

// AfterFunc runs f on a timer goroutine once d has passed; a negative delay
// fires immediately.
func (Direct) AfterFunc(d time.Duration, f func()) Timer {
	return Wrap(time.AfterFunc(clamp(d), f))
}

// Wrap adapts a standard library timer to Timer.
func Wrap(t *time.Timer) Timer { return realTimer{timer: t} }

// Clamp returns d, or zero when d is negative.
func Clamp(d time.Duration) time.Duration { return clamp(d) }

func clamp(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	return d
}

type realTimer struct{ timer *time.Timer }

func (r realTimer) Stop() bool { return r.timer.Stop() }

// Arm replaces key's timer: it bumps the key's generation, stops any timer
// already set and schedules run, handing it the new generation. A nil
// generations map is created, and returned for the caller to keep.
func Arm(timers map[int]Timer, generations map[int]uint64, key int, s Scheduler, d time.Duration, run func(generation uint64)) map[int]uint64 {
	if generations == nil {
		generations = map[int]uint64{}
	}
	generations[key]++
	generation := generations[key]
	Stop(timers, key)
	timers[key] = s.AfterFunc(d, func() { run(generation) })
	return generations
}

// Stop cancels and forgets key's timer.
func Stop(timers map[int]Timer, key int) {
	if timer, ok := timers[key]; ok {
		timer.Stop()
		delete(timers, key)
	}
}

// Claim reports whether a firing callback is still the live one for key: its
// generation is current and its timer was not stopped. A live firing is
// removed from the map so it runs once.
func Claim(timers map[int]Timer, generations map[int]uint64, key int, generation uint64) bool {
	if generations[key] != generation {
		return false
	}
	if _, ok := timers[key]; !ok {
		return false
	}
	delete(timers, key)
	return true
}
