package gathering

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gathering"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// watchProgress records the GatherProgress events a function queues.
func watchProgress(t *testing.T, f func()) []events.GatherProgress {
	t.Helper()
	var got []events.GatherProgress
	id := events.RegisterListener(events.GatherProgress{}, func(e events.Event) events.ListenerReturn {
		got = append(got, e.(events.GatherProgress))
		return events.Continue
	})
	defer events.UnregisterListener(events.GatherProgress{}, id)
	f()
	events.ProcessEvents()
	return got
}

// TestGatherProgressIsAnnouncedAndReadable (Phase 45): a start, a finish and
// a stop each queue a GatherProgress event for the web client, and the
// progress provider the prompt and status sheet read reports the time left.
func TestGatherProgressIsAnnouncedAndReadable(t *testing.T) {
	w := newWorld(t, "herbs")
	w.rolls = []int{0, 0, 0, 0, 0}
	gathering.SetProgressProvider(w.m.progress)
	t.Cleanup(func() { gathering.SetProgressProvider(nil) })

	_, busy := gathering.ProgressOf(7)
	assert.False(t, busy, "nothing in progress yet")

	started := watchProgress(t, func() { w.start(gathering.Herbs) })
	require.Len(t, started, 1)
	assert.Equal(t, "start", started[0].Phase)
	assert.Equal(t, 7, started[0].UserId)
	assert.Equal(t, "gathering herbs", started[0].Label)
	assert.Equal(t, 20, started[0].Seconds, "the client animates the bar over this many seconds")

	w.now = w.now.Add(5 * time.Second)
	p, busy := gathering.ProgressOf(7)
	require.True(t, busy)
	assert.Equal(t, gathering.Herbs, p.Kind)
	assert.Equal(t, 15*time.Second, p.Remaining)
	assert.Equal(t, 25, p.Percent())

	done := watchProgress(t, func() { w.finishAfter(gathering.Herbs) })
	require.Len(t, done, 1)
	assert.Equal(t, "done", done[0].Phase)
	assert.NotEmpty(t, done[0].Lines, "the result is carried with the finish")
	_, busy = gathering.ProgressOf(7)
	assert.False(t, busy, "and the work is no longer in progress")
}

func TestGatherStoppedIsAnnounced(t *testing.T) {
	w := newWorld(t, "herbs")
	w.start(gathering.Herbs)
	events.ProcessEvents() // the start announcement is not under test
	got := watchProgress(t, func() { w.m.onInput(events.Input{UserId: 7, InputText: "say hello"}) })
	require.Len(t, got, 1)
	assert.Equal(t, "stopped", got[0].Phase)
	assert.Contains(t, got[0].Lines[0], "abandoned")
}
