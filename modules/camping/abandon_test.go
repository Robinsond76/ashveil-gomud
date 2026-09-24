package camping

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAbandonForDeathRestingCamp(t *testing.T) {
	now := baseTime()
	store := &fakeStore{}
	scheduler := &fakeScheduler{}
	surv := &fakeSurvival{}
	module := newTestModule(store, scheduler, surv, func() time.Time { return now })
	user := campUser(t, 7, 100)
	module.establish(user, eligibleRoom())
	module.lightFire(user, eligibleRoom())
	module.startRest(user, eligibleRoom())
	require.True(t, module.RoomHasLitFire(100))

	require.NoError(t, module.AbandonForDeath(7))

	assert.Empty(t, module.camps)
	assert.Empty(t, store.saved.Camps, "saved at once")
	assert.False(t, module.RoomHasLitFire(100), "the fire goes with the camp")
	blocked, _ := module.MovementBlocked(7)
	assert.False(t, blocked)
	now = now.Add(camping.RestDuration + time.Minute)
	scheduler.fireLatest()
	assert.Zero(t, surv.applyCalls, "a rest cut short by death grants no recovery")
	assert.False(t, module.restedPending[7], "and no Rested")
}

func TestAbandonForDeathIdleCamp(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store, &fakeScheduler{}, &fakeSurvival{}, baseTime)
	user := campUser(t, 7, 100)
	module.establish(user, eligibleRoom())

	require.NoError(t, module.AbandonForDeath(7))

	assert.Empty(t, module.camps)
	assert.Empty(t, store.saved.Camps)
}

func TestAbandonForDeathFinishedRestKeepsRecovery(t *testing.T) {
	now := baseTime()
	store := &fakeStore{}
	surv := &fakeSurvival{}
	module := newTestModule(store, &fakeScheduler{}, surv, func() time.Time { return now })
	user := campUser(t, 7, 100)
	module.establish(user, eligibleRoom())
	module.lightFire(user, eligibleRoom())
	module.startRest(user, eligibleRoom())
	now = now.Add(camping.RestDuration + time.Second) // due, timer not yet fired

	require.NoError(t, module.AbandonForDeath(7))

	assert.Equal(t, 1, surv.applyCalls, "the finished rest keeps its recovery")
	assert.True(t, store.saved.RestedPending[7])
	assert.Empty(t, store.saved.Camps)
	assert.Empty(t, store.saved.RecoveryApplied)
}

func TestAbandonForDeathInnStay(t *testing.T) {
	e := newInnEnv(t)
	user := campUser(t, 7, 2003)
	user.Character.Gold = 25
	e.module.innRest(user, innRoom())
	require.Contains(t, e.store.saved.Stays, 7)

	require.NoError(t, e.module.AbandonForDeath(7))

	assert.Empty(t, e.module.stays)
	assert.Empty(t, e.store.saved.Stays)
	assert.Equal(t, 15, user.Character.Gold, "no refund")
	*e.now = e.now.Add(time.Hour)
	e.scheduler.fireLatest()
	assert.Zero(t, e.surv.applyCalls, "no recovery")
	assert.False(t, e.module.wellRestedPending[7], "and no Well Rested")
}

func TestAbandonForDeathCampSaveFailureRestores(t *testing.T) {
	now := baseTime()
	store := &fakeStore{}
	scheduler := &fakeScheduler{}
	surv := &fakeSurvival{}
	module := newTestModule(store, scheduler, surv, func() time.Time { return now })
	user := campUser(t, 7, 100)
	module.establish(user, eligibleRoom())
	module.lightFire(user, eligibleRoom())
	module.startRest(user, eligibleRoom())
	store.failNextSave = true

	assert.Error(t, module.AbandonForDeath(7))

	require.Contains(t, module.camps, 7, "the camp is kept")
	assert.Equal(t, camping.Resting, module.camps[7].Rest.State)
	assert.Contains(t, module.timers, 7, "with its timer")
	assert.True(t, module.RoomHasLitFire(100))
	now = now.Add(camping.RestDuration + time.Second)
	scheduler.fireLatest()
	assert.Equal(t, 1, surv.applyCalls, "the kept rest still completes")
}

func TestAbandonForDeathNothing(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store, &fakeScheduler{}, &fakeSurvival{}, baseTime)
	module.loadErr = assert.AnError

	assert.NoError(t, module.AbandonForDeath(7))
	assert.Zero(t, store.saveCalls)
}
