package expedition

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJourneyProgress (Phase 26a): a journey's progress for the summary,
// read without side effects; none when not travelling.
func TestJourneyProgress(t *testing.T) {
	user := travelUser(t, 7, 100)
	now := baseTime()
	store := &fakeStore{}
	module := newTestModule(store, &fakeScheduler{}, &fakeMover{user: user}, &fakeSurvival{}, func() time.Time { return now }, testProfiles())
	_, ok := module.JourneyProgress(7)
	assert.False(t, ok, "not travelling")

	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	saves := store.saveCalls
	p, ok := module.JourneyProgress(7)
	require.True(t, ok)
	assert.False(t, p.Interrupted)
	assert.Equal(t, 0, p.Percent)
	assert.Equal(t, "oak-road", p.Route)
	assert.Greater(t, p.Remaining, time.Duration(0))
	assert.Equal(t, saves, store.saveCalls, "reads only")

	module.sessions[7] = interruptedSession()
	p, ok = module.JourneyProgress(7)
	require.True(t, ok)
	assert.True(t, p.Interrupted)
}
