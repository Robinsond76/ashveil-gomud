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

func TestTravelProfileValidate(t *testing.T) {
	cases := []struct {
		name    string
		profile TravelProfile
		wantErr error
	}{
		{"valid", validProfile(), nil},
		{"zero exertion allowed", TravelProfile{Name: "flat", Duration: time.Second}, nil},
		{"empty name", TravelProfile{Duration: time.Second}, ErrInvalidProfileName},
		{"whitespace name", TravelProfile{Name: "  ", Duration: time.Second}, ErrInvalidProfileName},
		{"zero duration", TravelProfile{Name: "zero"}, ErrInvalidProfileDuration},
		{"negative duration", TravelProfile{Name: "neg", Duration: -time.Second}, ErrInvalidProfileDuration},
		{"negative hunger", TravelProfile{Name: "bad", Duration: time.Second, Exertion: survival.Exertion{Hunger: -1}}, ErrInvalidProfileExertion},
		{"negative thirst", TravelProfile{Name: "bad", Duration: time.Second, Exertion: survival.Exertion{Thirst: -1}}, ErrInvalidProfileExertion},
		{"negative fatigue", TravelProfile{Name: "bad", Duration: time.Second, Exertion: survival.Exertion{Fatigue: -1}}, ErrInvalidProfileExertion},
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

func TestNilProvidersAreInert(t *testing.T) {
	SetStartProvider(nil)
	SetViewProvider(nil)
	t.Cleanup(func() {
		SetStartProvider(nil)
		SetViewProvider(nil)
	})

	handled, err := Start(StartRequest{LeaderUserID: 7, ProfileName: "oak-road"})
	require.NoError(t, err)
	assert.False(t, handled)

	viewHandled, err := TravelView(7)
	require.NoError(t, err)
	assert.False(t, viewHandled)
}

func TestProvidersAreConsultedAndCleared(t *testing.T) {
	starter := &fakeStarter{handled: true}
	viewer := &fakeViewer{handled: true, err: errors.New("view failed")}
	SetStartProvider(starter)
	SetViewProvider(viewer)
	t.Cleanup(func() {
		SetStartProvider(nil)
		SetViewProvider(nil)
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

	SetStartProvider(nil)
	handled, err = Start(req)
	require.NoError(t, err)
	assert.False(t, handled)
	assert.Equal(t, 1, starter.calls, "cleared provider must not be called again")
}
