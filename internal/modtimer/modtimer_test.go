package modtimer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type fakeTimer struct{ stopped bool }

func (f *fakeTimer) Stop() bool { f.stopped = true; return true }

type fakeScheduler struct {
	timers    []*fakeTimer
	callbacks []func()
}

func (s *fakeScheduler) AfterFunc(_ time.Duration, f func()) Timer {
	t := &fakeTimer{}
	s.timers = append(s.timers, t)
	s.callbacks = append(s.callbacks, f)
	return t
}

func TestArmReplacesTimerAndStalesOldGeneration(t *testing.T) {
	s := &fakeScheduler{}
	timers := map[int]Timer{}
	var gens map[int]uint64
	var fired []uint64
	run := func(g uint64) { fired = append(fired, g) }

	gens = Arm(timers, gens, 7, s, time.Second, run)
	gens = Arm(timers, gens, 7, s, time.Second, run)

	assert.True(t, s.timers[0].stopped, "the first timer is stopped when replaced")
	assert.Len(t, timers, 1)
	s.callbacks[0]()
	s.callbacks[1]()
	assert.Equal(t, []uint64{1, 2}, fired)
	assert.False(t, Claim(timers, gens, 7, 1), "the replaced generation is stale")
	assert.True(t, Claim(timers, gens, 7, 2), "the live generation claims once")
	assert.Empty(t, timers)
	assert.False(t, Claim(timers, gens, 7, 2), "a second firing finds no timer")
}

func TestStopForgetsTimer(t *testing.T) {
	s := &fakeScheduler{}
	timers := map[int]Timer{}
	gens := Arm(timers, nil, 3, s, time.Second, func(uint64) {})
	Stop(timers, 3)
	assert.True(t, s.timers[0].stopped)
	assert.False(t, Claim(timers, gens, 3, 1), "a stopped timer cannot be claimed")
	Stop(timers, 3)
}

func TestDirectFiresImmediatelyForNegativeDelay(t *testing.T) {
	done := make(chan struct{})
	Direct{}.AfterFunc(-time.Hour, func() { close(done) })
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("negative delay never fired")
	}
}
