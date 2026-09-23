package camping

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "low", "", false)
	os.Exit(m.Run())
}

func baseTime() time.Time {
	return time.Date(2026, time.September, 22, 12, 0, 0, 0, time.UTC)
}

type fakeStore struct {
	saved        Registry
	loadErr      error
	saveErr      error
	failNextSave bool
	loadCalls    int
	saveCalls    int
	log          *orderLog
}

func (f *fakeStore) Load(registry *Registry) error {
	f.loadCalls++
	if f.loadErr != nil {
		return f.loadErr
	}
	*registry = f.saved.Clone()
	return nil
}

func (f *fakeStore) Save(registry Registry) error {
	f.saveCalls++
	if f.log != nil {
		f.log.entries = append(f.log.entries, "save")
	}
	if f.failNextSave {
		f.failNextSave = false
		return assert.AnError
	}
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = registry.Clone()
	return nil
}

type orderLog struct{ entries []string }

type fakeTimer struct{ stopped *bool }

func (t fakeTimer) Stop() bool {
	if t.stopped != nil {
		*t.stopped = true
	}
	return true
}

type fakeScheduler struct {
	log       *orderLog
	callbacks []func()
	delays    []time.Duration
}

func (f *fakeScheduler) AfterFunc(d time.Duration, cb func()) Timer {
	if f.log != nil {
		f.log.entries = append(f.log.entries, "schedule")
	}
	f.delays = append(f.delays, d)
	f.callbacks = append(f.callbacks, cb)
	return fakeTimer{}
}

func (f *fakeScheduler) fireLatest() {
	if n := len(f.callbacks); n > 0 {
		f.callbacks[n-1]()
	}
}

type fakeSurvival struct {
	availableErr error
	applyErr     error
	applyCalls   int
	appliedFor   []int
	needs        []survival.MemberNeeds
}

func (f *fakeSurvival) Available() error { return f.availableErr }

func (f *fakeSurvival) ApplyCompanyRestRecovery(leaderUserID int, _ string, _ int) ([]survival.ExertionResult, error) {
	f.applyCalls++
	if f.applyErr != nil {
		return nil, f.applyErr
	}
	f.appliedFor = append(f.appliedFor, leaderUserID)
	return nil, nil
}

func (f *fakeSurvival) CompanyNeeds(_ int) []survival.MemberNeeds { return f.needs }

func newTestModule(store Store, scheduler Scheduler, surv Survival, clock func() time.Time) *CampingModule {
	return &CampingModule{
		store:           store,
		scheduler:       scheduler,
		survival:        surv,
		clock:           clock,
		camps:           map[int]camping.Camp{},
		recoveryApplied: map[int]bool{},
		timers:          map[int]Timer{},
		timerGeneration: map[int]uint64{},
	}
}

func eligibleRoom() *rooms.Room {
	return &rooms.Room{RoomId: 100, Title: "A Clearing", Tags: []string{"camping"}}
}

func ineligibleRoom() *rooms.Room {
	return &rooms.Room{RoomId: 100, Title: "A Road", Tags: []string{}}
}

func campUser(t *testing.T, userId, roomId int) *users.UserRecord {
	t.Helper()
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(userId, 1)
	user.Character.RoomId = roomId
	users.SetTestUser(user)
	return user
}

func captureMessages(t *testing.T) *[]string {
	t.Helper()
	messages := []string{}
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		messages = append(messages, e.(events.Message).Text)
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	return &messages
}

// --- Task 3: eligibility, duplicate refusal, fire/rest preconditions, status ---

func TestEstablishRefusesIneligibleRoom(t *testing.T) {
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{}, baseTime)
	user := campUser(t, 7, 100)

	msg := module.establish(user, ineligibleRoom())
	assert.Contains(t, msg, "nowhere here")
	assert.Empty(t, module.camps)
	assert.Zero(t, module.store.(*fakeStore).saveCalls)
}

func TestEstablishRefusesDuplicateWithoutPersistenceChange(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store, &fakeScheduler{}, &fakeSurvival{}, baseTime)
	user := campUser(t, 7, 100)

	msg := module.establish(user, eligibleRoom())
	assert.Contains(t, msg, "make camp")
	original := store.saved.Camps[7]
	saveCallsAfterFirst := store.saveCalls

	msg = module.establish(user, eligibleRoom())
	assert.Contains(t, msg, "already have a camp")
	assert.Equal(t, original, store.saved.Camps[7])
	assert.Equal(t, saveCallsAfterFirst, store.saveCalls)
}

func TestEstablishPersistsEligibleCamp(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store, &fakeScheduler{}, &fakeSurvival{}, baseTime)
	user := campUser(t, 7, 100)

	msg := module.establish(user, eligibleRoom())
	assert.Contains(t, msg, "make camp")
	require.Contains(t, store.saved.Camps, 7)
	assert.Equal(t, 100, store.saved.Camps[7].RoomID)
	assert.False(t, store.saved.Camps[7].FireLit)
}

func TestFireRequiresExistingCampAtSameRoom(t *testing.T) {
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{}, baseTime)
	user := campUser(t, 7, 100)

	msg := module.lightFire(user, eligibleRoom())
	assert.Contains(t, msg, "no camp")

	module.establish(user, eligibleRoom())
	elsewhere := &rooms.Room{RoomId: 999, Tags: []string{"camping"}}
	msg = module.lightFire(user, elsewhere)
	assert.Contains(t, msg, "not here")

	msg = module.lightFire(user, eligibleRoom())
	assert.Contains(t, msg, "light a crackling campfire")
	assert.True(t, module.camps[7].FireLit)

	msg = module.lightFire(user, eligibleRoom())
	assert.Contains(t, msg, "already lit")
}

func TestRestRequiresFireAndSchedulesSixtySeconds(t *testing.T) {
	log := &orderLog{}
	store := &fakeStore{log: log}
	scheduler := &fakeScheduler{log: log}
	module := newTestModule(store, scheduler, &fakeSurvival{}, baseTime)
	user := campUser(t, 7, 100)

	module.establish(user, eligibleRoom())
	msg := module.startRest(user, eligibleRoom())
	assert.Contains(t, msg, "need a lit campfire")
	assert.Empty(t, scheduler.callbacks)

	module.lightFire(user, eligibleRoom())
	msg = module.startRest(user, eligibleRoom())
	assert.Contains(t, msg, "settle in")
	require.Len(t, scheduler.delays, 1)
	assert.Equal(t, camping.RestDuration, scheduler.delays[0])
	assert.Equal(t, camping.Resting, module.camps[7].Rest.State)
}

func TestRestRequiresSurvivalAvailable(t *testing.T) {
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{availableErr: assert.AnError}, baseTime)
	user := campUser(t, 7, 100)
	module.establish(user, eligibleRoom())
	module.lightFire(user, eligibleRoom())

	msg := module.startRest(user, eligibleRoom())
	assert.Contains(t, msg, assert.AnError.Error())
	assert.Nil(t, module.camps[7].Rest)
}

func TestStatusShowsCampFireAndRestState(t *testing.T) {
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{needs: []survival.MemberNeeds{
		{Name: "Hero", Needs: survival.Needs{Hunger: 90, Thirst: 90, Fatigue: 40}},
	}}, baseTime)
	user := campUser(t, 7, 100)
	module.establish(user, eligibleRoom())

	status := module.status(user.UserId)
	assert.Contains(t, status, "no fire lit")
	assert.Contains(t, status, "Hero")

	module.lightFire(user, eligibleRoom())
	status = module.status(user.UserId)
	assert.Contains(t, status, "campfire is lit")

	module.startRest(user, eligibleRoom())
	status = module.status(user.UserId)
	assert.Contains(t, status, "Resting: 0% complete")
}

func TestStatusWithoutCamp(t *testing.T) {
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{}, baseTime)
	assert.Equal(t, "You have no camp.", module.status(7))
}

// --- Task 4: completion, recovery idempotency, recovery, views, movement ---

func TestRestCompletionAppliesOneRecoveryOperation(t *testing.T) {
	now := baseTime()
	store := &fakeStore{}
	scheduler := &fakeScheduler{}
	surv := &fakeSurvival{}
	module := newTestModule(store, scheduler, surv, func() time.Time { return now })
	user := campUser(t, 7, 100)
	messages := captureMessages(t)

	module.establish(user, eligibleRoom())
	module.lightFire(user, eligibleRoom())
	module.startRest(user, eligibleRoom())

	now = now.Add(camping.RestDuration)
	scheduler.fireLatest()
	events.ProcessEvents()

	assert.Equal(t, 1, surv.applyCalls)
	assert.Equal(t, camping.Completed, module.camps[7].Rest.State)
	assert.True(t, module.recoveryApplied[7])
	assert.Contains(t, strings.Join(*messages, ""), "feels rested")

	// A second timer fire for the same generation must not recur; simulate a
	// stale callback firing again by directly invoking onTimer.
	module.onTimer(7, module.timerGeneration[7])
	assert.Equal(t, 1, surv.applyCalls, "stale/duplicate callback must not double-apply recovery")
}

func TestStaleTimerCallbackCannotCompleteReplacementRest(t *testing.T) {
	now := baseTime()
	store := &fakeStore{}
	scheduler := &fakeScheduler{}
	surv := &fakeSurvival{}
	module := newTestModule(store, scheduler, surv, func() time.Time { return now })
	user := campUser(t, 7, 100)

	module.establish(user, eligibleRoom())
	module.lightFire(user, eligibleRoom())
	module.startRest(user, eligibleRoom())
	staleGeneration := module.timerGeneration[7]

	// Reschedule (as recovery would) without completing, bumping the generation.
	module.scheduleLocked(module.camps[7])

	module.onTimer(7, staleGeneration)
	assert.Equal(t, 0, surv.applyCalls, "a stale generation callback must be a no-op")
	assert.Equal(t, camping.Resting, module.camps[7].Rest.State)
}

func TestFailedSurvivalRecoveryRetriesWithoutDoubleApplying(t *testing.T) {
	now := baseTime()
	store := &fakeStore{}
	scheduler := &fakeScheduler{}
	surv := &fakeSurvival{applyErr: assert.AnError}
	module := newTestModule(store, scheduler, surv, func() time.Time { return now })
	user := campUser(t, 7, 100)

	module.establish(user, eligibleRoom())
	module.lightFire(user, eligibleRoom())
	module.startRest(user, eligibleRoom())

	now = now.Add(camping.RestDuration)
	scheduler.fireLatest()
	events.ProcessEvents()

	// Completion persisted even though recovery failed; state must be Completed
	// and not yet marked recovered.
	require.NotNil(t, module.camps[7].Rest)
	assert.Equal(t, camping.Completed, module.camps[7].Rest.State)
	assert.False(t, module.recoveryApplied[7])
	assert.Equal(t, 1, surv.applyCalls)

	// A later status check retries recovery, now succeeding.
	surv.applyErr = nil
	module.status(user.UserId)
	assert.Equal(t, 2, surv.applyCalls)
	assert.True(t, module.recoveryApplied[7])

	// A further status check must not call survival again.
	module.status(user.UserId)
	assert.Equal(t, 2, surv.applyCalls)
}

func TestFailedFinalizationSaveRetriesCompletionOnNextSync(t *testing.T) {
	now := baseTime()
	store := &fakeStore{}
	scheduler := &fakeScheduler{}
	surv := &fakeSurvival{}
	module := newTestModule(store, scheduler, surv, func() time.Time { return now })
	user := campUser(t, 7, 100)

	module.establish(user, eligibleRoom())
	module.lightFire(user, eligibleRoom())
	module.startRest(user, eligibleRoom())

	now = now.Add(camping.RestDuration)
	store.failNextSave = true
	err := module.syncLocked(user.UserId)
	require.Error(t, err)
	assert.Equal(t, camping.Resting, module.camps[7].Rest.State, "failed persistence must not leave an unsaved Completed transition")
	assert.Zero(t, surv.applyCalls)

	err = module.syncLocked(user.UserId)
	require.NoError(t, err)
	assert.Equal(t, camping.Completed, module.camps[7].Rest.State)
	assert.Equal(t, 1, surv.applyCalls)
}

func TestRestartRecoveryReschedulesActiveRest(t *testing.T) {
	now := baseTime()
	saved := Registry{Camps: map[int]camping.Camp{
		7: mustResting(t, 7, 100, now.Add(-10*time.Second)),
	}}
	store := &fakeStore{saved: saved}
	scheduler := &fakeScheduler{}
	surv := &fakeSurvival{}
	module := newTestModule(store, scheduler, surv, func() time.Time { return now })

	module.load()

	require.Len(t, scheduler.delays, 1)
	assert.Equal(t, camping.RestDuration-10*time.Second, scheduler.delays[0])
	assert.Zero(t, surv.applyCalls)
}

func TestRestartRecoveryCompletesOverdueRestOnce(t *testing.T) {
	now := baseTime()
	saved := Registry{Camps: map[int]camping.Camp{
		7: mustResting(t, 7, 100, now.Add(-2*camping.RestDuration)),
	}}
	store := &fakeStore{saved: saved}
	scheduler := &fakeScheduler{}
	surv := &fakeSurvival{}
	module := newTestModule(store, scheduler, surv, func() time.Time { return now })

	module.load()

	assert.Equal(t, camping.Completed, module.camps[7].Rest.State)
	assert.Equal(t, 1, surv.applyCalls)
	assert.True(t, module.recoveryApplied[7])
	assert.Empty(t, scheduler.delays, "an overdue rest completes rather than rescheduling")
}

func TestRestartRecoveryRetriesOnlyPendingCompletedRecovery(t *testing.T) {
	now := baseTime()
	completed := mustResting(t, 7, 100, now.Add(-2*camping.RestDuration))
	completed, err := completed.CompleteRest(now.Add(-camping.RestDuration))
	require.NoError(t, err)
	saved := Registry{Camps: map[int]camping.Camp{7: completed}, RecoveryApplied: map[int]bool{}}
	store := &fakeStore{saved: saved}
	surv := &fakeSurvival{}
	module := newTestModule(store, &fakeScheduler{}, surv, func() time.Time { return now })

	module.load()

	assert.Equal(t, 1, surv.applyCalls)
	assert.True(t, module.recoveryApplied[7])
}

func TestRestartRecoveryRetainsInvalidCampForRepair(t *testing.T) {
	store := &fakeStore{saved: Registry{Camps: map[int]camping.Camp{
		7: {LeaderUserID: 7, RoomID: -1},
	}}}
	surv := &fakeSurvival{}
	module := newTestModule(store, &fakeScheduler{}, surv, baseTime)

	module.load()

	require.Contains(t, module.camps, 7)
	assert.Equal(t, -1, module.camps[7].RoomID)
	assert.Zero(t, surv.applyCalls)
}

func TestBreakCampRemovesIdleCampAndRefusesWhileResting(t *testing.T) {
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{}, baseTime)
	user := campUser(t, 7, 100)
	module.establish(user, eligibleRoom())
	module.lightFire(user, eligibleRoom())
	module.startRest(user, eligibleRoom())

	msg := module.breakCamp(user, eligibleRoom())
	assert.Contains(t, msg, "can't break camp while resting")
	assert.Contains(t, module.camps, 7)

	now := baseTime().Add(camping.RestDuration)
	module.clock = func() time.Time { return now }
	msg = module.breakCamp(user, eligibleRoom())
	assert.Contains(t, msg, "break camp")
	assert.NotContains(t, module.camps, 7)
	assert.NotContains(t, module.recoveryApplied, 7)
}

func TestMovementBlockedOnlyWhileResting(t *testing.T) {
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{}, baseTime)
	user := campUser(t, 7, 100)

	blocked, _ := module.MovementBlocked(7)
	assert.False(t, blocked, "no camp must never block movement")

	module.establish(user, eligibleRoom())
	blocked, _ = module.MovementBlocked(7)
	assert.False(t, blocked, "an idle camp must never block movement")

	module.lightFire(user, eligibleRoom())
	module.startRest(user, eligibleRoom())
	blocked, msg := module.MovementBlocked(7)
	assert.True(t, blocked)
	assert.Contains(t, msg, "resting")
}

func TestRenderCampViewOnlyWhileResting(t *testing.T) {
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{}, baseTime)
	user := campUser(t, 7, 100)
	captureMessages(t)

	handled, err := module.RenderCampView(7)
	require.NoError(t, err)
	assert.False(t, handled)

	module.establish(user, eligibleRoom())
	handled, err = module.RenderCampView(7)
	require.NoError(t, err)
	assert.False(t, handled, "an idle camp does not replace room rendering")

	module.lightFire(user, eligibleRoom())
	module.startRest(user, eligibleRoom())
	handled, err = module.RenderCampView(7)
	require.NoError(t, err)
	assert.True(t, handled)
}

func TestUserCommandDispatchAndUsage(t *testing.T) {
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{}, baseTime)
	user := campUser(t, 7, 100)
	messages := captureMessages(t)

	_, err := module.userCommand("bogus", user, eligibleRoom(), 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*messages, ""), campUsage)

	_, err = module.userCommand("", user, eligibleRoom(), 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*messages, ""), "make camp")
}

func mustResting(t *testing.T, leaderUserID, roomID int, startedAt time.Time) camping.Camp {
	t.Helper()
	camp, err := camping.Established(leaderUserID, roomID)
	require.NoError(t, err)
	camp, err = camp.LightFire()
	require.NoError(t, err)
	camp, err = camp.StartRest(startedAt)
	require.NoError(t, err)
	return camp
}

func TestLitCampfireIsLightFixture(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store, &fakeScheduler{}, &fakeSurvival{}, baseTime)
	user := campUser(t, 7, 100)

	module.establish(user, eligibleRoom())
	assert.False(t, module.RoomHasLitFire(100), "an unlit camp is not a light fixture")

	module.lightFire(user, eligibleRoom())
	assert.True(t, module.RoomHasLitFire(100), "a lit campfire lights its room")
	assert.False(t, module.RoomHasLitFire(300), "a room with no camp has no campfire")

	module.breakCamp(user, eligibleRoom())
	assert.False(t, module.RoomHasLitFire(100), "breaking camp puts the fire out")
}

func TestLitCampfireSnapshotFollowsFailedSaveRevert(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store, &fakeScheduler{}, &fakeSurvival{}, baseTime)
	user := campUser(t, 7, 100)
	module.establish(user, eligibleRoom())

	store.saveErr = errors.New("disk full")
	module.lightFire(user, eligibleRoom())
	assert.False(t, module.camps[7].FireLit, "a failed save reverts the fire")
	assert.False(t, module.RoomHasLitFire(100), "the light snapshot follows the reverted state")
}
