package camping

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestRestSessionRecoveryLegacyDefault(t *testing.T) {
	legacy := RestSession{StartedAtUTC: campTime, State: Resting}
	assert.Equal(t, FatigueRecovery, legacy.RecoveryAmount())
	scaled := RestSession{StartedAtUTC: campTime, State: Resting, Recovery: 15}
	assert.Equal(t, 15, scaled.RecoveryAmount())
	assert.ErrorIs(t, RestSession{StartedAtUTC: campTime, Recovery: -1}.Validate(), ErrInvalidCamp)

	var loaded Camp
	require.NoError(t, yaml.Unmarshal([]byte("leader_user_id: 7\nroom_id: 42\nfire_lit: true\nrest:\n  started_at_utc: 2026-09-22T12:00:00Z\n  state: 0\n"), &loaded))
	require.NoError(t, loaded.Validate())
	assert.Equal(t, FatigueRecovery, loaded.Rest.RecoveryAmount(), "a camp rest saved before Phase 16 restores 20")
}

func TestProgressForMatchesProgressAtAtSixtySeconds(t *testing.T) {
	camp, err := Established(7, 42)
	require.NoError(t, err)
	camp, _ = camp.LightFire()
	camp, err = camp.StartRest(campTime)
	require.NoError(t, err)
	for _, offset := range []time.Duration{-time.Second, 0, 15 * time.Second, 30 * time.Second, RestDuration, 2 * RestDuration} {
		now := campTime.Add(offset)
		assert.Equal(t, camp.ProgressAt(now), camp.Rest.ProgressFor(now, RestDuration), "offset %s", offset)
	}
	assert.Equal(t, 0.25, camp.Rest.ProgressFor(campTime.Add(30*time.Second), 120*time.Second))
	assert.Equal(t, 90*time.Second, camp.Rest.RemainingFor(campTime.Add(30*time.Second), 120*time.Second))
}

func TestInnStayTransitions(t *testing.T) {
	stay, err := StartInnStay(7, 2003, 10, campTime, 60*time.Second, 60)
	require.NoError(t, err)
	assert.True(t, stay.Resting())
	assert.Equal(t, 60, stay.Rest.RecoveryAmount())
	assert.False(t, stay.Due(campTime.Add(59*time.Second)))
	assert.Equal(t, 0.5, stay.ProgressAt(campTime.Add(30*time.Second)))
	assert.Equal(t, 30*time.Second, stay.RemainingAt(campTime.Add(30*time.Second)))

	_, err = stay.Complete(campTime.Add(30 * time.Second))
	assert.ErrorIs(t, err, ErrRestNotDue)

	done, err := stay.Complete(campTime.Add(60 * time.Second))
	require.NoError(t, err)
	assert.False(t, done.Resting())
	assert.Equal(t, 1.0, done.ProgressAt(campTime))
	assert.True(t, stay.Resting(), "Complete returns a copy")

	_, err = done.Complete(campTime.Add(time.Hour))
	assert.True(t, errors.Is(err, ErrRestAlreadyCompleted))
}

func TestInnStayValidation(t *testing.T) {
	cases := []struct {
		name                         string
		leader, room, paid, recovery int
		start                        time.Time
		duration                     time.Duration
	}{
		{"no leader", 0, 2003, 5, 60, campTime, time.Minute},
		{"no room", 7, 0, 5, 60, campTime, time.Minute},
		{"negative paid", 7, 2003, -1, 60, campTime, time.Minute},
		{"no recovery", 7, 2003, 5, 0, campTime, time.Minute},
		{"no start", 7, 2003, 5, 60, time.Time{}, time.Minute},
		{"no duration", 7, 2003, 5, 60, campTime, 0},
	}
	for _, tc := range cases {
		_, err := StartInnStay(tc.leader, tc.room, tc.paid, tc.start, tc.duration, tc.recovery)
		assert.ErrorIs(t, err, ErrInvalidStay, tc.name)
	}
	free, err := StartInnStay(7, 2003, 0, campTime, time.Minute, 60)
	require.NoError(t, err, "a free stay (price 0) is valid")
	assert.Zero(t, free.Paid)
}
