package expedition

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func interruptedSession() expedition.TravelSession {
	return expedition.TravelSession{LeaderUserID: 7, OriginRoomID: 100, DestinationRoomID: 200, ExitName: "north", ProfileName: "oak-road", StartedAtUTC: baseTime(), PausedAtUTC: baseTime().Add(5 * time.Second), InterruptionTriggered: true, Interruption: &expedition.TravelInterruption{Kind: expedition.FallenTree, Checkpoint: 5}, State: expedition.Interrupted}
}

// abandonUser drains the event queue after the test, so no arrival text
// leaks into a later test's message capture.
func abandonUser(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { events.ProcessEvents() })
}

func TestAbandonForDeathTraveling(t *testing.T) {
	abandonUser(t)
	user := travelUser(t, 7, 100)
	store := &fakeStore{}
	scheduler := &fakeScheduler{}
	mover := &fakeMover{user: user}
	now := baseTime()
	module := newTestModule(store, scheduler, mover, &fakeSurvival{}, func() time.Time { return now }, testProfiles())
	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	require.Len(t, scheduler.callbacks, 1)

	require.NoError(t, module.AbandonForDeath(7))

	assert.NotContains(t, module.sessions, 7)
	assert.Empty(t, store.saved.Sessions, "saved at once")
	blocked, _ := module.MovementBlocked(7)
	assert.False(t, blocked)
	now = baseTime().Add(time.Minute)
	scheduler.fire(0)
	assert.Empty(t, mover.moves, "the old timer does nothing: no move to the destination")
	assert.Equal(t, 100, user.Character.RoomId)
}

func TestAbandonForDeathInterrupted(t *testing.T) {
	user := travelUser(t, 7, 100)
	store := &fakeStore{}
	mover := &fakeMover{user: user}
	module := newTestModule(store, &fakeScheduler{}, mover, &fakeSurvival{}, baseTime, interruptionProfiles())
	module.sessions[7] = interruptedSession()

	require.NoError(t, module.AbandonForDeath(7))

	assert.NotContains(t, module.sessions, 7)
	assert.Empty(t, store.saved.Sessions)
	assert.Empty(t, mover.moves)
	assert.Equal(t, "You are not travelling.", module.status(7))
}

func TestAbandonForDeathCompletedRecord(t *testing.T) {
	user := travelUser(t, 7, 100)
	store := &fakeStore{saved: savedSession(expedition.Completed, baseTime())}
	mover := &fakeMover{user: user}
	module := newTestModule(store, &fakeScheduler{}, mover, &fakeSurvival{}, baseTime, testProfiles())
	module.sessions[7] = store.saved.Sessions[7]

	require.NoError(t, module.AbandonForDeath(7))
	module.load()

	assert.Empty(t, module.sessions, "recovery finds nothing to retry")
	assert.Empty(t, mover.moves, "a completed record would have moved the leader")
}

func TestAbandonForDeathSaveFailureRestores(t *testing.T) {
	abandonUser(t)
	user := travelUser(t, 7, 100)
	store := &fakeStore{}
	scheduler := &fakeScheduler{}
	mover := &fakeMover{user: user}
	now := baseTime()
	module := newTestModule(store, scheduler, mover, &fakeSurvival{}, func() time.Time { return now }, testProfiles())
	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	store.failNextSave = true

	assert.Error(t, module.AbandonForDeath(7))

	assert.Equal(t, expedition.Traveling, module.sessions[7].State, "the session is kept")
	assert.Contains(t, store.saved.Sessions, 7)
	assert.Contains(t, module.timers, 7, "and its timer")
	now = baseTime().Add(time.Minute)
	scheduler.fire(0)
	assert.Equal(t, []int{200}, mover.moves, "the kept journey still completes")
}

func TestAbandonForDeathNoSession(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store, &fakeScheduler{}, &fakeMover{}, &fakeSurvival{}, baseTime, testProfiles())

	assert.NoError(t, module.AbandonForDeath(7))
	assert.Zero(t, store.saveCalls, "nothing to save")

	module.loadErr = assert.AnError
	assert.Error(t, module.AbandonForDeath(7), "unreadable travel data may hold a journey")
	assert.Zero(t, store.saveCalls)
}

func TestAbandonForDeathUnavailablePersistence(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store, &fakeScheduler{}, &fakeMover{}, &fakeSurvival{}, baseTime, interruptionProfiles())
	module.sessions[7] = interruptedSession()
	module.loadErr = assert.AnError

	assert.Error(t, module.AbandonForDeath(7))
	assert.Contains(t, module.sessions, 7)
}
