package camping

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
)

// TestCampTimerRunsOnTheGameLoop (Phase 65 review): the real scheduler's
// timer only queues its callback, so the end of a rest (banter and the
// company's bonds, which write the company registry and may send a rival
// away) runs on the game loop, never on the timer's goroutine.
func TestCampTimerRunsOnTheGameLoop(t *testing.T) {
	// The module's init registered onCampTimerDue.
	var ran atomic.Int32
	realScheduler{}.AfterFunc(0, func() { ran.Add(1) })
	time.Sleep(50 * time.Millisecond) // the timer has fired, and queued
	assert.Zero(t, ran.Load(), "not run on the timer's goroutine")
	events.ProcessEvents()
	assert.Equal(t, int32(1), ran.Load(), "run once, on the loop")
}
