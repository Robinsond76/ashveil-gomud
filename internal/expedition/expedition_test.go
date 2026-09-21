package expedition

import (
	"errors"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baseTime() time.Time {
	return time.Date(2026, time.September, 17, 12, 0, 0, 0, time.UTC)
}

func validProfile() TravelProfile {
	return TravelProfile{
		Name:     "oak-road",
		Duration: 30 * time.Second,
		Exertion: survival.Exertion{Hunger: 7, Thirst: 11, Fatigue: 13},
	}
}

func validSession() TravelSession {
	return TravelSession{
		LeaderUserID:           7,
		OriginRoomID:           100,
		DestinationRoomID:      200,
		ExitName:               "north",
		ProfileName:            "oak-road",
		StartedAtUTC:           baseTime(),
		LastExertionCheckpoint: 0,
		State:                  Traveling,
	}
}

func validFallenTreeProfile() TravelProfile {
	profile := validProfile()
	profile.Interruption = &InterruptionProfile{Kind: FallenTree, Checkpoint: 5}
	return profile
}

func TestTravelProfileValidate(t *testing.T) {
	cases := []struct {
		name    string
		profile TravelProfile
		wantErr error
	}{
		{"valid", validProfile(), nil},
		{"zero exertion allowed", TravelProfile{Name: "flat", Duration: time.Second}, nil},
		{"valid fallen tree interruption", validFallenTreeProfile(), nil},
		{"valid interruption at last checkpoint", TravelProfile{Name: "oak-road", Duration: 30 * time.Second, Interruption: &InterruptionProfile{Kind: FallenTree, Checkpoint: CheckpointCount - 1}}, nil},
		{"empty name", TravelProfile{Duration: time.Second}, ErrInvalidProfileName},
		{"whitespace name", TravelProfile{Name: "  ", Duration: time.Second}, ErrInvalidProfileName},
		{"zero duration", TravelProfile{Name: "zero"}, ErrInvalidProfileDuration},
		{"negative duration", TravelProfile{Name: "neg", Duration: -time.Second}, ErrInvalidProfileDuration},
		{"negative hunger", TravelProfile{Name: "bad", Duration: time.Second, Exertion: survival.Exertion{Hunger: -1}}, ErrInvalidProfileExertion},
		{"negative thirst", TravelProfile{Name: "bad", Duration: time.Second, Exertion: survival.Exertion{Thirst: -1}}, ErrInvalidProfileExertion},
		{"negative fatigue", TravelProfile{Name: "bad", Duration: time.Second, Exertion: survival.Exertion{Fatigue: -1}}, ErrInvalidProfileExertion},
		{"empty interruption kind", TravelProfile{Name: "oak-road", Duration: 30 * time.Second, Interruption: &InterruptionProfile{Checkpoint: 5}}, ErrInvalidInterruption},
		{"unknown interruption kind", TravelProfile{Name: "oak-road", Duration: 30 * time.Second, Interruption: &InterruptionProfile{Kind: "rock_slide", Checkpoint: 5}}, ErrInvalidInterruption},
		{"interruption checkpoint zero", TravelProfile{Name: "oak-road", Duration: 30 * time.Second, Interruption: &InterruptionProfile{Kind: FallenTree, Checkpoint: 0}}, ErrInvalidInterruption},
		{"interruption checkpoint at route end", TravelProfile{Name: "oak-road", Duration: 30 * time.Second, Interruption: &InterruptionProfile{Kind: FallenTree, Checkpoint: CheckpointCount}}, ErrInvalidInterruption},
		{"interruption checkpoint beyond route end", TravelProfile{Name: "oak-road", Duration: 30 * time.Second, Interruption: &InterruptionProfile{Kind: FallenTree, Checkpoint: CheckpointCount + 1}}, ErrInvalidInterruption},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.profile.Validate()
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestTravelSessionValidate(t *testing.T) {
	require.NoError(t, validSession().Validate())

	bad := validSession()
	bad.LeaderUserID = 0
	require.ErrorIs(t, bad.Validate(), ErrInvalidSession)

	bad = validSession()
	bad.DestinationRoomID = bad.OriginRoomID
	require.ErrorIs(t, bad.Validate(), ErrInvalidSession)

	bad = validSession()
	bad.ProfileName = ""
	require.ErrorIs(t, bad.Validate(), ErrInvalidSession)

	bad = validSession()
	bad.StartedAtUTC = time.Time{}
	require.ErrorIs(t, bad.Validate(), ErrInvalidSession)

	bad = validSession()
	bad.LastExertionCheckpoint = CheckpointCount + 1
	require.ErrorIs(t, bad.Validate(), ErrInvalidSession)

	bad = validSession()
	bad.State = SessionState(99)
	require.ErrorIs(t, bad.Validate(), ErrInvalidSession)

	bad = validSession()
	bad.PausedDuration = -time.Second
	require.ErrorIs(t, bad.Validate(), ErrInvalidSession)

	// An interrupted session must carry both the immutable payload and the
	// instant it paused at.
	bad = validSession()
	bad.State = Interrupted
	bad.PausedAtUTC = baseTime().Add(15 * time.Second)
	bad.InterruptionTriggered = true
	require.ErrorIs(t, bad.Validate(), ErrInvalidSession)

	bad = validSession()
	bad.State = Interrupted
	bad.Interruption = &TravelInterruption{Kind: FallenTree, Checkpoint: 5}
	bad.InterruptionTriggered = true
	require.ErrorIs(t, bad.Validate(), ErrInvalidSession)

	// A payload outside the interrupted state is corrupt persisted data.
	bad = validSession()
	bad.Interruption = &TravelInterruption{Kind: FallenTree, Checkpoint: 5}
	require.ErrorIs(t, bad.Validate(), ErrInvalidSession)

	bad = validSession()
	bad.PausedAtUTC = baseTime().Add(15 * time.Second)
	require.ErrorIs(t, bad.Validate(), ErrInvalidSession)

	// The persisted payload's kind and checkpoint must each be legal.
	bad = validSession()
	bad.State = Interrupted
	bad.PausedAtUTC = baseTime().Add(15 * time.Second)
	bad.InterruptionTriggered = true
	bad.Interruption = &TravelInterruption{Kind: FallenTree, Checkpoint: 0}
	require.ErrorIs(t, bad.Validate(), ErrInvalidSession)

	bad = validSession()
	bad.State = Interrupted
	bad.PausedAtUTC = baseTime().Add(15 * time.Second)
	bad.InterruptionTriggered = true
	bad.Interruption = &TravelInterruption{Kind: FallenTree, Checkpoint: CheckpointCount}
	require.ErrorIs(t, bad.Validate(), ErrInvalidSession)

	bad = validSession()
	bad.State = Interrupted
	bad.PausedAtUTC = baseTime().Add(15 * time.Second)
	bad.InterruptionTriggered = true
	bad.Interruption = &TravelInterruption{Kind: "rock_slide", Checkpoint: 5}
	require.ErrorIs(t, bad.Validate(), ErrInvalidSession)

	bad = validSession()
	bad.State = Interrupted
	bad.PausedAtUTC = baseTime().Add(15 * time.Second)
	bad.InterruptionTriggered = true
	bad.Interruption = &TravelInterruption{Checkpoint: 5}
	require.ErrorIs(t, bad.Validate(), ErrInvalidSession)

	// A completed route may still carry the history of its pause.
	done := validSession()
	done.State = Completed
	done.PausedDuration = time.Hour
	done.InterruptionTriggered = true
	require.NoError(t, done.Validate())
}

func TestInterruptionProfileCheckpointRangeExcludesRouteEnd(t *testing.T) {
	for checkpoint := 0; checkpoint <= CheckpointCount; checkpoint++ {
		profile := validProfile()
		profile.Interruption = &InterruptionProfile{Kind: FallenTree, Checkpoint: uint8(checkpoint)}
		err := profile.Validate()
		if checkpoint >= 1 && checkpoint < CheckpointCount {
			require.NoError(t, err, "checkpoint %d must be valid", checkpoint)
			continue
		}
		require.ErrorIs(t, err, ErrInvalidInterruption, "checkpoint %d must be rejected", checkpoint)
	}
}

func TestProgressAtClampsBeforeStartAndAfterCompletion(t *testing.T) {
	profile := validProfile()
	session := validSession()

	assert.Equal(t, 0.0, session.ProgressAt(baseTime().Add(-time.Minute), profile.Duration))
	assert.Equal(t, 0.0, session.ProgressAt(baseTime(), profile.Duration))
	assert.InDelta(t, 0.5, session.ProgressAt(baseTime().Add(15*time.Second), profile.Duration), 0.0001)
	assert.Equal(t, 1.0, session.ProgressAt(baseTime().Add(30*time.Second), profile.Duration))
	assert.Equal(t, 1.0, session.ProgressAt(baseTime().Add(time.Hour), profile.Duration))
	assert.Equal(t, 1.0, session.ProgressAt(baseTime(), 0))
}

func TestCheckpointAtCoversTenCheckpoints(t *testing.T) {
	profile := validProfile()
	session := validSession()

	assert.Equal(t, uint8(0), session.CheckpointAt(baseTime().Add(-time.Second), profile.Duration))
	assert.Equal(t, uint8(0), session.CheckpointAt(baseTime(), profile.Duration))
	assert.Equal(t, uint8(1), session.CheckpointAt(baseTime().Add(3*time.Second), profile.Duration))
	assert.Equal(t, uint8(5), session.CheckpointAt(baseTime().Add(15*time.Second), profile.Duration))
	assert.Equal(t, uint8(9), session.CheckpointAt(baseTime().Add(29*time.Second), profile.Duration))
	assert.Equal(t, uint8(10), session.CheckpointAt(baseTime().Add(30*time.Second), profile.Duration))
	assert.Equal(t, uint8(10), session.CheckpointAt(baseTime().Add(time.Hour), profile.Duration))

	// Every tenth of the route advances exactly one checkpoint.
	for step := 0; step <= CheckpointCount; step++ {
		now := baseTime().Add(time.Duration(step) * profile.Duration / CheckpointCount)
		assert.Equal(t, uint8(step), session.CheckpointAt(now, profile.Duration), "step %d", step)
	}
}

func TestExertionDueIsIncrementalAndTotalsExactly(t *testing.T) {
	profile := validProfile()
	session := validSession()

	total := survival.Exertion{}
	for step := 1; step <= CheckpointCount; step++ {
		now := baseTime().Add(time.Duration(step) * profile.Duration / CheckpointCount)
		due := session.ExertionDue(now, profile)
		assert.GreaterOrEqual(t, due.Hunger, 0)
		assert.GreaterOrEqual(t, due.Thirst, 0)
		assert.GreaterOrEqual(t, due.Fatigue, 0)
		total.Hunger += due.Hunger
		total.Thirst += due.Thirst
		total.Fatigue += due.Fatigue
		session.LastExertionCheckpoint = session.CheckpointAt(now, profile.Duration)
	}

	assert.Equal(t, profile.Exertion, total)
}

func TestExertionDueAtCompletionEqualsProfileTotal(t *testing.T) {
	profile := validProfile()
	session := validSession()

	due := session.ExertionDue(baseTime().Add(profile.Duration), profile)
	assert.Equal(t, profile.Exertion, due)
}

func TestExertionDueIsZeroWhenNoCheckpointAdvanced(t *testing.T) {
	profile := validProfile()
	session := validSession()
	session.LastExertionCheckpoint = 4

	assert.Equal(t, survival.Exertion{}, session.ExertionDue(baseTime().Add(12*time.Second), profile))
	assert.Equal(t, survival.Exertion{}, session.ExertionDue(baseTime(), profile))
}

func TestExertionDueIsZeroForFlatProfile(t *testing.T) {
	profile := TravelProfile{Name: "flat", Duration: 10 * time.Second}
	session := validSession()

	assert.Equal(t, survival.Exertion{}, session.ExertionDue(baseTime().Add(10*time.Second), profile))
}

func TestSessionTransitions(t *testing.T) {
	completed, err := validSession().Transition(Completed)
	require.NoError(t, err)
	assert.Equal(t, Completed, completed.State)

	interrupted, err := validSession().Transition(Interrupted)
	require.NoError(t, err)
	assert.Equal(t, Interrupted, interrupted.State)

	cancelled, err := validSession().Transition(Cancelled)
	require.NoError(t, err)
	assert.Equal(t, Cancelled, cancelled.State)

	terminal := validSession()
	terminal.State = Completed
	_, err = terminal.Transition(Traveling)
	require.ErrorIs(t, err, ErrInvalidTransition)

	terminal.State = Cancelled
	_, err = terminal.Transition(Completed)
	require.ErrorIs(t, err, ErrInvalidTransition)
}

func TestSessionStateStrings(t *testing.T) {
	assert.Equal(t, "traveling", Traveling.String())
	assert.Equal(t, "interrupted", Interrupted.String())
	assert.Equal(t, "completed", Completed.String())
	assert.Equal(t, "cancelled", Cancelled.String())
	assert.Equal(t, "unknown", SessionState(99).String())
}

func TestPrepareExertionUsesStablePendingOperation(t *testing.T) {
	session := validSession()
	profile := TravelProfile{Name: "oak-road", Duration: 10 * time.Second, Exertion: survival.Exertion{Hunger: 10}}
	now := session.StartedAtUTC.Add(7 * time.Second)

	pending, ok, err := session.PrepareExertion(now, profile)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, uint8(7), pending.Checkpoint)
	assert.Equal(t, survival.Exertion{Hunger: 7}, pending.Cost)
	assert.NotEmpty(t, pending.OperationID)

	repeated, ok, err := session.PrepareExertion(now, profile)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, pending, repeated)
}

func TestActiveElapsedAtExcludesPausedTime(t *testing.T) {
	profile := validFallenTreeProfile()
	session := validSession()

	assert.Equal(t, time.Duration(0), session.ActiveElapsedAt(baseTime(), profile.Duration))
	assert.Equal(t, 15*time.Second, session.ActiveElapsedAt(baseTime().Add(15*time.Second), profile.Duration))
	assert.Equal(t, profile.Duration, session.ActiveElapsedAt(baseTime().Add(time.Hour), profile.Duration))

	interrupted, err := session.Interrupt(baseTime().Add(15*time.Second), profile)
	require.NoError(t, err)
	assert.Equal(t, 15*time.Second, interrupted.ActiveElapsedAt(baseTime().Add(time.Hour), profile.Duration))

	resumed, err := interrupted.Resume(baseTime().Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 15*time.Second, resumed.ActiveElapsedAt(baseTime().Add(time.Hour), profile.Duration))
	assert.Equal(t, 16*time.Second, resumed.ActiveElapsedAt(baseTime().Add(time.Hour+time.Second), profile.Duration))
}

func TestRemainingAtClampsAndSubtractsActiveTime(t *testing.T) {
	profile := validFallenTreeProfile()
	session := validSession()

	assert.Equal(t, profile.Duration, session.RemainingAt(baseTime(), profile.Duration))
	assert.Equal(t, 15*time.Second, session.RemainingAt(baseTime().Add(15*time.Second), profile.Duration))
	assert.Equal(t, time.Duration(0), session.RemainingAt(baseTime().Add(profile.Duration), profile.Duration))
	assert.Equal(t, time.Duration(0), session.RemainingAt(baseTime().Add(time.Hour), profile.Duration))
	assert.Equal(t, time.Duration(0), session.RemainingAt(baseTime(), 0))

	interrupted, err := session.Interrupt(baseTime().Add(15*time.Second), profile)
	require.NoError(t, err)
	assert.Equal(t, 15*time.Second, interrupted.RemainingAt(baseTime().Add(time.Hour), profile.Duration))
}

func TestInterruptionDueRequiresDueConfiguredRoute(t *testing.T) {
	profile := validFallenTreeProfile()
	session := validSession()

	assert.False(t, session.InterruptionDue(baseTime().Add(14*time.Second), profile))
	assert.True(t, session.InterruptionDue(baseTime().Add(15*time.Second), profile))
	assert.True(t, session.InterruptionDue(baseTime().Add(time.Hour), profile))

	// A profile without a configured interruption never fires.
	assert.False(t, session.InterruptionDue(baseTime().Add(time.Hour), validProfile()))

	// A session that already triggered the interruption never fires again.
	triggered := session
	triggered.InterruptionTriggered = true
	assert.False(t, triggered.InterruptionDue(baseTime().Add(time.Hour), profile))
}

func TestInterruptRejectsProfileThatIsNotDue(t *testing.T) {
	profile := validFallenTreeProfile()

	_, err := validSession().Interrupt(baseTime().Add(14*time.Second), profile)
	require.ErrorIs(t, err, ErrInvalidInterruption)

	_, err = validSession().Interrupt(baseTime().Add(15*time.Second), validProfile())
	require.ErrorIs(t, err, ErrInvalidInterruption)

	bad := validFallenTreeProfile()
	bad.Interruption = &InterruptionProfile{Kind: FallenTree, Checkpoint: CheckpointCount}
	_, err = validSession().Interrupt(baseTime().Add(time.Hour), bad)
	require.ErrorIs(t, err, ErrInvalidInterruption)
}

func TestInterruptRecordsDurablePayloadAndPauseInstant(t *testing.T) {
	profile := validFallenTreeProfile()
	session := validSession()
	pauseInstant := baseTime().Add(15 * time.Second)

	interrupted, err := session.Interrupt(pauseInstant, profile)
	require.NoError(t, err)

	assert.Equal(t, Interrupted, interrupted.State)
	assert.Equal(t, pauseInstant, interrupted.PausedAtUTC)
	assert.Equal(t, time.Duration(0), interrupted.PausedDuration)
	assert.True(t, interrupted.InterruptionTriggered)
	require.NotNil(t, interrupted.Interruption)
	assert.Equal(t, TravelInterruption{Kind: FallenTree, Checkpoint: 5}, *interrupted.Interruption)
	require.NoError(t, interrupted.Validate())

	// Transitions return copies: the original session is untouched.
	assert.Equal(t, Traveling, session.State)
	assert.True(t, session.PausedAtUTC.IsZero())
	assert.Nil(t, session.Interruption)
}

func TestInterruptedRoutePausesProgressAndResumesActiveTime(t *testing.T) {
	profile := validFallenTreeProfile()
	session := validSession()

	interrupted, err := session.Interrupt(baseTime().Add(15*time.Second), profile)
	require.NoError(t, err)

	// Progress and exertion stay frozen for as long as the route is interrupted.
	assert.InDelta(t, 0.5, interrupted.ProgressAt(baseTime().Add(time.Hour), profile.Duration), 0.0001)
	assert.Equal(t, uint8(5), interrupted.CheckpointAt(baseTime().Add(time.Hour), profile.Duration))
	assert.Equal(t, survival.Exertion{Hunger: 4, Thirst: 6, Fatigue: 7}, interrupted.ExertionDue(baseTime().Add(time.Hour), profile))

	resumed, err := interrupted.Resume(baseTime().Add(time.Hour))
	require.NoError(t, err)
	require.NoError(t, resumed.Validate())
	assert.Equal(t, Traveling, resumed.State)
	assert.Equal(t, time.Hour-15*time.Second, resumed.PausedDuration)
	assert.True(t, resumed.PausedAtUTC.IsZero())
	assert.Nil(t, resumed.Interruption)
	assert.Equal(t, 15*time.Second, resumed.RemainingAt(baseTime().Add(time.Hour), profile.Duration))

	// Progress tracks active time, not the hour of wall time spent paused.
	assert.InDelta(t, 25.0/30.0, resumed.ProgressAt(baseTime().Add(time.Hour+10*time.Second), profile.Duration), 0.0001)
	// Fifteen more active seconds complete the route, an hour of wall time later.
	assert.Equal(t, 1.0, resumed.ProgressAt(baseTime().Add(time.Hour+15*time.Second), profile.Duration))
	assert.Equal(t, uint8(CheckpointCount), resumed.CheckpointAt(baseTime().Add(time.Hour+15*time.Second), profile.Duration))
	assert.Equal(t, profile.Exertion, resumed.ExertionDue(baseTime().Add(time.Hour+15*time.Second), profile))
}

func TestResumePreservesOneShotInterruptionAndActiveProgress(t *testing.T) {
	profile := validProfile()
	profile.Interruption = &InterruptionProfile{Kind: FallenTree, Checkpoint: 5}
	session := validSession()

	interrupted, err := session.Interrupt(baseTime().Add(15*time.Second), profile)
	require.NoError(t, err)
	resumed, err := interrupted.Resume(baseTime().Add(time.Hour))
	require.NoError(t, err)

	assert.True(t, resumed.InterruptionTriggered)
	assert.False(t, resumed.InterruptionDue(baseTime().Add(time.Hour), profile))
	assert.Equal(t, 15*time.Second, resumed.RemainingAt(baseTime().Add(time.Hour), profile.Duration))
}

func TestResumeRequiresInterruptedSession(t *testing.T) {
	_, err := validSession().Resume(baseTime().Add(time.Hour))
	require.ErrorIs(t, err, ErrInvalidTransition)

	completed := validSession()
	completed.State = Completed
	_, err = completed.Resume(baseTime().Add(time.Hour))
	require.ErrorIs(t, err, ErrInvalidTransition)

	// An interrupted record with no pause instant is corrupt.
	bad := validSession()
	bad.State = Interrupted
	bad.Interruption = &TravelInterruption{Kind: FallenTree, Checkpoint: 5}
	bad.InterruptionTriggered = true
	_, err = bad.Resume(baseTime().Add(time.Hour))
	require.ErrorIs(t, err, ErrInvalidSession)
}

func TestReturnCancelsOnlyInterruptedSessions(t *testing.T) {
	profile := validFallenTreeProfile()
	interrupted, err := validSession().Interrupt(baseTime().Add(15*time.Second), profile)
	require.NoError(t, err)

	returned, err := interrupted.Return()
	require.NoError(t, err)
	assert.Equal(t, Cancelled, returned.State)
	require.NoError(t, returned.Validate())

	_, err = validSession().Return()
	require.ErrorIs(t, err, ErrInvalidTransition)

	completed := validSession()
	completed.State = Completed
	_, err = completed.Return()
	require.ErrorIs(t, err, ErrInvalidTransition)
}

func TestInterruptResumeAndReturnLeaveExertionStateAlone(t *testing.T) {
	profile := validFallenTreeProfile()
	session := validSession()
	session.LastExertionCheckpoint = 3
	pending, ok, err := session.PrepareExertion(baseTime().Add(15*time.Second), profile)
	require.NoError(t, err)
	require.True(t, ok)
	session.PendingExertion = &pending

	interrupted, err := session.Interrupt(baseTime().Add(15*time.Second), profile)
	require.NoError(t, err)
	assert.Equal(t, uint8(3), interrupted.LastExertionCheckpoint)
	require.NotNil(t, interrupted.PendingExertion)
	assert.Equal(t, pending, *interrupted.PendingExertion)

	resumed, err := interrupted.Resume(baseTime().Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, uint8(3), resumed.LastExertionCheckpoint)
	require.NotNil(t, resumed.PendingExertion)
	assert.Equal(t, pending, *resumed.PendingExertion)

	returned, err := interrupted.Return()
	require.NoError(t, err)
	assert.Equal(t, uint8(3), returned.LastExertionCheckpoint)
	require.NotNil(t, returned.PendingExertion)
	assert.Equal(t, pending, *returned.PendingExertion)
}

type fakeStarter struct {
	handled bool
	err     error
	last    StartRequest
	calls   int
}

func (f *fakeStarter) StartTravel(req StartRequest) (bool, error) {
	f.calls++
	f.last = req
	return f.handled, f.err
}

type fakeViewer struct {
	handled bool
	err     error
	last    int
	calls   int
}

func (f *fakeViewer) RenderTravelView(leaderUserID int) (bool, error) {
	f.calls++
	f.last = leaderUserID
	return f.handled, f.err
}

type fakeMovementProvider struct {
	blocked bool
	message string
	last    int
	calls   int
}

func (f *fakeMovementProvider) MovementBlocked(leaderUserID int) (bool, string) {
	f.calls++
	f.last = leaderUserID
	return f.blocked, f.message
}

func TestNilProvidersAreInert(t *testing.T) {
	SetStartProvider(nil)
	SetViewProvider(nil)
	SetMovementProvider(nil)
	t.Cleanup(func() {
		SetStartProvider(nil)
		SetViewProvider(nil)
		SetMovementProvider(nil)
	})

	handled, err := Start(StartRequest{LeaderUserID: 7, ProfileName: "oak-road"})
	require.NoError(t, err)
	assert.False(t, handled)

	viewHandled, err := TravelView(7)
	require.NoError(t, err)
	assert.False(t, viewHandled)

	blocked, message := MovementBlocked(7)
	assert.False(t, blocked)
	assert.Empty(t, message)
}

func TestProvidersAreConsultedAndCleared(t *testing.T) {
	starter := &fakeStarter{handled: true}
	viewer := &fakeViewer{handled: true, err: errors.New("view failed")}
	mover := &fakeMovementProvider{blocked: true, message: "already travelling"}
	SetStartProvider(starter)
	SetViewProvider(viewer)
	SetMovementProvider(mover)
	t.Cleanup(func() {
		SetStartProvider(nil)
		SetViewProvider(nil)
		SetMovementProvider(nil)
	})

	req := StartRequest{LeaderUserID: 7, OriginRoomID: 100, DestinationRoomID: 200, ExitName: "north", ProfileName: "oak-road"}
	handled, err := Start(req)
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Equal(t, 1, starter.calls)
	assert.Equal(t, req, starter.last)

	viewHandled, err := TravelView(7)
	require.Error(t, err)
	assert.True(t, viewHandled)
	assert.Equal(t, 7, viewer.last)

	blocked, message := MovementBlocked(7)
	assert.True(t, blocked)
	assert.Equal(t, "already travelling", message)

	SetStartProvider(nil)
	SetMovementProvider(nil)
	handled, err = Start(req)
	require.NoError(t, err)
	assert.False(t, handled)
	assert.Equal(t, 1, starter.calls, "cleared provider must not be called again")
	blocked, _ = MovementBlocked(7)
	assert.False(t, blocked)
}
